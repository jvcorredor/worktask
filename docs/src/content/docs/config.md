---
title: Configuration
description: Configuring worktask via tasks_dir, editor, XDG paths, and resolution order.
---

`worktask` is configured via a single optional TOML file. There is no `init` step: if the file is absent, `worktask` runs on defaults. If it is present, every key in it is optional.

## File path

`$XDG_CONFIG_HOME/worktask/config.toml`, falling back to `~/.config/worktask/config.toml`.

The file is never created automatically. Drop one in if you want to override a default.

## Format

```toml
tasks_dir = "/custom/path"
editor    = "code --wait"
```

## `tasks_dir`

Where task files live. See the data directory layout in [File & storage format](./storage.md). When unset, `worktask` uses `$XDG_DATA_HOME/worktask/`, falling back to `~/.local/share/worktask/`.

## `editor`

Command used by `worktask edit`. The value is run as a shell-style invocation with the resolved task path appended.

Resolution order:

1. `editor` from `config.toml`
2. `$EDITOR` environment variable
3. `vi`

The first value that is set wins. `vi` is the final fallback.
