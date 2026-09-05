package updatepolicy

import (
	"testing"

	"github.com/manuelzzz/versiongate/internal/release"
	"github.com/manuelzzz/versiongate/internal/version"
)

func mustParse(t *testing.T, s string) version.Version {
	t.Helper()
	v, err := version.Parse(s)
	if err != nil {
		t.Fatalf("setup: version.Parse(%q) failed: %v", s, err)
	}
	return v
}

func rel(t *testing.T, v string, build int, policy release.Policy) release.Release {
	t.Helper()
	return release.Release{
		Version:     mustParse(t, v),
		BuildNumber: build,
		Policy:      policy,
	}
}

// TestEvaluate covers every worked example and deterministic scenario
// from specs/domain/update-policy.md, per issue #31's acceptance
// criteria.
func TestEvaluate(t *testing.T) {
	tests := []struct {
		name          string
		clientVersion string
		releases      []release.Release
		wantAction    Action
		wantLatest    string // "" means Latest should be nil
	}{
		{
			name:          "no releases yet",
			clientVersion: "1.0.0",
			releases:      nil,
			wantAction:    ActionContinue,
			wantLatest:    "",
		},
		{
			name:          "client already on the latest release",
			clientVersion: "1.3.0",
			releases: []release.Release{
				rel(t, "1.3.0", 1, release.PolicyOptional),
			},
			wantAction: ActionContinue,
			wantLatest: "1.3.0",
		},
		{
			name:          "client ahead of every registered release",
			clientVersion: "9.0.0",
			releases: []release.Release{
				rel(t, "1.0.0", 1, release.PolicyRequired),
			},
			wantAction: ActionContinue,
			wantLatest: "1.0.0",
		},
		{
			name:          "client behind only optional releases",
			clientVersion: "1.0.0",
			releases: []release.Release{
				rel(t, "1.1.0", 1, release.PolicyOptional),
				rel(t, "1.2.0", 1, release.PolicyOptional),
			},
			wantAction: ActionOptional,
			wantLatest: "1.2.0",
		},
		{
			name:          "client behind exactly one required release",
			clientVersion: "1.0.0",
			releases: []release.Release{
				rel(t, "1.1.0", 1, release.PolicyRequired),
			},
			wantAction: ActionRequired,
			wantLatest: "1.1.0",
		},
		{
			name:          "client behind multiple releases with mixed policies",
			clientVersion: "1.0.0",
			releases: []release.Release{
				rel(t, "1.1.0", 1, release.PolicyOptional),
				rel(t, "1.2.0", 1, release.PolicyRequired),
				rel(t, "1.3.0", 1, release.PolicyOptional),
			},
			wantAction: ActionRequired,
			wantLatest: "1.3.0",
		},
		{
			// The spec's motivating example: a required boundary is not
			// weakened by a later optional release.
			name:          "required followed by newer optional — client behind the boundary",
			clientVersion: "1.0.0",
			releases: []release.Release{
				rel(t, "1.1.0", 1, release.PolicyOptional),
				rel(t, "1.2.0", 1, release.PolicyRequired),
				rel(t, "1.3.0", 1, release.PolicyOptional),
			},
			wantAction: ActionRequired,
			wantLatest: "1.3.0",
		},
		{
			// Same release set, but the client already satisfies the
			// required boundary (1.2.0) — only 1.3.0 (optional) remains
			// ahead.
			name:          "required followed by newer optional — client already satisfies the boundary",
			clientVersion: "1.2.0",
			releases: []release.Release{
				rel(t, "1.1.0", 1, release.PolicyOptional),
				rel(t, "1.2.0", 1, release.PolicyRequired),
				rel(t, "1.3.0", 1, release.PolicyOptional),
			},
			wantAction: ActionOptional,
			wantLatest: "1.3.0",
		},
		{
			name:          "reports the overall latest release, not the one that triggered required",
			clientVersion: "1.0.0",
			releases: []release.Release{
				rel(t, "1.2.0", 1, release.PolicyRequired),
				rel(t, "1.5.0", 1, release.PolicyOptional),
			},
			wantAction: ActionRequired,
			wantLatest: "1.5.0",
		},
		{
			name:          "build number never changes the outcome",
			clientVersion: "1.0.0",
			releases: []release.Release{
				// A higher build number under a lower version must not
				// outrank a higher version.
				rel(t, "1.1.0", 999, release.PolicyOptional),
				rel(t, "1.2.0", 1, release.PolicyRequired),
			},
			wantAction: ActionRequired,
			wantLatest: "1.2.0",
		},
		{
			name:          "build number tiebreaks latest between identical versions",
			clientVersion: "1.0.0",
			releases: []release.Release{
				rel(t, "1.5.0", 1, release.PolicyOptional),
				rel(t, "1.5.0", 2, release.PolicyOptional),
			},
			wantAction: ActionOptional,
			wantLatest: "1.5.0", // build 2 should win the tiebreak
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientVersion := mustParse(t, tt.clientVersion)

			got := Evaluate(clientVersion, tt.releases)

			if got.Action != tt.wantAction {
				t.Fatalf("Action = %q, want %q", got.Action, tt.wantAction)
			}

			if tt.wantLatest == "" {
				if got.Latest != nil {
					t.Fatalf("Latest = %+v, want nil", got.Latest)
				}
				return
			}
			if got.Latest == nil {
				t.Fatalf("Latest = nil, want version %s", tt.wantLatest)
			}
			wantVersion := mustParse(t, tt.wantLatest)
			if got.Latest.Version != wantVersion {
				t.Fatalf("Latest.Version = %+v, want %+v", got.Latest.Version, wantVersion)
			}
		})
	}
}

// TestEvaluate_LatestBuildTiebreak asserts the specific Release
// selected as latest when two share a version, not just its version
// string — proving the build-number tiebreak resolves to the right
// record, not just a coincidentally-equal one.
func TestEvaluate_LatestBuildTiebreak(t *testing.T) {
	older := rel(t, "1.5.0", 1, release.PolicyOptional)
	newer := rel(t, "1.5.0", 2, release.PolicyRequired)

	got := Evaluate(mustParse(t, "1.0.0"), []release.Release{older, newer})

	if got.Latest == nil || got.Latest.BuildNumber != 2 {
		t.Fatalf("Latest = %+v, want the build-2 release", got.Latest)
	}
}
