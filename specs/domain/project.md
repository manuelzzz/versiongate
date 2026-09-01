# Domain Concept: Project

## Purpose

A **Project** is the top-level organizational boundary in VersionGate. It
represents a single product or team's workspace: everything else in the
domain — Applications, Releases, Versions, API Tokens — exists *within* a
Project.

A Project exists to answer one question for every other concept: "whose is
this, and what else does it belong with?" It is the unit of isolation and
grouping, not a domain concept with behavior of its own (it does not
evaluate policies or hold release data itself).

## Responsibilities

A Project is responsible for:

- Providing a namespace that groups a related set of Applications (e.g. all
  the apps belonging to one product or organization).
- Being the scope at which access is granted: API Tokens are issued for a
  Project, not for VersionGate as a whole.
- Giving operators a natural unit to reason about, list, and manage
  ("show me everything under this Project").

A Project is **not** responsible for:

- Storing or evaluating update policies (that belongs to Release/Version
  and their policy rules).
- Knowing about specific platforms, binaries, or store metadata (that
  belongs to Application/Release).
- Authentication mechanics (that belongs to the API Token concept) — a
  Project only defines the scope a token is bound to.

## Ownership boundaries

- A Project owns zero or more Applications. An Application belongs to
  exactly one Project.
- A Project owns zero or more API Tokens. An API Token is scoped to exactly
  one Project.
- Ownership is exclusive: an Application or API Token cannot be shared
  across multiple Projects. Moving an Application to a different Project is
  a deliberate, explicit operation, not an implicit side effect.
- Everything nested under an Application (Releases, Versions) is
  transitively scoped to the Application's Project, but a Project does not
  reach into or manage that data directly — it delegates to Application.

## Relationship with Applications

- One Project has many Applications (e.g. "Acme Inc." as a Project might
  contain "Acme iOS" and "Acme Android" as separate Applications).
- An Application always belongs to exactly one Project; there is no such
  thing as an orphaned Application.
- The Project boundary is what allows two different Projects to have
  Applications with the same name without conflict — uniqueness of
  Application identifiers is scoped to the Project, not global.

## Relationship with API Tokens

- API Tokens are issued within the scope of a single Project and can only
  authorize actions on that Project's Applications and their data.
- A token scoped to one Project must never grant access to another
  Project's data. This is a hard isolation boundary, not a configurable
  default.
- Revoking or rotating tokens is a Project-level concern: an operator
  manages the tokens for a Project without affecting other Projects.

## Lifecycle considerations

- A Project is created explicitly by an operator before any Application can
  be created under it.
- A Project can be renamed or have its descriptive metadata updated without
  affecting the identity or data of the Applications it contains.
- A Project can be deactivated/archived to stop it (and everything under
  it) from serving policy evaluations, without necessarily deleting its
  data — this supports decommissioning a product without losing history.
- Deleting a Project is a destructive operation that cascades to everything
  it owns (Applications, and transitively Releases, Versions, and Tokens).
  Because of this, deletion should be an explicit, deliberate act — not an
  implicit consequence of another operation.

## Invariants

- A Project has a unique identifier within VersionGate.
- A Project cannot exist without a name/identifier assigned at creation.
- An Application, Release, Version, or API Token always resolves to exactly
  one owning Project — there is no shared or ambiguous ownership.
- A deactivated Project must not allow policy evaluation for any
  Application it owns, even if the Application itself is otherwise active.
- Data belonging to one Project must never be visible or mutable through
  another Project's scope, regardless of how a request is authenticated.

## Explicitly NOT part of the Project concept

- **User/team membership and permissions model** — who *within an
  organization* can manage a Project is an authorization concern, not part
  of what a Project fundamentally is. (May be defined separately if/when
  VersionGate needs multi-user access control.)
- **Billing, quotas, or plan limits** — VersionGate is self-hosted; a
  Project is not a billing unit.
- **Application platform details** (iOS/Android, bundle identifiers, store
  metadata) — those belong to the Application concept.
- **Release and Version data, or policy evaluation logic** — a Project is a
  grouping boundary, not where policy decisions live.
- **Token issuance mechanics** (how tokens are generated, hashed, or
  validated) — the Project defines the *scope* a token belongs to, not how
  tokens work.
