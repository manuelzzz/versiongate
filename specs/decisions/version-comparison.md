# Decision: Version Comparison Strategy

## Status

Accepted — initial decision, revisit if a concrete requirement outgrows it.

## Context

VersionGate's core function — deciding whether a client should continue,
be notified, or be required to update — depends entirely on being able to
compare versions deterministically. This decision formalizes *why* the
comparison approach already specified in `specs/domain/version.md` and
relied on by `specs/domain/release.md` and
`specs/protocols/release-publishing.md` was chosen, and what it rules out.

Several requirements converge on this:

- `specs/domain/release.md` defines "latest release" as a derived fact
  based on version ordering, explicitly **not** based on when a Release
  was written to storage — because two CI/CD pipelines may publish
  Releases out of chronological order (a delayed pipeline publishing an
  older version after a faster one has already published a newer one).
- `specs/protocols/release-publishing.md` requires publish requests to be
  safely retryable and order-independent; if "latest" depended on
  insertion time, retries and out-of-order arrivals could change which
  Release is considered latest in ways unrelated to what was actually
  published.
- Client apps report their own current version when asking for a policy
  decision. That reported version needs to be compared against registered
  Releases even when it doesn't match any of them exactly — including the
  case where a client is running a version newer than anything VersionGate
  has on record.
- Version strings and build numbers are supplied externally (by CI/CD
  publishers and by client apps) and are not guaranteed to be well-formed.

## Decision

1. **Versions follow a Semantic Versioning-shaped structure**:
   `MAJOR.MINOR.PATCH`, with optional pre-release identifiers recognized
   per the SemVer precedence rules but not required to be supported by the
   initial implementation (see `specs/domain/version.md`).
2. **Version and build number are independent values.** Build number is
   never folded into the version string (no SemVer build-metadata suffix
   usage) and is never used as a primary ordering axis.
3. **Version ordering is numeric, field-by-field** (MAJOR, then MINOR,
   then PATCH), never lexicographic/string comparison. `1.10.0 > 1.2.0`.
4. **Build ordering only applies as a tiebreaker between two Releases that
   share the identical version.** It never causes a Release with a lower
   version to be considered newer than one with a higher version.
5. **"Latest release" is always computed by comparing versions of
   currently non-revoked Releases for an Application.** It is never
   derived from database insertion order, row identity, auto-increment
   IDs, or any creation/publish timestamp.
6. **A client-reported version newer than any registered Release's version
   is treated as already up to date** — VersionGate must respond as if no
   update is needed, not as an error and not by suggesting an older
   registered Release.
7. **An invalid (malformed) version — from a publisher or from a client —
   is rejected, never coerced, truncated, or guessed into a "best effort"
   comparable value.** Comparison logic must only ever operate on values
   already confirmed to be well-formed.

## Rationale

- **Numeric field comparison is the only correct interpretation of
  MAJOR.MINOR.PATCH.** String/lexicographic comparison silently produces
  wrong results for any component reaching two digits (`1.10.0` sorting
  before `1.2.0`), which would make "latest release" wrong exactly when a
  product has been alive long enough to matter.
- **Excluding insertion time from "latest" is what makes the concurrency
  guarantees in `specs/protocols/release-publishing.md` possible.** If
  "latest" depended on write order, two pipelines racing or retrying would
  produce different, order-dependent answers to "what's latest" —
  non-deterministic behavior in exactly the scenario (CI/CD publishing)
  VersionGate is built to support. Deriving "latest" purely from version
  values makes the answer a pure function of *what has been published*,
  not *the order it was written*.
- **Keeping build number as a tiebreaker only, never a primary axis,**
  reflects reality: build numbers are platform tooling artifacts
  (`versionCode`, `CFBundleVersion`) with no cross-version semantic
  meaning. Letting a higher build number outrank a lower version would let
  an unrelated CI/CD counter override the actual semantic intent of a
  release.
- **Treating a client ahead of the latest registered Release as
  up-to-date is a deliberate, necessary rule, not an edge case being
  ignored.** This is a legitimate, expected situation — beta/dogfood
  builds, manually distributed test builds, or a client that already
  adopted a Release VersionGate hasn't been told about yet (e.g. a delayed
  publish). Treating it as an error, or worse, telling that client to
  "update" to an older registered Release, would be actively wrong and
  would erode trust in the system's decisions.
- **Rejecting invalid versions outright (fail closed) protects the
  comparison logic's correctness.** Comparison is only meaningful over
  well-formed values; accepting a "best guess" interpretation of malformed
  input would let ambiguous data quietly influence policy decisions —
  exactly the kind of silent-corruption risk a service making
  update-or-not decisions cannot afford.

## Consequences

- "Latest release" must always be computed by comparing the version values
  of an Application's non-revoked Releases — it must never be cached or
  stored as a static flag on "whichever Release was inserted last," and
  any caching introduced later must be invalidated based on version
  comparison outcomes, not write recency.
- Publishers can safely publish out of order or retry publish requests
  (per `specs/protocols/release-publishing.md`) without any risk of
  corrupting which Release is considered latest.
- Policy evaluation logic must explicitly branch on "client version newer
  than latest known Release" as its own case — distinct from "client is
  on the latest" and from "client is behind" — and must resolve it to "no
  update needed."
- Malformed version input (from a publisher or from a client request) must
  be handled as a distinct validation failure, never silently treated as
  either "up to date" or "needs update."
- Comparison logic becomes a well-isolated, pure piece of domain logic
  (values in, ordering out) with no dependency on storage or time — this
  keeps it independent of infrastructure, consistent with
  `.rules/architecture.md`'s domain-independence rule, and straightforward
  to test exhaustively with table-driven tests per `.rules/testing.md`.

## Alternatives considered

- **Lexicographic (string) comparison of version strings.** Rejected: it
  is simply incorrect for multi-digit version components (`1.10.0` vs.
  `1.2.0`), which will occur in any project with a normal release
  cadence. Not a viable option at any scale.
- **Determining "latest release" by insertion timestamp or
  auto-increment ID.** Rejected per the explicit requirement: this
  approach fails as soon as two publishers publish out of order — the
  exact scenario CI/CD pipelines are expected to produce — and makes
  "latest" a function of infrastructure timing rather than actual release
  content.
- **Using build number as the primary ordering signal** (e.g. "highest
  build number wins"). Rejected: build numbers are platform-specific
  incrementing counters without guaranteed semantic meaning relative to
  version, and different platforms/pipelines may not keep them globally
  monotonic in a way that reflects "newer" in the way version does.
- **Treating a client version ahead of the latest registered Release as
  an error or invalid state.** Rejected: this is an expected occurrence
  (beta builds, delayed publishing, manual test builds) and erroring would
  actively break well-behaved clients that happen to be ahead of what
  VersionGate has on record.
- **Supporting full SemVer semantics (pre-release precedence, build
  -metadata parsing) from day one.** Deferred, not rejected outright: the
  ordering rules for pre-release identifiers are already defined
  conceptually in `specs/domain/version.md` so they can be added without
  redesign, but implementing them now would be speculative scope beyond
  VersionGate's current needs (see `.rules/architecture.md`'s
  premature-abstraction guidance).
- **Adopting a third-party SemVer parsing/comparison library by
  default.** Considered but not mandated: the comparison rules VersionGate
  needs today (three numeric fields, straightforward field-by-field
  ordering) are simple enough to implement directly with the standard
  library, consistent with `.rules/go.md`'s standard-library preference.
  There is no compelling reason yet to take on an external dependency for
  this; it can be reconsidered if/when full SemVer pre-release/build
  -metadata precedence support is actually needed and proves nontrivial to
  implement correctly by hand.
