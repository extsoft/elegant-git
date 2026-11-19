---
layout: default
title: eg work accept
parent: '"work" guide'
nav_order: 10
permalink: /work/accept/
---

# eg work accept

`work accept` adds modifications to the default development branch.

```bash
eg work accept my-change
```

The branch is required. In interactive mode it is asked if you leave it out, with completion over
local and remote branches. From bare `eg work` on a feature branch, picking `accept` passes the
branch you are already on.

A dirty tree uses the same `carry`, `stash`, `discard`, and `cancel` choice as
[eg work sync](08-06-work-sync.md), including `--untracked` on discard. `carry` restores the changes on the branch you started on,
when that branch still exists, and keeps the staged and unstaged split. If that branch was
deleted, the backup stays and is not applied to another branch. The stash entry stays in the
list. Without a terminal, a dirty tree fails until `--dirty` is set. The
[branch pipe](reference/pipes.md) restores the branch you were on when that local ref still exists.
If you accepted the branch you started on, that local branch is deleted and you stay on the default
development branch.

If a rebase is already in progress on the helper branch `__eg`, the command continues that rebase
before the dirty choice and, when that rebase finishes, goes on to the merge and push. The name you
pass has to be the branch that rebase started from; a different name is refused before the rebase
is continued. With no name, accept uses the one local branch that still points at that starting
commit. Conflict resolutions are not stashed or discarded. Any other active rebase, merge,
cherry-pick, revert, or bisect is an error before that choice; finish it first. Repeating accept
after you continue or abort outside Elegant Git does not push or delete again. If `__eg` already
has commits that are not on the accepted branch, accept refuses before it stashes or resets.

Otherwise it fetches, checks the accepted branch out as `__eg` (tracking a remote of that name
when there is no local branch), rebases `__eg` onto the freshest source, and leaves out commits
whose changes are already there. Untracked files that replay would overwrite are moved aside until
accept finishes checking out; ignored files are left in place. See [eg work sync](08-06-work-sync.md).
A delivered source, including one whose upstream is gone, is treated as the default development
branch. It then checks out the default development branch, fast-forward merges `__eg`, and deletes
the helper. If there are remotes it pushes the default branch to `origin`. When the accepted branch's upstream
is `origin/<name>`, and that name is the accepted branch, not the default branch, and not
protected, it deletes that remote branch and reports the deletion. A failed fetch or rebase says the
remote was not updated. A failed push says the local merge finished and the backup was kept. The
recorded source for the accepted branch is cleared. When the accepted name was a local branch that
is not protected, that local branch is deleted last, so a failed push still leaves it available to
retry. If Git cannot delete it (for example it is checked out in another worktree), accept still
succeeds and the branch is left in place.

This is the intended path onto a protected default branch, so there is no protected-branch
refusal.
