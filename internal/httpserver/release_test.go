package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/release"
	"github.com/manuelzzz/versiongate/internal/version"
)

// fakeReleaseRepository is an in-memory release.Repository used to
// test the HTTP layer without a database, per .rules/testing.md.
type fakeReleaseRepository struct {
	byKey  map[string]release.Release
	nextID int
}

func newFakeReleaseRepository() *fakeReleaseRepository {
	return &fakeReleaseRepository{byKey: make(map[string]release.Release)}
}

func releaseTestKey(applicationID application.ID, v version.Version, buildNumber int) string {
	return string(applicationID) + "|" + formatVersion(v) + "|" + itoa(buildNumber)
}

func (f *fakeReleaseRepository) Create(_ context.Context, applicationID application.ID, v version.Version, buildNumber int, policy release.Policy) (release.Release, error) {
	key := releaseTestKey(applicationID, v, buildNumber)
	if _, exists := f.byKey[key]; exists {
		return release.Release{}, release.ErrAlreadyExists
	}
	f.nextID++
	r := release.Release{
		ID:            release.ID(itoa(f.nextID)),
		ApplicationID: applicationID,
		Version:       v,
		BuildNumber:   buildNumber,
		Policy:        policy,
	}
	f.byKey[key] = r
	return r, nil
}

func (f *fakeReleaseRepository) GetByVersion(_ context.Context, applicationID application.ID, v version.Version, buildNumber int) (release.Release, error) {
	r, ok := f.byKey[releaseTestKey(applicationID, v, buildNumber)]
	if !ok {
		return release.Release{}, release.ErrNotFound
	}
	return r, nil
}

func (f *fakeReleaseRepository) ListByApplication(_ context.Context, applicationID application.ID) ([]release.Release, error) {
	var releases []release.Release
	for _, r := range f.byKey {
		if r.ApplicationID == applicationID {
			releases = append(releases, r)
		}
	}
	return releases, nil
}

func TestPublishRelease(t *testing.T) {
	newServer := func() (http.Handler, *fakeTokenRepository, *fakeApplicationRepository, *fakeReleaseRepository) {
		tokens := newFakeTokenRepository()
		apps := newFakeApplicationRepository()
		releases := newFakeReleaseRepository()
		srv := New(Dependencies{Tokens: tokens, Applications: apps, Releases: releases})
		return srv, tokens, apps, releases
	}

	publishBody := func(v, policy string, build int) []byte {
		b, _ := json.Marshal(publishReleaseRequest{Version: v, BuildNumber: build, Policy: policy})
		return b
	}

	t.Run("requires a valid Project token", func(t *testing.T) {
		srv, _, _, _ := newServer()
		req := httptest.NewRequest(http.MethodPost, "/applications/x/releases", bytes.NewReader(publishBody("1.0.0", "optional", 1)))
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("publishes a fresh Release as 201", func(t *testing.T) {
		srv, tokens, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, project.ID("project-a"), "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost,
			"/applications/"+string(a.ID)+"/releases", publishBody("1.0.0", "optional", 42))
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var got releaseResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if got.Version != "1.0.0" || got.BuildNumber != 42 || got.Policy != "optional" {
			t.Fatalf("unexpected response body: %+v", got)
		}
	})

	t.Run("an identical retry is a 200 no-op, not a 201", func(t *testing.T) {
		srv, tokens, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, project.ID("project-a"), "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}
		body := publishBody("1.0.0", "optional", 42)

		first := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost, "/applications/"+string(a.ID)+"/releases", body)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, first)
		if rec.Code != http.StatusCreated {
			t.Fatalf("setup: first publish status = %d, want %d", rec.Code, http.StatusCreated)
		}

		second := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost, "/applications/"+string(a.ID)+"/releases", body)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, second)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (idempotent no-op)", rec.Code, http.StatusOK)
		}
	})

	t.Run("a same-version-and-build different-policy request is a conflict", func(t *testing.T) {
		srv, tokens, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, project.ID("project-a"), "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		first := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost,
			"/applications/"+string(a.ID)+"/releases", publishBody("1.0.0", "optional", 42))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, first)
		if rec.Code != http.StatusCreated {
			t.Fatalf("setup: first publish status = %d, want %d", rec.Code, http.StatusCreated)
		}

		second := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost,
			"/applications/"+string(a.ID)+"/releases", publishBody("1.0.0", "required", 42))
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, second)

		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
		}
		var got errorEnvelope
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if got.Error.Code != CodeConflict {
			t.Fatalf("Error.Code = %q, want %q", got.Error.Code, CodeConflict)
		}
	})

	t.Run("a malformed version is a validation error", func(t *testing.T) {
		srv, tokens, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, project.ID("project-a"), "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost,
			"/applications/"+string(a.ID)+"/releases", publishBody("1.10", "optional", 1))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("an invalid policy is a validation error", func(t *testing.T) {
		srv, tokens, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, project.ID("project-a"), "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost,
			"/applications/"+string(a.ID)+"/releases", publishBody("1.0.0", "mandatory", 1))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("a token from a different Project cannot publish to this Application", func(t *testing.T) {
		srv, tokens, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, project.ID("project-a"), "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := newAuthedRequest(t, tokens, project.ID("project-b"), http.MethodPost,
			"/applications/"+string(a.ID)+"/releases", publishBody("1.0.0", "optional", 1))
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (validation_error, per release-publishing.md's unknown-Application rule)", rec.Code, http.StatusBadRequest)
		}
	})
}
