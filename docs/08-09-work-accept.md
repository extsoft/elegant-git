---
layout: default
title: eg work accept
parent: '"work" guide'
nav_order: 9
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

The [stash pipe](reference/pipes.md) wraps the whole run, so uncommitted changes are preserved and
restored when the branch they came from still exists. If you accepted that branch, it is deleted
and the stash is left in place instead of being popped onto the default development branch. The
[branch pipe](reference/pipes.md) restores the branch you were on when that local ref still exists.
If you accepted the branch you started on, that local branch is deleted and you stay on the default
development branch.

If a rebase is already in progress on the helper branch `__eg`, the command continues that rebase.
Any other active rebase is an error; finish it first.

Otherwise it fetches, checks the accepted branch out as `__eg` (tracking a remote of that name
when there is no local branch), rebases `__eg` onto the freshest source, checks out the default
development branch, fast-forward merges `__eg`, and deletes the helper. If there are remotes it
pushes the default branch to `origin`, and if the accepted branch's upstream was on `origin` it
deletes that remote branch. The recorded source for the accepted branch is cleared. When the
accepted name was a local branch that is not protected, that local branch is deleted last, so a
failed push still leaves it available to retry.

This is the intended path onto a protected default branch, so there is no protected-branch
refusal.
