---
layout: default
title: eg repo prune
parent: '"repo" guide'
nav_order: 6
permalink: /repo/prune/
---

# eg repo prune

`repo prune` deletes branches, stashes, and worktrees you no longer need, in phases you choose.

```bash
eg repo prune [phase...]
```

## What each phase removes

Every candidate starts selected. `stashes` has no selection.

- `local` — deletes local branches merged into the default development branch or a protected
  branch, or whose upstream is gone.
- `remote` — deletes remote branches merged into the default development branch or a protected
  branch, on the default upstream remote (`origin`) itself.
- `stashes` — lists ordinary stashes and does not drop them. Elegant Git cannot drop one stash
  safely by commit. The phase prints `git stash drop stash@{n}` for you to run yourself.
- `worktrees` — prunes worktrees whose directory no longer exists.

Protected branches and the default development branch are never removed.

## How a run goes

Interactive mode asks `What to clean up`, with every phase selected, before it checks anything
out. Naming phases on the command line skips that question. Accepting none stops before checkout.
Cancelling that list fails the command and leaves the working tree where it was. Non-interactive
mode with no phases runs `local` only. `stashes` does not ask, so naming it without a
terminal lists the ordinary stashes and the `git stash drop` command.

When the tree has tracked edits or non-ignored untracked files, the next question is the same
`carry`, `stash`, `discard`, and `cancel` choice as
[work start](08-01-work-start.md). `carry` restores those changes on the default development
branch, with the staged and unstaged split kept. If this command deletes the branch you were on,
that backup stays in the stash and is not applied to the default branch. `stash` leaves an
ordinary stash and does not apply it. `discard` restores committed-file edits and keeps
new-path files unless `--untracked=delete` is set with `--dirty=discard`. Delete removes
only that listed set and keeps no backup. An ignored file the checkout or rebase would
write is refused first, and so is an untracked file for `carry` or for discard unless
delete is removing that listed new-path file. The stash name is asked before any fetch. A phase that fails after checkout restores
a carried backup when HEAD is on the default branch and no rebase is in progress, then
reports that phase error. A rebase still in progress, a deleted source branch, or a
destination that is not the default branch leaves the backup in place. With `stash`, a
failed phase also leaves that backup suspended.
`cancel`, or cancelling the question, stops before any stash or reset. Without a
terminal, a dirty tree fails until `--dirty` is set. A clean tree does not need it. The phase
list is asked first. An in-progress Git operation is refused after that list and before the
dirty question. If the branch you were on was deleted, the backup stays suspended. A later
`repo prune`, even with a clean tree, does not run again until `eg repo doctor` clears it,
and it does not repeat those deletions.

The command then fetches every remote with `--prune` when the default branch has an upstream (and
says so if fetch fails, then continues with the local version), checks out that default branch,
and rebases onto the upstream when the fetch worked. The fetch deletes remote-tracking branches
whose remote branch is gone, so `local` sees those upstreams as gone. Candidates are read after
that, so a stash created for this command does not change which older stash is which.

Phases run in the order of the table, whichever ones you named.

Each phase prints what it is about to remove, then scans. On a terminal, a progress line shows the
item being checked and stays up for at least 1.5 seconds before it is erased. The examples use
`main` as the default development branch:

```text
==>> Pruning origin branches merged into main or a protected branch (deletes them on origin)...
Scanning remote branches 179/902 : origin/feat
```

In interactive mode, the phase then opens a multi-select picker named for the action, such as
`Branches to delete on origin`. Esc skips the phase. After you accept a selection, a confirmation
asks before anything changes, with `no` as the default:

```text
Delete 2 branches on origin? [yes/no] (press enter to 'no'):
```

Answering `no`, accepting an empty selection, or pressing Esc prints `Nothing deleted.` (`Nothing
pruned.` for worktrees) and moves to the next phase. `stashes` does not ask. That skip
does not undo the checkout, and it does not mean the carried changes failed to come back. After a
phase applies, it reports what it did, such as `Deleted 2 branches on origin.` A phase with no
candidates says why, such as `No origin branches are merged into main or a protected branch.`,
or `No remotes; skipping origin branches.` The messages name the actual branch and remote.

Non-interactive mode neither picks nor confirms: it removes every candidate in the phases you
named and reports the count.

## How candidates are listed and chosen

Branches and stashes are listed from oldest to newest. Each row starts with the committer date in
fixed-width columns, relative first, then the name; names longer than 40 characters are trimmed
with `...` in the list, but the full name is what gets removed:

```text
[ 6 y ago]  2019-10-10  old-feature  merged into main
[ 2 y ago]  2024-03-04  only-master  merged into master
[21 d ago]  2025-09-18  fix/1132-staging-cron-jobs-agreement-...  upstream origin/fix/1132 is gone
```

A branch counts as merged when its tip is already in the default development branch, or in a
protected branch. The default development branch is the local one, just updated. Each other
protected branch is its remote-tracking branch on the default upstream remote when that exists,
otherwise the local branch, and it is skipped when neither exists. The row names the first branch
it is merged into. A local branch that still has an upstream configured is removed only when that
upstream is gone, and the row says so.

- `local` never removes the default development branch or a protected branch. Deleting a branch
  clears its branch source from repo memory.
- `remote` never removes the remote's `HEAD`, the default development branch, or a protected
  branch. The branches you select are deleted on the default upstream remote with one push. This
  phase does nothing when the repository has no remotes.
- `stashes` lists ordinary stashes, oldest first, with the subject after the date and
  `stash@{n}`. Recovery backups are omitted. It then prints: Elegant Git cannot drop one
  stash safely by commit. Drop one yourself with `git stash drop stash@{n}`. The phase
  succeeds either way, including when the list is empty.
- `worktrees` removes worktrees whose directory is gone and that are not locked. Unselected stale
  worktrees are locked for the duration of the prune, then unlocked, so only the ones you picked
  are removed.
