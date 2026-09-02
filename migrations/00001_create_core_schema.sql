-- +goose Up

-- Project: the top-level organizational boundary (specs/domain/project.md).
CREATE TABLE projects (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Application: belongs to exactly one Project. `identifier` is the
-- stable, immutable handle Releases/requests reference; it only needs
-- to be unique within its owning Project (specs/domain/application.md).
CREATE TABLE applications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    identifier   TEXT NOT NULL,
    display_name TEXT NOT NULL,
    platform     TEXT NOT NULL CHECK (platform IN ('ios', 'android')),
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, identifier)
);

-- Release: belongs to exactly one Application. Version is stored as
-- three separate non-negative integer columns (not a string) so that
-- ordering by version in SQL (ORDER BY major, minor, patch) is correct
-- numeric comparison by construction, never lexicographic
-- (specs/decisions/version-comparison.md). Build number disambiguates
-- Releases that would otherwise share a version; (application_id,
-- version, build_number) must be unique (specs/domain/release.md's
-- Duplicate releases section). revoked_at is NULL while active; setting
-- it is how a Release is excluded from policy evaluation without being
-- deleted (specs/domain/release.md's Revocation section).
CREATE TABLE releases (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications (id) ON DELETE CASCADE,
    major          INTEGER NOT NULL CHECK (major >= 0),
    minor          INTEGER NOT NULL CHECK (minor >= 0),
    patch          INTEGER NOT NULL CHECK (patch >= 0),
    build_number   INTEGER NOT NULL CHECK (build_number >= 0),
    policy         TEXT NOT NULL CHECK (policy IN ('optional', 'required')),
    revoked_at     TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (application_id, major, minor, patch, build_number)
);

-- Supports both "latest non-revoked release for an Application" lookups
-- and ordering by version within an Application.
CREATE INDEX releases_application_version_idx
    ON releases (application_id, major, minor, patch, build_number);

-- API Token: scoped to exactly one Project (specs/decisions/authentication.md).
-- Only a hash of the token is ever stored — the raw value is shown to
-- the operator once, at creation, and is never persisted or
-- retrievable again. revoked_at NULL means the token is active;
-- revocation must take effect immediately and is independent of any
-- other token's state.
CREATE TABLE api_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX api_tokens_project_id_idx ON api_tokens (project_id);

-- +goose Down

DROP TABLE api_tokens;
DROP TABLE releases;
DROP TABLE applications;
DROP TABLE projects;
