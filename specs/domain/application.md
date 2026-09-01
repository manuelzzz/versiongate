# Domain Concept: Application

## Purpose

An **Application** represents a single distributable mobile application
managed by VersionGate — e.g. "Acme iOS" or "Acme Android". It is the
concept that Releases and Versions are evaluated against: when a client
asks "should I update?", it is asking on behalf of a specific Application.

An Application is where identity (what app is this?) and platform (what
kind of client is asking?) are established. It does not itself decide
update outcomes — that is the responsibility of Release/Version and their
policy rules (see `specs/domain/release.md`, `specs/domain/version.md`).

## Relationship with Project

- Every Application belongs to exactly one [[project]] (see
  `specs/domain/project.md`); there is no orphaned or project-less
  Application.
- An Application's identifier only needs to be unique within its owning
  Project, not globally across VersionGate — two different Projects may
  each have an Application identified the same way.
- A Project's deactivation affects every Application it owns (see
  Project's lifecycle rules); an Application cannot override or bypass
  that.
- Moving an Application between Projects is not a normal operation. If
  supported at all, it must be explicit and deliberate, not an implicit
  side effect of another action.

## Application identity

- An Application is identified by a stable, human-assigned identifier
  (a slug or key) that is unique within its Project and does not change
  over the Application's lifetime — clients and integrations depend on it
  staying constant.
- An Application also has a display name, which is descriptive metadata
  and may change freely without affecting identity.
- Identity is what Releases and Versions attach to, and what API requests
  reference — it is the stable handle for everything downstream.

## Platform considerations

- Every Application targets exactly one platform (initially: **iOS** or
  **Android**). A single codebase distributed on multiple platforms is
  represented as multiple Applications (e.g. one for iOS, one for
  Android), each with its own identity, releases, and policies.
- This is a deliberate choice, not an oversight: iOS and Android have
  independent release cadences, version numbering, and store review
  processes, so update policy needs to be evaluated independently per
  platform.
- Platform is fixed at creation and does not change — an Application does
  not migrate from one platform to another.
- Do not design an abstraction layer for hypothetical future platforms
  (web, desktop, etc.) beyond a plain enumeration of "iOS" and "Android".
  Add real platform support when there's a concrete need, not
  speculatively.

## Relationship with Releases

- An Application has many Releases; a Release always belongs to exactly
  one Application.
- Releases represent the history and current state of what has shipped (or
  is shippable) for that Application; the Application itself holds no
  version or policy data directly — it delegates all of that to its
  Releases.
- Policy evaluation for a client request is always scoped to one
  Application: a request identifies the Application (and platform,
  implicitly, since platform is fixed per Application) it is asking about.

## Lifecycle

- An Application is created explicitly within a Project, with its
  identifier and platform fixed at creation time.
- Display name and other descriptive metadata can be updated freely.
- An Application can be deactivated to stop it from serving policy
  evaluations without deleting its history of Releases — useful for
  retiring an app or pausing it temporarily.
- Deleting an Application is destructive and cascades to its Releases and
  Versions. This should be an explicit, deliberate act, distinct from
  deactivation.

## Invariants

- An Application has an identifier that is unique within its owning
  Project and immutable after creation.
- An Application always belongs to exactly one Project.
- An Application always targets exactly one platform, fixed at creation.
- A deactivated Application (or one whose owning Project is deactivated)
  must not produce policy evaluation results.
- A Release always resolves to exactly one owning Application; there is no
  shared or ambiguous ownership.

## Constraints

- Platform is restricted to a known, closed set (iOS, Android) — not a
  free-form string. Adding a new platform is a deliberate domain change,
  not configuration.
- An Application cannot exist without an owning Project.
- Application identity (identifier) is immutable; only descriptive
  metadata (display name, etc.) is mutable.
- Cross-platform behavior for "the same app" is not modeled at the
  Application level — if a policy needs to reason across iOS and Android
  variants of one product, that reasoning happens above/outside the
  Application concept, not by merging their identities.
