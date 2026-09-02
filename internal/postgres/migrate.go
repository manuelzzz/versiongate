package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"github.com/manuelzzz/versiongate/migrations"
)

// provider builds a goose provider reading migrations from the embedded
// migrations.FS, against db.
func provider(db *sql.DB) (*goose.Provider, error) {
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("postgres: build migration provider: %w", err)
	}
	return p, nil
}

// MigrateUp applies all pending migrations.
func MigrateUp(ctx context.Context, db *sql.DB) error {
	p, err := provider(db)
	if err != nil {
		return err
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("postgres: migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back the most recently applied migration.
func MigrateDown(ctx context.Context, db *sql.DB) error {
	p, err := provider(db)
	if err != nil {
		return err
	}
	if _, err := p.Down(ctx); err != nil {
		return fmt.Errorf("postgres: migrate down: %w", err)
	}
	return nil
}

// MigrationStatus returns the applied/pending status of every known
// migration, in order.
func MigrationStatus(ctx context.Context, db *sql.DB) ([]*goose.MigrationStatus, error) {
	p, err := provider(db)
	if err != nil {
		return nil, err
	}
	status, err := p.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: migration status: %w", err)
	}
	return status, nil
}
