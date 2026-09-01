# Decision: Initial Persistence Approach

## Status

Accepted — initial decision, revisit if a concrete requirement outgrows it.

## Decision

VersionGate will use **PostgreSQL** as its initial and only persistence
engine, run as a single self-hosted instance (no read replicas, no
sharding, no clustering) for the foreseeable initial phase of the project.

No additional datastore (cache, queue, search index — including Redis) is
introduced at this stage. If a concrete need for one emerges later, it
will be evaluated and documented as its own decision.

This decision covers *where and how* VersionGate persists data. It does
not define tables, schemas, or an ORM/query-building approach — those are
separate, later decisions.

## Context

VersionGate needs durable storage for the domain concepts already
specified: [[project]], [[application]], [[release]] (which embeds
version and build number), and API Tokens (see `specs/domain/project.md`,
`specs/domain/application.md`, `specs/domain/release.md`,
`specs/domain/version.md`).

Relevant requirements, as given:

- **Self-hosted deployment** — operators run VersionGate themselves, often
  with limited operational expertise or dedicated infrastructure staff.
  Whatever we depend on, they must be able to install, run, and maintain.
- **Operational simplicity** — fewer moving parts is better; every
  additional service is something an operator has to run, monitor, back
  up, and upgrade.
- **Reliability** — release and policy data is the whole point of the
  system; it must not be lost or corrupted.
- **Transactional consistency** — several invariants already specified
  depend on it directly: the (version, build number) uniqueness constraint
  per Application (`specs/domain/release.md`), and the idempotent
  publish/conflict semantics in `specs/protocols/release-publishing.md`
  (a publish request must atomically check-and-create a Release, with no
  window for a duplicate to slip in under concurrent/retried requests).
- **Indexing** — policy evaluation and "latest release" resolution
  (`specs/domain/release.md`) require efficient lookups and ordering by
  Application and by version; this needs real index support, not just
  key-value lookups.
- **Horizontal scaling of the stateless API** — the VersionGate API
  service itself is intended to be stateless and horizontally scalable;
  all instances must see the same, immediately consistent data, which
  means the datastore must be reachable over the network by multiple
  concurrent processes, not embedded per-instance.
- **Moderate write volume** — Releases are published by CI/CD pipelines,
  which is infrequent relative to reads (on the order of per-deploy, not
  per-request).
- **Potentially high read volume** — every client app checking for
  updates results in a policy-evaluation read; this can scale with end
  -user traffic and may become the dominant load pattern.

## Rationale

- **PostgreSQL satisfies transactional consistency directly.** ACID
  transactions and unique constraints let VersionGate enforce the
  (version, build number) invariant and the idempotent-publish/conflict
  behavior at the database level, rather than relying on application-level
  locking or accepting race conditions under retried/concurrent publish
  requests.
- **PostgreSQL satisfies the horizontal-scaling requirement.** It is a
  network-accessible, client-server database: any number of stateless API
  instances can connect to the same PostgreSQL instance concurrently, with
  a single consistent view of the data. This is a hard requirement the
  API's statelessness depends on, and it rules out purely embedded,
  single-process datastores as the *default* choice (see Alternatives).
- **PostgreSQL gives real indexing and query capabilities** needed for
  ordering Releases by version and efficiently resolving "latest release"
  per Application — capabilities that a simpler key-value store would
  require VersionGate to reimplement in application code.
- **A single instance keeps operational simplicity intact for now.**
  Reliability and horizontal read scaling can both be improved later
  (backups, replicas, connection pooling) without changing this decision's
  core: PostgreSQL as the engine. What's deferred is *topology*
  (single instance vs. replicated), not the technology choice.
- **This matches the "moderate write / potentially high read" profile.**
  PostgreSQL handles moderate write volume comfortably without any special
  tuning, and read-side scaling (indexes now, read replicas or caching
  later, only if actually needed) is a well-understood path to grow into
  if read volume becomes the bottleneck — without having to introduce that
  complexity today.
- **Self-hosting PostgreSQL is a well-worn, well-documented path.**
  Operators already run PostgreSQL for countless self-hosted applications;
  it is not an unusual operational burden to ask of them, and mature
  backup/restore tooling exists. This is a materially smaller ask than
  operating a distributed database or a multi-service persistence stack.

## Consequences

- VersionGate requires operators to provision and maintain a PostgreSQL
  instance (or use a managed PostgreSQL service) — this is a real
  operational dependency, not optional.
- All domain invariants that benefit from transactional guarantees
  (uniqueness, idempotent publishing) can be enforced at the database
  layer, simplifying the corresponding application logic.
- The initial deployment topology (single instance) means reliability
  currently depends on the operator's own backup/replication practices;
  VersionGate does not provide built-in high availability at this stage.
  This is an accepted tradeoff for initial simplicity, not a permanent
  ceiling — it can be revisited if reliability requirements grow.
- If read volume grows enough to matter, the natural next steps are
  read-optimized indexes, connection pooling, and — only if genuinely
  needed — read replicas or a caching layer in front of hot read paths
  (e.g. policy evaluation). None of that is built now; this decision
  intentionally defers it until there's a real signal it's needed.
- Domain code must remain independent of PostgreSQL specifics (see
  `.rules/architecture.md`'s dependency-direction rule): this decision
  fixes the infrastructure choice, not the domain model, which continues
  to depend only on interfaces it defines.

## Alternatives considered

- **SQLite.** Excellent operational simplicity (zero separate process,
  single file, trivial backup) and would be a strong fit for a
  single-instance, embedded deployment mode. Rejected as the *initial
  default* because it does not satisfy the stated horizontal-scaling
  requirement: SQLite is not designed for multiple processes across
  multiple hosts writing/reading concurrently over the network, which the
  stateless, horizontally-scalable API depends on. It may be worth
  revisiting later as an *optional* lightweight deployment mode for
  single-instance/small-scale operators, but that would be a distinct,
  explicitly scoped decision — not assumed here.
- **MySQL/MariaDB.** Would satisfy most of the same requirements
  (transactions, indexing, network access, wide self-hosting familiarity).
  Not chosen because PostgreSQL offers no disadvantage here and has
  stronger defaults around constraint/transaction correctness and richer
  indexing features that fit VersionGate's ordering/uniqueness needs
  slightly better. This is a mild preference, not a strong technical
  requirement — MySQL was a viable alternative.
- **A distributed/NewSQL database** (e.g. CockroachDB, YugabyteDB).
  Rejected as unnecessary and operationally heavy for this stage:
  VersionGate's write volume is moderate and a single PostgreSQL instance
  comfortably meets current needs. Introducing distributed-database
  operational complexity now would be optimizing for a scale problem that
  does not yet exist.
- **A key-value or document store** (e.g. a generic NoSQL store).
  Rejected because VersionGate's data is inherently relational (Projects
  own Applications, Applications own Releases, uniqueness spans
  Application + version + build number) and its query needs include
  ordering and range-style comparisons (version ordering) that relational
  indexing handles naturally and a key-value model would push into
  application code.
- **Redis** (cache, queue, or primary store). Rejected for now: there is
  no current requirement it uniquely solves. Caching isn't yet justified
  (no confirmed read-latency problem), and VersionGate has no queueing or
  pub/sub need at this stage. Introducing Redis now would add an
  operational dependency (per operational-simplicity concerns above)
  without a concrete problem behind it. If high read volume from policy
  evaluation later proves to be a real bottleneck that indexing and
  connection pooling can't address, a caching layer — Redis or otherwise
  — can be introduced then, as its own decision.
