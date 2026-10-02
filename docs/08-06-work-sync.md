---
layout: default
title: eg work sync
parent: '"work" guide'
nav_order: 6
permalink: /work/sync/
---

# eg work sync

`work sync` rebases onto the branch this one was created from, or onto a branch you name.

```bash
eg work sync [branch-name]
```

The branch name is optional. In interactive mode, Elegant Git fetches once when there are remotes,
then asks which branch to rebase onto. The picker lists the source branch first (the default
selection), the default development branch, and this branch's upstream when those refs exist,
then every other local and remote branch. The current branch is left out of that list; its
upstream can still appear as a pinned row. It does not fetch again after you choose. With a name
on the command line, it rebases onto that branch, fetching once first when the name is a remote
branch. In non-interactive mode, an omitted name rebases onto the source branch without asking.
Shell completion uses the same list.

The [stash pipe](reference/pipes.md) always wraps the rebase, so uncommitted changes are
preserved and restored. If a rebase is already in progress, it continues that rebase first.

Protected branches are allowed. Syncing them does not commit or rewrite in the sense that
`save` and `polish` refuse.
