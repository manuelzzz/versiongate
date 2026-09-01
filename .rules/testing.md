# Testing Rules

## Test behavior, not implementation

- Tests should verify observable behavior (inputs → outputs, state
  changes, errors) through a package's public API, not internal
  implementation details.
- If a refactor that preserves behavior breaks tests, the tests were
  probably coupled to implementation rather than behavior — fix the tests'
  approach, not just the refactor.

## Table-driven tests

- Use table-driven tests where a function has multiple input/output cases
  worth covering, especially policy evaluation and other branching domain
  logic.
- Keep each case focused and named clearly enough that a failure message
  alone points at the scenario.
- Don't force a table for a single case or for tests that don't share
  structure — a plain test function is fine.

## Domain logic gets priority

- Domain logic (release metadata rules, update policy evaluation) must be
  covered by tests that don't touch the network, a database, or the
  filesystem. If domain code needs those to be tested, it's leaking
  infrastructure concerns — see `.rules/architecture.md`.
- Prioritize test coverage on domain logic over transport/glue code; glue
  code should be thin enough that it needs little testing of its own.

## Avoid implementation-detail tests

- Don't assert on private state, call counts, or internal call order unless
  that ordering is itself the contract being tested.
- Don't write tests that mock out so much of a component that the test only
  checks that mocks were called — assert on real outcomes instead.
- Avoid snapshot/golden-file tests for anything that isn't inherently about
  exact output format (e.g. serialized API responses); prefer explicit
  assertions elsewhere.
