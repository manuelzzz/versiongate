# Go Rules

## Idiomatic Go

- Follow standard Go conventions: `gofmt`/`goimports` clean, `go vet` clean,
  and the guidance in [Effective Go](https://go.dev/doc/effective_go) and
  the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Prefer clarity over cleverness. Optimize code for the next reader, not for
  minimizing line count.
- Keep names short but meaningful; let package names provide context so
  identifiers inside them don't have to repeat it (`release.New`, not
  `release.NewRelease`).

## Standard library preference

- Reach for the standard library first (`net/http`, `encoding/json`,
  `context`, `errors`, `testing`, etc.). Only add a third-party dependency
  when the standard library is genuinely insufficient.
- This applies especially to HTTP routing/middleware and JSON handling —
  don't pull in a framework to do what `net/http` already does.

## Package organization

- Organize packages by domain concept, not by technical kind. Avoid generic
  dumping-ground packages like `utils`, `common`, or `helpers`.
- Keep package boundaries aligned with the module boundaries described in
  `.rules/architecture.md`: a package should have one clear responsibility.
- Avoid import cycles by construction: if two packages need each other,
  that's a sign one of them is drawing its boundary in the wrong place.

## Error handling

- Handle errors where you have enough context to act on them; otherwise
  wrap and propagate with `fmt.Errorf("...: %w", err)` so context
  accumulates up the stack.
- Don't discard errors (no bare `_ = err`) unless it is genuinely safe to
  ignore, and say why in a comment when it's not obvious.
- Use sentinel errors or typed errors (`errors.Is`/`errors.As`) for
  conditions callers need to branch on. Don't parse error strings.
- Don't use panics for expected error conditions; reserve `panic` for
  programmer errors/invariant violations.

## Context usage

- Functions that do I/O or can block (HTTP calls, DB queries, etc.) take a
  `context.Context` as their first parameter, named `ctx`.
- Don't store request-scoped values in struct fields; pass `ctx` explicitly
  through the call chain instead.
- Don't use `context.Context` to smuggle in optional dependencies or
  business data that should just be a normal function argument.

## Avoiding unnecessary interfaces

- Don't define an interface until there are at least two real
  implementations, or a boundary from `.rules/architecture.md` requires one
  (e.g. domain-defined ports for infrastructure).
- Define interfaces at the consumer, not the producer: a package should
  accept the smallest interface it needs, not the concrete type exported by
  its dependency.
- Prefer concrete types for internal, single-implementation code. Interfaces
  are for decoupling, not a default habit.
