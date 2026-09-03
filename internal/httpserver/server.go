package httpserver

import (
	"net/http"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/release"
	"github.com/manuelzzz/versiongate/internal/token"
)

// Dependencies are the repositories New's routes need. Passed
// explicitly (.rules/architecture.md's Explicit dependencies) rather
// than constructed internally, so httpserver never decides how they're
// backed — that's cmd/server's job.
type Dependencies struct {
	Tokens       token.Repository
	Applications application.Repository
	Releases     release.Repository
}

func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /internal/_test-echo", testEchoHandler)

	requireToken := RequireToken(deps.Tokens)
	mux.Handle("POST /applications", requireToken(createApplicationHandler(deps.Applications)))
	mux.Handle("GET /applications/{id}", requireToken(getApplicationHandler(deps.Applications)))
	mux.Handle("POST /applications/{applicationID}/releases",
		requireToken(publishReleaseHandler(deps.Releases, deps.Applications)))

	return mux
}
