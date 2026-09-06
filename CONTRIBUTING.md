# Contributing to VersionGate

## Before you start

Read [`AGENTS.md`](AGENTS.md) first — it explains how repository
knowledge is organized:

- [`specs/`](specs/) is the source of truth for *why* the system is
  shaped the way it is (domain concepts, protocols, architectural
  decisions).
- [`.rules/`](.rules/) governs *how* code is written (style, structure,
  testing).
- [`docs/`](docs/) is user-facing documentation for *what* VersionGate
  does.

Before changing an area of the code, read the parts of `specs/` and
`.rules/` relevant to it. If you make a non-trivial architectural
decision along the way, record it under `specs/decisions/` rather than
leaving it implicit in the code or commit message — that's the
specs → decision → implementation workflow this repository follows.

## Building and testing locally

Requires Go (see `go.mod` for the version) and, for anything touching
Postgres, a running PostgreSQL instance (`docker compose up postgres`
starts one — see [Installation](https://manuelzzz.github.io/versiongate/installation/)).

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .   # should print nothing; anything listed needs `gofmt -w`
```

Domain logic (`internal/release`, `internal/updatepolicy`,
`internal/version`, etc.) is tested without a database, per
[`.rules/testing.md`](.rules/testing.md). Only `internal/postgres`'s
tests touch a real connection, and only for failure-path behavior that
doesn't require a live Postgres instance to be running — see that
package's test file.

### Running the service locally

```bash
export VERSIONGATE_DATABASE_DSN="postgres://versiongate:versiongate@localhost:5432/versiongate?sslmode=disable"

go run ./cmd/versiongate migrate up
go run ./cmd/versiongate bootstrap --name "Dev"
go run ./cmd/server
```

### Running the docs site locally

`docs/` is a separate Astro + Starlight (Node.js) project — see
[`specs/decisions/docs-site.md`](specs/decisions/docs-site.md). It has no
effect on the Go build or `go test ./...`.

```bash
cd docs
npm install
npm run dev   # http://localhost:4321
```

## Coding conventions

Covered in `.rules/`, most relevantly:

- [`.rules/go.md`](.rules/go.md) — idiomatic Go, standard-library
  preference, package organization, error handling.
- [`.rules/architecture.md`](.rules/architecture.md) — modular monolith,
  dependency direction (domain never depends on transport or storage),
  avoiding premature abstraction.
- [`.rules/testing.md`](.rules/testing.md) — table-driven tests, testing
  behavior over implementation, domain logic coverage priority.
- [`.rules/git.md`](.rules/git.md) — commit conventions (below).

## Commits and pull requests

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
(`feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`, `ci`), each
commit atomic and self-contained — see `.rules/git.md` for the full
guidance, including keeping unrelated changes out of the same commit.

Fill out the pull request template; reference the issue(s) the PR closes.
CI runs on every pull request and must pass: build, `go vet`, `go test`,
and a `gofmt` check.

## Documentation

If a change affects user-facing behavior (the API, configuration,
deployment, or anything a consumer of VersionGate would notice), update
the relevant page(s) under `docs/src/content/docs/` as part of the same
change, per `AGENTS.md`.
