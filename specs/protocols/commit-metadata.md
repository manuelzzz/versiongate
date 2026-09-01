# Protocol: Commit Metadata

## Purpose

The Commit Metadata Protocol lets developers communicate release metadata
to VersionGate through Git commit messages, so a CI/CD pipeline can derive
that metadata automatically when publishing a [[release]] (see
`specs/domain/release.md`) instead of requiring a separate manual step.

The initial, and currently only, use case is communicating the **update
policy** (see `specs/domain/release.md`) a Release should carry: whether
adopting it is optional or required.

This protocol defines a textual convention embedded in commit messages. It
does not define how a CI/CD pipeline extracts, transmits, or submits that
information to VersionGate — that is an integration concern, out of scope
here.

## Syntax

A commit message may contain a metadata tag anywhere in its text, in the
form:

```
[versiongate:<key>=<value>]
```

- The tag is enclosed in square brackets and begins with the literal
  namespace `versiongate:`.
- `<key>` identifies what is being communicated (currently only `update`
  is defined — see Supported metadata below).
- `<value>` is the value for that key.
- The tag may appear on any line of the commit message (subject or body).
  It does not need to be the only content on its line.
- A commit message may contain at most one tag per key. If a key appears
  more than once in the same commit message, the commit's metadata for
  that key is invalid (see Behavior for invalid metadata).

## Supported metadata

| Key      | Meaning                                    |
|----------|---------------------------------------------|
| `update` | The update policy this commit indicates for the resulting Release. |

This is intentionally the only key defined today. The tag syntax
(`[versiongate:<key>=<value>]`) is designed to accommodate additional keys
in the future without changing the protocol itself, but no other keys are
defined until a real need for them exists.

## Valid update policy values

The `update` key accepts exactly two values:

- `optional` — adopting this Release is a suggestion; clients should be
  notified but not blocked.
- `required` — adopting this Release is mandatory; clients should be
  blocked until they update.

```
[versiongate:update=optional]
[versiongate:update=required]
```

No other value is valid. Values are case-sensitive and must match exactly
(`optional`/`required`, not `Optional`, `REQUIRED`, `opt`, etc.).

## Parsing rules

- A commit's metadata is extracted by scanning its full message (subject +
  body) for well-formed `[versiongate:<key>=<value>]` tags.
- Whitespace inside the brackets (e.g. `[versiongate: update = optional]`)
  is not part of the defined syntax and must not be accepted — the tag
  must match the exact form `key=value` with no surrounding spaces.
- Tags found outside this exact form (typos, wrong casing on the
  namespace, malformed brackets) are not recognized as VersionGate
  metadata at all — they are ordinary commit text, not invalid metadata.
  Invalid metadata (below) only applies to text that *is* recognized as a
  tag but carries an unsupported key or value.
- A single commit contributes at most one resolved value per key: if
  exactly one well-formed `update` tag is present, that is the commit's
  update policy. If none is present, the commit carries no update policy.

## Behavior when multiple commits contain metadata

A Release is typically generated from a range of commits (e.g. everything
since the last Release). Multiple commits in that range may each carry an
`update` tag, and they are not required to agree.

Example:

```
Commit A: [versiongate:update=optional]
Commit B: [versiongate:update=required]
```

## Precedence rules

- The resolved update policy for a Release is the **most restrictive**
  value found among all commits in the range: `required` outranks
  `optional`.
- Concretely: if *any* commit in the range specifies `required`, the
  Release's update policy is `required`, regardless of how many other
  commits specify `optional` or specify nothing.
- This rule is deterministic and independent of commit order, authorship
  order, or how commits were merged/rebased — it depends only on the
  *set* of tags present in the range, not their sequence. This matters
  because Git history order is not a reliable signal (rebases, squashes,
  and parallel branches can all reorder or restructure commits), so the
  protocol must not depend on it.
- Applying this to the example above: Commit A says `optional`, Commit B
  says `required` → the resulting Release's update policy is `required`,
  regardless of whether A or B was authored/merged first.

## Behavior for invalid metadata

- A recognized tag (`[versiongate:<key>=...]` with a known key) that has
  an unsupported value (e.g. `[versiongate:update=mandatory]`) is invalid
  metadata for that key on that commit.
- An invalid tag on a commit must not be silently ignored and must not be
  silently treated as if the commit carried no metadata. A pipeline
  integrating this protocol should surface invalid metadata as an error
  so the developer can fix the commit message, rather than allowing an
  ambiguous or unintended policy to reach a Release.
- An invalid tag on one commit does not, by itself, invalidate other
  commits' valid tags — but it must block automatic resolution for the
  Release until corrected, since the protocol cannot determine what the
  author intended.
- An unrecognized key (e.g. `[versiongate:channel=beta]` today, since only
  `update` is defined) is not an error — it is simply metadata this
  version of the protocol does not act on, preserving room for future
  keys without breaking older tooling.

## Behavior when metadata is absent

- If no commit in the range carries an `update` tag, the protocol
  resolves no update policy from commit metadata. This is not an error
  condition in the protocol itself.
- What happens in that case (e.g. falling back to a default policy,
  requiring the policy to be supplied some other way, or rejecting the
  Release) is a decision for whatever is publishing the Release, not part
  of this protocol.

## Examples

Single commit, unambiguous:

```
feat: add offline sync support

[versiongate:update=optional]
```
→ Resolved update policy: `optional`.

Multiple commits, mixed policy (most restrictive wins):

```
Commit A: fix: correct typo in settings screen
          [versiongate:update=optional]

Commit B: fix: patch critical crash on launch
          [versiongate:update=required]

Commit C: chore: update dependencies
          (no tag)
```
→ Resolved update policy: `required`.

Invalid value:

```
fix: patch critical crash on launch

[versiongate:update=forced]
```
→ Invalid metadata: `forced` is not a supported value; must be surfaced as
an error, not defaulted.

No metadata present:

```
chore: bump internal build tooling
```
→ No update policy resolved from commit metadata.
