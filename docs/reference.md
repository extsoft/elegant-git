---
layout: default
title: Reference
nav_order: 11
has_children: true
has_toc: true
---

# Reference

The reference pages are the lookup material — complete lists, exact paths, and exact names. If you
are trying to understand how a command behaves while you use it, start with the object chapters
instead: ["self" guide](05-00-self.md), ["workspace" guide](06-00-workspace.md),
["repo" guide](07-00-repo.md), ["work" guide](08-00-work.md), ["release" guide](09-00-release.md),
["hook" guide](10-00-hook.md).

- [Configuration](reference/configuration.md) — what `eg self configure` and
  `eg repo configure` apply.
- [Memory](reference/memory.md) — where Elegant Git keeps its own state.
- [Migrations](reference/migrations.md) — how state is rewritten between versions, and when `doctor` asks first.
- [Interaction](reference/interaction.md) — interactive and non-interactive execution, and the
  shape of every question.
- [Pipes](reference/pipes.md) — stash and branch preserved around a command, then restored.
- [Hook files](reference/hook-files.md) — where hooks live, how they are named, and when they run.
- [Exit codes](reference/exit-codes.md) — what a non-zero exit means.
