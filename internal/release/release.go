// Package release implements VersionGate's Release domain concept: a
// published version of an Application for its platform
// (specs/domain/release.md). VersionGate does not distribute binaries —
// a Release is metadata only. Like internal/application, this package
// has no dependency on any specific storage technology.
package release

import (
	"context"
	"errors"
	"time"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/version"
)

// ID identifies a Release. Like application.ID, it is assigned by a
// Repository at creation time.
type ID string

// Policy is the update behavior a Release implies for clients relative
// to their reported version — optional (notify) or mandatory (require
// update). This is a closed set, not a free-form string.
type Policy string

const (
	PolicyOptional Policy = "optional"
	PolicyRequired Policy = "required"
)

// Valid reports whether p is one of the known policies.
func (p Policy) Valid() bool {
	switch p {
	case PolicyOptional, PolicyRequired:
		return true
	default:
		return false
	}
}

// Release is a published version of an Application. Publishing is a
// single, atomic act — there is no mutable "in progress" state
// (specs/domain/release.md's Release lifecycle). Once created, a
// Release's Version, BuildNumber, and Policy are immutable: this
// package exposes no way to change them after Create. Revocation
// (specs/domain/release.md's Revocation section) is explicitly out of
// scope for this package — see #28's issue scope.
type Release struct {
	ID            ID
	ApplicationID application.ID
	Version       version.Version
	BuildNumber   int
	Policy        Policy
	CreatedAt     time.Time
}

// ErrNotFound is returned by GetByVersion when no Release matches the
// requested (Application, version, build number).
var ErrNotFound = errors.New("release: not found")

// ErrAlreadyExists is returned by Create when the (Application,
// version, build number) triple already has a Release — this is the
// same invariant enforced by the database's unique constraint
// (specs/domain/release.md's Duplicate releases). It does not by itself
// distinguish an idempotent retry from a genuine conflict; that
// resolution is Publish's job (see publish.go).
var ErrAlreadyExists = errors.New("release: version and build number already published for this application")

// ErrApplicationNotFound is returned by Create when applicationID does
// not reference an existing Application. A Release cannot exist without
// an owning Application.
var ErrApplicationNotFound = errors.New("release: application not found")

// Repository persists and retrieves Releases, scoped to an Application.
// Infrastructure provides the implementation; this package only
// declares what it needs from it. Create is expected to rely on a
// database-level uniqueness constraint on (applicationID, version,
// buildNumber) rather than a check-then-insert race — see Publish.
type Repository interface {
	Create(ctx context.Context, applicationID application.ID, v version.Version, buildNumber int, policy Policy) (Release, error)
	GetByVersion(ctx context.Context, applicationID application.ID, v version.Version, buildNumber int) (Release, error)

	// ListByApplication returns applicationID's current non-revoked
	// Releases, for update policy evaluation
	// (specs/domain/update-policy.md). Revocation doesn't exist yet (see
	// #28's issue scope), so today this is simply every Release under
	// the Application; a caller filters once revocation lands.
	ListByApplication(ctx context.Context, applicationID application.ID) ([]Release, error)
}
