---
title: Contribute
description: Build, test, and contribute to VersionGate.
---

VersionGate is open source. This page is a quick on-ramp; the
authoritative, most up-to-date version of this guidance is always
[`CONTRIBUTING.md`](https://github.com/manuelzzz/versiongate/blob/main/CONTRIBUTING.md)
in the repository.

## Before you start

Read [`AGENTS.md`](https://github.com/manuelzzz/versiongate/blob/main/AGENTS.md)
first — it explains how repository knowledge is organized:

- [`specs/`](https://github.com/manuelzzz/versiongate/tree/main/specs) is
  the source of truth for *why* the system is shaped the way it is
  (domain concepts, protocols, architectural decisions).
- [`.rules/`](https://github.com/manuelzzz/versiongate/tree/main/.rules)
  governs *how* code is written (style, structure, testing).
- This site (built from `docs/`) is user-facing documentation for *what*
  VersionGate does.

Before changing an area of the code, read the parts of `specs/` and
`.rules/` relevant to it. If you make a non-trivial architectural
decision along the way, record it under `specs/decisions/` rather than
leaving it implicit in the code or commit message — that's the
specs → decision → implementation workflow this repository follows.

## Building and testing locally

Requires Go (see `go.mod` for the version) and, for anything touching
Postgres, a running PostgreSQL instance (`docker compose up postgres`
starts one — see [Installation](/versiongate/installation/)).

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .   # should print nothing; anything listed needs `gofmt -w`
```

Domain logic (release metadata, update policy evaluation, version
comparison) is tested without a database. Only `internal/postgres`'s
tests touch a real connection, and only for failure-path behavior that
doesn't require a live Postgres instance to be running.

### Running the service locally

```bash
export VERSIONGATE_DATABASE_DSN="postgres://versiongate:versiongate@localhost:5432/versiongate?sslmode=disable"

go run ./cmd/versiongate migrate up
go run ./cmd/versiongate bootstrap --name "Dev"
go run ./cmd/server
```

### Running this docs site locally

This site is a separate Astro + Starlight (Node.js) project under
`docs/` — it has no effect on the Go build or `go test ./...`.

```bash
cd docs
npm install
npm run dev   # http://localhost:4321
```

## Coding conventions

Covered in `.rules/`, most relevantly:

- [`.rules/go.md`](https://github.com/manuelzzz/versiongate/blob/main/.rules/go.md)
  — idiomatic Go, standard-library preference, package organization,
  error handling.
- [`.rules/architecture.md`](https://github.com/manuelzzz/versiongate/blob/main/.rules/architecture.md)
  — modular monolith, dependency direction, avoiding premature
  abstraction.
- [`.rules/testing.md`](https://github.com/manuelzzz/versiongate/blob/main/.rules/testing.md)
  — table-driven tests, testing behavior over implementation.
- [`.rules/git.md`](https://github.com/manuelzzz/versiongate/blob/main/.rules/git.md)
  — commit conventions (below).

## Commits and pull requests

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
(`feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`, `ci`), each
commit atomic and self-contained. Fill out the pull request template and
reference the issue(s) it closes. CI runs on every pull request and must
pass: build, `go vet`, `go test`, and a `gofmt` check.

## Documentation

If a change affects user-facing behavior (the API, configuration,
deployment, or anything a consumer of VersionGate would notice), update
the relevant page(s) under `docs/src/content/docs/` as part of the same
change.
