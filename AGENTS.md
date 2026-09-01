# AGENTS.md

This file is the primary entry point for AI agents (and humans acting like
one) working on the VersionGate repository. Read it before making any change.

## What VersionGate is

VersionGate is a lightweight, self-hosted, API-first service for managing
mobile application releases and evaluating update policies. Given a client's
current version (and other request context), it decides whether the app
should:

- continue normally;
- notify the user that an update is available;
- require an update before continuing.

VersionGate does **not** distribute application binaries. It only manages
release metadata and update policies.

## How repository knowledge is organized

- `.rules/` — development rules and coding guidelines. These govern *how*
  code is written in this repository (style, structure, testing, tooling,
  etc.).
- `specs/` — internal engineering knowledge: domain concepts, protocols, and
  architectural decisions. This is the source of truth for *why* the system
  is shaped the way it is.
- `docs/` — public documentation for users and contributors. This is
  user-facing and describes *what* VersionGate does and how to use it.

Before making a change, read the parts of `.rules/` and `specs/` relevant to
the area you're touching. Don't guess at conventions or domain rules that
are already written down.

If you make a non-trivial architectural decision, record it in `specs/`
rather than leaving it implicit in the code or the commit message.

## Architectural stance

- VersionGate starts as a **modular monolith**. Do not introduce
  microservices, message queues, or other distributed-systems machinery
  unless a spec explicitly calls for it.
- Favor simplicity. Avoid premature abstraction, speculative
  configurability, or generalizing for use cases the project doesn't have
  yet. Solve the problem in front of you.
- Any architectural decision that isn't obvious from the code must be
  justified — either in `specs/` or in the change that introduces it. "It
  might be useful later" is not sufficient justification.
- The domain (release metadata, update policy evaluation) must stay
  independent from infrastructure (HTTP transport, storage, external
  services). Domain code should not depend on frameworks or drivers.

## Documentation

When a change affects user-facing behavior (the API, configuration,
deployment, or anything a consumer of VersionGate would notice), update the
relevant pages in `docs/` as part of the same change.
