package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/release"
	"github.com/manuelzzz/versiongate/internal/version"
)

// publishReleaseRequest is the request body for
// POST /applications/{applicationID}/releases, per
// specs/protocols/release-publishing.md's Release metadata. The target
// Application comes from the URL path; the owning Project comes from
// the caller's token — neither is a client-supplied body field.
type publishReleaseRequest struct {
	Version     string `json:"version"`
	BuildNumber int    `json:"build_number"`
	Policy      string `json:"policy"`
}

type releaseResponse struct {
	ID            string    `json:"id"`
	ApplicationID string    `json:"application_id"`
	Version       string    `json:"version"`
	BuildNumber   int       `json:"build_number"`
	Policy        string    `json:"policy"`
	CreatedAt     time.Time `json:"created_at"`
}

func toReleaseResponse(r release.Release) releaseResponse {
	return releaseResponse{
		ID:            string(r.ID),
		ApplicationID: string(r.ApplicationID),
		Version:       formatVersion(r.Version),
		BuildNumber:   r.BuildNumber,
		Policy:        string(r.Policy),
		CreatedAt:     r.CreatedAt,
	}
}

func formatVersion(v version.Version) string {
	return itoa(v.Major) + "." + itoa(v.Minor) + "." + itoa(v.Patch)
}

func itoa(n int) string {
	// Local, dependency-free int->string for the three small,
	// non-negative version components — not worth pulling in strconv
	// for.
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// publishReleaseHandler exposes release.Publish over HTTP, per
// specs/protocols/release-publishing.md and specs/protocols/http.md.
func publishReleaseHandler(releases release.Repository, applications application.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, ok := ProjectIDFromContext(r.Context())
		if !ok {
			WriteError(w, CodeInternalError, "missing project scope")
			return
		}

		applicationID := application.ID(r.PathValue("applicationID"))

		var body publishReleaseRequest
		if !DecodeJSON(w, r, &body) {
			return
		}

		v, err := version.Parse(body.Version)
		if err != nil {
			WriteError(w, CodeValidationError, "version must be MAJOR.MINOR.PATCH")
			return
		}

		rel, created, err := release.Publish(
			r.Context(), releases, applications,
			projectID, applicationID,
			v, body.BuildNumber, release.Policy(body.Policy),
		)
		if err != nil {
			writeReleaseError(w, err)
			return
		}

		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(w, status, toReleaseResponse(rel))
	}
}

func writeReleaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, release.ErrInvalidBuildNumber),
		errors.Is(err, release.ErrInvalidPolicy),
		errors.Is(err, release.ErrApplicationNotFound),
		errors.Is(err, release.ErrApplicationInactive):
		// Per release-publishing.md, an unknown or inactive Application
		// is a validation failure for this endpoint specifically — not
		// not_found, unlike a direct Application lookup (#27).
		WriteError(w, CodeValidationError, err.Error())
	case errors.Is(err, release.ErrConflict):
		WriteError(w, CodeConflict, err.Error())
	default:
		WriteError(w, CodeInternalError, "internal error")
	}
}
