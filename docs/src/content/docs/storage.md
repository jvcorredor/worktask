---
title: File & storage format
description: How worktask lays out tasks on disk and the on-disk shape of each task file.
---

`worktask` keeps each task as its own markdown file. Status is encoded by which subdirectory the file lives in: `open/` or `closed/`. The data directory is plain enough that `cat`, `grep`, `rg`, and `vim` work directly without going through the binary.

## Data directory

Default: `$XDG_DATA_HOME/worktask/`, falling back to `~/.local/share/worktask/`. Override with the `tasks_dir` key in the config file (see [Configuration](./config.md)).

```
<tasks_dir>/
  open/
    2026-04-29T11-30_abcdef12_buy-milk.md
  closed/
    2026-04-15T09-12_12345678_old-thing.md
```

## Filename

`<created-ts>_<id>_<slug>.md`.

- `created-ts` is filesystem-safe ISO: `YYYY-MM-DDTHH-MM`, with the time component's colons replaced by hyphens.
- `id` is an 8-character lowercase hex string from `crypto/rand`.
- `slug` is generated from the description at creation and frozen. `update` never renames the file.

## File format

YAML frontmatter followed by a free markdown body:

```
---
id: abcdef12
created: 2026-04-29T11:30:00Z
---
buy milk
remember the brand
```

The first body line is the canonical description and is the target of `update`. Everything below it is freeform — `append` adds to the body, preserving everything above.

`completed:` is added to the frontmatter when a task is closed and removed when it is reopened. Open tasks have no `completed` field.
