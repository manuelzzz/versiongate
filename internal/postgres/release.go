package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/release"
	"github.com/manuelzzz/versiongate/internal/version"
)

// ReleaseRepository is a PostgreSQL-backed implementation of
// release.Repository. It relies entirely on the database's unique
// constraint on (application_id, major, minor, patch, build_number) to
// detect duplicates — Create makes no separate existence check before
// inserting, so it is safe under concurrent or retried calls.
type ReleaseRepository struct {
	db *sql.DB
}

// NewReleaseRepository builds a ReleaseRepository backed by db.
func NewReleaseRepository(db *sql.DB) *ReleaseRepository {
	return &ReleaseRepository{db: db}
}

func (r *ReleaseRepository) Create(ctx context.Context, applicationID application.ID, v version.Version, buildNumber int, policy release.Policy) (release.Release, error) {
	const query = `
		INSERT INTO releases (application_id, major, minor, patch, build_number, policy)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, major, minor, patch, build_number, policy, created_at`

	rel, err := scanRelease(r.db.QueryRowContext(ctx, query,
		applicationID, v.Major, v.Minor, v.Patch, buildNumber, string(policy)))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgErrUniqueViolation:
				return release.Release{}, release.ErrAlreadyExists
			case pgErrForeignKeyViolation:
				return release.Release{}, release.ErrApplicationNotFound
			}
		}
		return release.Release{}, fmt.Errorf("postgres: create release: %w", err)
	}
	return rel, nil
}

func (r *ReleaseRepository) GetByVersion(ctx context.Context, applicationID application.ID, v version.Version, buildNumber int) (release.Release, error) {
	const query = `
		SELECT id, application_id, major, minor, patch, build_number, policy, created_at
		FROM releases
		WHERE application_id = $1 AND major = $2 AND minor = $3 AND patch = $4 AND build_number = $5`

	rel, err := scanRelease(r.db.QueryRowContext(ctx, query,
		applicationID, v.Major, v.Minor, v.Patch, buildNumber))
	if errors.Is(err, sql.ErrNoRows) {
		return release.Release{}, release.ErrNotFound
	}
	if err != nil {
		return release.Release{}, fmt.Errorf("postgres: get release: %w", err)
	}
	return rel, nil
}

func scanRelease(row *sql.Row) (release.Release, error) {
	var (
		rel    release.Release
		v      version.Version
		policy string
	)
	err := row.Scan(&rel.ID, &rel.ApplicationID, &v.Major, &v.Minor, &v.Patch, &rel.BuildNumber, &policy, &rel.CreatedAt)
	if err != nil {
		return release.Release{}, err
	}
	rel.Version = v
	rel.Policy = release.Policy(policy)
	return rel, nil
}
