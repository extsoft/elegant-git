---
layout: default
title: eg work start
parent: '"work" guide'
nav_order: 1
permalink: /work/start/
---

# eg work start

`work start` creates a new branch. This is the usual way off a protected branch, because Elegant
Git does not commit there.

```bash
eg work start my-change
```

The name is required. `from-ref` is optional and defaults to the default development branch. In
interactive mode a missing name is asked; in non-interactive mode it fails without one.

If the repository has remotes, it fetches first (and says so if fetch fails). The new branch is
created with `checkout -b` from the upstream of that start point when one exists, otherwise from
the start point itself, and is not set to track anything. The start point is recorded as the
branch source, so later `polish`, `sync`, and `list` can tell your unique commits from the ones
you inherited.

## Carrying uncommitted changes over

When the work tree is dirty, interactive mode asks whether to add the changes to the new branch,
reset them, or cancel. `add` is the default: the [stash pipe](reference/pipes.md) carries the
modifications over. `reset` hard-resets first. `cancel` stops the command.

In non-interactive mode dirty changes are stashed and moved onto the new branch.
