package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/token"
)

// fakeTokenRepository is an in-memory token.Repository used to test
// RequireToken without a database, per .rules/testing.md. It mirrors
// the fake in internal/token's own tests — kept local here since
// test-only fakes aren't exported across packages.
type fakeTokenRepository struct {
	byHash map[string]token.Token
	byID   map[token.ID]string
	nextID int
}

func newFakeTokenRepository() *fakeTokenRepository {
	return &fakeTokenRepository{byHash: make(map[string]token.Token), byID: make(map[token.ID]string)}
}

func (f *fakeTokenRepository) Create(_ context.Context, projectID project.ID, tokenHash string) (token.Token, error) {
	f.nextID++
	t := token.Token{ID: token.ID(string(rune('a' + f.nextID))), ProjectID: projectID}
	f.byHash[tokenHash] = t
	f.byID[t.ID] = tokenHash
	return t, nil
}

func (f *fakeTokenRepository) GetByHash(_ context.Context, tokenHash string) (token.Token, error) {
	t, ok := f.byHash[tokenHash]
	if !ok {
		return token.Token{}, token.ErrNotFound
	}
	return t, nil
}

func (f *fakeTokenRepository) Revoke(_ context.Context, id token.ID) (token.Token, error) {
	hash := f.byID[id]
	t := f.byHash[hash]
	now := t.CreatedAt
	t.RevokedAt = &now
	f.byHash[hash] = t
	return t, nil
}

// echoProjectID is a protected handler standing in for a future real
// endpoint: it writes back the Project scope RequireToken resolved.
func echoProjectID(w http.ResponseWriter, r *http.Request) {
	id, ok := ProjectIDFromContext(r.Context())
	if !ok {
		WriteError(w, CodeInternalError, "no project in context")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"project_id": string(id)})
}

func TestRequireToken(t *testing.T) {
	repo := newFakeTokenRepository()
	projectA := project.ID("project-a")
	issued, raw, err := token.Issue(context.Background(), repo, projectA)
	if err != nil {
		t.Fatalf("setup: Issue() failed: %v", err)
	}

	handler := RequireToken(repo)(http.HandlerFunc(echoProjectID))

	t.Run("missing Authorization header is unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("malformed Authorization header is unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", raw) // missing "Bearer " prefix
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("unknown token is unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer vg_does-not-exist")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("revoked token is unauthorized", func(t *testing.T) {
		if _, err := token.Revoke(context.Background(), repo, issued.ID); err != nil {
			t.Fatalf("setup: Revoke() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+raw)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid token attaches the resolved Project to the context", func(t *testing.T) {
		_, freshRaw, err := token.Issue(context.Background(), repo, projectA)
		if err != nil {
			t.Fatalf("setup: Issue() failed: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+freshRaw)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})
}

// TestRequireToken_CrossProjectIsolation demonstrates the pattern
// future write endpoints must follow: comparing ProjectIDFromContext
// against a target resource's owning Project, and reporting a mismatch
// as not_found rather than unauthorized (specs/protocols/http.md).
func TestRequireToken_CrossProjectIsolation(t *testing.T) {
	repo := newFakeTokenRepository()
	projectA := project.ID("project-a")
	projectB := project.ID("project-b")

	_, rawA, err := token.Issue(context.Background(), repo, projectA)
	if err != nil {
		t.Fatalf("setup: Issue() for project A failed: %v", err)
	}

	// Stands in for a future handler like "revoke a Release under
	// Application X" — X belongs to projectB here.
	resourceOwner := projectB
	protectedHandler := func(w http.ResponseWriter, r *http.Request) {
		id, _ := ProjectIDFromContext(r.Context())
		if id != resourceOwner {
			WriteError(w, CodeNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	}

	handler := RequireToken(repo)(http.HandlerFunc(protectedHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawA)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d — a token from project A must never authorize access to project B's resource", rec.Code, http.StatusNotFound)
	}
}
