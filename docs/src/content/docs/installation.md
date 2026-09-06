---
title: Installation (Docker Compose)
description: Run a self-hosted VersionGate instance with Docker Compose.
---

This guide gets a self-hosted VersionGate instance running with Docker
Compose: one `server` container and one PostgreSQL container, per
[`specs/decisions/database.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/decisions/database.md)
(single Postgres instance, no replicas or clustering).

## Prerequisites

- Docker and Docker Compose.
- A Go toolchain, to run the `versiongate` CLI. The CLI isn't built into
  the server image — it's a separate binary you run against the same
  database (see [Applying migrations](#applying-migrations) below and
  [CLI bootstrap](/versiongate/bootstrap/)).

## 1. Get the source

```bash
git clone https://github.com/manuelzzz/versiongate.git
cd versiongate
```

The `server` image is built locally from the repository's `Dockerfile` —
there is no published pre-built image.

## 2. Start Postgres and the server

```bash
docker compose up -d
```

This starts:

- `postgres` — PostgreSQL 16, with credentials and database name defined
  in `docker-compose.yml`. Those are development defaults; override them
  (via a `.env` file or a compose override) for anything beyond local
  use.
- `server` — the VersionGate HTTP API, built from `Dockerfile`, listening
  on `:8888` (mapped to the host as `8888:8888`).

The `server` container waits for `postgres`'s healthcheck before
starting, but **does not** apply database migrations on startup — that's
a deliberate, explicit step (next).

## Applying migrations

Migrations are applied with the `versiongate` CLI, not automatically by
the server. Install the CLI with Go:

```bash
go install github.com/manuelzzz/versiongate/cmd/versiongate@latest
```

`docker-compose.yml` exposes Postgres on the host at `localhost:5432`
specifically so the CLI can reach it from outside the container network.
Point the CLI at it and apply migrations:

```bash
export VERSIONGATE_DATABASE_DSN="postgres://versiongate:versiongate@localhost:5432/versiongate?sslmode=disable"

versiongate migrate up
```

`versiongate migrate status` shows which migrations have been applied —
see [Managing the database schema](/versiongate/migrations/) for the
full command reference, including rolling back.

## 3. Verify

```bash
curl http://localhost:8888/health
# {"status":"ok"}
```

## Next step

The server has no data yet — no Project, Application, or API Token
exists. Continue to [CLI bootstrap](/versiongate/bootstrap/) to create the
first Project and its API Token.

## Uninstall

```bash
docker compose down
```

Stops and removes the `server` and `postgres` containers, but **keeps**
the `postgres-data` volume — your Projects, Applications, Releases, and
Tokens are preserved for next time.

To also delete all data:

```bash
docker compose down -v
```

This removes the `postgres-data` volume along with the containers —
**irreversible**. There is no backup/export step in this guide; take one
yourself first if you need to keep the data.

If you installed the CLI with `go install`, remove its binary too:

```bash
rm "$(go env GOPATH)/bin/versiongate"
```

## Configuration reference

The server reads its configuration from environment variables (already
set in `docker-compose.yml`):

| Variable                    | Required | Default | Meaning |
|------------------------------|----------|---------|---------|
| `VERSIONGATE_DATABASE_DSN`   | yes      | —       | PostgreSQL connection string. |
| `VERSIONGATE_LISTEN_ADDR`    | no       | `:8888` | Address the HTTP server listens on. |

The CLI reads the same two variables, since it shares `internal/config`
with the server.
