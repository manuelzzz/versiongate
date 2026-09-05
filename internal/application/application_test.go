package application

import (
	"context"
	"testing"
	"time"

	"github.com/manuelzzz/versiongate/internal/project"
)

// fakeRepository is an in-memory Repository used to test the domain
// logic in this package without a database, per .rules/testing.md. It
// mirrors the real per-Project identifier-uniqueness constraint so
// tests can exercise it directly.
type fakeRepository struct {
	byKey  map[[2]string]Application // key: (projectID, identifier)
	byID   map[ID]Application
	nextID int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byKey: make(map[[2]string]Application),
		byID:  make(map[ID]Application),
	}
}

func (f *fakeRepository) Create(_ context.Context, projectID project.ID, identifier, displayName string, platform Platform) (Application, error) {
	key := [2]string{string(projectID), identifier}
	if _, exists := f.byKey[key]; exists {
		return Application{}, ErrIdentifierTaken
	}

	f.nextID++
	a := Application{
		ID:          ID(string(rune('a' + f.nextID))),
		ProjectID:   projectID,
		Identifier:  identifier,
		DisplayName: displayName,
		Platform:    platform,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	f.byKey[key] = a
	f.byID[a.ID] = a
	return a, nil
}

func (f *fakeRepository) Get(_ context.Context, projectID project.ID, id ID) (Application, error) {
	a, ok := f.byID[id]
	if !ok || a.ProjectID != projectID {
		return Application{}, ErrNotFound
	}
	return a, nil
}

func (f *fakeRepository) GetByIdentifier(_ context.Context, identifier string) (Application, error) {
	var matches []Application
	for _, a := range f.byID {
		if a.Identifier == identifier {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 0:
		return Application{}, ErrNotFound
	case 1:
		return matches[0], nil
	default:
		return Application{}, ErrIdentifierAmbiguous
	}
}

func (f *fakeRepository) Deactivate(_ context.Context, projectID project.ID, id ID) (Application, error) {
	a, ok := f.byID[id]
	if !ok || a.ProjectID != projectID {
		return Application{}, ErrNotFound
	}
	a.Active = false
	a.UpdatedAt = time.Now()
	f.byID[id] = a
	f.byKey[[2]string{string(a.ProjectID), a.Identifier}] = a
	return a, nil
}

const (
	projectA = project.ID("project-a")
	projectB = project.ID("project-b")
)

func TestCreate(t *testing.T) {
	t.Run("rejects an empty identifier", func(t *testing.T) {
		repo := newFakeRepository()
		_, err := Create(context.Background(), repo, projectA, "  ", "Acme iOS", PlatformIOS)
		if err != ErrIdentifierRequired {
			t.Fatalf("err = %v, want ErrIdentifierRequired", err)
		}
	})

	t.Run("rejects an empty display name", func(t *testing.T) {
		repo := newFakeRepository()
		_, err := Create(context.Background(), repo, projectA, "acme-ios", "  ", PlatformIOS)
		if err != ErrDisplayNameRequired {
			t.Fatalf("err = %v, want ErrDisplayNameRequired", err)
		}
	})

	t.Run("rejects an unknown platform", func(t *testing.T) {
		repo := newFakeRepository()
		_, err := Create(context.Background(), repo, projectA, "acme-web", "Acme Web", Platform("web"))
		if err != ErrInvalidPlatform {
			t.Fatalf("err = %v, want ErrInvalidPlatform", err)
		}
	})

	t.Run("persists a new active Application", func(t *testing.T) {
		repo := newFakeRepository()
		a, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS", PlatformIOS)
		if err != nil {
			t.Fatalf("Create() returned unexpected error: %v", err)
		}
		if !a.Active {
			t.Fatal("Active = false, want true for a newly created Application")
		}
		if a.Platform != PlatformIOS {
			t.Fatalf("Platform = %v, want %v", a.Platform, PlatformIOS)
		}
	})

	t.Run("same identifier is rejected within the same Project", func(t *testing.T) {
		repo := newFakeRepository()
		if _, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS", PlatformIOS); err != nil {
			t.Fatalf("setup: first Create() failed: %v", err)
		}
		_, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS (again)", PlatformIOS)
		if err != ErrIdentifierTaken {
			t.Fatalf("err = %v, want ErrIdentifierTaken", err)
		}
	})

	t.Run("same identifier is allowed across two different Projects", func(t *testing.T) {
		repo := newFakeRepository()
		if _, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS", PlatformIOS); err != nil {
			t.Fatalf("setup: Create() for project A failed: %v", err)
		}
		_, err := Create(context.Background(), repo, projectB, "acme-ios", "Acme iOS (project B)", PlatformIOS)
		if err != nil {
			t.Fatalf("Create() for project B returned unexpected error: %v", err)
		}
	})
}

func TestGet_CrossProjectIsolation(t *testing.T) {
	repo := newFakeRepository()
	a, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS", PlatformIOS)
	if err != nil {
		t.Fatalf("setup: Create() failed: %v", err)
	}

	if _, err := repo.Get(context.Background(), projectA, a.ID); err != nil {
		t.Fatalf("Get() from the owning Project returned unexpected error: %v", err)
	}

	_, err = repo.Get(context.Background(), projectB, a.ID)
	if err != ErrNotFound {
		t.Fatalf("Get() from a different Project: err = %v, want ErrNotFound", err)
	}
}

func TestDeactivate(t *testing.T) {
	t.Run("flips Active to false", func(t *testing.T) {
		repo := newFakeRepository()
		a, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS", PlatformIOS)
		if err != nil {
			t.Fatalf("setup: Create() failed: %v", err)
		}

		deactivated, err := Deactivate(context.Background(), repo, projectA, a.ID)
		if err != nil {
			t.Fatalf("Deactivate() returned unexpected error: %v", err)
		}
		if deactivated.Active {
			t.Fatal("Active = true, want false after Deactivate")
		}
	})

	t.Run("deactivating twice is idempotent, not an error", func(t *testing.T) {
		repo := newFakeRepository()
		a, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS", PlatformIOS)
		if err != nil {
			t.Fatalf("setup: Create() failed: %v", err)
		}
		if _, err := Deactivate(context.Background(), repo, projectA, a.ID); err != nil {
			t.Fatalf("first Deactivate() returned unexpected error: %v", err)
		}
		if _, err := Deactivate(context.Background(), repo, projectA, a.ID); err != nil {
			t.Fatalf("second Deactivate() returned unexpected error: %v", err)
		}
	})

	t.Run("a different Project cannot deactivate another Project's Application", func(t *testing.T) {
		repo := newFakeRepository()
		a, err := Create(context.Background(), repo, projectA, "acme-ios", "Acme iOS", PlatformIOS)
		if err != nil {
			t.Fatalf("setup: Create() failed: %v", err)
		}

		_, err = Deactivate(context.Background(), repo, projectB, a.ID)
		if err != ErrNotFound {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
