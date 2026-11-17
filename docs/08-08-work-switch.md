---
layout: default
title: eg work switch
parent: '"work" guide'
nav_order: 8
permalink: /work/switch/
---

# eg work switch

`work switch` checks out a local branch, creates a local branch that tracks a remote
branch, or creates a local branch at a tag or other revision. HEAD stays attached.

```bash
eg work switch feat
```

The ref is optional in a terminal. Leaving it out offers a picker of local branches, remote
branches, and tags, then `other revision`, which asks you to type any revision.
Non-interactive mode requires the ref. A second argument sets the local branch name when the
ref is a remote branch, a tag, or another revision. Passing that argument for a local branch
is a usage error.

A ref that starts with `-` is a usage error, and it is never passed to Git as an option. The
command also stops while a merge, cherry-pick, revert, rebase, or bisect is in progress, and
the error names that operation and how to finish or abort it. A bare repository is refused
because it has no working tree.

An exact local branch is checked out without fetching. Any other ref fetches first when the
repository has remotes; a failed fetch is reported and the last known refs are used. Resolution
tries an exact local branch, then an exact remote-tracking branch, then a bare name that matches
one remote branch, then an exact tag, then any other revision. If nothing matches exactly, a
unique substring of a local or remote branch name is used. Several matches, or none, is a
[workflow error](reference/exit-codes.md).

Qualified forms select the kind: `heads/x` and `refs/heads/x` are local, `remotes/origin/x`
and `refs/remotes/origin/x` are remote, and `tags/x` and `refs/tags/x` are a tag. When one
short name is both a local branch and a tag, the local branch is used and a notice names
`tags/x`.

Uncommitted changes are carried onto the branch you switch to when you started on a branch,
and only while that branch is still checked out in this worktree. Starting from a detached
HEAD leaves the stash in place. In a terminal you can keep them,
discard them, or cancel. Non-interactive mode keeps them. If the checkout fails, those changes
are put back. A repository with no commits yet cannot carry uncommitted changes, because
stashing needs a commit. Switching to the branch you are already on prints a notice and changes
nothing.

A free local name for a remote branch is created with `checkout -b` so it tracks that remote.
When that local branch already tracks the remote, a terminal asks whether to check it out,
fast-forward it, reset it, or pick another name. The fast-forward choice is the default when
the local branch can move forward. Non-interactive mode fast-forwards when it can, and otherwise
checks the branch out and leaves its tip where it is.

## Tags and other revisions

A tag or any other revision, such as a commit or `HEAD~1`, becomes a new local branch at that
commit. An annotated tag uses the commit it points at, not the tag object. The branch is
created with `--no-track`. HEAD is never detached.

A terminal asks for the local branch name, the same way it does for a remote branch. The
suggested name is `tag-<tag>` for a tag and `rev-<short-sha>` for any other revision, or the
local branch name you passed. Non-interactive mode uses that name without asking. A name that
already exists, clashes with another branch, or is otherwise unusable is refused. In a terminal
you can type another name. Non-interactive mode stops with a workflow error.

The new branch records the tag or revision you gave as its source branch. [eg work
sync](08-06-work-sync.md) then rebases onto that ref.

A range, a path, a tree, or a blob is not a revision, so it is rejected and nothing is checked
out.

Hook exit codes are ignored. An after hook for `work switch` still runs when the switch fails.
`work track` hooks do not run for this command.
