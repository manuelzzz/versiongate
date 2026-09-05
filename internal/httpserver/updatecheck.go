package httpserver

import (
	"errors"
	"net/http"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/release"
	"github.com/manuelzzz/versiongate/internal/updatepolicy"
	"github.com/manuelzzz/versiongate/internal/version"
)

type updateCheckResponse struct {
	Action        string                 `json:"action"`
	LatestRelease *latestReleaseResponse `json:"latest_release,omitempty"`
}

type latestReleaseResponse struct {
	Version     string `json:"version"`
	BuildNumber int    `json:"build_number"`
}

func toUpdateCheckResponse(result updatepolicy.Result) updateCheckResponse {
	resp := updateCheckResponse{Action: string(result.Action)}
	if result.Latest != nil {
		resp.LatestRelease = &latestReleaseResponse{
			Version:     formatVersion(result.Latest.Version),
			BuildNumber: result.Latest.BuildNumber,
		}
	}
	return resp
}

// updateCheckHandler exposes updatepolicy.Evaluate over HTTP, per
// specs/protocols/update-check.md. It is deliberately unauthenticated
// (specs/decisions/authentication.md) — see server.go's routing, where
// this route is the one endpoint not wrapped in RequireToken.
func updateCheckHandler(applications application.Repository, releases release.Repository, projects project.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		identifier := query.Get("application_identifier")
		if identifier == "" {
			WriteError(w, CodeValidationError, "application_identifier is required")
			return
		}

		clientVersion, err := version.Parse(query.Get("version"))
		if err != nil {
			WriteError(w, CodeValidationError, "version must be MAJOR.MINOR.PATCH")
			return
		}

		// Build number is accepted for descriptive symmetry with
		// publishing, but is inert — it never affects the outcome
		// (specs/protocols/update-check.md). Still validated if present:
		// a malformed value in a field the client chose to send is a
		// malformed request, not something to silently ignore.
		if raw := query.Get("build_number"); raw != "" {
			if _, err := version.ParseBuildNumber(raw); err != nil {
				WriteError(w, CodeValidationError, "build_number must be a non-negative integer")
				return
			}
		}

		app, err := applications.GetByIdentifier(r.Context(), identifier)
		if err != nil {
			writeUpdateCheckError(w, err)
			return
		}
		// A deactivated Application (or one whose Project is
		// deactivated) must not produce an evaluation result — treated
		// identically to not found (specs/protocols/update-check.md),
		// since Application identifiers are public and this must never
		// be distinguishable from "no such Application."
		if !app.Active {
			WriteError(w, CodeNotFound, "application not found")
			return
		}

		proj, err := projects.Get(r.Context(), app.ProjectID)
		if err != nil {
			writeUpdateCheckError(w, err)
			return
		}
		if !proj.Active {
			WriteError(w, CodeNotFound, "application not found")
			return
		}

		rels, err := releases.ListByApplication(r.Context(), app.ID)
		if err != nil {
			WriteError(w, CodeInternalError, "internal error")
			return
		}

		result := updatepolicy.Evaluate(clientVersion, rels)
		writeJSON(w, http.StatusOK, toUpdateCheckResponse(result))
	}
}

func writeUpdateCheckError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrNotFound), errors.Is(err, project.ErrNotFound):
		WriteError(w, CodeNotFound, "application not found")
	default:
		// Includes application.ErrIdentifierAmbiguous: an operator data
		// issue, not something the client did wrong.
		WriteError(w, CodeInternalError, "internal error")
	}
}
