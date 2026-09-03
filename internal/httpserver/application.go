package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/manuelzzz/versiongate/internal/application"
)

// createApplicationRequest is the request body for POST /applications.
// The owning Project is never a client-supplied field — it is always
// the Project resolved from the caller's token (ProjectIDFromContext),
// per specs/decisions/authentication.md: a token only ever acts within
// its own Project's scope.
type createApplicationRequest struct {
	Identifier  string `json:"identifier"`
	DisplayName string `json:"display_name"`
	Platform    string `json:"platform"`
}

type applicationResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Identifier  string    `json:"identifier"`
	DisplayName string    `json:"display_name"`
	Platform    string    `json:"platform"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toApplicationResponse(a application.Application) applicationResponse {
	return applicationResponse{
		ID:          string(a.ID),
		ProjectID:   string(a.ProjectID),
		Identifier:  a.Identifier,
		DisplayName: a.DisplayName,
		Platform:    string(a.Platform),
		Active:      a.Active,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

func createApplicationHandler(repo application.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, ok := ProjectIDFromContext(r.Context())
		if !ok {
			WriteError(w, CodeInternalError, "missing project scope")
			return
		}

		var body createApplicationRequest
		if !DecodeJSON(w, r, &body) {
			return
		}

		a, err := application.Create(r.Context(), repo, projectID, body.Identifier, body.DisplayName, application.Platform(body.Platform))
		if err != nil {
			writeApplicationError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, toApplicationResponse(a))
	}
}

func getApplicationHandler(repo application.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, ok := ProjectIDFromContext(r.Context())
		if !ok {
			WriteError(w, CodeInternalError, "missing project scope")
			return
		}

		id := application.ID(r.PathValue("id"))

		a, err := repo.Get(r.Context(), projectID, id)
		if err != nil {
			writeApplicationError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, toApplicationResponse(a))
	}
}

// writeApplicationError maps a domain error from internal/application
// to the shared HTTP error envelope, per specs/protocols/http.md.
func writeApplicationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrIdentifierRequired),
		errors.Is(err, application.ErrDisplayNameRequired),
		errors.Is(err, application.ErrInvalidPlatform):
		WriteError(w, CodeValidationError, err.Error())
	case errors.Is(err, application.ErrIdentifierTaken):
		WriteError(w, CodeConflict, err.Error())
	case errors.Is(err, application.ErrNotFound), errors.Is(err, application.ErrProjectNotFound):
		// A missing owning Project can only happen if the authenticated
		// token somehow outlived its Project; treated identically to
		// "Application not found" — never distinguished for the caller.
		WriteError(w, CodeNotFound, "application not found")
	default:
		WriteError(w, CodeInternalError, "internal error")
	}
}
