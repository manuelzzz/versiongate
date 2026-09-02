package token

import (
	"context"
	"testing"
	"time"

	"github.com/manuelzzz/versiongate/internal/project"
)

// fakeRepository is an in-memory Repository used to test the domain
// logic in this package without a database, per .rules/testing.md.
type fakeRepository struct {
	byHash map[string]Token
	byID   map[ID]string // ID -> hash, so Revoke can look a token up by ID
	nextID int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byHash: make(map[string]Token),
		byID:   make(map[ID]string),
	}
}

func (f *fakeRepository) Create(_ context.Context, projectID project.ID, tokenHash string) (Token, error) {
	f.nextID++
	t := Token{
		ID:        ID(string(rune('a' + f.nextID))),
		ProjectID: projectID,
		CreatedAt: time.Now(),
	}
	f.byHash[tokenHash] = t
	f.byID[t.ID] = tokenHash
	return t, nil
}

func (f *fakeRepository) GetByHash(_ context.Context, tokenHash string) (Token, error) {
	t, ok := f.byHash[tokenHash]
	if !ok {
		return Token{}, ErrNotFound
	}
	return t, nil
}

func (f *fakeRepository) Revoke(_ context.Context, id ID) (Token, error) {
	hash, ok := f.byID[id]
	if !ok {
		return Token{}, ErrNotFound
	}
	t := f.byHash[hash]
	if t.RevokedAt == nil {
		now := time.Now()
		t.RevokedAt = &now
	}
	f.byHash[hash] = t
	return t, nil
}

func TestIssueAndVerify(t *testing.T) {
	repo := newFakeRepository()
	acme := project.ID("acme")

	issued, raw, err := Issue(context.Background(), repo, acme)
	if err != nil {
		t.Fatalf("Issue() returned unexpected error: %v", err)
	}
	if raw == "" {
		t.Fatal("Issue() returned an empty raw token")
	}

	t.Run("only a hash reaches the repository, never the raw value", func(t *testing.T) {
		if _, ok := repo.byHash[raw]; ok {
			t.Fatal("repository has an entry keyed by the raw token value")
		}
		storedHash, ok := repo.byID[issued.ID]
		if !ok {
			t.Fatal("repository has no entry for the issued token's ID")
		}
		if storedHash == raw {
			t.Fatal("stored hash equals the raw token value")
		}
		if storedHash != hashRaw(raw) {
			t.Fatal("stored hash does not match SHA-256(raw) — cannot verify what was actually persisted")
		}
	})

	t.Run("verifying the raw value resolves the correct token and Project", func(t *testing.T) {
		got, err := Verify(context.Background(), repo, raw)
		if err != nil {
			t.Fatalf("Verify() returned unexpected error: %v", err)
		}
		if got.ID != issued.ID {
			t.Fatalf("ID = %v, want %v", got.ID, issued.ID)
		}
		if got.ProjectID != acme {
			t.Fatalf("ProjectID = %v, want %v", got.ProjectID, acme)
		}
	})

	t.Run("verifying a wrong value fails", func(t *testing.T) {
		_, err := Verify(context.Background(), repo, "vg_not-the-right-token")
		if err != ErrInvalid {
			t.Fatalf("err = %v, want ErrInvalid", err)
		}
	})
}

func TestRevoke(t *testing.T) {
	repo := newFakeRepository()
	issued, raw, err := Issue(context.Background(), repo, project.ID("acme"))
	if err != nil {
		t.Fatalf("setup: Issue() failed: %v", err)
	}

	t.Run("revoked tokens fail verification immediately", func(t *testing.T) {
		if _, err := Revoke(context.Background(), repo, issued.ID); err != nil {
			t.Fatalf("Revoke() returned unexpected error: %v", err)
		}

		_, err := Verify(context.Background(), repo, raw)
		if err != ErrInvalid {
			t.Fatalf("Verify() after revoke: err = %v, want ErrInvalid", err)
		}
	})

	t.Run("revoking twice is idempotent, not an error", func(t *testing.T) {
		if _, err := Revoke(context.Background(), repo, issued.ID); err != nil {
			t.Fatalf("second Revoke() returned unexpected error: %v", err)
		}
	})
}

func TestCrossProjectIsolation(t *testing.T) {
	repo := newFakeRepository()
	projectA := project.ID("project-a")
	projectB := project.ID("project-b")

	_, rawA, err := Issue(context.Background(), repo, projectA)
	if err != nil {
		t.Fatalf("setup: Issue() for project A failed: %v", err)
	}
	_, _, err = Issue(context.Background(), repo, projectB)
	if err != nil {
		t.Fatalf("setup: Issue() for project B failed: %v", err)
	}

	got, err := Verify(context.Background(), repo, rawA)
	if err != nil {
		t.Fatalf("Verify() returned unexpected error: %v", err)
	}
	if got.ProjectID != projectA {
		t.Fatalf("token issued for project A resolved to ProjectID %v, want %v", got.ProjectID, projectA)
	}
	if got.ProjectID == projectB {
		t.Fatal("token issued for project A resolved to project B")
	}
}
