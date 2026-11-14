---
layout: default
title: eg work switch
parent: '"work" guide'
nav_order: 8
permalink: /work/switch/
---

# eg work switch

`work switch` checks out a local branch, or creates a local branch that tracks a remote
branch. A tag or any other revision is refused: that input is not a branch.

```bash
eg work switch feat
```

The ref is optional in a terminal. Leaving it out offers a picker of local branches, remote
branches, and tags. Non-interactive mode requires it. A second argument sets the local branch
name when the ref is a remote branch. Passing that argument for a local branch is a usage error.

A ref that starts with `-` is a usage error, and it is never passed to Git as an option. The
command also stops while a merge, cherry-pick, revert, rebase, or bisect is in progress, and
the error names that operation and how to finish or abort it. A bare repository is refused
because it has no working tree.

An exact local branch is checked out without fetching. Any other ref fetches first when the
repository has remotes; a failed fetch is reported and the last known refs are used. Resolution
tries an exact local branch, then an exact remote-tracking branch, then a bare name that matches
one remote branch. If nothing matches exactly, a unique substring of a local or remote branch
name is used. Several matches, or none, is a [workflow error](reference/exit-codes.md).

Qualified forms select the kind: `heads/x` and `refs/heads/x` are local, and `remotes/origin/x`
and `refs/remotes/origin/x` are remote. `tags/x` and `refs/tags/x` name a tag, which this
command does not check out. When one short name is both a local branch and a tag, the local
branch is used and a notice names `tags/x`.

Uncommitted changes are carried onto the branch you switch to. In a terminal you can keep them,
discard them, or cancel. Non-interactive mode keeps them. If the checkout fails, those changes
are put back. A repository with no commits yet cannot carry uncommitted changes, because
stashing needs a commit. Switching to the branch you are already on prints a notice and changes
nothing.

A free local name for a remote branch is created with `checkout -b` so it tracks that remote.
When that local branch already tracks the remote, a terminal asks whether to check it out,
fast-forward it, reset it, or pick another name. The fast-forward choice is the default when
the local branch can move forward. Non-interactive mode fast-forwards when it can, and otherwise
checks the branch out and leaves its tip where it is.

Hook exit codes are ignored. An after hook for `work switch` still runs when the switch fails.
`work track` hooks do not run for this command.
