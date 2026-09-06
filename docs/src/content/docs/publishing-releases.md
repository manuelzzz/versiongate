---
title: Publishing a Release from CI/CD
description: Register an Application and publish Releases for it from a CI/CD pipeline.
---

This guide covers the write side of VersionGate: registering an
Application once, then publishing Releases for it from a CI/CD pipeline.
Both require the Project-scoped API Token from
[CLI bootstrap](/versiongate/bootstrap/).

All requests below use:

```
Authorization: Bearer <your-token>
Content-Type: application/json
```

Request and response bodies are JSON, with `snake_case` field names, per
[`specs/protocols/http.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/protocols/http.md).

## 1. Create an Application (one-time)

An Application represents one distributable app on one platform — e.g.
your iOS app and your Android app are two separate Applications, each
with their own identifier and Release history
(see [`specs/domain/application.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/domain/application.md)).
Create it once, before publishing any Releases:

```bash
curl -X POST http://localhost:8888/applications \
  -H "Authorization: Bearer $VERSIONGATE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "identifier": "acme-ios",
        "display_name": "Acme (iOS)",
        "platform": "ios"
      }'
```

```json
{
  "id": "b1e2...",
  "project_id": "3f2a...",
  "identifier": "acme-ios",
  "display_name": "Acme (iOS)",
  "platform": "ios",
  "active": true,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

`platform` must be `ios` or `android`. `identifier` must be unique within
your Project (not globally) and does not change afterward — it's the
same value a mobile client will later send to `/update-check`
(see [Update Check](/versiongate/update-check/)), so pick something stable
before shipping it in a client binary.

A duplicate `identifier` within the same Project returns `409 conflict`.

## 2. Publish a Release

```bash
curl -X POST "http://localhost:8888/applications/$APPLICATION_ID/releases" \
  -H "Authorization: Bearer $VERSIONGATE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "version": "1.4.0",
        "build_number": 214,
        "policy": "optional"
      }'
```

```json
{
  "id": "c9d0...",
  "application_id": "b1e2...",
  "version": "1.4.0",
  "build_number": 214,
  "policy": "optional",
  "created_at": "2026-01-01T00:00:00Z"
}
```

- `version` is `MAJOR.MINOR.PATCH` (see
  [`specs/domain/version.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/domain/version.md)).
- `build_number` is a non-negative integer.
- `policy` is `optional` (notify) or `required` (block until updated).

The response is `201 Created` for a newly created Release, or `200 OK` if
the identical (version, build number, policy) was already published —
see Idempotency below. Either status means the Release exists as
described.

## Idempotency and retries

Publishing is idempotent on (Application, version, build number), per
[`specs/protocols/release-publishing.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/protocols/release-publishing.md).
If your pipeline can't tell whether a previous publish request actually
reached VersionGate (timeout, dropped connection), the safe action is
always to retry the identical request:

- Same version, build number, and policy as an existing Release → the
  request succeeds (`200 OK`) without creating anything new.
- Same version and build number but a **different** policy → `409
  conflict`. This means something upstream sent inconsistent data for
  the same build; it will not resolve by retrying, and needs a
  deliberate fix (correct the request, or publish under a different
  build number).

## Deriving `policy` from commit messages (optional)

VersionGate's publish endpoint only accepts a resolved `policy` value —
it does not read your Git history. If you want to derive that value from
commit messages instead of setting it manually, your pipeline can
implement the convention in
[`specs/protocols/commit-metadata.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/protocols/commit-metadata.md):
tag commits with `[versiongate:update=optional]` or
`[versiongate:update=required]`, and resolve the most restrictive tag
across the commit range (`required` wins over `optional`) before calling
this endpoint. This resolution step is entirely your pipeline's
responsibility — VersionGate has no part in it beyond accepting the
resulting `policy` field.

## Error responses

Errors use the shared envelope from
[`specs/protocols/http.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/protocols/http.md):

```json
{ "error": { "code": "validation_error", "message": "..." } }
```

| Cause | `code` | HTTP status |
|---|---|---|
| Missing/invalid/revoked token | `unauthorized` | 401 |
| Malformed `version`, invalid `build_number`, invalid `policy`, unknown or inactive Application | `validation_error` | 400 |
| Duplicate `identifier` (Applications) / conflicting metadata for an existing (version, build number) (Releases) | `conflict` | 409 |
| Unexpected server-side failure | `internal_error` | 500 |

An unknown or inactive Application on the publish endpoint specifically
returns `validation_error`, not `not_found` — see
[`specs/protocols/release-publishing.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/protocols/release-publishing.md).
