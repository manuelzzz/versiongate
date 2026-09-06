# VersionGate docs site

This is the source for VersionGate's public documentation site, built
with [Astro](https://astro.build) and
[Starlight](https://starlight.astro.build). It's a separate, self
-contained Node.js project — it does not affect the Go module at the
repository root, and the Go build/CI pipeline never touches it.

Deployed to GitHub Pages at <https://manuelzzz.github.io/versiongate/>
via `.github/workflows/docs.yml`, on every push to `main` that touches
this directory.

## Structure

```
docs/
├── src/content/docs/   # the actual documentation pages (Markdown/MDX)
├── astro.config.mjs    # site title, sidebar, GitHub Pages base path
└── package.json
```

Each `.md`/`.mdx` file under `src/content/docs/` is one page, routed by
its file name (e.g. `installation.md` → `/installation/`).

## Commands

Run from this directory (`docs/`):

| Command           | Action                                       |
| ------------------ | --------------------------------------------- |
| `npm install`      | Install dependencies                          |
| `npm run dev`       | Start the local dev server at `localhost:4321` |
| `npm run build`     | Build the production site to `./dist/`        |
| `npm run preview`   | Preview the production build locally          |

## Accuracy

Per `specs/decisions/docs-site.md` and the same bar the Markdown content
was originally held to: every documented command/endpoint here describes
real, implemented VersionGate behavior — not aspirational features.
