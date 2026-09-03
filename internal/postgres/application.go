package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/project"
)

// Postgres error codes this package translates into domain sentinels.
// See https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
)

// ApplicationRepository is a PostgreSQL-backed implementation of
// application.Repository.
type ApplicationRepository struct {
	db *sql.DB
}

// NewApplicationRepository builds an ApplicationRepository backed by db.
func NewApplicationRepository(db *sql.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

func (r *ApplicationRepository) Create(ctx context.Context, projectID project.ID, identifier, displayName string, platform application.Platform) (application.Application, error) {
	const query = `
		INSERT INTO applications (project_id, identifier, display_name, platform)
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, identifier, display_name, platform, active, created_at, updated_at`

	a, err := scanApplication(r.db.QueryRowContext(ctx, query, projectID, identifier, displayName, string(platform)))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgErrUniqueViolation:
				return application.Application{}, application.ErrIdentifierTaken
			case pgErrForeignKeyViolation:
				return application.Application{}, application.ErrProjectNotFound
			}
		}
		return application.Application{}, fmt.Errorf("postgres: create application: %w", err)
	}
	return a, nil
}

func (r *ApplicationRepository) Get(ctx context.Context, projectID project.ID, id application.ID) (application.Application, error) {
	const query = `
		SELECT id, project_id, identifier, display_name, platform, active, created_at, updated_at
		FROM applications
		WHERE id = $1 AND project_id = $2`

	a, err := scanApplication(r.db.QueryRowContext(ctx, query, id, projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return application.Application{}, application.ErrNotFound
	}
	if err != nil {
		return application.Application{}, fmt.Errorf("postgres: get application: %w", err)
	}
	return a, nil
}

func (r *ApplicationRepository) Deactivate(ctx context.Context, projectID project.ID, id application.ID) (application.Application, error) {
	const query = `
		UPDATE applications
		SET active = FALSE, updated_at = now()
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, identifier, display_name, platform, active, created_at, updated_at`

	a, err := scanApplication(r.db.QueryRowContext(ctx, query, id, projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return application.Application{}, application.ErrNotFound
	}
	if err != nil {
		return application.Application{}, fmt.Errorf("postgres: deactivate application: %w", err)
	}
	return a, nil
}

func scanApplication(row *sql.Row) (application.Application, error) {
	var (
		a        application.Application
		platform string
	)
	err := row.Scan(&a.ID, &a.ProjectID, &a.Identifier, &a.DisplayName, &platform, &a.Active, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return application.Application{}, err
	}
	a.Platform = application.Platform(platform)
	return a, nil
}
