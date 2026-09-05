package httpserver

import (
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

// fakeProjectRepository is an in-memory project.Repository used to
// test the HTTP layer without a database, per .rules/testing.md.
type fakeProjectRepository struct {
	byID map[project.ID]project.Project
}

func newFakeProjectRepository() *fakeProjectRepository {
	return &fakeProjectRepository{byID: make(map[project.ID]project.Project)}
}

func (f *fakeProjectRepository) put(p project.Project) {
	f.byID[p.ID] = p
}

func (f *fakeProjectRepository) Create(context.Context, string) (project.Project, error) {
	panic("not used by these tests")
}

func (f *fakeProjectRepository) Get(_ context.Context, id project.ID) (project.Project, error) {
	p, ok := f.byID[id]
	if !ok {
		return project.Project{}, project.ErrNotFound
	}
	return p, nil
}

func (f *fakeProjectRepository) Deactivate(_ context.Context, id project.ID) (project.Project, error) {
	p, ok := f.byID[id]
	if !ok {
		return project.Project{}, project.ErrNotFound
	}
	p.Active = false
	f.byID[id] = p
	return p, nil
}

func TestUpdateCheck(t *testing.T) {
	const activeProjectID = project.ID("project-active")
	const inactiveProjectID = project.ID("project-inactive")

	newServer := func() (http.Handler, *fakeApplicationRepository, *fakeReleaseRepository) {
		projects := newFakeProjectRepository()
		projects.put(project.Project{ID: activeProjectID, Name: "Active Project", Active: true})
		projects.put(project.Project{ID: inactiveProjectID, Name: "Inactive Project", Active: false})

		apps := newFakeApplicationRepository()
		releases := newFakeReleaseRepository()
		srv := New(Dependencies{Projects: projects, Applications: apps, Releases: releases})
		return srv, apps, releases
	}

	t.Run("requires no auth header at all", func(t *testing.T) {
		srv, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, activeProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=1.0.0", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("no releases yet resolves to continue", func(t *testing.T) {
		srv, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, activeProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=1.0.0", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		var got updateCheckResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if got.Action != "continue" {
			t.Fatalf("Action = %q, want continue", got.Action)
		}
		if got.LatestRelease != nil {
			t.Fatalf("LatestRelease = %+v, want nil", got.LatestRelease)
		}
	})

	t.Run("required release behind the client resolves to required, with the latest release reported", func(t *testing.T) {
		srv, apps, releases := newServer()
		a, err := application.Create(context.Background(), apps, activeProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}
		v110, _ := version.Parse("1.1.0")
		v130, _ := version.Parse("1.3.0")
		if _, err := releases.Create(context.Background(), a.ID, v110, 1, release.PolicyRequired); err != nil {
			t.Fatalf("setup: releases.Create() failed: %v", err)
		}
		if _, err := releases.Create(context.Background(), a.ID, v130, 1, release.PolicyOptional); err != nil {
			t.Fatalf("setup: releases.Create() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=1.0.0", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		var got updateCheckResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if got.Action != "required" {
			t.Fatalf("Action = %q, want required", got.Action)
		}
		if got.LatestRelease == nil || got.LatestRelease.Version != "1.3.0" {
			t.Fatalf("LatestRelease = %+v, want version 1.3.0 (the overall latest, not the triggering release)", got.LatestRelease)
		}
	})

	t.Run("build_number is accepted and inert", func(t *testing.T) {
		srv, apps, releases := newServer()
		a, err := application.Create(context.Background(), apps, activeProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}
		v := version.Version{Major: 1, Minor: 1, Patch: 0}
		if _, err := releases.Create(context.Background(), a.ID, v, 999, release.PolicyOptional); err != nil {
			t.Fatalf("setup: releases.Create() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=1.0.0&build_number=1", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("malformed client version is a validation error, never defaulted", func(t *testing.T) {
		srv, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, activeProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=not-a-version", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
		var got errorEnvelope
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if got.Error.Code != CodeValidationError {
			t.Fatalf("Error.Code = %q, want %q", got.Error.Code, CodeValidationError)
		}
	})

	t.Run("malformed build_number is a validation error", func(t *testing.T) {
		srv, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, activeProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=1.0.0&build_number=-1", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("unknown Application is not_found, never continue", func(t *testing.T) {
		srv, _, _ := newServer()

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier=does-not-exist&version=1.0.0", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
		var got errorEnvelope
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if got.Error.Code != CodeNotFound {
			t.Fatalf("Error.Code = %q, want %q", got.Error.Code, CodeNotFound)
		}
	})

	t.Run("deactivated Application is not_found, never continue", func(t *testing.T) {
		srv, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, activeProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}
		if _, err := application.Deactivate(context.Background(), apps, activeProjectID, a.ID); err != nil {
			t.Fatalf("setup: application.Deactivate() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=1.0.0", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("Application under a deactivated Project is not_found, never continue", func(t *testing.T) {
		srv, apps, _ := newServer()
		a, err := application.Create(context.Background(), apps, inactiveProjectID, "acme-ios", "Acme iOS", application.PlatformIOS)
		if err != nil {
			t.Fatalf("setup: application.Create() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/update-check?application_identifier="+a.Identifier+"&version=1.0.0", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("missing application_identifier is a validation error", func(t *testing.T) {
		srv, _, _ := newServer()

		req := httptest.NewRequest(http.MethodGet, "/update-check?version=1.0.0", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
