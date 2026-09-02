// Package token implements VersionGate's API Token domain concept:
// issuing, verifying, and revoking Project-scoped bearer credentials
// (specs/decisions/authentication.md). Like internal/project, this
// package has no dependency on any specific storage technology.
package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/manuelzzz/versiongate/internal/project"
)

// ID identifies an API Token record. Like project.ID, it is assigned by
// a Repository at creation time.
type ID string

// tokenPrefix makes issued tokens recognizable (in logs, in config
// files) as VersionGate tokens specifically, the same way GitHub's
// "ghp_"-prefixed tokens are — a small operational nicety, not a
// security property.
const tokenPrefix = "vg_"

// rawByteLength is the amount of cryptographically random data encoded
// into each issued token's secret portion (256 bits).
const rawByteLength = 32

// Token is metadata about an issued API Token. The raw secret value
// itself is never part of this type — see Issue's return value for the
// only place it is ever available.
type Token struct {
	ID        ID
	ProjectID project.ID
	CreatedAt time.Time
	RevokedAt *time.Time // nil means the token is active.
}

// Revoked reports whether the token has been revoked.
func (t Token) Revoked() bool {
	return t.RevokedAt != nil
}

// ErrNotFound is returned by a Repository when no Token matches a given
// hash or ID.
var ErrNotFound = errors.New("token: not found")

// ErrInvalid is returned by Verify when raw does not correspond to any
// active token. It covers both "no such token" and "revoked token" —
// callers cannot distinguish which, by design: telling them apart would
// leak whether a given token value ever existed
// (specs/decisions/authentication.md).
var ErrInvalid = errors.New("token: invalid or revoked")

// Repository persists and retrieves Tokens. Infrastructure provides the
// implementation; only a hash is ever passed to or read from it — the
// raw token value never reaches storage.
type Repository interface {
	Create(ctx context.Context, projectID project.ID, tokenHash string) (Token, error)
	GetByHash(ctx context.Context, tokenHash string) (Token, error)
	Revoke(ctx context.Context, id ID) (Token, error)
}

// Issue generates a new random token for projectID, persists only its
// hash via repo, and returns both the Token record and the raw secret
// value. The raw value is never stored and is only ever available here,
// at creation — callers must show it to the operator immediately, since
// it cannot be recovered afterward.
func Issue(ctx context.Context, repo Repository, projectID project.ID) (Token, string, error) {
	raw, err := generateRaw()
	if err != nil {
		return Token{}, "", fmt.Errorf("token: generate: %w", err)
	}

	t, err := repo.Create(ctx, projectID, hashRaw(raw))
	if err != nil {
		return Token{}, "", err
	}
	return t, raw, nil
}

// Verify resolves raw to its Token record, if raw corresponds to a
// currently active (non-revoked) token. Any other case — no matching
// token, or a matching but revoked one — is reported identically as
// ErrInvalid.
func Verify(ctx context.Context, repo Repository, raw string) (Token, error) {
	t, err := repo.GetByHash(ctx, hashRaw(raw))
	if errors.Is(err, ErrNotFound) {
		return Token{}, ErrInvalid
	}
	if err != nil {
		return Token{}, err
	}
	if t.Revoked() {
		return Token{}, ErrInvalid
	}
	return t, nil
}

// Revoke immediately invalidates the token identified by id. Revoking
// an already-revoked token is not an error — it is idempotent, and does
// not change the original revocation time.
func Revoke(ctx context.Context, repo Repository, id ID) (Token, error) {
	return repo.Revoke(ctx, id)
}

func generateRaw() (string, error) {
	buf := make([]byte, rawByteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashRaw derives the value actually persisted for a token. SHA-256 is
// used rather than a slow password-hashing algorithm (bcrypt, argon2)
// deliberately: those defend against guessing a low-entropy human
// password, but an issued token is already a high-entropy random
// secret — a fast cryptographic hash is the right primitive here, and
// keeps verification cheap on VersionGate's high-volume write path
// (specs/decisions/authentication.md's Token storage security doesn't
// mandate a specific algorithm; this is the implementation detail it
// leaves open).
func hashRaw(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
