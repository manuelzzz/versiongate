// Package application implements VersionGate's Application domain
// concept: a single distributable mobile app, scoped to exactly one
// Project, that Releases and Versions are evaluated against
// (specs/domain/application.md). Like internal/project, this package
// has no dependency on any specific storage technology.
package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/manuelzzz/versiongate/internal/project"
)

// ID identifies an Application. Like project.ID, it is assigned by a
// Repository at creation time.
type ID string

// Platform is the closed set of platforms an Application can target.
// This is a deliberate enumeration, not a free-form string
// (specs/domain/application.md's Constraints) — adding a platform is a
// domain change, not configuration.
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

// Valid reports whether p is one of the known platforms.
func (p Platform) Valid() bool {
	switch p {
	case PlatformIOS, PlatformAndroid:
		return true
	default:
		return false
	}
}

// Application represents a single distributable mobile app under a
// Project. Platform is fixed at creation — this package exposes no way
// to change it afterward, by design (specs/domain/application.md's
// Platform considerations: "Platform is fixed at creation and does not
// change").
type Application struct {
	ID          ID
	ProjectID   project.ID
	Identifier  string
	DisplayName string
	Platform    Platform
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ErrNotFound is returned by a Repository when no Application matches
// the requested (Project, ID) pair — including when the ID exists but
// belongs to a different Project, so a cross-Project lookup is
// indistinguishable from a genuinely missing Application
// (specs/protocols/http.md).
var ErrNotFound = errors.New("application: not found")

// ErrIdentifierRequired is returned by Create when identifier is empty.
var ErrIdentifierRequired = errors.New("application: identifier is required")

// ErrDisplayNameRequired is returned by Create when displayName is
// empty.
var ErrDisplayNameRequired = errors.New("application: display name is required")

// ErrInvalidPlatform is returned by Create when platform is not one of
// the known Platform values.
var ErrInvalidPlatform = errors.New("application: platform must be ios or android")

// ErrIdentifierTaken is returned by Create when identifier is already
// in use by another Application under the same Project — identifiers
// only need to be unique within their owning Project
// (specs/domain/application.md's Relationship with Project).
var ErrIdentifierTaken = errors.New("application: identifier already in use for this project")

// ErrProjectNotFound is returned by Create when projectID does not
// reference an existing Project. An Application cannot exist without
// an owning Project (specs/domain/application.md's Constraints).
var ErrProjectNotFound = errors.New("application: project not found")

// ErrIdentifierAmbiguous is returned by GetByIdentifier if more than
// one Application shares identifier across Projects. Identifiers are
// only guaranteed unique *within* a Project
// (specs/domain/application.md's Relationship with Project) — two
// Projects legitimately reusing the same identifier is an accepted
// scenario for Project-scoped access, but the unauthenticated
// update-check lookup (specs/protocols/update-check.md) has no Project
// context to disambiguate with. This is treated as an operator data
// issue (an internal error), not a client error: a well-behaved
// deployment should never actually hit it.
var ErrIdentifierAmbiguous = errors.New("application: identifier is not unique across projects")

// Repository persists and retrieves Applications, scoped to a Project.
// Infrastructure provides the implementation; this package only
// declares what it needs from it.
type Repository interface {
	Create(ctx context.Context, projectID project.ID, identifier, displayName string, platform Platform) (Application, error)
	Get(ctx context.Context, projectID project.ID, id ID) (Application, error)
	Deactivate(ctx context.Context, projectID project.ID, id ID) (Application, error)

	// GetByIdentifier looks up an Application by its public identifier
	// alone, with no Project scope — used only by the unauthenticated
	// update-check path (specs/protocols/update-check.md), where a
	// client has no token and thus no Project context. Every other
	// caller should use Get, which is Project-scoped.
	GetByIdentifier(ctx context.Context, identifier string) (Application, error)
}

// Create validates identifier, displayName, and platform, then
// persists a new, active Application under projectID via repo. This is
// the one place those invariants are enforced, so every caller (HTTP
// handlers, future CLI commands) gets them for free.
func Create(ctx context.Context, repo Repository, projectID project.ID, identifier, displayName string, platform Platform) (Application, error) {
	if strings.TrimSpace(identifier) == "" {
		return Application{}, ErrIdentifierRequired
	}
	if strings.TrimSpace(displayName) == "" {
		return Application{}, ErrDisplayNameRequired
	}
	if !platform.Valid() {
		return Application{}, ErrInvalidPlatform
	}
	return repo.Create(ctx, projectID, identifier, displayName, platform)
}

// Deactivate stops an Application from serving policy evaluations
// without deleting its history of Releases (specs/domain/application.md's
// Lifecycle). Deactivating an already-inactive Application is not an
// error — it is idempotent.
func Deactivate(ctx context.Context, repo Repository, projectID project.ID, id ID) (Application, error) {
	return repo.Deactivate(ctx, projectID, id)
}
