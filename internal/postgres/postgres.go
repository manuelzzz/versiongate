// Package postgres wires VersionGate to its PostgreSQL database:
// connection pooling and schema migrations (specs/decisions/database.md).
// It is infrastructure — nothing under internal/version or future
// domain packages may import it (.rules/architecture.md).
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Registers the "pgx" driver with database/sql. VersionGate talks to
	// Postgres through the standard database/sql interface rather than
	// pgx's native API, so this is the only place that needs to know
	// which driver implementation is in use.
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 25
	connMaxLifetime = 5 * time.Minute
	pingTimeout     = 5 * time.Second
)

// Open connects to Postgres using dsn and verifies connectivity before
// returning, so a misconfigured or unreachable database fails clearly
// at startup rather than on the first query.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}

	return db, nil
}
