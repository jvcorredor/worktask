---
title: Configuration
description: Configuring bytheway via tasks_dir, editor, research_model, research_prompt_path, research_extra_tools, XDG paths, and resolution order.
---

`btw` is configured via a single optional TOML file. There is no `init` step: if the file is absent, `btw` runs on defaults. If it is present, every key in it is optional.

## File path

`$XDG_CONFIG_HOME/worktask/config.toml`, falling back to `~/.config/worktask/config.toml`.

The file is never created automatically. Drop one in if you want to override a default.

## Format

```toml
tasks_dir = "/custom/path"
editor    = "code --wait"

research_model       = "claude-sonnet-4-6"
research_prompt_path = "/custom/path/to/research-prompt.md"
research_extra_tools = [
  "mcp__claude_ai_Slack__slack_read_thread",
  "Bash(gh pr view:*)",
]
```

## `tasks_dir`

Where task files live. See the data directory layout in [File & storage format](./storage.md). When unset, `btw` uses `$XDG_DATA_HOME/worktask/`, falling back to `~/.local/share/worktask/`.

## `editor`

Command used by `btw edit`. The value is run as a shell-style invocation with the resolved task path appended.

Resolution order:

1. `editor` from `config.toml`
2. `$EDITOR` environment variable
3. `vi`

The first value that is set wins. `vi` is the final fallback.

## `research_model`

Claude model id used by the headless research agent (`btw research`).

When unset, the runner picks its built-in default, currently `claude-sonnet-4-6`. Sonnet is the right default for retrieval+summarization across MCP tools — the interactive default (often Opus) would burn the usage limit on every batch sweep without a meaningful quality gain here.

The `--model` flag on `btw research` overrides this key for a single invocation.

## `research_prompt_path`

Path to a custom research prompt template. When unset, `btw` ships an embedded default that is used as-is.

The default is `<config_dir>/research-prompt.md` — i.e. a sibling of `config.toml`. Drop a file at that path (or anywhere `research_prompt_path` points) to override the embedded prompt without recompiling the binary.

There is no auto-install of a starter prompt on first run. If the file does not exist, the embedded default is used.

## `research_extra_tools`

A list of allowlist entries appended verbatim to the research agent's `--allowed-tools` argument, after the in-source baseline (`Read`, `Glob`, `Grep`, `WebSearch`, `WebFetch`).

Use this key to give the research agent access to your machine's MCP servers and to pattern-restricted `Bash(...)` invocations. The shipped binary deliberately ships no MCP entries — those are install-specific and live in your `config.toml`.

```toml
research_extra_tools = [
  "mcp__claude_ai_Slack__slack_read_thread",
  "Bash(gh pr view:*)",
]
```

When the key is absent, the agent runs against only the vanilla baseline.

### Threat model

The research agent is spawned by `btw` with `--permission-mode=bypassPermissions`. It runs unattended. The `--allowed-tools` list is the security boundary: anything on it can be invoked without prompting; anything off it is denied.

That makes the contents of `research_extra_tools` security-meaningful. Two rules follow:

1. **Only add read-only entries.** The validator blocks the obvious built-in writes (see [Validation](#validation) below), but it cannot enumerate every write-capable MCP tool — there are infinite MCP server names. Per-MCP safety is your responsibility. If your Slack MCP exposes both `slack_read_thread` and `slack_send_message`, list only the read tool.
2. **`gh` is allowed only under explicit subcommand patterns.** `Bash(gh:*)` is a glob escape hatch and is not safe — `gh` has many write subcommands (`create`, `edit`, `close`, `merge`, `comment`, `delete`, `review`, `lock`, `pin`, `develop`) and `gh api` issues arbitrary HTTP methods. Pin to specific read verbs: `Bash(gh pr view:*)`, `Bash(gh issue list:*)`.

### Validation

`btw` rejects entries that would let the unattended agent write to disk or run arbitrary shell commands at config load. The error names the offending entry, its index in the list, and the path of the `config.toml` it came from, so you can locate and fix it in seconds.

Rejected entries:

| Entry | Reason |
|-------|--------|
| `Edit` | Built-in write tool. |
| `Write` | Built-in write tool. |
| `NotebookEdit` | Built-in write tool. |
| `TodoWrite` | Built-in write tool. |
| `Bash` | Bare `Bash` is unrestricted shell access. |
| `Bash(*)` | Wildcard escape hatch — equivalent to bare `Bash`. |
| `Bash(*:...)` | Wildcard binary in front of any subcommand — equivalent to bare `Bash`. |

MCP write methods are not enumerated. They are not blocked. Per-MCP safety is your responsibility per the threat model above.

### Starter snippets

Drop one or more of these into your `config.toml` under `research_extra_tools`. Each block is read-only by design. Mix and match to match the MCPs you actually have installed.

#### Atlassian (Jira + Confluence) — read-only

```toml
research_extra_tools = [
  "mcp__claude_ai_Atlassian__getJiraIssue",
  "mcp__claude_ai_Atlassian__searchJiraIssuesUsingJql",
  "mcp__claude_ai_Atlassian__getJiraIssueRemoteIssueLinks",
  "mcp__claude_ai_Atlassian__getConfluencePage",
  "mcp__claude_ai_Atlassian__searchConfluenceUsingCql",
  "mcp__claude_ai_Atlassian__getConfluencePageDescendants",
  "mcp__claude_ai_Atlassian__getConfluencePageFooterComments",
  "mcp__claude_ai_Atlassian__getConfluencePageInlineComments",
  "mcp__claude_ai_Atlassian__getConfluenceCommentChildren",
  "mcp__claude_ai_Atlassian__getPagesInConfluenceSpace",
  "mcp__claude_ai_Atlassian__getConfluenceSpaces",
  "mcp__claude_ai_Atlassian__getIssueLinkTypes",
  "mcp__claude_ai_Atlassian__getJiraProjectIssueTypesMetadata",
  "mcp__claude_ai_Atlassian__getJiraIssueTypeMetaWithFields",
  "mcp__claude_ai_Atlassian__getTransitionsForJiraIssue",
  "mcp__claude_ai_Atlassian__getVisibleJiraProjects",
  "mcp__claude_ai_Atlassian__lookupJiraAccountId",
  "mcp__claude_ai_Atlassian__atlassianUserInfo",
  "mcp__claude_ai_Atlassian__getAccessibleAtlassianResources",
  "mcp__claude_ai_Atlassian__search",
  "mcp__claude_ai_Atlassian__fetch",
]
```

#### Slack — read-only

```toml
research_extra_tools = [
  "mcp__claude_ai_Slack__slack_read_channel",
  "mcp__claude_ai_Slack__slack_read_thread",
  "mcp__claude_ai_Slack__slack_read_canvas",
  "mcp__claude_ai_Slack__slack_read_user_profile",
  "mcp__claude_ai_Slack__slack_search_channels",
  "mcp__claude_ai_Slack__slack_search_public",
  "mcp__claude_ai_Slack__slack_search_public_and_private",
  "mcp__claude_ai_Slack__slack_search_users",
]
```

#### Datadog — read-only

```toml
research_extra_tools = [
  "mcp__claude_ai_Datadog__aggregate_events",
  "mcp__claude_ai_Datadog__aggregate_rum_events",
  "mcp__claude_ai_Datadog__aggregate_spans",
  "mcp__claude_ai_Datadog__analyze_datadog_logs",
  "mcp__claude_ai_Datadog__get_datadog_dashboard",
  "mcp__claude_ai_Datadog__get_datadog_incident",
  "mcp__claude_ai_Datadog__get_datadog_metric",
  "mcp__claude_ai_Datadog__get_datadog_metric_context",
  "mcp__claude_ai_Datadog__get_datadog_notebook",
  "mcp__claude_ai_Datadog__get_datadog_trace",
  "mcp__claude_ai_Datadog__search_datadog_dashboards",
  "mcp__claude_ai_Datadog__search_datadog_events",
  "mcp__claude_ai_Datadog__search_datadog_hosts",
  "mcp__claude_ai_Datadog__search_datadog_incidents",
  "mcp__claude_ai_Datadog__search_datadog_logs",
  "mcp__claude_ai_Datadog__search_datadog_metrics",
  "mcp__claude_ai_Datadog__search_datadog_metrics_v2",
  "mcp__claude_ai_Datadog__search_datadog_monitors",
  "mcp__claude_ai_Datadog__search_datadog_notebooks",
  "mcp__claude_ai_Datadog__search_datadog_rum_events",
  "mcp__claude_ai_Datadog__search_datadog_service_dependencies",
  "mcp__claude_ai_Datadog__search_datadog_services",
  "mcp__claude_ai_Datadog__search_datadog_spans",
]
```

#### context7 — library / SDK / framework documentation

```toml
research_extra_tools = [
  "mcp__context7__resolve-library-id",
  "mcp__context7__query-docs",
]
```

#### `gh` CLI — read-only Bash patterns

```toml
research_extra_tools = [
  "Bash(gh issue view:*)",
  "Bash(gh issue list:*)",
  "Bash(gh pr view:*)",
  "Bash(gh pr list:*)",
  "Bash(gh pr diff:*)",
  "Bash(gh search:*)",
]
```

`gh api` is intentionally not on this list. It can issue arbitrary HTTP methods (`-X POST/PATCH/DELETE`, or `-f field=value` which auto-switches to POST), so the read/write boundary collapses to the `gh` auth token's scopes rather than the argv pattern.

### Migrating from older `btw`

Earlier versions of `btw` shipped a personal allowlist hard-coded into the binary — Atlassian, Slack, Datadog, and `gh` MCP tool names were all in-source. That is no longer the case: the shipped binary's allowlist is `Read`, `Glob`, `Grep`, `WebSearch`, `WebFetch`, and nothing else.

If you were relying on the old in-source allowlist, add a `research_extra_tools` block to your `config.toml` per machine. The starter snippets above are drop-in equivalents to the previously-bundled lists.
