package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/token"
)

// TokenRepository is a PostgreSQL-backed implementation of
// token.Repository. It only ever reads or writes token_hash — the raw
// token value never reaches this package.
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository builds a TokenRepository backed by db.
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Create(ctx context.Context, projectID project.ID, tokenHash string) (token.Token, error) {
	const query = `
		INSERT INTO api_tokens (project_id, token_hash)
		VALUES ($1, $2)
		RETURNING id, project_id, created_at, revoked_at`

	return scanToken(r.db.QueryRowContext(ctx, query, projectID, tokenHash), "create token")
}

func (r *TokenRepository) GetByHash(ctx context.Context, tokenHash string) (token.Token, error) {
	const query = `
		SELECT id, project_id, created_at, revoked_at
		FROM api_tokens
		WHERE token_hash = $1`

	return scanToken(r.db.QueryRowContext(ctx, query, tokenHash), "get token")
}

func (r *TokenRepository) Revoke(ctx context.Context, id token.ID) (token.Token, error) {
	// COALESCE keeps the original revocation time if the token was
	// already revoked, so a repeated Revoke call is a true no-op rather
	// than moving the audit timestamp forward.
	const query = `
		UPDATE api_tokens
		SET revoked_at = COALESCE(revoked_at, now())
		WHERE id = $1
		RETURNING id, project_id, created_at, revoked_at`

	return scanToken(r.db.QueryRowContext(ctx, query, id), "revoke token")
}

func scanToken(row *sql.Row, op string) (token.Token, error) {
	var (
		t         token.Token
		revokedAt sql.NullTime
	)
	err := row.Scan(&t.ID, &t.ProjectID, &t.CreatedAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return token.Token{}, token.ErrNotFound
	}
	if err != nil {
		return token.Token{}, fmt.Errorf("postgres: %s: %w", op, err)
	}
	if revokedAt.Valid {
		t.RevokedAt = &revokedAt.Time
	}
	return t, nil
}
