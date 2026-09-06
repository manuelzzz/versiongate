# Decision: Public Documentation Site

## Status

Accepted — initial decision, revisit if a concrete requirement outgrows
it.

## Decision

VersionGate's public documentation (`docs/`) is built into a static site
with [Astro](https://astro.build) and its
[Starlight](https://starlight.astro.build) documentation template, and
deployed to **GitHub Pages** at the default project-site URL
(`https://manuelzzz.github.io/versiongate/`) — no custom domain.

The Markdown pages under `docs/` are the **single source of truth** for
documentation content: they live inside the Astro project itself
(`docs/src/content/docs/*.md`), not as a separate, duplicated copy kept
in sync by hand. There is one copy of each page, and it is both what a
repository browser sees under `docs/` and what the built site renders.

## Context

`specs/decisions/authentication.md` through `specs/protocols/update-check.md`
established VersionGate's technical decisions; this one covers a
repository-tooling choice, not a domain or protocol concern, per
`AGENTS.md`'s guidance to record non-trivial architectural decisions
rather than leave them implicit.

Prior to this decision, `docs/` was four flat Markdown files, rendered
as-is by GitHub — sufficient for the MVP but with no dedicated
navigation, search, or a stable public URL to link to from outside the
repository (see `specs/decisions/database.md`-style precedent: this
project favors deferring infrastructure until there's a concrete need,
and by this point there was one — see the originating issue).

Three sub-decisions had to be made, each with real alternatives:

1. **Hosting.** Where the built site is served from.
2. **Domain.** Whether it needs a custom domain now.
3. **Content ownership.** Whether the existing plain-Markdown pages stay
   a separate, flat copy (with an Astro project elsewhere duplicating or
   importing them) or get absorbed as the Astro project's own content.

## Rationale

- **GitHub Pages** requires no new external account, no billing, and no
  secrets beyond what GitHub Actions already provides (`GITHUB_TOKEN`).
  It deploys from the same repository and the same CI system
  (`.github/workflows/`) already used for the Go build, keeping the
  project's operational surface area small — consistent with
  `.rules/architecture.md`'s avoiding-unnecessary-frameworks and
  operational-simplicity guidance applied to tooling, not just runtime
  infrastructure. Vercel/Netlify would work too, but add an external
  account and a second place secrets/config can drift, for no concrete
  benefit VersionGate needs today.
- **No custom domain, for now.** The default
  `manuelzzz.github.io/versiongate` URL is a working, stable, linkable
  address with zero DNS setup. A custom domain is a purely additive
  change later (add a `CNAME` file and an Astro `site` config update) —
  deferring it doesn't foreclose anything, and there's no concrete
  requirement (branding, a purchased domain) driving it yet.
- **Absorbing Markdown into the Astro project, not duplicating it.**
  Keeping the plain files as a separate flat copy while a second Astro
  project elsewhere imported or re-exported them would create two
  representations of the same content that could drift out of sync — the
  exact "trustworthy record" problem this project avoids elsewhere (e.g.
  `specs/domain/release.md`'s immutability rationale, applied here to
  documentation instead of release data). A single copy, addressed
  directly by both a repository browser and the built site, has no
  drift risk by construction.
- **Astro + Starlight specifically**, rather than a hand-rolled static
  site: Starlight is Astro's official, purpose-built documentation
  template — it ships navigation, search, and page routing out of the
  box. Building that by hand would be exactly the kind of reinvention
  `.rules/go.md`'s standard-library-preference principle (applied here to
  the Node ecosystem) and `.rules/architecture.md`'s
  avoiding-unnecessary-frameworks guidance argue against: there is no
  reason to build what a well-maintained, purpose-fit tool already does.

## Consequences

- `docs/` is now a self-contained Node.js project (`package.json`,
  `astro.config.mjs`, etc.) alongside its Markdown content, entirely
  separate from the Go module at the repository root. The Go build,
  tests, and CI (`.github/workflows/ci.yml`) never invoke Node tooling
  and are unaffected by anything under `docs/`.
- A second GitHub Actions workflow (`.github/workflows/docs.yml`) builds
  and deploys the site to GitHub Pages, scoped to pushes to `main` that
  touch `docs/**`. The repository's GitHub Pages source must be set to
  "GitHub Actions" (a one-time repository setting) for that workflow's
  deployment step to take effect.
- Every documentation page lives under `docs/src/content/docs/`, not
  directly under `docs/`. Cross-references between pages use
  site-relative paths (e.g. `/bootstrap/`); references to files that only
  exist in the repository (`specs/`, `.rules/`, `AGENTS.md`) use absolute
  GitHub blob URLs, since those files aren't part of the built site.
- Adding a documentation page means adding a Markdown/MDX file under
  `docs/src/content/docs/` and an entry in `astro.config.mjs`'s
  `sidebar` — not just a flat file anywhere under `docs/`.
- If a future requirement calls for a custom domain or a different host,
  that's a small, additive change to this decision (an Astro config
  update and, for a host change, a new deploy workflow) — not a content
  restructuring, since content ownership is unaffected by where the site
  is served from.

## Alternatives considered

- **Vercel or Netlify.** Both would work well and offer preview
  deployments per pull request, which GitHub Pages does not. Rejected for
  now: they require creating and maintaining an external account and
  project configuration for a benefit (PR previews) no current workflow
  in this project depends on. Revisit if PR-preview review of docs
  changes becomes a real pain point.
- **A custom domain from the start.** Rejected as premature: no domain is
  currently owned or required for this documentation to be useful, and
  adding one later doesn't require undoing anything decided here.
- **Keeping `docs/*.md` flat and adding Astro in a subdirectory** (e.g.
  `docs/site/`) that imports or duplicates the Markdown. Rejected per the
  Rationale above: two representations of the same content is a
  synchronization liability with no offsetting benefit, since Starlight
  can serve the content directly from where it already needs to live.
- **A hand-rolled Astro site (no Starlight).** Rejected: would mean
  reimplementing navigation, search, and responsive layout that Starlight
  already provides, maintained by the Astro team, for no concrete
  requirement Starlight doesn't already satisfy.
- **Docusaurus, MkDocs, or another documentation generator.** Not
  rejected on technical merits — any would likely work — but Astro was
  the tool named when this was proposed, and Starlight is its
  purpose-built documentation template; there was no concrete reason to
  evaluate alternatives once a suitable, well-maintained option in the
  requested ecosystem existed.
