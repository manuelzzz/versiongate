# Decision: Initial Project and Token Bootstrap

## Status

Accepted — initial decision, revisit if a concrete requirement outgrows it.

## Decision

VersionGate is initialized through a **CLI-based bootstrap mechanism**.
An operator uses the VersionGate CLI, running against the same
application/domain layer and persistence infrastructure as the API, to
create the initial [[project]] and its first API Token
(`specs/decisions/authentication.md`).

No publicly exposed, unauthenticated HTTP endpoint is used to create
Projects or Tokens. The CLI is conceptually similar to:

```bash
versiongate bootstrap
```

or, once at least one Project exists:

```bash
versiongate project create
```

The exact command structure, flags, and UX are implementation detail and
are not fixed by this decision — what's fixed is that bootstrap happens
through an operator-invoked CLI against the domain layer, not through an
unauthenticated network-facing endpoint.

## Context

Every write operation in VersionGate requires a Project-scoped API Token
(`specs/decisions/authentication.md`). But creating a Project — and
issuing its first Token — is itself a write operation. This is a genuine
bootstrap problem: there is no token yet capable of authorizing the
request that would create the first token.

VersionGate is self-hosted (`specs/decisions/database.md`'s context) and
has no user/team account model (`specs/domain/project.md` explicitly
excludes membership/permissions). The operator running the software *is*
the trust boundary — there is no separate "sign up" flow to lean on, and
no existing decision anticipates one.

## Rationale

- **A CLI avoids introducing a privileged, unauthenticated HTTP endpoint.**
  Any network-facing endpoint capable of creating Projects or Tokens
  without a prior credential is a meaningful attack surface, especially
  for self-hosted operators who may not tightly control network exposure
  by default. A CLI that must be invoked with access to the host (or its
  database connection) doesn't add that surface.
- **It fits VersionGate's self-hosted, non-GUI philosophy.** VersionGate
  has no web dashboard (deliberately out of scope, per prior MVP
  discussion) and already leans on the standard library and simple
  tooling over frameworks (`.rules/architecture.md`'s
  avoiding-unnecessary-frameworks rule). A CLI is consistent with how the
  rest of the project is being built, rather than introducing a new
  interaction paradigm just for bootstrap.
- **It gives an explicit, operator-controlled initialization step.**
  Bootstrap is not implicit (no auto-generated credentials appearing in
  logs on first startup) and not silently defaulted — the operator takes a
  deliberate action and receives the resulting credential directly,
  consistent with `specs/decisions/authentication.md`'s rule that a token
  is only ever shown once, at creation.
- **The CLI reuses the same domain layer as the API, rather than
  bypassing it.** This keeps the dependency-direction and
  domain-independence rules in `.rules/architecture.md` intact: the CLI is
  another caller of the application/domain layer, not a separate path that
  writes to storage directly and risks violating the same invariants
  (uniqueness, scoping) the API enforces.
- **It naturally supports future automation and scripting** (e.g.
  infrastructure-as-code provisioning a VersionGate instance) without
  requiring a decision reversal — a CLI command is scriptable by
  construction.

## Consequences

- Operators need shell/process access to the environment VersionGate runs
  in (or a way to invoke its CLI against the target database) to perform
  initial setup. This is an accepted operational requirement, consistent
  with self-hosted deployment expectations already established in
  `specs/decisions/database.md`.
- There is no self-service way to create a new Project without CLI (or
  host) access. If a future requirement calls for provisioning Projects
  without direct host access (e.g. a multi-tenant hosted offering), that
  would need its own decision — this one deliberately does not anticipate
  it.
- Public documentation (`docs/`) must cover the CLI bootstrap flow as part
  of installation/getting-started guidance once the CLI exists.
- The application/domain layer must expose the "create Project" and
  "issue Token" capabilities in a form usable by both the CLI and the HTTP
  admin endpoints (see the Project/Application management work in the
  implementation backlog), rather than the CLI reimplementing that logic
  separately.
- Token storage and handling for CLI-issued tokens follow the same rules
  as any other token (`specs/decisions/authentication.md`): hashed at
  rest, shown once, independently revocable.

## Alternatives considered

- **Environment-provided bootstrap credentials** (e.g. an admin token read
  from an environment variable at startup). Rejected: this pushes secret
  management onto process configuration and deployment tooling, is easy
  to leave in place longer than intended, and gives a less explicit,
  less auditable initialization moment than an operator deliberately
  running a command.
- **Automatic token generation on first startup.** Rejected: implicit
  credential creation risks the value ending up in logs or being missed
  entirely, and doesn't give the operator a deliberate control point for
  *when* initialization happens — VersionGate might start (e.g. in a
  container orchestrator) well before an operator is watching.
- **Administrative HTTP endpoint** (e.g. an unauthenticated or
  single-use-token-protected `/bootstrap` route). Rejected as the
  privileged-unauthenticated-endpoint risk described in Rationale;
  also inconsistent with the Project-scoped-token-only authentication
  model in `specs/decisions/authentication.md`, which has no concept of
  a "super-admin" credential today.
- **Database seeding** (an operator manually inserting rows). Rejected:
  bypasses domain validation and invariants entirely (uniqueness,
  hashing, scoping), and is fragile across schema changes — exactly what
  `.rules/architecture.md`'s domain-independence rule is meant to prevent
  infrastructure from doing.
- **CLI bootstrap** — selected, for the reasons above.
