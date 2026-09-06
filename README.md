# VersionGate

[![CI](https://github.com/manuelzzz/versiongate/actions/workflows/ci.yml/badge.svg)](https://github.com/manuelzzz/versiongate/actions/workflows/ci.yml)

A lightweight, self-hosted, API-first service for managing mobile
application releases and evaluating update policies. Given a client's
current version, it decides whether the app should continue normally, be
notified that an update is available, or be required to update before
continuing.

VersionGate does not distribute application binaries — it only manages
release metadata and update policies.

## Quick start

Requires Docker and Docker Compose, plus a Go toolchain for the CLI
(bootstrap/migrations):

```bash
git clone https://github.com/manuelzzz/versiongate.git
cd versiongate
docker compose up -d
```

Then apply the database schema and create your first Project and API
Token with the CLI:

```bash
go install github.com/manuelzzz/versiongate/cmd/versiongate@latest

export VERSIONGATE_DATABASE_DSN="postgres://versiongate:versiongate@localhost:5432/versiongate?sslmode=disable"
versiongate migrate up
versiongate bootstrap --name "My Project"
```

See [Installation](https://manuelzzz.github.io/versiongate/installation/)
and [CLI bootstrap](https://manuelzzz.github.io/versiongate/bootstrap/)
for the full walkthrough.

## Documentation

Full docs: <https://manuelzzz.github.io/versiongate/> (source under
[`docs/`](docs/), built with Astro + Starlight per
[`specs/decisions/docs-site.md`](specs/decisions/docs-site.md)).

- [Installation](https://manuelzzz.github.io/versiongate/installation/)
  — Docker-based install.
- [CLI bootstrap](https://manuelzzz.github.io/versiongate/bootstrap/) —
  creating the first Project and API Token with the CLI.
- [Publishing Releases](https://manuelzzz.github.io/versiongate/publishing-releases/)
  — publishing a Release from CI/CD.
- [Update Check](https://manuelzzz.github.io/versiongate/update-check/)
  — integrating the update-check endpoint in a mobile client.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Project knowledge

- [`specs/`](specs/) — domain concepts, protocols, and architectural
  decisions: the source of truth for *why* VersionGate is shaped the way
  it is.
- [`.rules/`](.rules/) — development rules and coding guidelines
  governing *how* code is written in this repository.
- [`AGENTS.md`](AGENTS.md) — entry point for how repository knowledge is
  organized, for humans and AI agents alike.

## License

[MIT](LICENSE)
