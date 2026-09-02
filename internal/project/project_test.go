package project

import (
	"context"
	"testing"
	"time"
)

// fakeRepository is an in-memory Repository used to test the domain
// logic in this package without a database, per .rules/testing.md.
type fakeRepository struct {
	byID   map[ID]Project
	nextID int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: make(map[ID]Project)}
}

func (f *fakeRepository) Create(_ context.Context, name string) (Project, error) {
	f.nextID++
	p := Project{
		ID:        ID(string(rune('a' + f.nextID))),
		Name:      name,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeRepository) Get(_ context.Context, id ID) (Project, error) {
	p, ok := f.byID[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	return p, nil
}

func (f *fakeRepository) Deactivate(_ context.Context, id ID) (Project, error) {
	p, ok := f.byID[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	p.Active = false
	p.UpdatedAt = time.Now()
	f.byID[id] = p
	return p, nil
}

func TestCreate(t *testing.T) {
	t.Run("rejects an empty name", func(t *testing.T) {
		repo := newFakeRepository()
		_, err := Create(context.Background(), repo, "   ")
		if err != ErrNameRequired {
			t.Fatalf("err = %v, want ErrNameRequired", err)
		}
	})

	t.Run("persists a new active Project", func(t *testing.T) {
		repo := newFakeRepository()
		p, err := Create(context.Background(), repo, "Acme")
		if err != nil {
			t.Fatalf("Create() returned unexpected error: %v", err)
		}
		if p.Name != "Acme" {
			t.Fatalf("Name = %q, want Acme", p.Name)
		}
		if !p.Active {
			t.Fatal("Active = false, want true for a newly created Project")
		}
	})
}

func TestDeactivate(t *testing.T) {
	t.Run("flips Active to false", func(t *testing.T) {
		repo := newFakeRepository()
		created, err := Create(context.Background(), repo, "Acme")
		if err != nil {
			t.Fatalf("setup: Create() failed: %v", err)
		}

		deactivated, err := Deactivate(context.Background(), repo, created.ID)
		if err != nil {
			t.Fatalf("Deactivate() returned unexpected error: %v", err)
		}
		if deactivated.Active {
			t.Fatal("Active = true, want false after Deactivate")
		}
	})

	t.Run("deactivating twice is idempotent, not an error", func(t *testing.T) {
		repo := newFakeRepository()
		created, err := Create(context.Background(), repo, "Acme")
		if err != nil {
			t.Fatalf("setup: Create() failed: %v", err)
		}

		if _, err := Deactivate(context.Background(), repo, created.ID); err != nil {
			t.Fatalf("first Deactivate() returned unexpected error: %v", err)
		}
		if _, err := Deactivate(context.Background(), repo, created.ID); err != nil {
			t.Fatalf("second Deactivate() returned unexpected error: %v", err)
		}
	})

	t.Run("unknown ID is ErrNotFound", func(t *testing.T) {
		repo := newFakeRepository()
		_, err := Deactivate(context.Background(), repo, ID("missing"))
		if err != ErrNotFound {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
