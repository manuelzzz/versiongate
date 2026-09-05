package release

import (
	"context"
	"testing"
	"time"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/version"
)

// fakeReleaseRepository is an in-memory Repository used to test Publish
// without a database, per .rules/testing.md. It mirrors the real
// uniqueness constraint on (applicationID, version, buildNumber) and
// exposes a CreateCalls counter so tests can assert Create is attempted
// exactly once per Publish call — proving Publish never does a
// check-then-insert (a second call to Create is what a race would look
// like).
type fakeReleaseRepository struct {
	byKey       map[string]Release
	nextID      int
	CreateCalls int
}

func newFakeReleaseRepository() *fakeReleaseRepository {
	return &fakeReleaseRepository{byKey: make(map[string]Release)}
}

func releaseKey(applicationID application.ID, v version.Version, buildNumber int) string {
	return string(applicationID) + "|" +
		itoa(v.Major) + "." + itoa(v.Minor) + "." + itoa(v.Patch) + "|" +
		itoa(buildNumber)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (f *fakeReleaseRepository) Create(_ context.Context, applicationID application.ID, v version.Version, buildNumber int, policy Policy) (Release, error) {
	f.CreateCalls++
	key := releaseKey(applicationID, v, buildNumber)
	if _, exists := f.byKey[key]; exists {
		return Release{}, ErrAlreadyExists
	}
	f.nextID++
	r := Release{
		ID:            ID(itoa(f.nextID)),
		ApplicationID: applicationID,
		Version:       v,
		BuildNumber:   buildNumber,
		Policy:        policy,
		CreatedAt:     time.Now(),
	}
	f.byKey[key] = r
	return r, nil
}

func (f *fakeReleaseRepository) GetByVersion(_ context.Context, applicationID application.ID, v version.Version, buildNumber int) (Release, error) {
	r, ok := f.byKey[releaseKey(applicationID, v, buildNumber)]
	if !ok {
		return Release{}, ErrNotFound
	}
	return r, nil
}

func (f *fakeReleaseRepository) ListByApplication(_ context.Context, applicationID application.ID) ([]Release, error) {
	var releases []Release
	for _, r := range f.byKey {
		if r.ApplicationID == applicationID {
			releases = append(releases, r)
		}
	}
	return releases, nil
}

// fakeApplicationRepository is a minimal in-memory application.Repository
// sufficient for Publish's needs: Get and Active-state checking.
type fakeApplicationRepository struct {
	byID map[application.ID]application.Application
}

func newFakeApplicationRepository(apps ...application.Application) *fakeApplicationRepository {
	f := &fakeApplicationRepository{byID: make(map[application.ID]application.Application)}
	for _, a := range apps {
		f.byID[a.ID] = a
	}
	return f
}

func (f *fakeApplicationRepository) Create(context.Context, project.ID, string, string, application.Platform) (application.Application, error) {
	panic("not used by these tests")
}

func (f *fakeApplicationRepository) GetByIdentifier(context.Context, string) (application.Application, error) {
	panic("not used by these tests")
}

func (f *fakeApplicationRepository) Get(_ context.Context, projectID project.ID, id application.ID) (application.Application, error) {
	a, ok := f.byID[id]
	if !ok || a.ProjectID != projectID {
		return application.Application{}, application.ErrNotFound
	}
	return a, nil
}

func (f *fakeApplicationRepository) Deactivate(context.Context, project.ID, application.ID) (application.Application, error) {
	panic("not used by these tests")
}

const testProjectID = project.ID("project-a")

var activeApp = application.Application{
	ID:        application.ID("app-1"),
	ProjectID: testProjectID,
	Active:    true,
}

var inactiveApp = application.Application{
	ID:        application.ID("app-2"),
	ProjectID: testProjectID,
	Active:    false,
}

func mustParseVersion(t *testing.T, s string) version.Version {
	t.Helper()
	v, err := version.Parse(s)
	if err != nil {
		t.Fatalf("setup: version.Parse(%q) failed: %v", s, err)
	}
	return v
}

func TestPublish_FreshRelease(t *testing.T) {
	releases := newFakeReleaseRepository()
	apps := newFakeApplicationRepository(activeApp)
	v := mustParseVersion(t, "1.0.0")

	r, created, err := Publish(context.Background(), releases, apps, testProjectID, activeApp.ID, v, 42, PolicyOptional)
	if err != nil {
		t.Fatalf("Publish() returned unexpected error: %v", err)
	}
	if !created {
		t.Fatal("created = false, want true for a brand-new (version, build)")
	}
	if r.Version != v || r.BuildNumber != 42 || r.Policy != PolicyOptional {
		t.Fatalf("unexpected Release: %+v", r)
	}
}

func TestPublish_IdempotentRetry(t *testing.T) {
	releases := newFakeReleaseRepository()
	apps := newFakeApplicationRepository(activeApp)
	v := mustParseVersion(t, "1.0.0")

	first, created, err := Publish(context.Background(), releases, apps, testProjectID, activeApp.ID, v, 42, PolicyOptional)
	if err != nil {
		t.Fatalf("first Publish() returned unexpected error: %v", err)
	}
	if !created {
		t.Fatal("first Publish(): created = false, want true")
	}

	second, created, err := Publish(context.Background(), releases, apps, testProjectID, activeApp.ID, v, 42, PolicyOptional)
	if err != nil {
		t.Fatalf("retried Publish() with identical metadata returned an error, want a no-op success: %v", err)
	}
	if created {
		t.Fatal("retried Publish(): created = true, want false — this is a no-op, not a new Release")
	}
	if second.ID != first.ID {
		t.Fatalf("retried Publish() returned a different Release (ID %v) than the original (ID %v)", second.ID, first.ID)
	}
	if releases.CreateCalls != 2 {
		t.Fatalf("Create was called %d times, want exactly 2 (one attempt per Publish call, no check-then-insert)", releases.CreateCalls)
	}
}

func TestPublish_ConflictOnDifferentMetadata(t *testing.T) {
	releases := newFakeReleaseRepository()
	apps := newFakeApplicationRepository(activeApp)
	v := mustParseVersion(t, "1.0.0")

	if _, _, err := Publish(context.Background(), releases, apps, testProjectID, activeApp.ID, v, 42, PolicyOptional); err != nil {
		t.Fatalf("setup: first Publish() failed: %v", err)
	}

	_, created, err := Publish(context.Background(), releases, apps, testProjectID, activeApp.ID, v, 42, PolicyRequired)
	if err != ErrConflict {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
	if created {
		t.Fatal("created = true on a conflict, want false")
	}
}

func TestPublish_ValidationFailuresHaveNoSideEffects(t *testing.T) {
	v := mustParseVersion(t, "1.0.0")

	t.Run("negative build number", func(t *testing.T) {
		releases := newFakeReleaseRepository()
		apps := newFakeApplicationRepository(activeApp)

		_, _, err := Publish(context.Background(), releases, apps, testProjectID, activeApp.ID, v, -1, PolicyOptional)
		if err != ErrInvalidBuildNumber {
			t.Fatalf("err = %v, want ErrInvalidBuildNumber", err)
		}
		if releases.CreateCalls != 0 {
			t.Fatalf("Create was called %d times, want 0 — validation failure must have no side effects", releases.CreateCalls)
		}
	})

	t.Run("invalid policy", func(t *testing.T) {
		releases := newFakeReleaseRepository()
		apps := newFakeApplicationRepository(activeApp)

		_, _, err := Publish(context.Background(), releases, apps, testProjectID, activeApp.ID, v, 1, Policy("mandatory"))
		if err != ErrInvalidPolicy {
			t.Fatalf("err = %v, want ErrInvalidPolicy", err)
		}
		if releases.CreateCalls != 0 {
			t.Fatalf("Create was called %d times, want 0", releases.CreateCalls)
		}
	})

	t.Run("unknown application", func(t *testing.T) {
		releases := newFakeReleaseRepository()
		apps := newFakeApplicationRepository() // empty

		_, _, err := Publish(context.Background(), releases, apps, testProjectID, application.ID("does-not-exist"), v, 1, PolicyOptional)
		if err != ErrApplicationNotFound {
			t.Fatalf("err = %v, want ErrApplicationNotFound", err)
		}
		if releases.CreateCalls != 0 {
			t.Fatalf("Create was called %d times, want 0", releases.CreateCalls)
		}
	})

	t.Run("inactive application", func(t *testing.T) {
		releases := newFakeReleaseRepository()
		apps := newFakeApplicationRepository(inactiveApp)

		_, _, err := Publish(context.Background(), releases, apps, testProjectID, inactiveApp.ID, v, 1, PolicyOptional)
		if err != ErrApplicationInactive {
			t.Fatalf("err = %v, want ErrApplicationInactive", err)
		}
		if releases.CreateCalls != 0 {
			t.Fatalf("Create was called %d times, want 0", releases.CreateCalls)
		}
	})

	t.Run("application from a different project is reported the same as unknown", func(t *testing.T) {
		releases := newFakeReleaseRepository()
		apps := newFakeApplicationRepository(activeApp)

		_, _, err := Publish(context.Background(), releases, apps, project.ID("someone-elses-project"), activeApp.ID, v, 1, PolicyOptional)
		if err != ErrApplicationNotFound {
			t.Fatalf("err = %v, want ErrApplicationNotFound", err)
		}
	})
}
