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

## Already delivered work

Commits whose changes are already in the target are left out of the rebase. A branch stacked on
work that was squash-merged, polished, or partly cherry-picked keeps only its own commits. The
check uses the commits on the branch now, not a saved history.

A source branch counts as delivered when its upstream is gone, or when its changes are already in
the default development branch. The rebase then uses the default development branch. A local
source branch is used instead of its remote-tracking branch when the two differ and the local
branch already contains that remote.

The old commits are still replayed, and that can conflict, when the source was amended to
different content, a later squash changed the same lines, the squash was edited on those lines, or
the default development branch changed those lines after the squash.

## Files in the way

A replayed commit, or the checkout onto the target, can need to create a path that is already on
disk as an untracked or ignored file. Git will not overwrite that file. Before the rebase, Elegant
Git moves each such file to `<repo>/.git/elegant-git/aside/`, keeping its relative path. It moves
the file back when that command is done rebasing and checking out, replacing whatever is at that
path. A path the rebase left tracked is written back after the checkout, so the checkout still
runs. If the rebase stops, the files stay aside until it finishes through `work sync`, `work push`,
`work accept`, or `work polish`. A rebase you finish or abort with plain Git is cleaned up the
next time one of those commands rebases.

Protected branches are allowed. Syncing them does not commit or rewrite in the sense that
`save` and `polish` refuse.
