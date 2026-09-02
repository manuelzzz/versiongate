# Decision: Authentication Strategy

## Status

Accepted — initial decision, revisit if a concrete requirement outgrows it.

## Context

VersionGate has two distinct access patterns with very different trust
characteristics:

1. **CI/CD pipelines publishing Releases** (see
   `specs/protocols/release-publishing.md`) — a trusted, operator-controlled
   system, performing write operations that create durable records.
2. **Mobile applications checking update status** — an untrusted client
   population, performing read-only policy evaluation, at potentially high
   volume (see `specs/decisions/database.md`'s read-volume analysis).

These two patterns cannot share one authentication model:

- Write access must be restricted — publishing a Release is a meaningful,
  auditable action (`specs/domain/release.md`'s Auditability section) and
  must only be performed by someone authorized to do so for a given
  [[project]].
- Mobile clients **cannot safely store secrets**. Anything embedded in a
  distributed mobile binary — an API key, a signing secret, anything — can
  be extracted by decompiling the app. Treating such a value as a
  confidential credential would be a false security guarantee, not a real
  one.
- The data a mobile client needs (whether to notify or require an update)
  is not sensitive in the way write access is: it's information the
  Application's own users are, by definition, entitled to learn by simply
  running the app. This asymmetry is what allows the two access patterns
  to be treated differently rather than forcing one uniform scheme onto
  both.

## Decision

VersionGate adopts a **two-tier access model**:

1. **Write operations** (publishing Releases, managing Projects,
   Applications, and Tokens) require a **Project-scoped API Token**,
   presented as a bearer credential over TLS. This is the same token
   concept already established in `specs/domain/project.md`'s
   "Relationship with API Tokens" and used by the publishing protocol in
   `specs/protocols/release-publishing.md`.
2. **Read operations used for update evaluation** (a mobile client asking
   "should I update?") require **no secret credential**. The request is
   scoped by the Application's public identifier
   (`specs/domain/application.md`'s Application identity) — a value that
   is not treated as confidential and is safe to embed in a distributed
   mobile binary, because it grants no write capability and exposes only
   the Application's own (non-sensitive) update-policy information.

No other authentication mechanism (per-install keys, mobile-embedded
secrets, session-based auth, human user accounts) is introduced at this
stage.

## Rationale

- **Project-scoped tokens for write access give real least privilege.** A
  token compromised from one CI/CD pipeline exposes at most one Project's
  data — it cannot read or write another Project's Applications, Releases,
  or Tokens (per `specs/domain/project.md`'s isolation invariant). This
  bounds the blast radius of any single leaked credential without needing
  a more granular (and currently unjustified) permission system.
- **Public read access is the honest response to "mobile clients can't
  store secrets."** Requiring a secret that cannot actually be kept secret
  would not improve security — it would only create the illusion of
  access control while adding complexity (issuing, embedding, and
  rotating per-install credentials) for no real protection. Since the
  information being served is not sensitive, no credential is needed to
  read it.
- **Scoping reads by Application identifier, not by secret, keeps the
  boundary honest about what it actually protects.** The identifier
  controls *what data a request is asking about*, not *whether the
  requester is authorized* — because there is nothing to authorize for a
  read that exposes only an Application's own public update policy.
- **A single token type keeps the model simple**, consistent with
  `.rules/architecture.md`'s guidance to avoid premature abstraction:
  there is one real actor type performing writes today (a CI/CD pipeline
  with a Project-scoped token), so there is no current need for
  role-based access control, per-user accounts, or fine-grained
  per-action permissions. Those can be introduced later if a concrete
  requirement — e.g. multiple humans/teams managing one Project — emerges.

## Consequences

- Every write-capable request must carry a valid Project-scoped token;
  VersionGate must be able to reject requests with a missing, invalid, or
  wrong-scope token before any write is attempted (already required by
  `specs/protocols/release-publishing.md`'s failure behavior).
- Read endpoints for update evaluation must be designed to work correctly
  and safely with no authentication at all — meaning they must never leak
  anything beyond the requesting Application's own public update-policy
  data, regardless of what identifier is supplied.
- Because Application identifiers are not secret, VersionGate must not
  treat "knows the Application identifier" as proof of anything beyond
  "is asking about this Application." No write or administrative action
  may ever be gated on that identifier alone.
- Operators are responsible for issuing, distributing, and safeguarding
  Project-scoped tokens for their own CI/CD systems; VersionGate's
  obligation is to make tokens easy to rotate and revoke (below), not to
  prevent operators from mishandling them.
- How the very first Project and its first token come to exist — before
  any token is available to authenticate a creation request — is a
  separate bootstrap problem, resolved in
  `specs/decisions/bootstrap-mechanism.md`.
- If a future requirement introduces multiple trust levels among write
  operations (e.g. a human admin vs. an automated pipeline) or per-user
  accountability, that will require a new or extended decision — this
  document intentionally does not anticipate that.

### Token storage security

- Tokens must be stored in a form that is not directly usable if the
  underlying storage is compromised — i.e. VersionGate stores a verifier
  derived from the token (comparable to password hashing practice), not
  the token itself in plain form.
- The full token value is only ever shown to the operator once, at
  creation time. VersionGate does not retain the ability to display an
  existing token again.
- This document does not choose a specific hashing algorithm or
  implementation — that is an implementation detail, not part of the
  durable decision.

### Token rotation

- Rotating a token must not require downtime for the systems using it: an
  operator must be able to issue a new token for a Project while the old
  one remains valid, update their CI/CD configuration, and only then
  revoke the old token.
- Rotation is an operator-initiated action, not automatic or
  time-forced at this stage (no mandatory expiry is introduced — see
  Alternatives considered).

### Token revocation

- Revocation must take effect immediately: a revoked token must be
  rejected on the next request, with no grace period.
- Revoking one token must not affect other tokens issued for the same or
  a different Project — tokens are independent of one another.
- Revocation is the primary incident-response mechanism for a leaked
  token; it must be simple and fast enough for an operator to use under
  pressure.

### Least privilege

- A token's authority is bounded to exactly one Project — never global
  administrative access, and never another Project's data (restated from
  `specs/domain/project.md`).
- Within its Project, this decision does not further subdivide a token's
  authority (e.g. "publish-only" vs. "manage Applications" tokens). That
  would be over-engineering authorization ahead of a concrete need;
  Project scoping is judged sufficient least-privilege granularity for
  the current, single-actor-type access pattern.

## Threat considerations

- **Token leakage from CI/CD systems** (logs, misconfigured environment
  variables, compromised pipeline). Mitigated by Project-scoped blast
  radius, hashed-at-rest storage (a database leak alone doesn't yield
  usable tokens), and fast revocation/rotation for incident response.
- **Mobile app decompilation.** Mitigated by design: nothing confidential
  is ever embedded in the mobile client, so decompilation yields no
  usable credential — there is no secret to extract.
- **Interception in transit.** Tokens are bearer credentials; transport
  must be encrypted (TLS) so a token cannot be captured on the network.
  This document assumes TLS is a baseline deployment requirement, defined
  further in `docs/` deployment guidance rather than here.
- **Token replay after theft.** Without mandatory expiry, a stolen token
  remains valid until explicitly revoked. This is an accepted tradeoff
  (see Alternatives) mitigated by making revocation fast and by keeping
  blast radius scoped to one Project; operators who want time-boxed
  credentials can rotate manually on their own cadence.
- **Abuse/flooding of public read endpoints**, since they require no
  credential. This is a legitimate concern but is an availability/rate
  -limiting problem, not an authentication problem — it does not change
  this decision and should be addressed separately (e.g. at a reverse
  proxy or infrastructure layer) if and when it becomes a real issue.
- **Read-endpoint data exposure.** Because reads are unauthenticated by
  design, any data returned from update-evaluation reads must be treated
  as public. VersionGate must never allow a read endpoint to return
  anything beyond an Application's own update-policy information (never
  other Applications', other Projects', or token/administrative data).

## Alternatives considered

- **Per-application tokens instead of per-project.** Would offer finer
  granularity but adds real operational overhead (issuing/rotating a
  token per Application instead of once per Project) without a concrete
  requirement driving it — most CI/CD setups publish for all Applications
  in a Project from the same pipeline. Rejected as unnecessary granularity
  for now; can be revisited if a real multi-team-per-project need arises.
- **Embedding a secret/API key in the mobile client for read access.**
  Rejected outright: this contradicts the stated constraint that mobile
  clients cannot safely store secrets. It would add implementation and
  distribution complexity while providing no real security benefit, since
  the "secret" would be trivially extractable.
- **mTLS for CI/CD authentication.** Would be strong, but is a
  significant operational burden (certificate issuance, distribution,
  rotation) for self-hosted operators, most of whom are already well
  served by a simple bearer token over TLS. Rejected for now as
  disproportionate to the actual threat model; may be reconsidered if
  operators with stronger requirements request it.
- **OAuth2/JWT with short-lived access tokens and refresh tokens.**
  Rejected as unnecessary complexity for a machine-to-machine, single
  -actor-type write path. This pattern is built for delegated
  user-authorization flows VersionGate doesn't have yet; adopting it now
  would be solving a problem VersionGate doesn't have, at the cost of
  operator-facing complexity.
- **Mandatory token expiry.** Considered as a way to bound replay risk
  automatically. Rejected for the initial decision because it forces
  operators into credential-refresh automation before there's a concrete
  driver for it, and manual rotation combined with fast revocation
  already gives operators the tools to bound risk themselves. This can be
  added later as an optional feature without contradicting this decision.
- **Requiring authentication for the update-check read endpoint** (e.g. a
  per-install or per-device key). Rejected because the data being
  protected isn't sensitive enough to justify the complexity, and because
  any credential a mobile client could present would not be a real secret
  in the first place — see Threat considerations.
