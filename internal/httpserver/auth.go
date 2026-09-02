package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/token"
)

// contextKey is an unexported type so keys set by this package can
// never collide with a context key set by another package.
type contextKey int

const projectIDContextKey contextKey = iota

// ProjectIDFromContext returns the Project scope RequireToken resolved
// for the current request, if any. Endpoint handlers that need to
// enforce cross-Project isolation on a specific resource compare this
// against that resource's owning Project — see RequireToken's doc
// comment for why that comparison belongs at the handler, not here.
func ProjectIDFromContext(ctx context.Context) (project.ID, bool) {
	id, ok := ctx.Value(projectIDContextKey).(project.ID)
	return id, ok
}

// RequireToken returns middleware enforcing a valid, Project-scoped
// bearer token (specs/decisions/authentication.md) on every request it
// wraps, attaching the resolved Project scope to the request context on
// success.
//
// Missing, malformed, invalid, and revoked tokens are all rejected
// identically with unauthorized — a response never reveals which of
// those applied, matching token.Verify's contract.
//
// This middleware only establishes *who* is asking (which Project a
// token belongs to). It does not know which resource a given endpoint
// is about, so it cannot by itself enforce "a token from Project X can
// never authorize a request scoped to Project Y" — that comparison
// (this Project's ID vs. the target resource's owning Project) is each
// endpoint handler's responsibility, using ProjectIDFromContext. Per
// specs/protocols/http.md, a mismatch there must be reported as
// not_found, not unauthorized, so a cross-Project access attempt is
// indistinguishable from the resource genuinely not existing.
func RequireToken(repo token.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r)
			if !ok {
				WriteError(w, CodeUnauthorized, "missing or malformed Authorization header")
				return
			}

			t, err := token.Verify(r.Context(), repo, raw)
			if err != nil {
				WriteError(w, CodeUnauthorized, "invalid or revoked token")
				return
			}

			ctx := context.WithValue(r.Context(), projectIDContextKey, t.ProjectID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "

	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}

	raw := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if raw == "" {
		return "", false
	}
	return raw, true
}
