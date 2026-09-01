# Domain Concept: Release

## Purpose

A **Release** represents a published version of an [[application]] for its
platform — a concrete, identifiable point in that Application's release
history that VersionGate can evaluate update policy against.

VersionGate does not distribute binaries. A Release is metadata: it
describes *that* a given version exists, what update behavior it implies
for clients, and how it relates to other Releases — never the artifact
itself.

## Relationship with Application

- A Release always belongs to exactly one [[application]] (see
  `specs/domain/application.md`); there is no Release without an owning
  Application.
- Because platform is fixed on the Application, a Release does not carry
  its own separate platform field — it inherits platform identity from its
  Application. A "release" for the same product on a different platform is
  a Release under a different Application.
- All Releases for a given Application are directly comparable to one
  another (same platform, same versioning context). Releases across
  different Applications are never compared to each other.

## Platform

- Platform is not a property of Release — it is inherited from the owning
  Application. This avoids representing an impossible state (a Release
  whose platform disagrees with its Application's).

## Version

- A Release carries a version (see `specs/domain/version.md` for the
  version concept and its comparison rules). The version is what expresses
  *how new* a Release is relative to others.
- A Release's version must be meaningful within the versioning scheme its
  platform uses, and must be comparable to every other Release's version
  under the same Application.

## Build number

- Beyond the user-facing version, a Release carries a build number (or
  equivalent platform build identifier) — a monotonically distinguishing
  value used by platform stores and CI/CD systems that may not itself
  change the user-facing version string.
- Build number disambiguates Releases that could otherwise share a version
  (e.g. two builds shipped under the same version string during a staged
  rollout), but it does not replace version as the primary axis of
  comparison for update policy — see Ordering below.

## Update policy

- A Release carries the update policy information clients need to
  evaluate against it: at minimum, whether adopting this Release (or newer)
  is optional (notify) or mandatory (require update), relative to a
  client's reported version.
- The Release is where policy *data* lives; the *evaluation* of a specific
  client request against that data is a separate concern (policy
  evaluation logic), not part of what defines a Release.

## Release lifecycle

- A Release is created (published) with its version, build number, and
  policy data fixed at creation.
- A Release does not have a mutable "in progress" state — publishing is a
  single, deliberate, atomic act. VersionGate does not model draft or
  partially-published Releases.
- After publication, a Release can transition to **revoked** (see below)
  but its identifying and descriptive data does not change.

## Immutability

- Once published, a Release's version and build number are immutable —
  they are its identity and must never change.
- Update policy fields attached to a Release should be treated as
  effectively immutable after publication. If a policy mistake needs
  correcting, the correct action is to publish a new Release (or revoke
  the incorrect one), not to silently mutate history that clients and
  audit trails already rely on.
- Immutability is what makes a Release trustworthy as an audit record: two
  observers looking at the same Release at different times must see the
  same facts.

## Revocation or deactivation

- A Release can be **revoked** to stop it from being considered by policy
  evaluation (e.g. a build was pulled from the store, or found to be
  broken) without deleting its historical record.
- Revocation is not deletion: the Release still exists for audit and
  history purposes, but is excluded from "latest release" and from update
  decisions going forward.
- A revoked Release must never be resurrected implicitly (e.g. by
  becoming "latest" again through some recalculation) — un-revoking, if
  ever supported, must be an explicit act.
- Revocation does not affect other Releases' identity or ordering; it only
  removes the revoked Release from consideration.

## Ordering

- Releases for an Application are ordered by **version**, using the
  comparison rules defined by the platform's versioning scheme (see
  `specs/domain/version.md`) — never by database insertion time, creation
  timestamp, or any other artifact of when the record was written.
- "Latest release" means: the highest-ordered, non-revoked Release for an
  Application, by version comparison. It is a derived fact computed from
  existing Releases, not a flag stored on one particular record and not
  something that depends on arrival order.
- This matters directly for concurrency: two CI/CD pipelines may publish
  Releases out of chronological order (e.g. a delayed pipeline publishes
  version 2.3.0 after a faster pipeline has already published 2.4.0).
  Because ordering is by version and not by publish time, this does not
  corrupt "latest" — 2.4.0 remains latest regardless of which Release was
  written to storage first or second.
- Consequently, publishing a Release with a lower version than an
  already-published Release is a valid, expected occurrence (e.g. a hotfix
  for an older line, or exactly the out-of-order CI scenario above) — it
  does not become "latest," but it is not an error in itself.

## Duplicate releases

- Two Releases under the same Application must not share the same version
  and build number combination — that would be the same Release published
  twice, not two distinct ones.
- A version may legitimately be re-associated with a new build number only
  in schemes where the platform allows multiple builds under one version
  string (see Build number above); the (version, build number) pair is
  what must stay unique, not version alone in every platform scheme.
- Attempting to publish a duplicate must be rejected rather than silently
  overwriting or ignoring the existing Release — publishing is not an
  upsert.

## Auditability

- Every Release is a permanent record of a point in the Application's
  history: what was published, when, with what policy. Because of
  immutability, this record is trustworthy without needing to guard
  against later mutation.
- Revocation is itself an auditable event: a revoked Release should be
  distinguishable from one that was never published, and it should be
  possible to tell that it *was* once eligible for evaluation and later
  withdrawn.
- The audit trail is the sequence of publish and revoke events over time,
  not a single mutable "current state" per Release — this is what lets
  operators reconstruct what any client would have been told to do at any
  point in the past.
