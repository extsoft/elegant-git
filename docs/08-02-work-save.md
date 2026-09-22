---
layout: default
title: eg work save
parent: '"work" guide'
nav_order: 2
permalink: /work/save/
---

# eg work save

`work save` commits current modifications, or folds them into a unique commit already on the
branch.

```bash
eg work save
```

There are no required arguments. First it asks how to save when this branch has unique commits
versus its source — the branch recorded when you ran [`work start`](08-01-work-start.md) (`from-ref`,
or the default development branch). That is the same comparison [`work list`](08-04-work-list.md)
and [`work polish`](08-05-work-polish.md) use, including the freshest remote-tracking ref of that
source when one exists. Optional `target` is `new`, `HEAD`, or a unique commit hash (short or full).

The first option is `new` (the default): create a new commit. After that come the unique commits,
newest first. Each one shows a fixed-width relative time like `[ 2 h ago]`, then the hash, then
the subject, so the last commit sits right under `new`. Then it runs `git add --interactive` so
you pick the hunks in the usual Git way.

If there are no unique commits, or you omit `target` in non-interactive mode, it skips the picker
and creates a new commit.

Picking the last commit amends it (`git commit --amend`). That path still runs
`work-amend-ahead` and `work-amend-after` in addition to the `work-save-*` hooks. Picking an
older unique commit creates a fixup for that commit and rebases with autosquash so the change
lands there. Leftover unstaged hunks are autostashed only for that rebase, then restored. If the
rebase fails, the `fixup!` commit is dropped (`rebase --abort` then `reset --soft` to the pre-fixup
HEAD).

In non-interactive mode pass the same choice you would pick, so scripts can amend or fix up
without a prompt:

```bash
eg work save new
eg work save HEAD
eg work save abc1234
```

`target` must be `new`, `HEAD`, or a unique commit (any spelling `git rev-parse` resolves to one).
An unknown value is an error.

On a protected branch, Elegant Git does not commit. On a TTY it warns and runs
[`work start`](08-01-work-start.md) first, then saves on the new branch. In non-interactive mode
it refuses with the protected-branch exit code.

After a successful commit on a feature branch, when the repository has at least one remote,
interactive mode asks `Push?` in the picker. The default is publishing to the branch name
[`work push`](08-07-work-push.md) would use for the current branch (usually the local name). You
can choose that branch, enter a different remote branch name, or skip. A different name must be a
valid git branch name; invalid input is rejected and asked again. Cancelling leaves the commit
local. Non-interactive mode skips the question and does not push.

This command does not use the stash pipe. Uncommitted changes you do not stage stay in the work
tree, except during the autosquash rebase noted above.
