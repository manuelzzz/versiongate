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
	"github.com/manuelzzz/versiongate/internal/token"
)

// fakeApplicationRepository is an in-memory application.Repository used
// to test the HTTP layer without a database, per .rules/testing.md.
type fakeApplicationRepository struct {
	byKey  map[[2]string]application.Application
	byID   map[application.ID]application.Application
	nextID int
}

func newFakeApplicationRepository() *fakeApplicationRepository {
	return &fakeApplicationRepository{
		byKey: make(map[[2]string]application.Application),
		byID:  make(map[application.ID]application.Application),
	}
}

func (f *fakeApplicationRepository) Create(_ context.Context, projectID project.ID, identifier, displayName string, platform application.Platform) (application.Application, error) {
	key := [2]string{string(projectID), identifier}
	if _, exists := f.byKey[key]; exists {
		return application.Application{}, application.ErrIdentifierTaken
	}
	f.nextID++
	a := application.Application{
		ID:          application.ID(string(rune('a' + f.nextID))),
		ProjectID:   projectID,
		Identifier:  identifier,
		DisplayName: displayName,
		Platform:    platform,
		Active:      true,
	}
	f.byKey[key] = a
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeApplicationRepository) Get(_ context.Context, projectID project.ID, id application.ID) (application.Application, error) {
	a, ok := f.byID[id]
	if !ok || a.ProjectID != projectID {
		return application.Application{}, application.ErrNotFound
	}
	return a, nil
}

func (f *fakeApplicationRepository) Deactivate(_ context.Context, projectID project.ID, id application.ID) (application.Application, error) {
	a, ok := f.byID[id]
	if !ok || a.ProjectID != projectID {
		return application.Application{}, application.ErrNotFound
	}
	a.Active = false
	f.byID[id] = a
	return a, nil
}

// newAuthedRequest builds a request carrying a valid bearer token
// scoped to projectID, backed by tokens.
func newAuthedRequest(t *testing.T, tokens token.Repository, projectID project.ID, method, target string, body []byte) *http.Request {
	t.Helper()

	_, raw, err := token.Issue(context.Background(), tokens, projectID)
	if err != nil {
		t.Fatalf("setup: token.Issue() failed: %v", err)
	}

	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCreateApplication(t *testing.T) {
	newServer := func() (http.Handler, *fakeTokenRepository, *fakeApplicationRepository) {
		tokens := newFakeTokenRepository()
		apps := newFakeApplicationRepository()
		srv := New(Dependencies{Tokens: tokens, Applications: apps})
		return srv, tokens, apps
	}

	t.Run("requires a valid Project token", func(t *testing.T) {
		srv, _, _ := newServer()
		req := httptest.NewRequest(http.MethodPost, "/applications", bytes.NewReader(nil))
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("creates an Application scoped to the authenticated Project", func(t *testing.T) {
		srv, tokens, _ := newServer()
		body, _ := json.Marshal(createApplicationRequest{
			Identifier:  "acme-ios",
			DisplayName: "Acme iOS",
			Platform:    "ios",
		})
		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost, "/applications", body)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}

		var got applicationResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if got.Identifier != "acme-ios" || got.Platform != "ios" || !got.Active {
			t.Fatalf("unexpected response body: %+v", got)
		}
		if got.ProjectID != "project-a" {
			t.Fatalf("ProjectID = %q, want project-a", got.ProjectID)
		}
	})

	t.Run("validation failures use the shared error envelope", func(t *testing.T) {
		srv, tokens, _ := newServer()
		body, _ := json.Marshal(createApplicationRequest{
			Identifier:  "",
			DisplayName: "Acme iOS",
			Platform:    "ios",
		})
		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost, "/applications", body)
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

	t.Run("an invalid platform is a validation error", func(t *testing.T) {
		srv, tokens, _ := newServer()
		body, _ := json.Marshal(createApplicationRequest{
			Identifier:  "acme-web",
			DisplayName: "Acme Web",
			Platform:    "web",
		})
		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost, "/applications", body)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("a duplicate identifier within the same Project is a conflict", func(t *testing.T) {
		srv, tokens, _ := newServer()
		body, _ := json.Marshal(createApplicationRequest{Identifier: "acme-ios", DisplayName: "Acme iOS", Platform: "ios"})

		first := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost, "/applications", body)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, first)
		if rec.Code != http.StatusCreated {
			t.Fatalf("setup: first create status = %d, want %d", rec.Code, http.StatusCreated)
		}

		second := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodPost, "/applications", body)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, second)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
		}
	})
}

func TestGetApplication(t *testing.T) {
	tokens := newFakeTokenRepository()
	apps := newFakeApplicationRepository()
	srv := New(Dependencies{Tokens: tokens, Applications: apps})

	created, err := application.Create(context.Background(), apps, project.ID("project-a"), "acme-ios", "Acme iOS", application.PlatformIOS)
	if err != nil {
		t.Fatalf("setup: application.Create() failed: %v", err)
	}

	t.Run("returns the Application for its owning Project", func(t *testing.T) {
		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodGet, "/applications/"+string(created.ID), nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("a token from a different Project gets not_found, not unauthorized", func(t *testing.T) {
		req := newAuthedRequest(t, tokens, project.ID("project-b"), http.MethodGet, "/applications/"+string(created.ID), nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d — a token from a different Project must never see this Application", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("an unknown ID is not_found", func(t *testing.T) {
		req := newAuthedRequest(t, tokens, project.ID("project-a"), http.MethodGet, "/applications/does-not-exist", nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("requires a valid Project token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/applications/"+string(created.ID), nil)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}
