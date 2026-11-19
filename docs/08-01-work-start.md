---
layout: default
title: eg work start
parent: '"work" guide'
nav_order: 1
permalink: /work/start/
---

# eg work start

`work start` creates a new branch. This is the usual way off a protected branch, because Elegant
Git does not commit there.

```bash
eg work start my-change
```

The name is required. `from-ref` is optional and defaults to the default development branch. In
interactive mode a missing name is asked; in non-interactive mode it fails without one.

An unusable name is refused before any fetch, stash, or reset. That includes an invalid branch
name, a name that already exists, and a name that conflicts across a slash with an existing
branch (`feature` and `feature/x`, including when only the letter case differs). A difference
in letter case alone is refused when Git resolves it to the existing branch, which happens on
a case-insensitive filesystem while that ref is loose. Once the ref is packed, Git may allow
the other spelling, and Elegant Git allows it too. Surrounding whitespace on a supplied name
is trimmed. In a terminal, interactive mode shows the problem under the name while you type
and ignores Enter until the name is usable. That includes names Git rejects, such as one that
starts with `-`, a component that starts with `.`, or a component that ends with `.lock`. A
name given on the command line is reported first, then the prompt starts empty. If you submit
a name that the prompt accepted and a last check still rejects it, the prompt opens again with
that name filled in. Where to start is asked only after the name is usable, so a rejected
name does not fetch. Non-interactive mode stops with a
[workflow error](reference/exit-codes.md).

Any [ahead hook](reference/hook-files.md) for this command still runs before the name check.
An after hook still runs when the name is rejected.

If the repository has remotes, it then fetches (and says so if fetch fails). The new branch is
created with `checkout -b` from the upstream of that start point when one exists, otherwise from
the start point itself, and is not set to track anything. The start point is recorded as the
branch source, so later `polish`, `sync`, and `list` can tell your unique commits from the ones
you inherited.

## Uncommitted changes

A clean tree does not ask, and it does not need a flag. When something is uncommitted,
`carry`, `stash`, `discard`, and `cancel` are the only names, in the picker and in
`--dirty`. The picker starts on `carry`. Passing `--dirty` skips that picker and nothing
else: a missing branch name is still asked. `cancel`, end of input, and leaving the picker
stop before any stash or reset. Tracked edits and non-ignored untracked files are what
`carry` and `stash` keep, including the staged and unstaged split. Ignored files stay
where they are. If checkout would overwrite one, the command stops and leaves it, even
when the rest of the tree is clean.

`carry` moves that work onto the new branch, including when HEAD is detached, and then removes
the temporary stash. It stops
before any stash when an untracked or ignored path would be overwritten by a file or
directory on that branch. `stash` still saves a non-ignored untracked path that the
branch also has, and checks out a clean branch. An ignored collision still stops it.
The command suggests `WIP on <branch> — <local date and time>`
(or `WIP at <short commit>` when HEAD is detached). Enter accepts the suggestion.
Typing adds to the end of it; erase the suggestion first to replace it,
and cancelling stops before the stash exists. `--stash-name` sets that suggestion. Without
a terminal, `--dirty=stash` uses the suggestion when `--stash-name` is omitted. The label
can repeat; the stash id does not. The command prints the label, the stash selector, the
object id, and `git stash apply --index <oid>`. It does not apply the stash on the new
branch, and coming back later does not apply it either. A finished stash does not block
later commands. If the handoff does not finish, the recovery record stays and the command
does not claim the stash was handed off.

`discard` restores modifications and deletions of committed files. It shows the
staged, unstaged, and untracked paths first, with names quoted so spaces and control
characters stay unambiguous. New-path files — ordinary untracked files, staged
additions, and rename targets — stay in place unless you pass `--untracked=delete`
with `--dirty=discard`. Without a terminal, omitting `--untracked` keeps them. In a
terminal, omitting it asks keep or delete only when those files exist. Delete removes
that listed set permanently and keeps no backup. A path that would be overwritten is
still refused before anything is changed, except a listed new-path file that delete is
removing. A path that escapes the repository, git metadata, a submodule, a nested
repository, a symlink ancestor, a directory, or a special file refuses the whole
delete before anything is changed.
`--untracked` with any other `--dirty` value is a usage error.

In non-interactive mode a dirty tree fails until you pass `--dirty`. That failure is
immediate: there is no window where a dirty `work start` still stashes on its own. This
is the one place the usual compatibility wait was waived. Command names, hooks, and
recovery records already on disk are unchanged. `preserve` and `fail` are not accepted.

A repository with no commits yet cannot carry or stash uncommitted changes. A merge,
cherry-pick, revert, rebase, or bisect stops the command even when the tree is clean, as
does an unmerged index. If checkout fails, the changes go back to the branch you started
on when that is still safe. Otherwise the command names the backup and leaves it. If the
destination changed the same tracked lines, carry does not predict that conflict. Apply
can stop after part of the backup is already on the branch. The error names that stash
commit and says the destination may already contain part of it. An ahead
hook may already have run; cancelling does not undo that hook.
