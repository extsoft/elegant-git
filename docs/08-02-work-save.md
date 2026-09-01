---
layout: default
title: eg work save
parent: '"work" guide'
nav_order: 2
permalink: /work/save/
---

# eg work save

`work save` commits current modifications.

```bash
eg work save
```

There are no arguments. It runs `git add --interactive` and then `git commit`, so you pick the
hunks and write the message in the usual Git way.

On a protected branch, Elegant Git does not commit. On a TTY it warns and runs
[`work start`](08-01-work-start.md) first, then saves on the new branch. In non-interactive mode
it refuses with the protected-branch exit code.

This command does not use the stash pipe. Uncommitted changes you do not stage stay in the work
tree.
