---
layout: default
title: eg work push
parent: '"work" guide'
nav_order: 7
permalink: /work/push/
---

# eg work push

`work push` publishes HEAD to a remote repository.

```bash
eg work push
```

The remote branch name is optional and is never asked. Without it, the command uses the upstream
branch when one is set, otherwise the local name.

A new push of a protected branch is refused — delivering there is `accept`, or a plain
`git push` if you really mean it. A rebase already in progress is continued first, including
on a protected branch. Conflict resolutions are not stashed or discarded.

Otherwise it fetches and rebases onto the freshest source. Commits whose changes are already
in that source are left out. A delivered source, including one whose upstream is gone, is
treated as the default development branch. Then it force-pushes with `--set-upstream` to the
remote (from the upstream remote, or `origin`). HTTP URLs in the push output are opened when
`open` is available. A dirty tree uses the same choice as [eg work sync](08-06-work-sync.md).
`carry` restores the changes on this branch. A failed fetch or rebase says the remote was not
updated. A failed push says the local rebase finished. The backup stays. Untracked files a
replayed commit would overwrite are moved aside until the rebase finishes, as described in
[eg work sync](08-06-work-sync.md). Ignored files are left in place. The cases that can still
conflict are listed there too.
