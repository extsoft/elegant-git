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

If a rebase is already in progress, it continues that rebase first. Conflict resolutions are
not stashed or discarded. Otherwise a dirty tree asks you to `carry`, `stash`, `discard`, or
`cancel`, the same choice as [eg work start](08-01-work-start.md). `carry` restores tracked
and non-ignored untracked files on this branch and keeps the staged and unstaged split.
`stash` leaves them in the stash. `discard` restores committed-file edits and keeps new-path files unless `--untracked=delete` is set with `--dirty=discard`, as described for [eg work start](08-01-work-start.md). An ignored file the incoming rebase would write, and for carry or keep an untracked file, is refused before any stash or reset. Delete removes only the listed new-path files and keeps no backup. Without a terminal,
a dirty tree fails until `--dirty` is set. A clean tree does not need it. Non-interactive
mode does not replace Git's rebase editor. The backup stays on this branch in this worktree.
Another branch, including one at the same commit, does not receive it.

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
disk as an untracked file. Git will not overwrite that file. Before the rebase, Elegant Git moves
each such untracked file to `<repo>/.git/elegant-git/aside/`, keeping its relative path. An
ignored file is not moved. The command stops and leaves it in place. Files already in the aside
are still restored when the destination is free. A new run does not delete an aside copy that is
already there. It moves a file back when that command is done rebasing and checking out. If the
destination is occupied, the aside copy stays. If the rebase stops, the files stay aside until it
finishes through `work sync`, `work push`, `work accept`, or `work polish`. A rebase you finish
or abort with plain Git is cleaned up the next time one of those commands rebases.

Protected branches are allowed. Syncing them does not commit or rewrite in the sense that
`save` and `polish` refuse.
