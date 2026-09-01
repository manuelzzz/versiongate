# Git Rules

## Conventional Commits

- Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):
  `<type>[optional scope]: <description>`.
- Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`,
  `ci`. Use `!` or a `BREAKING CHANGE:` footer for breaking changes.
- Scope, when used, should name the affected module or area (e.g.
  `feat(policy): ...`, `fix(release): ...`).

## Atomic commits

- Each commit should represent one coherent, self-contained change that
  leaves the repository in a working state (builds, tests pass).
- Prefer several small, well-scoped commits over one large commit — it
  makes review and `git bisect` meaningful.

## Avoid unrelated changes in the same commit

- Don't mix formatting-only changes, unrelated refactors, or drive-by fixes
  into a commit whose subject is something else. Split them into separate
  commits (or PRs).
- If you notice something unrelated while working, fix it in its own
  commit rather than folding it into the current one.

## Clear commit messages

- Subject line: imperative mood, no trailing period, ideally under ~72
  characters (e.g. `fix(policy): handle missing minimum version`).
- Use the commit body to explain *why* a change was made when it's not
  obvious from the diff — the diff already shows *what* changed.
- Reference related issues/specs where relevant instead of restating their
  content.
