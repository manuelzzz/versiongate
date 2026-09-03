package release

import (
	"context"
	"errors"

	"github.com/manuelzzz/versiongate/internal/application"
	"github.com/manuelzzz/versiongate/internal/project"
	"github.com/manuelzzz/versiongate/internal/version"
)

// ErrInvalidBuildNumber is returned by Publish when buildNumber is
// negative.
var ErrInvalidBuildNumber = errors.New("release: build number must be non-negative")

// ErrInvalidPolicy is returned by Publish when policy is not one of the
// known Policy values.
var ErrInvalidPolicy = errors.New("release: policy must be optional or required")

// ErrApplicationInactive is returned by Publish when the target
// Application exists but is deactivated. Per
// specs/protocols/release-publishing.md, an unknown or inactive
// Application is a validation failure, not a not-found — distinct from
// how a direct Application lookup (specs/domain/application.md, #27)
// reports a missing Application.
var ErrApplicationInactive = errors.New("release: application is not active")

// ErrConflict is returned by Publish when the (version, build number)
// pair already has a Release under this Application, but with
// different metadata (a different policy) than the request — a
// genuine conflict, distinct from an idempotent retry
// (specs/protocols/release-publishing.md's Duplicate publishing).
var ErrConflict = errors.New("release: version and build number already published with different metadata")

// Publish implements the idempotent publish decision from
// specs/protocols/release-publishing.md:
//
//  1. Validate the request (build number, policy, and — via
//     applications — that the target Application exists, is active,
//     and belongs to projectID).
//  2. Attempt to create the Release directly, relying on the
//     database's uniqueness constraint (see Repository.Create) rather
//     than checking for existence first — a check-then-insert would
//     race under concurrent or retried requests.
//  3. If creation hits an existing (version, build number) pair,
//     resolve the ambiguity by reading back what's actually stored:
//     identical metadata is a safe no-op (the publisher's request
//     already happened), different metadata is a conflict.
//
// v is expected to already be validated (e.g. via version.Parse) —
// Publish does not re-validate its structure, only that it is
// accompanied by a valid build number and policy.
//
// The returned bool is true if this call created a new Release, and
// false if it resolved to an idempotent no-op against an existing one
// — both are success outcomes (specs/protocols/http.md), but callers
// (e.g. the HTTP layer) may want to distinguish 201 from 200.
func Publish(
	ctx context.Context,
	releases Repository,
	applications application.Repository,
	projectID project.ID,
	applicationID application.ID,
	v version.Version,
	buildNumber int,
	policy Policy,
) (Release, bool, error) {
	if buildNumber < 0 {
		return Release{}, false, ErrInvalidBuildNumber
	}
	if !policy.Valid() {
		return Release{}, false, ErrInvalidPolicy
	}

	app, err := applications.Get(ctx, projectID, applicationID)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return Release{}, false, ErrApplicationNotFound
		}
		return Release{}, false, err
	}
	if !app.Active {
		return Release{}, false, ErrApplicationInactive
	}

	created, err := releases.Create(ctx, applicationID, v, buildNumber, policy)
	if err == nil {
		return created, true, nil
	}
	if !errors.Is(err, ErrAlreadyExists) {
		return Release{}, false, err
	}

	existing, err := releases.GetByVersion(ctx, applicationID, v, buildNumber)
	if err != nil {
		return Release{}, false, err
	}
	if existing.Policy != policy {
		return Release{}, false, ErrConflict
	}
	return existing, false, nil
}
