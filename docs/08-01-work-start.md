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

An unusable name is refused before any fetch, stash, or reset. That includes an invalid branch
name, a name that already exists, and a name that conflicts across a slash with an existing
branch (`feature` and `feature/x`, including when only the letter case differs). A difference
in letter case alone is refused when Git resolves it to the existing branch, which happens on
a case-insensitive filesystem while that ref is loose. Once the ref is packed, Git may allow
the other spelling, and Elegant Git allows it too. Surrounding whitespace on a supplied name
is trimmed. Interactive mode prints the problem and asks for a name again, including when the
name was given on the command line. Where to start is asked only after the name is usable, so a
rejected name does not fetch. Non-interactive mode stops with a
[workflow error](reference/exit-codes.md).

Any [ahead hook](reference/hook-files.md) for this command still runs before the name check.
An after hook still runs when the name is rejected.

If the repository has remotes, it then fetches (and says so if fetch fails). The new branch is
created with `checkout -b` from the upstream of that start point when one exists, otherwise from
the start point itself, and is not set to track anything. The start point is recorded as the
branch source, so later `polish`, `sync`, and `list` can tell your unique commits from the ones
you inherited.

## Carrying uncommitted changes over

When the work tree is dirty, interactive mode asks whether to add the changes to the new branch,
reset them, or cancel. `add` is the default: the [stash pipe](reference/pipes.md) carries the
modifications over. `reset` hard-resets first. `cancel` stops the command.

In non-interactive mode dirty changes are stashed and moved onto the new branch.
