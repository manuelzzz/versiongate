# Domain Concept: Update Policy Evaluation

## Purpose

Update Policy Evaluation is the derived decision VersionGate makes when
asked whether a client on a given version should continue, be notified of
an available update, or be required to update. It is not a stored entity —
it is a pure function over a client's reported version and an
[[application]]'s current [[release]] history.

This concept exists separately from Release because it operates *across*
a sequence of Releases, not on any single one. `specs/domain/release.md`
defines the policy value (`optional`/`required`) carried by an individual
Release; this document defines how those individual values combine into
one deterministic answer for a specific client.

## Inputs

- The client's reported version (see `specs/domain/version.md`).
- The set of the Application's currently non-revoked Releases, each with
  its own version and update policy (`optional` or `required`).

Build number plays no role in this evaluation — see
`specs/decisions/version-comparison.md` and Version/build ordering
interactions, below.

## The rule

> If any non-revoked Release with a version strictly greater than the
> client's reported version carries a `required` policy, the resolved
> outcome is **required** — regardless of whether a newer `optional`
> Release also exists.

Equivalently: a `required` Release establishes a **minimum acceptable
version boundary**. Any client below that boundary must update; a later
`optional` Release does not lower or clear a boundary a `required` Release
already established.

## Algorithm

Given a client version `v` and an Application's non-revoked Releases:

1. Let `ahead` = the set of Releases whose version is strictly greater
   than `v`.
2. If `ahead` is empty → outcome is **continue** (the client is already on
   the latest known version, or ahead of it).
3. Otherwise, if any Release in `ahead` has policy `required` → outcome is
   **required**.
4. Otherwise (every Release in `ahead` is `optional`) → outcome is
   **optional**.

In both the `required` and `optional` outcomes, the Release reported back
to the client (see `specs/protocols/update-check.md`) is the latest
Release overall (highest version among non-revoked Releases) — not
necessarily the specific Release that triggered a `required` outcome.
Directing the client to the latest Release is always correct: by
definition its version is greater than or equal to every `required`
Release's version, so adopting it satisfies every boundary that applies.

## Deterministic scenarios

- **Client already on the latest Release.** `ahead` is empty → **continue**.
- **Client ahead of every registered Release** (see
  `specs/decisions/version-comparison.md`). `ahead` is empty → **continue**.
  This is the same case as above, structurally: there is nothing with a
  greater version, so it doesn't matter whether the client matches or
  exceeds the latest known version.
- **Client behind only optional Releases.** Every Release in `ahead` is
  `optional` → **optional**.
- **Client behind exactly one required Release.** `ahead` contains one
  `required` Release → **required**.
- **Client behind multiple Releases with mixed policies.** If *any* member
  of `ahead` is `required`, the outcome is **required**, regardless of how
  many `optional` Releases are also in `ahead` or how they're interleaved.
- **A required Release followed by a newer optional Release.** The newer
  `optional` Release does not remove the boundary the `required` Release
  established — the outcome remains **required** as long as the client is
  behind that `required` Release. This is the scenario given as this
  decision's motivating example:

  ```
  Client:  1.0.0
  1.1.0 → optional
  1.2.0 → required
  1.3.0 → optional (latest)

  ahead = {1.1.0 (optional), 1.2.0 (required), 1.3.0 (optional)}
  → contains a required Release → outcome: required
  ```

  A client at `1.2.0` or higher, by contrast, has no `required` Release in
  `ahead` (only `1.3.0`, optional) → outcome: **optional**.

## Version/build ordering interactions

- Only version ordering (per `specs/decisions/version-comparison.md`)
  determines which Releases are in `ahead`. Build number is never used to
  decide whether a Release counts as "ahead" of the client, and never
  changes the resolved outcome.
- This mirrors `specs/domain/release.md`'s rule that build number is a
  tiebreaker between Releases sharing an identical version, not an axis
  of comparison against a client's reported state.

## Invariants

- The outcome is a pure function of (client version, current non-revoked
  Release set) — never of evaluation time, request order, or how the
  Releases were published (`specs/protocols/release-publishing.md`'s
  concurrency guarantees depend on this remaining true).
- A `required` boundary, once established by a Release, cannot be
  weakened by any later Release with a lower-or-equal-strength policy.
  Only revoking the `required` Release itself (`specs/domain/release.md`'s
  Revocation) removes it from consideration, since evaluation only
  considers non-revoked Releases.
- Revoking a Release is therefore the only way a previously `required`
  boundary stops applying — this is a consequence of the existing
  Revocation rule, not a new rule introduced here.
