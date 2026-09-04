// Package updatepolicy implements the Update Policy Evaluation domain
// concept: the derived decision of whether a client on a given version
// should continue, be notified of an available update, or be required
// to update (specs/domain/update-policy.md). Unlike most packages in
// this module, there is nothing to persist here — Evaluate is a pure
// function over its inputs, with no dependency on storage, time, or
// any other domain package's Repository.
package updatepolicy

import (
	"github.com/manuelzzz/versiongate/internal/release"
	"github.com/manuelzzz/versiongate/internal/version"
)

// Action is the resolved outcome of evaluation.
type Action string

const (
	ActionContinue Action = "continue"
	ActionOptional Action = "optional"
	ActionRequired Action = "required"
)

// Result is Evaluate's resolved outcome. Latest is nil only when
// releases was empty — there is nothing to report back.
type Result struct {
	Action Action
	Latest *release.Release
}

// Evaluate implements the algorithm from specs/domain/update-policy.md:
//
//  1. ahead = every Release in releases whose version is strictly
//     greater than clientVersion.
//  2. If ahead is empty, the outcome is continue — this covers both
//     "client is already on the latest" and "client is ahead of every
//     registered Release," which are the same case structurally.
//  3. If any Release in ahead is required, the outcome is required,
//     regardless of how many optional Releases are also ahead or how
//     they're interleaved — a required Release establishes a minimum
//     acceptable version boundary that a later optional Release does
//     not weaken.
//  4. Otherwise (every Release in ahead is optional), the outcome is
//     optional.
//
// releases is expected to already be the Application's current
// non-revoked Releases (there is no revocation yet — see
// internal/release's issue scope — so today that means "all of the
// Application's Releases"; a caller filters once revocation exists).
//
// Build number plays no role: only version ordering decides ahead
// membership and the resolved Action, per
// specs/decisions/version-comparison.md. It does, however, still serve
// as a tiebreaker when selecting which Release is "latest" among
// Releases that share an identical version — that's a property of
// Release ordering in general (specs/domain/release.md), not something
// special to evaluation.
func Evaluate(clientVersion version.Version, releases []release.Release) Result {
	if len(releases) == 0 {
		return Result{Action: ActionContinue}
	}

	var latest *release.Release
	var anyRequiredAhead bool
	var anyAhead bool

	for i := range releases {
		r := &releases[i]

		if latest == nil || version.CompareWithBuild(r.Version, r.BuildNumber, latest.Version, latest.BuildNumber) > 0 {
			latest = r
		}

		if r.Version.Compare(clientVersion) > 0 {
			anyAhead = true
			if r.Policy == release.PolicyRequired {
				anyRequiredAhead = true
			}
		}
	}

	switch {
	case !anyAhead:
		return Result{Action: ActionContinue, Latest: latest}
	case anyRequiredAhead:
		return Result{Action: ActionRequired, Latest: latest}
	default:
		return Result{Action: ActionOptional, Latest: latest}
	}
}
