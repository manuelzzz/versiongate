# Protocol: Update Check

## Purpose

The Update Check protocol defines the conceptual interaction between a
mobile client and VersionGate when a client asks whether it should
continue normally, be notified of an available update, or be required to
update.

This is a read-only, unauthenticated interaction (see
`specs/decisions/authentication.md`) — the counterpart to
`specs/protocols/release-publishing.md`, which defines how a Release gets
into VersionGate in the first place.

This document defines the interaction contract: what a client provides,
what VersionGate evaluates, and what a client receives back. It does not
define HTTP paths or wire-format details — those are centralized in
`specs/protocols/http.md`. It does not define the evaluation algorithm
itself — that is a domain concern, defined in
`specs/domain/update-policy.md`.

## Application identity

- A client identifies which [[application]] it belongs to using the
  Application's public identifier (see `specs/domain/application.md`'s
  Application identity) — the same stable, immutable identifier used
  throughout the domain.
- This identifier is not a secret (see `specs/decisions/authentication.md`)
  and is expected to be embedded directly in the mobile client.

## Platform

- Platform is not a separate field a client provides. As established in
  `specs/domain/application.md` and `specs/domain/release.md`, platform is
  fixed per Application — an iOS build and an Android build of the same
  product are already distinct Applications with distinct identifiers. The
  Application identifier alone fully determines platform.

## Client version representation

- The client reports its own current version, in the same
  `MAJOR.MINOR.PATCH` form defined in `specs/domain/version.md`.
- This is the primary input to evaluation — see What VersionGate
  evaluates, below.

## Client build number representation

- The client may report its current build number, in the same form
  defined in `specs/domain/version.md`.
- Build number is accepted for descriptive purposes and to keep the
  request shape symmetric with how Releases are published (version +
  build number), but it does not currently influence the update decision
  — evaluation is driven by version alone (see
  `specs/domain/update-policy.md`). This keeps the protocol simple and
  avoids inventing a comparison rule between a client-reported build
  number and a published Release's build number that no current
  requirement calls for.

## What VersionGate evaluates

- VersionGate compares the client's reported version against the
  Application's release history (all non-revoked Releases for that
  Application) and resolves an outcome using the algorithm defined in
  `specs/domain/update-policy.md`.
- Evaluation is a pure function of the client's reported version and the
  current set of non-revoked Releases — never of request time, request
  order, or how many times a client has asked before.

## Possible update actions

The outcome of evaluation is exactly one of:

- **Continue** — the client is already up to date; no action needed.
- **Optional update available** — a newer Release exists, but adopting it
  is not mandatory.
- **Required update** — the client must update; at least one Release
  ahead of the client's version is mandatory, per
  `specs/domain/update-policy.md`.

These map directly to the two Release-level policy values defined in
`specs/domain/release.md` (`optional`, `required`), plus the "nothing
ahead" case, which is not itself a policy value but a resolved state.

## What metadata the client receives

- The resolved action (continue / optional / required).
- The latest applicable Release's version (and, for descriptive purposes,
  its build number), so the client can inform its user what version is
  available.
- Nothing else. In particular: no binary, no download URL, no store
  metadata — VersionGate does not distribute binaries
  (`specs/domain/release.md`), so it has none of that to return.

## Behavior for unknown Applications

- A request identifying an Application that does not exist must be
  answered with a distinct "not found" outcome — never silently treated
  as "continue." Since Application identifiers are not confidential (see
  `specs/decisions/authentication.md`), returning a clear not-found
  response does not leak sensitive information; it is simply necessary
  feedback for a misconfigured client.
- A request identifying an Application whose owning Application or
  [[project]] is deactivated must not produce an evaluation result,
  per the invariants in `specs/domain/application.md` and
  `specs/domain/project.md`. Conceptually this is treated the same as
  "not found" from the evaluation's point of view: no policy decision is
  returned. Whether this is distinguishable from a true not-found at the
  HTTP level is a wire-format decision deferred to
  `specs/protocols/http.md` / implementation, not part of this contract.

## Behavior for unknown or invalid client versions

- A client version that does not parse as a well-formed version, per
  `specs/domain/version.md`'s invalid-version rule, must be rejected
  outright — never coerced, never defaulted to any of the three update
  actions. This is a validation failure, not an evaluation outcome.
- A client version that is well-formed but higher than any registered
  Release's version is a legitimate, expected case (see
  `specs/decisions/version-comparison.md`) and resolves to **Continue** —
  it is never an error.
- An Application with no Releases published yet has no history to compare
  against; a client asking about it resolves to **Continue** — there is
  vacuously nothing ahead of the client to notify or require.
