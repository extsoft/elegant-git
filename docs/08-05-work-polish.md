---
layout: default
title: eg work polish
parent: '"work" guide'
nav_order: 5
permalink: /work/polish/
---

# eg work polish

`work polish` rebases HEAD interactively.

```bash
eg work polish
```

There are no arguments. On a protected branch it refuses, because rewriting that history is not
allowed.

If a rebase is already in progress, it continues that rebase. If there are no unique commits
against the freshest source, it says so and stops. Otherwise the [stash pipe](reference/pipes.md)
wraps an interactive rebase of those unique commits, so uncommitted changes are preserved and
restored.

When you started this rebase from `accept`, the helper branch is `__eg` and bare `eg work`
continues with `accept` instead of `polish`. See [the branch lifecycle](08-00-work.md#the-branch-lifecycle).
