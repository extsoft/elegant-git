---
layout: default
title: eg work sync
parent: '"work" guide'
nav_order: 6
permalink: /work/sync/
---

# eg work sync

`work sync` actualizes the branch with upstream commits.

```bash
eg work sync
```

The branch name is optional and is never asked. Without it, Elegant Git fetches if there are
remotes and rebases onto the freshest source of the current branch. With a name, it rebases onto
that branch, fetching first when the name is a remote branch.

The [stash pipe](reference/pipes.md) always wraps the rebase, so uncommitted changes are
preserved and restored. If a rebase is already in progress, it continues that rebase first.

Protected branches are allowed — syncing them with upstream does not commit or rewrite in the
sense that `save` and `polish` refuse.
