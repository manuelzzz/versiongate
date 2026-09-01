# Architecture Rules

## Modular monolith

- VersionGate is a single deployable service, organized into internal
  modules by domain concept (e.g. releases, policies), not by technical
  layer (e.g. "handlers", "services", "repositories" as top-level packages).
- Modules communicate through explicit Go APIs (exported functions/types),
  not through shared global state or hidden coupling.
- Do not split the service into multiple deployables (microservices,
  separate binaries talking over the network, message queues, etc.) unless
  a spec in `specs/` justifies it. There is no such justification today.

## Dependency direction

- Dependencies point inward: infrastructure depends on domain, never the
  other way around.
- Transport code (HTTP handlers, CLI, etc.) depends on the domain to do
  work; the domain has no knowledge that HTTP, a CLI, or any other
  transport exists.
- Storage code (database, cache, external APIs) implements interfaces
  defined by the domain, not the reverse. The domain does not import
  storage packages.

## Domain independence

- Domain packages (release metadata, policy evaluation) must not import
  framework, driver, or transport packages (HTTP routers, SQL drivers, gRPC,
  etc.).
- Domain code should be testable with no network, no database, and no
  environment setup.
- If the domain needs something from the outside world (persistence, time,
  randomness), it declares a small interface for it; infrastructure
  provides the implementation.

## Avoiding premature abstractions

- Don't introduce a layer, interface, or plugin point for a use case
  VersionGate doesn't have yet. Add it when a second real implementation or
  a concrete requirement shows up, not before.
- Prefer one obvious way to do something over configurable/pluggable
  behavior "for flexibility."
- If you're unsure whether an abstraction is justified, it probably isn't
  yet — write it down as a question in `specs/` instead of building it.

## Explicit dependencies

- Pass dependencies explicitly (constructor parameters, struct fields).
  Avoid package-level mutable state, `init()` magic, and global singletons.
- A package's dependencies should be visible from its constructor
  signature, not discovered by reading its implementation.

## Avoiding unnecessary frameworks

- Prefer the standard library and small, focused libraries over large
  frameworks, especially for HTTP routing, dependency injection, and ORMs.
- Any non-standard-library dependency must earn its place: it should solve
  a real problem the standard library doesn't solve well, not save a few
  lines of boilerplate.
- Justify new dependencies in the PR/commit that introduces them.
