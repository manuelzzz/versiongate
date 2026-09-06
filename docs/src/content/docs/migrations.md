---
title: "CLI: managing the database schema"
description: Apply, roll back, and inspect VersionGate's database migrations with the versiongate CLI.
---

VersionGate's database schema is applied and inspected with the
`versiongate` CLI's `migrate` subcommands — never automatically by the
`server` binary (see [Installation](/versiongate/installation/)). This
assumes the CLI is already installed and pointed at your database; see
[CLI bootstrap](/versiongate/bootstrap/) if you haven't done that yet.

```bash
export VERSIONGATE_DATABASE_DSN="postgres://versiongate:versiongate@localhost:5432/versiongate?sslmode=disable"
```

## `versiongate migrate up`

Applies all pending migrations.

```bash
versiongate migrate up
# migrations applied
```

Safe to run any number of times: if every migration is already applied,
it's a no-op, not an error.

## `versiongate migrate status`

Shows every known migration and whether it's been applied.

```bash
versiongate migrate status
# 00001_create_core_schema.sql   pending
```

After `migrate up`, the same command shows when it was applied:

```bash
versiongate migrate status
# 00001_create_core_schema.sql   applied at 2026-01-01 00:00:00 +0000 UTC
```

## `versiongate migrate down`

Rolls back **only the single most recently applied** migration — not
every migration, and not to a specific version.

```bash
versiongate migrate down
# last migration rolled back
```

Running `migrate status` afterward shows that migration as `pending`
again.

## Current schema

There is currently one migration
([`00001_create_core_schema.sql`](https://github.com/manuelzzz/versiongate/blob/main/migrations/00001_create_core_schema.sql)),
covering Projects, Applications, Releases, and API Tokens. As the schema
evolves, new migration files are added — `migrate up` always applies
whatever is pending, so the commands above don't change as that happens.
