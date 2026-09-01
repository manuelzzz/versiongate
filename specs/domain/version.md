# Domain Concept: Version

## Purpose

Version is what lets VersionGate answer "is this newer?" It defines how a
[[release]]'s version and build number are structured, compared, and
ordered so that policy evaluation and "latest release" (see
`specs/domain/release.md`) are well-defined and deterministic — regardless
of when a Release was published or in what order.

Version is a value concept: two versions with the same structure and
components are the same version, with no identity of their own beyond
their value.

## Semantic versions

- VersionGate represents a version as three non-negative integers:
  `MAJOR.MINOR.PATCH` (e.g. `1.0.0`, `1.10.0`, `2.0.0`), following
  [Semantic Versioning](https://semver.org)'s structure for these three
  fields.
- Optional pre-release identifiers (e.g. `1.0.0-beta.1`) are recognized as
  part of the SemVer spec, but VersionGate does not need to support them on
  day one — this document defines how they would compare (below) without
  requiring the initial implementation to accept them. Do not build
  speculative handling beyond what SemVer already specifies.
- VersionGate does not use SemVer's optional build-metadata suffix
  (`+<metadata>`, e.g. the `+42` in `1.2.0+42`) as part of the version. If a
  client or CI system reports a version string with a `+` suffix, that
  suffix is not part of the domain's Version value — see Build numbers
  below for how VersionGate represents that information instead.

## Version comparison

- Versions are compared **numerically, field by field**, in order: MAJOR,
  then MINOR, then PATCH. This is not a string/lexicographic comparison.
- Example: `1.10.0 > 1.2.0`, because MINOR `10 > 2` numerically — even
  though the string `"1.10.0"` would sort before `"1.2.0"`
  lexicographically. Comparing versions as strings is explicitly wrong and
  must never be used.
- Given the examples `1.0.0`, `1.0.1`, `1.10.0`, `2.0.0`, the correct
  ascending order is: `1.0.0 < 1.0.1 < 1.10.0 < 2.0.0`.
- If pre-release identifiers are supported, a pre-release version is
  ordered before its corresponding release version (e.g.
  `1.0.0-beta.1 < 1.0.0`), per SemVer precedence rules.

## Build numbers

- A build number is a separate, independent value from version — a
  platform-native build identifier (e.g. Android `versionCode`, iOS
  `CFBundleVersion`) supplied by the platform's build/store tooling.
- Build numbers are compared numerically as well, but only ever as a
  tiebreaker within an identical version — they are not a substitute
  axis for "is this newer" across different versions. A higher build
  number under a lower version does not make that Release newer overall
  (see Ordering in `specs/domain/release.md`).

## Relationship between version and build

- Version expresses the user-facing, semantic notion of "how new" a
  Release is. Build number exists to disambiguate distinct artifacts that
  a platform or CI/CD pipeline produced under what may be the same version
  string (e.g. two builds shipped during a staged rollout, both `1.2.0`,
  distinguished by build `41` vs. `42`).
- Version is the primary axis of comparison; build number only matters
  when comparing two Releases that share the same version.

## Independent representation

- Version and build number are represented **independently** — two
  distinct fields, not one derived from the other and not encoded together
  in a single string.
- This is a deliberate choice: platform build numbers don't follow SemVer
  syntax or semantics (they're typically a simple incrementing integer),
  and conflating them with the semantic version (e.g. via SemVer's
  build-metadata suffix) would tie VersionGate's comparison logic to a
  convention it doesn't need and platforms don't consistently follow.

## Equality

- Two versions are equal if and only if their MAJOR, MINOR, and PATCH
  components are all equal (and, when supported, their pre-release
  identifiers are equal). Build metadata, since it is not part of the
  Version value at all, plays no role in equality.
- Two Releases are equal in version only if both version and build number
  match — but as established in `specs/domain/release.md`, that exact
  (version, build number) combination must not be duplicated across
  Releases of the same Application.

## Ordering

- Version ordering is a strict total order: for any two valid versions,
  exactly one of `<`, `=`, `>` holds, per the numeric field-by-field rule
  above.
- This total order is what makes "latest release" well-defined as a pure
  function of the set of published, non-revoked Releases — independent of
  publish time, insertion order, or which system produced the Release.

## Invalid versions

- A version must consist of three non-negative integer components
  (MAJOR.MINOR.PATCH); anything that doesn't parse into that structure
  (missing components, non-numeric components, negative numbers, leading
  content, etc.) is an invalid version.
- An invalid version must be rejected outright — VersionGate must not
  attempt to coerce, truncate, or guess a "best effort" interpretation of a
  malformed version string.
- A build number must be a non-negative integer; a malformed build number
  is likewise invalid and must be rejected rather than coerced.

## Duplicate version/build combinations

- Within a single Application, the pair (version, build number) must be
  unique — this is the same invariant stated in
  `specs/domain/release.md`'s Duplicate releases section, restated here
  from the Version concept's perspective: uniqueness is evaluated using
  the comparison and equality rules defined above, not string equality of
  however the version was originally submitted.
- Two different textual representations that normalize to the same
  version (if VersionGate ever accepts more than one input format) must be
  treated as the same version for duplicate detection — equality is a
  property of the parsed value, never of the raw input string.
