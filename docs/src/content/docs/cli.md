---
title: CLI reference
description: Subcommand reference for the bytheway CLI — flags, examples, and match-resolution behavior.
---

Every subcommand `btw` exposes is documented on this page. For the on-disk shape of task files, see [File & storage format](./storage.md). For JSON output and error envelopes, see [Agentic usage](./agentic.md).

## Global flags

`--format=human|json` (default `human`).

JSON output is intended for agentic consumers; the schema is documented as stable. Human output is the default and is what you see when running `btw` directly in a terminal.

`--version` (alias `-v`) prints the binary's version on a single line and exits — see the [`version`](#version) subcommand below for the underlying resolution rules.

## Subcommands at a glance

```
btw add <description>
btw list [--all] [--closed] [--limit N] [--tag TAG]
btw show <fragment>
bytheway update <fragment> <new description>
bytheway append <fragment> <text>
btw close <fragment>
bytheway reopen <fragment>
btw edit <fragment>
btw tag add <fragment> <tag>
btw tag rm <fragment> <tag>
btw tag ls [--all] [--closed]
btw research [<fragment>]
btw version
```

`<fragment>` is anything that resolves to a single task. See [Match resolution](#match-resolution) below.

There is no `delete` subcommand by design. Removing a task means `rm` against the data directory.

## `add`

Creates a new task in `open/`, prints `added <id>`.

```
$ btw add buy milk
added abcdef12
```

The slug is generated from the description and frozen on the file. `update` later replaces the description without renaming the file.

## `list`

Lists open tasks by default.

Flags:

- `--all` includes closed tasks (truncated to the most recent 20).
- `--closed` shows closed tasks only.
- `--limit N` overrides the truncation.
- `--tag TAG` filters to tasks carrying the tag (single value, exact match after lowercase normalization). Composes as logical AND with `--all`/`--closed`. A non-matching tag returns an empty list with no error; an invalid tag value (whitespace, uppercase that doesn't normalize cleanly, characters outside `[a-z0-9-]`) is rejected.

```
$ btw list
$ btw list --all
$ btw list --closed --limit 50
$ btw list --tag bug
$ btw list --tag infra --all
```

In `--format=human`, the rendering is TTY-aware:

- When stdout is a terminal, rows render as a borderless aligned table with a subdued `ID  CREATED  DESCRIPTION` header, IDs in an accent colour, dates faint and date-only (`YYYY-MM-DD`), and descriptions hard-truncated with `…` to fit the terminal width. `NO_COLOR` disables colour while preserving layout.
- When stdout is piped, redirected, or running in a non-TTY environment (CI, scripts), output is plain `id  YYYY-MM-DD HH:MM  description` rows with no header and no ANSI escapes — the contract callers rely on for `grep`, `awk`, and other text tools.

In `--format=json`, output is a JSON array; the schema is documented in [Agentic usage](./agentic.md) and is unchanged by TTY detection.

## `show`

Prints a single task.

In `--format=human`, the rendering is TTY-aware:

- When stdout is a terminal, the body is rendered as styled Markdown (headings, code blocks, lists, tables) via [glamour](https://github.com/charmbracelet/glamour) using its `auto` style, wrapped at the terminal width capped at 120 columns. A subdued one-line metadata strip — `{id} · {YYYY-MM-DD created} · {open|closed}` — is printed above the body, followed by a second faint line bearing the absolute task file path (no label, no quoting — bare path only). The path line lets a double-click select the whole path cleanly; terminal soft-wrap handles overflow on narrow terminals. Operational fields like `last_researched` and `last_research_log` are intentionally omitted; agents can still get them via `--format=json`. `NO_COLOR` disables colour while preserving layout.
- When stdout is piped, redirected, or running in a non-TTY environment (CI, scripts), the raw markdown file (frontmatter included) is written verbatim — unchanged and byte-identical to the pre-path-line behaviour — preserving any workflow that pipes `show` output into an editor or other tooling.

In `--format=json`, it prints a JSON object with `id`, `created`, `description`, `body`, `path`, and (if closed) `completed`. When the task has been researched, the object also carries `last_researched` (RFC 3339 UTC) and `last_research_log` (absolute path to the latest run's JSONL log); both are omitted when un-researched. `path` is the absolute, cleaned filesystem path of the task file; symlinks in the configured tasks directory are preserved verbatim. JSON output is unchanged by TTY detection.

```
$ btw show milk
$ btw --format=json show abcdef12
$ btw --format=json show abcdef12 | jq -r .path               # the file on disk
$ btw --format=json show abcdef12 | jq -r .last_research_log  # latest research log
```

Older releases shipped a dedicated `btw research-log <fragment>` subcommand that printed only the latest research-log path; it has been removed in favour of the `show --format=json | jq -r .last_research_log` form above. Existing scripts that called `btw research-log` should switch to that pipeline.

## `update`

Replaces the first body line — the canonical description — without touching frontmatter, ID, slug, or filename.

```
$ bytheway update milk buy oat milk instead
```

## `append`

Adds text to the body of a task, preserving everything above it.

```
$ bytheway append milk remember the brand
```

## `close`

Moves the file from `open/` to `closed/` via `os.Rename` and stamps `completed:` into the frontmatter.

```
$ btw close milk
```

## `reopen`

The inverse of `close`: moves the file back to `open/` and clears the `completed:` field.

```
$ bytheway reopen milk
```

## `tag add`

Adds a tag to an existing task. The tag is normalized (lower-cased, trimmed) and validated against `[a-z0-9-]` up to 40 characters. Re-running with the same tag is a no-op.

```
$ btw tag add milk urgent
tagged abcdef12 with urgent
```

## `tag rm`

Removes a tag from an existing task. Re-running with an absent tag is a no-op. When the last tag is removed, the `tags:` frontmatter line is omitted entirely.

```
$ btw tag rm milk urgent
removed urgent from abcdef12
```

## `tag ls`

Reports each distinct tag in the corpus along with the count of tasks carrying it, sorted alphabetically by tag name. Tags with zero occurrences never appear — the listing is derived from task files; tags are not pre-registered.

Flags:

- `--all` includes closed tasks alongside open.
- `--closed` scans closed tasks only.

Default is open tasks only — same semantics as [`list`](#list).

In `--format=human`, output is a single space-padded line of `name (count)` entries, no header, no ANSI escapes. An empty corpus emits a blank line.

```
$ btw tag ls
bug (3)  infra (5)  urgent (2)
```

In `--format=json`, output is `{"tags": [{"name": ..., "count": ...}, ...]}`. The `tags` key is always present (empty array when none) so consumers can `jq '.tags'` without null-checking.

```
$ btw --format=json tag ls
{
  "tags": [
    {"name": "bug",    "count": 3},
    {"name": "infra",  "count": 5},
    {"name": "urgent", "count": 2}
  ]
}
```

## `edit`

`syscall.Exec`s `$EDITOR` (or the configured override, falling back to `vi`) on the resolved task path. See [Configuration](./config.md) for editor resolution.

```
$ btw edit milk
```

This command is not usable from an agent shell because it replaces the calling process with the editor. The reference slash command instructs the user to run `btw edit <id>` from their own shell instead.

## `research`

Spawns a headless Claude Code agent to gather context for one task or sweep every open task. The agent runs unattended with `--permission-mode=bypassPermissions`; the security boundary is the `--allowed-tools` list passed to it.

```
$ btw research milk        # research a single task
$ btw research              # batch-research every open task
```

Flags:

- `--concurrency N` (default `3`): max concurrent in-flight research runs in batch mode.
- `--all`: force re-research of every open task, ignoring `last_researched`.
- `--stale DURATION`: re-research tasks whose `last_researched` is older than this duration.
- `--tag TAG` (batch mode only): scope the sweep to open tasks carrying this tag (exact match, lowercased before filtering). Composes as an additional AND with `--all`/`--stale` — tag filter is applied first, then staleness logic runs over the surviving tasks. A non-matching tag yields a normal completion with zero tasks researched. Rejected with an error in single-task mode (when a `<fragment>` is supplied).
- `--timeout DURATION` (default `10m`): per-task hard timeout.
- `--model ID`: claude model id; overrides `research_model` from `config.toml`.

The shipped binary's `--allowed-tools` baseline contains five built-in read-only tools and nothing else: `Read`, `Glob`, `Grep`, `WebSearch`, `WebFetch`. To extend the allowlist with MCP tools or pattern-restricted `Bash(...)` invocations on a per-machine basis, add a [`research_extra_tools`](./config.md#research_extra_tools) block to your `config.toml`. There is no `--extra-tool` flag; tool availability is a per-machine property, not a per-invocation one.

### Batch outcome reporting

In a batch run (`btw research` or `btw research --all`), every task makes it into the final summary, even when prep- or post-run plumbing fails. Two cases that previously fell through the cracks now surface as `failed` outcomes:

- **Writeback errors** (the agent succeeded but the cmd could not append the Research section, set the `last_researched` frontmatter, or append the worklog line — for example, the task file went missing mid-run, or the worklog directory is unwritable) emit a `task_failed` event and a `failed` row in the summary. The agent's original summary text is preserved in the row's `summary` field; the writeback error message goes into `error`. Previously these were silently reported as if the writeback had succeeded.
- **Per-task `prompt.Render` failures during prep** (a malformed override template, or a task whose frontmatter does not satisfy a custom template's field references) emit a synthetic `task_failed` event for that task and a `failed` row in the summary. Other tasks in the batch still run. Previously a single render failure aborted the whole sweep before any task ran.

Both cases count toward `totals.failed` in the summary; the summary's `totals.skipped` continues to count only the staleness-based skip path.

JSON output is documented in [Agentic usage](./agentic.md).

## `version`

Prints the binary's build identity. The same identity is what `btw --version` and `btw -v` report on a single line via cobra's built-in version flag.

```
$ btw version
bytheway v1.2.3 (commit abcdef1, built 2026-05-03T10:00:00Z)

$ btw --format=json version
{
  "version": "v1.2.3",
  "commit": "abcdef1",
  "date": "2026-05-03T10:00:00Z",
  "source": "ldflags"
}
```

The JSON schema is `{"version": string, "commit": string, "date": string, "source": string}`. The `source` field records which resolution branch produced the values: `"ldflags"` for release builds with `-ldflags="-X ..."`, `"buildinfo"` for binaries produced by `go install module@version` (where the version is read from the Go module proxy via `runtime/debug.ReadBuildInfo`), and `"default"` for unannotated developer builds (which report `"version": "dev"`).

## Match resolution

For commands that take `<fragment>`, `btw` tries strategies in order and stops at the first one that yields any matches:

1. ID exact
2. ID prefix
3. Description exact (case-insensitive)
4. Description substring (case-insensitive)

If a strategy yields exactly one match, that is the result. If it yields more than one, the command exits non-zero with `ErrAmbiguous` and lists the candidates. If no strategy matches, it exits non-zero with `ErrNoMatch` and lists the current open tasks so the caller can pick.

In `--format=json`, both error cases produce a JSON envelope with a stable `error` discriminator (`"ambiguous"` or `"no_match"`) so agents can branch without parsing prose. See [Agentic usage](./agentic.md) for the envelope shapes.

## Shell completion

Cobra emits completion scripts for bash, zsh, fish, and powershell via the hidden `completion <shell>` subcommand. Generate once and write to your shell's completions directory; the script delegates back to the binary at <kbd>Tab</kbd> time, so completions stay in sync as new subcommands ship without regenerating the script.

```
# fish
btw completion fish > ~/.config/fish/completions/btw.fish

# zsh (paths vary by setup)
btw completion zsh > "${fpath[1]}/_btw"

# bash
btw completion bash > /etc/bash_completion.d/btw
```

For commands that take `<fragment>` — `show`, `close`, `reopen`, `edit`, `update`, `append`, `research`, `tag add`, `tag rm` — pressing <kbd>Tab</kbd> on the fragment position lists candidate tasks as `<id>\t<first body line>`, which fish and zsh render as the id alongside the task description. The candidate set is scoped per command:

- `close`: open tasks only
- `reopen`: closed tasks only
- `research`: open tasks only
- everything else: open and closed

Subsequent positional arguments (e.g. the `<text>` of `update <fragment> <text>`) return no candidates and suppress filename fallback, so a stray <kbd>Tab</kbd> on the second arg does not flood the menu with files from the current directory.
