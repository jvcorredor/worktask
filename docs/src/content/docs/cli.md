---
title: CLI reference
description: Subcommand reference for the worktask CLI — flags, examples, and match-resolution behavior.
---

Every subcommand `worktask` exposes is documented on this page. For the on-disk shape of task files, see [File & storage format](./storage.md). For JSON output and error envelopes, see [Agentic usage](./agentic.md).

## Global flags

`--format=human|json` (default `human`).

JSON output is intended for agentic consumers; the schema is documented as stable. Human output is the default and is what you see when running `worktask` directly in a terminal.

`--version` (alias `-v`) prints the binary's version on a single line and exits — see the [`version`](#version) subcommand below for the underlying resolution rules.

## Subcommands at a glance

```
worktask add <description>
worktask list [--all] [--closed] [--limit N]
worktask show <fragment>
worktask update <fragment> <new description>
worktask append <fragment> <text>
worktask complete <fragment>
worktask reopen <fragment>
worktask edit <fragment>
worktask research [<fragment>]
worktask version
```

`<fragment>` is anything that resolves to a single task. See [Match resolution](#match-resolution) below.

There is no `delete` subcommand by design. Removing a task means `rm` against the data directory.

## `add`

Creates a new task in `open/`, prints `added <id>`.

```
$ worktask add buy milk
added abcdef12
```

The slug is generated from the description and frozen on the file. `update` later replaces the description without renaming the file.

## `list`

Lists open tasks by default.

Flags:

- `--all` includes closed tasks (truncated to the most recent 20).
- `--closed` shows closed tasks only.
- `--limit N` overrides the truncation.

```
$ worktask list
$ worktask list --all
$ worktask list --closed --limit 50
```

## `show`

Prints a single task.

In `--format=human`, the rendering is TTY-aware:

- When stdout is a terminal, the body is rendered as styled Markdown (headings, code blocks, lists, tables) via [glamour](https://github.com/charmbracelet/glamour) using its `auto` style, wrapped at the terminal width capped at 120 columns. A subdued one-line metadata strip — `{id} · {YYYY-MM-DD created} · {open|closed}` — is printed above the body. Operational fields like `last_researched` and `last_research_log` are intentionally omitted; agents can still get them via `--format=json`. `NO_COLOR` disables colour while preserving layout.
- When stdout is piped, redirected, or running in a non-TTY environment (CI, scripts), the raw markdown file (frontmatter included) is written verbatim, preserving any workflow that pipes `show` output into an editor or other tooling.

In `--format=json`, it prints a JSON object with `id`, `created`, `description`, `body`, and (if closed) `completed`. JSON output is unchanged by TTY detection.

```
$ worktask show milk
$ worktask --format=json show abcdef12
```

## `update`

Replaces the first body line — the canonical description — without touching frontmatter, ID, slug, or filename.

```
$ worktask update milk buy oat milk instead
```

## `append`

Adds text to the body of a task, preserving everything above it.

```
$ worktask append milk remember the brand
```

## `complete`

Moves the file from `open/` to `closed/` via `os.Rename` and stamps `completed:` into the frontmatter.

```
$ worktask complete milk
```

## `reopen`

The inverse of `complete`: moves the file back to `open/` and clears the `completed:` field.

```
$ worktask reopen milk
```

## `edit`

`syscall.Exec`s `$EDITOR` (or the configured override, falling back to `vi`) on the resolved task path. See [Configuration](./config.md) for editor resolution.

```
$ worktask edit milk
```

This command is not usable from an agent shell because it replaces the calling process with the editor. The reference slash command instructs the user to run `worktask edit <id>` from their own shell instead.

## `research`

Spawns a headless Claude Code agent to gather context for one task or sweep every open task. The agent runs unattended with `--permission-mode=bypassPermissions`; the security boundary is the `--allowed-tools` list passed to it.

```
$ worktask research milk        # research a single task
$ worktask research              # batch-research every open task
```

Flags:

- `--concurrency N` (default `3`): max concurrent in-flight research runs in batch mode.
- `--all`: force re-research of every open task, ignoring `last_researched`.
- `--stale DURATION`: re-research tasks whose `last_researched` is older than this duration.
- `--timeout DURATION` (default `10m`): per-task hard timeout.
- `--model ID`: claude model id; overrides `research_model` from `config.toml`.

The shipped binary's `--allowed-tools` baseline contains five built-in read-only tools and nothing else: `Read`, `Glob`, `Grep`, `WebSearch`, `WebFetch`. To extend the allowlist with MCP tools or pattern-restricted `Bash(...)` invocations on a per-machine basis, add a [`research_extra_tools`](./config.md#research_extra_tools) block to your `config.toml`. There is no `--extra-tool` flag; tool availability is a per-machine property, not a per-invocation one.

JSON output is documented in [Agentic usage](./agentic.md).

## `version`

Prints the binary's build identity. The same identity is what `worktask --version` and `worktask -v` report on a single line via cobra's built-in version flag.

```
$ worktask version
worktask v1.2.3 (commit abcdef1, built 2026-05-03T10:00:00Z)

$ worktask --format=json version
{
  "version": "v1.2.3",
  "commit": "abcdef1",
  "date": "2026-05-03T10:00:00Z",
  "source": "ldflags"
}
```

The JSON schema is `{"version": string, "commit": string, "date": string, "source": string}`. The `source` field records which resolution branch produced the values: `"ldflags"` for release builds with `-ldflags="-X ..."`, `"buildinfo"` for binaries produced by `go install module@version` (where the version is read from the Go module proxy via `runtime/debug.ReadBuildInfo`), and `"default"` for unannotated developer builds (which report `"version": "dev"`).

## Match resolution

For commands that take `<fragment>`, `worktask` tries strategies in order and stops at the first one that yields any matches:

1. ID exact
2. ID prefix
3. Description exact (case-insensitive)
4. Description substring (case-insensitive)

If a strategy yields exactly one match, that is the result. If it yields more than one, the command exits non-zero with `ErrAmbiguous` and lists the candidates. If no strategy matches, it exits non-zero with `ErrNoMatch` and lists the current open tasks so the caller can pick.

In `--format=json`, both error cases produce a JSON envelope with a stable `error` discriminator (`"ambiguous"` or `"no_match"`) so agents can branch without parsing prose. See [Agentic usage](./agentic.md) for the envelope shapes.
