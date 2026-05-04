---
title: Agentic usage
description: Using worktask from an LLM agent — JSON schema, error envelopes, and the reference slash command pattern.
---

`worktask` is built primarily as a tool for an LLM agent (Claude Code) to call from a thin slash command. Match resolution, ID generation, slug generation, and formatting all live in the CLI so the agent never has to read or rewrite task files itself. A read-only `list` or `show` costs the agent only the bytes the CLI prints, not the entire file store.

This page documents the contract between the CLI and an agent wrapper: the JSON schema returned by every subcommand, the error envelopes the agent must branch on, and the dispatch pattern the reference slash command uses.

## JSON output

Pass `--format=json` to any subcommand to get machine-readable output. The schema is documented as stable.

`list` returns an array of:

```json
{
  "id": "abcdef12",
  "created": "2026-04-29T11:30:00Z",
  "description": "buy milk",
  "completed": "2026-04-30T08:00:00Z"
}
```

`completed` is omitted on open tasks.

`show` returns the same fields plus `body` (the full body string after the first description line) and `path` (the absolute, cleaned filesystem path of the task file; symlinks in the configured tasks directory are preserved verbatim). Two optional research fields are emitted only when the task has been researched: `last_researched` (RFC 3339 timestamp in UTC) and `last_research_log` (absolute path to the research log file). Both mirror the YAML frontmatter keys and are omitted when empty.

Golden fixtures for the JSON schema live in `internal/render/testdata/` in the repo and are the authoritative reference for any wire-format edge cases.

## Error envelopes

When a `<fragment>` resolves ambiguously or not at all, the command exits non-zero. In `--format=json`, both cases produce a stable JSON envelope with an `error` discriminator so agents can branch without parsing prose.

```json
{ "error": "ambiguous", "fragment": "milk", "matches": [{ "id": "...", "description": "...", "tags": [] }] }
```

```json
{ "error": "no_match",  "fragment": "zzz",  "open_tasks": [ /* same shape as list */ ] }
```

For the underlying resolution rules, see [Match resolution](./cli.md#match-resolution) on the CLI reference page.

## Slash-command dispatch

The reference slash command for Claude Code is a thin wrapper. It does not read or parse task files directly — the CLI owns all of that.

| Subcommands | How the wrapper invokes them | Notes |
|-------------|------------------------------|-------|
| `add`, `list`, `show`, `update`, `append`, `close`, `reopen` | Shell out with `--format=json`. On exit 0, pass success output through. On non-zero exit, branch on the `error` discriminator. | Uniform shape across all of them. |
| `edit` | Does **not** shell out. Validate the fragment via `show`, then tell the user to run `worktask edit <id>` from their own shell. | The binary `syscall.Exec`s `$EDITOR`, which is unusable from an agent shell. |

## Reference slash command

A vendored reference implementation of the Claude Code slash command lives in this repo at `docs/src/content/docs/examples/worktask-slash-command.md`. The full source is rendered on the [Reference slash command](./examples/worktask-slash-command.md) page — read it end-to-end to see the dispatch table, error-envelope branching, and the `edit` / `research` special cases in their working form.

:::tip[Curl the canonical source]
For templating into another agent, fetch the raw markdown directly:

```
curl -fsSL https://raw.githubusercontent.com/jvcorredor/worktask/main/docs/src/content/docs/examples/worktask-slash-command.md
```

The file's frontmatter is Claude-Code-compatible (`description:`, `argument-hint:`); the additional `title:` key Starlight needs is ignored by Claude Code, so the raw download is a drop-in slash command source.
:::

## Research-agent allowlist

`worktask research` spawns a headless Claude Code subagent under `--permission-mode=bypassPermissions`, with an explicit `--allowed-tools` list. That list is the security boundary, and it is config-driven on a per-machine basis: the shipped binary contributes a vanilla baseline of five built-in read-only tools (`Read`, `Glob`, `Grep`, `WebSearch`, `WebFetch`), and the user's `config.toml` extends it via [`research_extra_tools`](./config.md#research_extra_tools). MCP tool names and pattern-restricted `Bash(...)` invocations belong in the config, not in the binary.

If you are wrapping `worktask research` from another agent, you do not need to thread tool names through the wrapper — they live in the operator's `config.toml`. The CLI surface (subcommand name, flags, JSON output, exit-code envelope) is unchanged by the allowlist mechanism.

## Skills

Claude Code also exposes a wrapper surface called *skills* alongside slash commands. A `worktask` skill is a viable future direction — no worked example is included here yet because there isn't one to vendor.
