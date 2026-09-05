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

// GetByIdentifier looks up an Application by identifier alone, across
// all Projects — see application.Repository.GetByIdentifier's doc
// comment for why, and for ErrIdentifierAmbiguous's meaning. LIMIT 2
// (rather than 1) is deliberate: it lets this method distinguish "found
// exactly one" from "found more than one" in a single query.
func (r *ApplicationRepository) GetByIdentifier(ctx context.Context, identifier string) (application.Application, error) {
	const query = `
		SELECT id, project_id, identifier, display_name, platform, active, created_at, updated_at
		FROM applications
		WHERE identifier = $1
		LIMIT 2`

	rows, err := r.db.QueryContext(ctx, query, identifier)
	if err != nil {
		return application.Application{}, fmt.Errorf("postgres: get application by identifier: %w", err)
	}
	defer rows.Close()

	var matches []application.Application
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return application.Application{}, fmt.Errorf("postgres: get application by identifier: %w", err)
		}
		matches = append(matches, a)
	}
	if err := rows.Err(); err != nil {
		return application.Application{}, fmt.Errorf("postgres: get application by identifier: %w", err)
	}

	switch len(matches) {
	case 0:
		return application.Application{}, application.ErrNotFound
	case 1:
		return matches[0], nil
	default:
		return application.Application{}, application.ErrIdentifierAmbiguous
	}
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows, letting
// scanApplication serve a single-row QueryRowContext result and a
// multi-row QueryContext result (see GetByIdentifier) identically.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanApplication(row rowScanner) (application.Application, error) {
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
