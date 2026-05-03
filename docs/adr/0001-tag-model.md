# ADR 0001: Tag model

## Status

Accepted

## Context

Tasks need lightweight categorisation that survives the file-based storage format and is accessible to both human users and programmatic consumers (agents, scripts). The design must fit the existing frontmatter codec without breaking backward compatibility.

## Decision

### Tag format

Tags are lowercase alphanumeric strings that may contain hyphens, up to 40 characters long. The `internal/tag` package enforces this with three pure functions:

- `Normalize`: lowercases and trims whitespace.
- `Validate`: rejects characters outside `[a-z0-9-]` and lengths over 40.
- `Deduplicate`: removes duplicates preserving first-occurrence order.

### On-disk format

`task.Task` gains a `Tags []string` field. `Encode` writes `tags: [a, b]\n` when Tags is non-empty (YAML flow-style list) and omits the line when Tags is empty, following the zero-value omission convention for other optional fields. `Decode` parses the `tags:` key leniently: each tag is normalised, invalid tags are silently dropped, and duplicates are removed. Unknown frontmatter keys are already tolerated, so older binaries reading newer files with `tags:` will ignore the field.

### Store interface

`store.Add` gains a variadic `tags ...string` parameter. Tags are normalised and validated before encoding; an invalid tag causes Add to return an error. Other store operations (Close, Reopen, Update, Append, SetResearchMeta) decode the full task and re-encode it, so tags survive by construction — no special handling needed.

### CLI

`add` gains a repeatable `--tag` string flag (`add --tag bug --tag urgent "fix login"`). Invalid characters produce an error at the CLI level; uppercase is silently normalised to lowercase.

### JSON output

`render.jsonTask` and `jsonShowTask` include `Tags []string` with `json:"tags"` (no omitempty). The field is always present — an empty array `[]` rather than `null` — so agents can `jq '.tags'` without null-checking.

## Consequences

- Existing task files without `tags:` remain valid; Decode returns an empty Tags slice.
- The frontmatter format gains one optional field; the codec is backward-compatible.
- Tags cannot be added or removed after creation without editing the file or using a future `tag` subcommand (out of scope for this slice).
- The 40-character limit and the alphanumeric+hyphen restriction keep tags machine-friendly and filename-safe for future use (e.g. directory-based tag views).
