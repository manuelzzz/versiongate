# Protocol: Release Publishing

## Purpose

The Release Publishing protocol defines the conceptual interaction between
a **publisher** — typically a CI/CD pipeline — and VersionGate, when a new
[[release]] (see `specs/domain/release.md`) is published for an
[[application]] (see `specs/domain/application.md`).

It defines what a publisher must provide, what guarantees VersionGate
offers in return, and how the interaction behaves under network failure
and retries — not the transport or wire format. This document does not
define HTTP endpoint paths, request/response schemas, or database
structures; it defines the protocol's semantics so those can be designed
consistently later.

## Publisher identity

- A publisher is not a first-class domain concept with its own identity in
  VersionGate — what VersionGate knows about a publish request is the
  [[project]] and Application it targets, established through
  authentication (below).
- VersionGate does not distinguish "which CI/CD system" made a request; a
  publisher is simply "whoever holds a valid token scoped to this
  Project." Attribution beyond that (which pipeline, which commit, which
  human triggered it) is the publisher's own concern, optionally carried
  as descriptive release metadata, not a protocol requirement.

## Authentication through project-scoped tokens

- Every publish request is authenticated using an API Token scoped to
  exactly one Project (see `specs/domain/project.md`'s "Relationship with
  API Tokens").
- A publish request may only create Releases for Applications belonging to
  the token's Project. A token must never be usable to publish a Release
  under a different Project, regardless of how the request identifies the
  target Application.
- This document does not define how tokens are issued, formatted, or
  validated — only that a valid, Project-scoped token is a precondition
  for every publish request.

## Release metadata

A publish request carries the information needed to create one Release
(see `specs/domain/release.md` and `specs/domain/version.md` for the
underlying concepts):

- the target Application (implicitly scoped to the authenticated Project);
- version (`MAJOR.MINOR.PATCH`);
- build number;
- update policy (`optional` or `required`) — which may be supplied
  directly, or derived upstream by the publisher using the Commit Metadata
  Protocol (`specs/protocols/commit-metadata.md`); VersionGate's publishing
  protocol only cares that a resolved policy value arrives with the
  request, not how the publisher arrived at it.

VersionGate does not accept or store a binary artifact as part of this
metadata — VersionGate does not distribute binaries (see
`specs/domain/release.md`).

## Idempotency

- A Release is uniquely identified, within its Application, by the pair
  (version, build number) — this is the same invariant defined in
  `specs/domain/release.md` and `specs/domain/version.md`.
- That pair is also the **idempotency key** for publish requests: a
  publish request for a (version, build number) that has already been
  successfully published, with identical release metadata, must be safe
  to repeat — it must not create a second Release, and it must not error
  in a way that implies something went wrong.
- If a publish request repeats a (version, build number) that already
  exists but with *different* metadata (e.g. a different update policy),
  that is a conflict, not an idempotent retry — see Duplicate publishing
  below.
- Idempotency is what makes the protocol safe under retries: a publisher
  does not need to track, on its own side, whether a previous attempt
  actually reached VersionGate before failing.

## Retry behavior

- CI/CD pipelines may retry a publish request because of network failures
  (timeouts, dropped connections, ambiguous responses) without knowing
  whether the original request was received and processed.
- A publisher retrying the exact same publish request (same Application,
  version, build number, and metadata) must be able to do so any number of
  times and observe the same outcome as the original successful request —
  publishing the same Release twice must not create inconsistent state,
  duplicate Releases, or duplicate side effects.
- Because ordering between concurrent or retried requests is not
  guaranteed, VersionGate must not rely on request arrival order to decide
  outcomes — this mirrors `specs/domain/release.md`'s rule that "latest
  release" is derived from version comparison, not from write order.
- A publisher should treat "no confirmed response" the same way regardless
  of cause (timeout, connection drop, 5xx) — the safe action is always to
  retry the identical request, never to alter it in an attempt to "make
  it work."

## Duplicate publishing

- A publish request that exactly matches an already-published Release
  (same Application, version, build number, and metadata) is a successful
  no-op from the publisher's point of view: the Release exists, and the
  outcome is indistinguishable from having just published it.
- A publish request that reuses an existing (version, build number) pair
  under the same Application but with *different* metadata (e.g. a
  different update policy) must be rejected as a conflict. VersionGate
  must not silently overwrite the existing Release — see
  `specs/domain/release.md`'s Immutability rule: publishing is not an
  upsert.
- Resolving such a conflict is a deliberate, explicit action outside this
  protocol (e.g. revoking the existing Release and publishing a corrected
  one) — the publish request itself must fail clearly rather than guess
  at intent.

## Validation expectations

Before a Release is created, a publish request must be validated against
the domain rules already defined for its constituent concepts:

- the target Application must exist, be active, and belong to the
  authenticated Project (`specs/domain/application.md`);
- version and build number must be well-formed, per
  `specs/domain/version.md` (non-negative integer components; malformed
  values are rejected outright, never coerced);
- update policy must be one of the values defined by
  `specs/protocols/commit-metadata.md` (`optional` or `required`);
  anything else is invalid.
- the (version, build number) pair must not already exist for this
  Application with different metadata (see Duplicate publishing).

A request that fails any of these checks must be rejected without
creating a Release or any partial state — validation failure has no side
effects.

## Publishing lifecycle

1. Publisher authenticates using a Project-scoped token.
2. Publisher submits release metadata for a target Application.
3. VersionGate validates the request (see Validation expectations).
4. VersionGate checks the (version, build number) pair against existing
   Releases for that Application:
   - if no such Release exists, it is created — publishing succeeds.
   - if an identical Release already exists, the request is treated as an
     idempotent no-op — publishing succeeds without creating anything.
   - if a conflicting Release already exists (same pair, different
     metadata), the request fails as a conflict.
5. Once created, a Release is immediately eligible for policy evaluation,
   subject only to the ordering rules in `specs/domain/release.md`
   (i.e. it becomes "latest" only if its version is in fact the highest
   among non-revoked Releases — publishing does not force a Release to be
   treated as latest by any other means).

Publishing is a single atomic step from the publisher's perspective: there
is no partially-published Release, and no separate step to "finalize" or
"activate" a Release after creation (revocation, defined in
`specs/domain/release.md`, is a distinct, later, and separate action).

## Expected failure behavior

- **Authentication failure** (missing/invalid/wrong-scope token): the
  request must fail before any validation of release metadata occurs, and
  must not reveal whether the target Application or Project exists.
- **Validation failure** (malformed version/build number, invalid update
  policy, unknown or inactive Application): the request fails with no
  side effects, as above.
- **Conflict** (same version/build number, different metadata): the
  request fails distinctly from a validation failure, so a publisher can
  tell "you sent bad data" apart from "this exact release already exists
  differently" — the latter usually requires a human decision, not a
  retry.
- **Transient/infrastructure failure** (timeout, network error, ambiguous
  response before a response is confirmed): the publisher cannot assume
  success or failure. The correct action is always to retry the identical
  request; idempotency guarantees this is safe.
- In all failure cases, VersionGate must leave no partial or inconsistent
  state — a failed publish attempt, retried or not, must never result in
  two Releases for the same (Application, version, build number), and
  must never result in a Release with metadata that doesn't match any
  single request the publisher actually sent.
