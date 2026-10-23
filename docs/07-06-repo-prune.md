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

Every candidate starts selected, except stashes.

- `local` — deletes local branches merged into the default development branch or a protected
  branch, or whose upstream is gone.
- `remote` — deletes remote branches merged into the default development branch or a protected
  branch, on the default upstream remote (`origin`) itself.
- `stashes` — drops only the stashes you pick; none start selected.
- `worktrees` — prunes worktrees whose directory no longer exists.

Protected branches and the default development branch are never removed.

## How a run goes

The command checks out the default development branch first, fetches every remote with `--prune`
if an upstream exists (and says so if fetch fails, then continues with the local version), and
rebases onto that upstream when it can. The fetch deletes remote-tracking branches whose remote
branch is gone, so `local` sees those upstreams as gone.

Phases run in the order of the table, whichever ones you name. Leave them off and interactive
mode asks `What to clean up`, with every phase selected. Naming phases on the command line skips
that question. Non-interactive mode with no phases runs `local` only. `stashes` needs you to pick,
so naming it in non-interactive mode fails before anything is checked out or fetched.

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
dropped.` for stashes, `Nothing pruned.` for worktrees) and moves to the next phase. After a phase
applies, it reports what it did, such as `Deleted 2 branches on origin.` A phase with no
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
- `stashes` lists every stash, with the subject after it, and drops the ones you select. Drops
  run from the highest `stash@{n}` down so the remaining names stay valid.
- `worktrees` removes worktrees whose directory is gone and that are not locked. Unselected stale
  worktrees are locked for the duration of the prune, then unlocked, so only the ones you picked
  are removed.
