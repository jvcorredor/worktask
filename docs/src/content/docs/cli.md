---
title: CLI reference
description: Subcommand reference for the worktask CLI — flags, examples, and match-resolution behavior.
---

Every subcommand `worktask` exposes is documented on this page. For the on-disk shape of task files, see [File & storage format](./storage.md). For JSON output and error envelopes, see [Agentic usage](./agentic.md).

## Global flag

`--format=human|json` (default `human`).

JSON output is intended for agentic consumers; the schema is documented as stable. Human output is the default and is what you see when running `worktask` directly in a terminal.

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

Prints a single task. In `--format=human`, it prints the raw markdown file. In `--format=json`, it prints a JSON object with `id`, `created`, `description`, `body`, and (if closed) `completed`.

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

## Match resolution

For commands that take `<fragment>`, `worktask` tries strategies in order and stops at the first one that yields any matches:

1. ID exact
2. ID prefix
3. Description exact (case-insensitive)
4. Description substring (case-insensitive)

If a strategy yields exactly one match, that is the result. If it yields more than one, the command exits non-zero with `ErrAmbiguous` and lists the candidates. If no strategy matches, it exits non-zero with `ErrNoMatch` and lists the current open tasks so the caller can pick.

In `--format=json`, both error cases produce a JSON envelope with a stable `error` discriminator (`"ambiguous"` or `"no_match"`) so agents can branch without parsing prose. See [Agentic usage](./agentic.md) for the envelope shapes.
