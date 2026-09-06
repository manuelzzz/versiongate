---
title: "CLI: bootstrapping the first Project and Token"
description: Create the initial Project and its first API Token with the versiongate CLI.
---

Every write operation in VersionGate — publishing a Release, creating an
Application — requires a Project-scoped API Token
(see [`specs/decisions/authentication.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/decisions/authentication.md)).
Creating the very first Project and Token is itself a write operation, so
it can't go through the authenticated HTTP API — there is no token yet to
authorize it.

VersionGate resolves this with an operator-invoked CLI command, per
[`specs/decisions/bootstrap-mechanism.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/decisions/bootstrap-mechanism.md).
This is the only way to create a Project: there is no HTTP endpoint for
it.

This assumes you already have VersionGate and its database running — see
[Installation](/versiongate/installation/).

## Install the CLI

```bash
go install github.com/manuelzzz/versiongate/cmd/versiongate@latest
```

## Point it at your database

```bash
export VERSIONGATE_DATABASE_DSN="postgres://versiongate:versiongate@localhost:5432/versiongate?sslmode=disable"
```

(Adjust the DSN to match your deployment. If you're following the
[Installation](/versiongate/installation/) guide's Docker Compose setup,
this is the value already shown there.)

## Create the Project and Token

```bash
versiongate bootstrap --name "Acme"
```

```
Project created: Acme (3f2a1c9e-...)

API Token (save this now — it will not be shown again):
vg_8f3a9c...
```

The token is only ever displayed at this moment. VersionGate stores a
hash of it, not the raw value, and has no way to show it to you again
(see [`specs/decisions/authentication.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/decisions/authentication.md)'s
Token storage security). Save it in your CI/CD system's secret store now.

`--name` is required; `versiongate bootstrap` without it fails before
attempting to connect to the database.

## What this token can do

The token is scoped to the Project it was issued for
(see [`specs/domain/project.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/domain/project.md)).
Use it as a bearer credential on every authenticated request:

```
Authorization: Bearer vg_8f3a9c...
```

It authorizes creating Applications and publishing Releases under that
Project — see [Publishing Releases](/versiongate/publishing-releases/).

## Current limitations

- `versiongate bootstrap` always creates a **new** Project along with its
  Token. There is currently no separate CLI command to issue an
  additional Token for an existing Project (rotation) — if you need a
  second Token today, running `bootstrap` again creates a new Project,
  not a new Token for the same one.
- There is no CLI or HTTP command to revoke a Token yet, even though the
  domain layer supports it (`internal/token`). Until that's exposed,
  treat a leaked token as an incident to resolve at the database level.

## Other CLI commands

```bash
versiongate version        # print the CLI/server build version
versiongate migrate up     # apply pending migrations
versiongate migrate down   # roll back the most recently applied migration
versiongate migrate status # show which migrations have been applied
```
