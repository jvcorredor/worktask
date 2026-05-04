---
title: Agentic recipes
description: The capture / soak / investigate / hand off funnel as recipes against existing bytheway primitives.
---

`btw` is built around a four-stage funnel: **capture** a fast inbound, **soak** it as more context arrives, **investigate** once the seed material is dense enough, and **hand off** the investigated brief to a downstream skill that shapes it.

The CLI exposes user-triggered primitives for capture and investigation. It does not chain stages together — every stage fires when you (or the calling agent) say so. There is no `capture` composite verb, no `add --research` flag, and no shipped wrapper skill. The recipes below show the canonical end-to-end pattern using only the primitives the binary already exposes.

## Role boundary

Bytheway owns **capture** and **investigation**. Shaping the investigated brief into a plan, a PRD, an issue, or a draft message — and delivering it to wherever it needs to go — are downstream responsibilities. Skills like `/grill-me` and `/grill-with-docs` are typical handoff targets; bytheway's job ends when it has written investigated starting points back into the task body.

## 1. Capture

Catch the inbound before it disappears. One line of intent is enough — slug, ID, and timestamp are generated for you.

```
$ btw add "slack thread from leadership about renewal churn"
added 1e3900e4
```

The returned ID is the only handle you need for everything that follows. A description fragment also resolves to the task in later commands (see [Match resolution](./cli.md#match-resolution)) — you do not have to remember IDs.

Capture is deliberately cheap. Do not stop to write context, attach links, or pick tags at this stage; that belongs in soak.

## 2. Soak

Soak is the gap between capture and investigation. As more context trickles in — a follow-up Slack message, a meeting decision, a related ticket — append it to the captured task. The body grows over time until the seed material is dense enough that an investigation run will produce something useful.

Append free-form context with [`append`](./cli.md#append):

```
$ bytheway append 1e3900e4 "ARR impact called out in Q1 board deck — see drive link in #renewal-ops"
$ bytheway append 1e3900e4 "engineering counterpoint: cohort 2024-Q3 was a one-off, not a trend"
```

Replace the canonical description with [`update`](./cli.md#update) when the framing of the task itself shifts:

```
$ bytheway update 1e3900e4 "renewal churn — cohort 2024-Q3 anomaly vs. trend"
```

For larger edits — restructuring the body, pasting a long quote — open the file in your editor with [`edit`](./cli.md#edit). The CLI hands off to `$EDITOR` directly; agents validate the fragment with `show` and then ask the user to run `btw edit <id>` from their own shell, since `syscall.Exec` is unusable from an agent shell.

Tagging is optional and belongs to soak. Tags are triage labels, not generic categorization — use them when you want to group captures for a later batch sweep:

```
$ btw tag add 1e3900e4 renewal
```

You can later scope investigation to that triage bucket with `btw research --tag renewal` (see below).

### When *not* to fire research yet

Investigation runs cost an agent run each, and the quality of the output is bounded by the seed material. Resist firing `research` against:

- A capture that is still a one-line headline with no surrounding context.
- A capture whose framing is still in flux — if you expect to `update` the description in the next hour or day, soak first.
- A capture you do not yet plan to act on. Investigation does not pre-warm a queue; it produces starting points that go stale if left unread.

Soak until the body would tell a stranger what the task is *about*, then move to investigate.

## 3. Investigate

Once the seed is dense enough, fire [`research`](./cli.md#research). The CLI spawns a headless Claude Code subagent that gathers context using a config-driven allowlist of read-only tools, and writes findings back into the task body. See [Agentic usage → Research-agent allowlist](./agentic.md#research-agent-allowlist) for the security boundary.

Single task:

```
$ btw research 1e3900e4
```

Batch every open task that has not been researched yet:

```
$ btw research
```

Sweep just the soak-triage bucket you tagged earlier:

```
$ btw research --tag renewal
```

Research runs are unattended and can take minutes per task. From a slash command, dispatch via a background bash and return an immediate ack to the user; the harness notifies the agent when the run lands. The vendored [reference slash command](./examples/bytheway-slash-command.md) shows the full pattern.

## 4. Hand off

After investigation, the task body carries the agent's findings, and the YAML frontmatter carries two timestamps the next stage needs to read. Both are surfaced verbatim in `show --format=json`:

- `last_researched` — RFC 3339 UTC timestamp of the most recent successful run. Omitted on un-researched tasks.
- `last_research_log` — absolute filesystem path to the run's log file. Omitted on un-researched tasks.

A handoff recipe pulls the body and the log path out of `show` and feeds them to the downstream skill:

```
$ btw --format=json show 1e3900e4 | jq -r .body
$ btw --format=json show 1e3900e4 | jq -r .last_research_log
$ btw --format=json show 1e3900e4 | jq -r .last_researched
```

A downstream slash command — `/grill-me`, `/grill-with-docs`, or any project-local equivalent — reads those fields, optionally `cat`s the log, and turns the investigated material into a plan, a PRD, an issue, or a draft message. That shaping step is outside bytheway's scope by design.

Branching on whether a task has been investigated is a one-liner against the same JSON:

```
$ btw --format=json show 1e3900e4 | jq -e 'has("last_researched")'
```

Returns 0 if the task has been researched, non-zero otherwise — useful for a wrapper that wants to soak-or-handoff dispatch without a second CLI call.

## Where to go next

- [Agentic usage](./agentic.md) — JSON schema, error envelopes, and the slash-command dispatch pattern.
- [Reference slash command](./examples/bytheway-slash-command.md) — the vendored Claude Code slash command, end-to-end.
- [CLI reference](./cli.md) — every primitive cited above, with flags and examples.
