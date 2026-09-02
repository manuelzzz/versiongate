# Protocol: Shared HTTP API Conventions

## Purpose

This document defines the cross-cutting HTTP-level conventions shared by
every VersionGate API endpoint, so individual protocols
(`specs/protocols/release-publishing.md`,
`specs/protocols/update-check.md`) and their eventual implementations
don't each invent their own JSON shape, error format, or status-code
mapping.

This document does not enumerate endpoints, request/response payloads for
specific operations, or authentication mechanics beyond the header
convention below — those belong to each operation's own protocol
(interaction contract) and to `specs/decisions/authentication.md`
(credential semantics).

## Content type

- Requests and responses use JSON (`application/json`), UTF-8 encoded.
- VersionGate does not need to support alternate representations (XML,
  form-encoded, etc.); introducing one is out of scope unless a concrete
  requirement emerges.

## JSON field naming

- JSON field names use `snake_case` (e.g. `build_number`,
  `update_policy`), consistently across every endpoint. This is a
  one-time convention choice made here so it never needs to be
  rediscovered per endpoint.

## Shared error envelope

Every error response — regardless of endpoint — uses the same shape:

```json
{
  "error": {
    "code": "validation_error",
    "message": "human-readable description"
  }
}
```

- `code` is a stable, machine-readable identifier (see Error codes,
  below) that a caller (a CI/CD pipeline, a mobile client) can branch on
  without parsing `message`.
- `message` is a human-readable explanation for logs/debugging. It is not
  a stable contract and may change wording over time.
- Successful responses do not use this envelope; they return the
  operation's own result shape directly, as defined by that operation's
  protocol.

## Error codes and status code semantics

| `code`              | HTTP status | Meaning |
|---------------------|-------------|---------|
| `validation_error`  | 400 | The request is malformed or fails a domain validation rule (e.g. an invalid version string, per `specs/domain/version.md`). No side effects occur. |
| `unauthorized`      | 401 | The request is missing a token, or the token is invalid/revoked, on an endpoint that requires one. |
| `not_found`         | 404 | The referenced resource (e.g. Application) does not exist, is not visible to the authenticated token's Project, or — per `specs/decisions/authentication.md` — belongs to a different Project entirely. A cross-Project access attempt is represented identically to a genuinely missing resource, so a response never confirms another Project's resource exists. |
| `conflict`          | 409 | The request repeats an identity that already exists with different data (e.g. `specs/protocols/release-publishing.md`'s duplicate-with-different-metadata case). Distinct from `validation_error` because retrying with the same input will not help — the conflict requires a deliberate decision. |
| `internal_error`    | 500 | An unexpected failure on VersionGate's side. Safe to retry per each operation's own idempotency guarantees. |

Success responses use `200 OK` (reads) or `201 Created` (a Release or
other resource was newly created). An idempotent publish that resolves to
"already exists, identical" (`specs/protocols/release-publishing.md`)
still returns a success status — it is not an error.

This table only defines the *shared* codes. An operation-specific
protocol may reference these but must not invent new ones without adding
them here first, to keep the set small and stable.

## Authentication header convention

- Requests that require a Project-scoped API Token
  (`specs/decisions/authentication.md`) present it as:

  ```
  Authorization: Bearer <token>
  ```

- Read endpoints that are unauthenticated by design (e.g.
  `specs/protocols/update-check.md`) simply omit this header; its absence
  on those endpoints is expected, not an error.

## Request validation behavior

- Validation failures are fail-closed: a request that doesn't meet an
  operation's documented requirements is rejected with `validation_error`
  and no side effects — never partially applied, never coerced into a
  "best effort" interpretation. This restates, at the HTTP level, the
  fail-closed principle already established in `specs/domain/version.md`
  and `specs/protocols/release-publishing.md`.
- Unknown fields in a request body are ignored rather than rejected, to
  allow forward-compatible additions to request shapes without breaking
  older or newer clients mid-rollout. This is the one deliberate
  exception to fail-closed validation, scoped narrowly to unrecognized
  fields only.

## Versioning

- No URL or header-based API versioning scheme is introduced for the
  MVP. There is one current contract per operation, defined by that
  operation's protocol document. If a breaking change is needed later,
  versioning will be introduced deliberately, as its own decision — it is
  not designed in speculatively now.
