---
title: Integrating update-check in a mobile client
description: Use the unauthenticated update-check endpoint from a mobile client.
---

`/update-check` is how a mobile client asks VersionGate whether it should
continue normally, notify the user of an available update, or require an
update before continuing. It's the read counterpart to
[Publishing Releases](/versiongate/publishing-releases/).

## No authentication required

This endpoint takes **no** `Authorization` header and needs none — see
[`specs/decisions/authentication.md`](https://github.com/manuelzzz/versiongate/blob/main/specs/decisions/authentication.md).
The Application identifier it takes is not a secret: it's safe to embed
directly in your mobile app's source or build configuration. The data
returned (whether an update is available, and for what version) is
information your own app's users are already entitled to see by running
the app.

## Request

```
GET /update-check?application_identifier=acme-ios&version=1.2.0&build_number=180
```

| Parameter | Required | Meaning |
|---|---|---|
| `application_identifier` | yes | The Application's public identifier, set when it was created (see [Publishing Releases](/versiongate/publishing-releases/)). |
| `version` | yes | The client's current version, `MAJOR.MINOR.PATCH`. |
| `build_number` | no | The client's current build number. Accepted for descriptive symmetry with publishing, but it never affects the outcome — only `version` does. If present, it's still validated (a malformed value is a bad request, not silently ignored). |

## Response

```json
{
  "action": "required",
  "latest_release": {
    "version": "1.4.0",
    "build_number": 214
  }
}
```

- `action` is one of `continue`, `optional`, or `required`.
- `latest_release` is the highest-versioned non-revoked Release known for
  the Application — present whenever the Application has at least one
  Release, regardless of `action`. It's omitted only when the Application
  has no Releases at all.

Client handling:

- `continue` — nothing to do. This also covers a client already ahead of
  every registered Release (e.g. a beta build) — that's expected, not an
  error.
- `optional` — a newer Release exists; you may prompt the user, but
  should not block them.
- `required` — the client must update before continuing. `latest_release`
  is always safe to direct the user to, even if a different, older
  Release was the one that actually established the requirement.

Nothing else is returned: no binary, no download URL, no store metadata —
VersionGate doesn't distribute binaries, only release metadata.

## Error responses

```json
{ "error": { "code": "validation_error", "message": "..." } }
```

| Cause | `code` | HTTP status |
|---|---|---|
| Missing `application_identifier`, malformed `version`, or malformed `build_number` | `validation_error` | 400 |
| Unknown Application, or an Application/Project that has been deactivated | `not_found` | 404 |
| Unexpected server-side failure | `internal_error` | 500 |

A deactivated Application (or one whose Project is deactivated) is
reported identically to an Application that doesn't exist at all — never
as `continue`, and never distinguishable from a genuinely unknown
identifier.

A client version that fails to parse as `MAJOR.MINOR.PATCH` is rejected
outright (`validation_error`) — it is never coerced or defaulted to any
outcome.
