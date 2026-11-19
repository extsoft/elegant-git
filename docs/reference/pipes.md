---
layout: default
title: Pipes
parent: Reference
nav_order: 5
permalink: /reference/pipes/
---

# Pipes

There are a lot of situations when a current Git state needs to be reserved prior to Elegant Git
commands execution. For instance, you are working on something. And now, urgently, you need to
accept someone's critical work — `eg work accept some-critical-branch`. But there are
uncommitted modifications that need to be stashed prior to accepting work. From the other side, it
will be good to back into the previous working branch and restore modification from the stash when
the needed work is accepted. That's why there are **pipes** which are doing preserve and restore
automatically.

There are the following pipes:

- **branch pipe** which preserves and restores current branch when that local ref still exists
- **stash pipe** which preserves and restores uncommitted changes

`work sync`, `work polish`, `work push`, `work accept`, `release new`, and `repo prune` ask
`carry`, `stash`, `discard`, or `cancel` before they change a dirty tree, the same four
words as `work start` and `work switch`. A non-interactive run fails until `--dirty` is set.
A clean tree does not ask. For sync, polish, push, accept, and release new, an in-progress
rebase is continued first, and a merge, cherry-pick, revert, or bisect is refused, before
that question. `repo prune` refuses any in-progress Git operation before the question.
Conflict resolutions are not stashed or reset. `carry` and `stash` keep tracked and non-ignored untracked files and
the staged and unstaged split. `discard` does not create a stash. With `--untracked=delete` it removes only the
listed new-path files and keeps no backup; otherwise those files stay. `carry` restores that backup on the branch the command is
allowed to return to: the branch you stayed on for sync, polish, and push, the branch
you started on for accept and release new, or the default development branch for
`repo prune`. When that restore finishes, the temporary stash is removed. If prune deletes
the branch the changes came from, the backup stays and is not applied to that default branch.
`stash` checks that the stash is still there,
then drops the recovery record and the private ref so the stash is an ordinary one you can
restore yourself with `git stash apply --index <oid>`. It is not applied afterward. If that
handoff fails, the record stays. If the branch the stash was taken from is gone — for
example because `work accept` deleted it — the changes are not restored onto the current
branch, and that stash entry stays in the list. A failed command keeps the backup and does not
run the push, tag, or delete again when you repeat it. The branch pipe is used by
`work accept` and `release new`, so both of them can walk away from your branch and bring
you back to it. If the saved branch is gone — for example because `work accept` deleted the
local branch you were on — the pipe leaves you on whatever branch the command checked out.

If a "piped" command is used, each pipe stores the state in per-repo command memory
(`.git/elegant-git/commands.json`), runs the original command, and restores saved state if the
command is successful. A failed command keeps the backup. Re-running it does not repeat the
command when recovery is already in progress; inspect that backup as described below. An older
record that only has the stash message does not run the command again. If that stash
is not in the list, the command says it was not found and other stashes were not
changed. Inspect it with `eg repo doctor`. Confirming it clears the record and does
not apply or drop the stash.

The file itself is described with the rest of [per-repo memory](memory.md#per-repo-memory).

## Recovery

A stash pipe keeps a backup until restore finishes. The backup is a stash of tracked and
non-ignored untracked files, a private ref `refs/elegant-git/recovery/<id>`, and a journal at
`<git-common-dir>/elegant-git/recovery.json`. The id in the stash subject is the identity.
`stash@{n}` is not.

Preparation needs Git 2.40.0 or newer on Linux, macOS, or Windows, the same floor as every other
Git-backed command. Older Git, a missing Git, or another operating system refuses before it
stashes or writes the journal. A backup that is already pending is left in place. Upgrade Git,
then run the command again. Elegant Git does not delete the stash, the private ref, or the
journal to get past a version refusal. A command-state file set with
`ELEGANT_GIT_CMD_STATE_FILE` must stay inside that worktree's `<git-dir>/elegant-git` directory.
A path outside it, a symlink that leaves it, or another worktree's file is refused.

The journal phases are `preparing`, `preserved`, `running`, `suspended`, `applying`, and
`applied`. A crash while `applying` is never applied a second time. Re-running the command does
not repeat the body when the outcome is unknown (`running`). A `suspended` backup, where the
command already returned an error, does not run that command again just to restore the stash.
The backup stays until you inspect it. A later rebase continues only when its
branch, onto commit, and original HEAD are the ones recorded for that backup.
Any other rebase is left alone, and the backup is not applied or dropped.
Finishing or aborting the rebase outside Elegant Git leaves the backup
recorded and does not run the command again.
Merge, cherry-pick, revert, and bisect are refused before any backup, including on a
clean tree. `applied` only finishes cleanup and does
not run the command again. After a confirmed restore, cleanup drops that temporary stash, then
the private ref, then the matching command-memory note and the journal entry. The drop uses a
fresh `stash@{n}` for that commit and the full stash message, so a different stash stays. An
entry that is already gone is cleaned up without applying again. A changed message, or two
entries with the same identity, stops cleanup. The error says the changes were restored and names
the commit; the stash, private ref, and journal stay for a later retry. That retry does not run
the command or apply the stash again. Git drops by selector only. A change made after the
selector is chosen and before the drop can remove a different entry. That window is not a
guarantee. If creating the stash fails, or Git saves nothing, the preparing record is removed and the
command runs without a backup.

If the source branch is gone, or a detached HEAD has moved or is no longer detached, the
stash stays in place. The command
tells you to inspect it. Restore also requires the worktree that saved the backup and HEAD
on the branch that command is allowed to restore to: the branch the stash was taken from for
`sync`, `polish`, `push`, `accept`, and `release new`, the branch `start` or `switch`
checked out, or the default development branch for `repo prune`. A phase error while HEAD is
still on that default branch, with no rebase in progress, restores a carried backup and then
reports the error. A stash choice stays in the stash. If prune deleted the branch the changes came from, or a rebase is still in progress,
that backup stays and prune does not run again until `eg repo doctor`
clears it. Another branch at the same commit is not that destination. The check is read
again immediately before `git stash apply` and again after it returns. Another process can
still change the branch while `git stash apply` is running. Git does not give Elegant Git a
way to prevent that. If the branch no longer matches when apply returns, the command does
not report success. It keeps the stash entry, the journal, and the private ref, and it does
not apply that backup again.
Another command that has its own changes will not start while that
backup is pending. A clean command still runs, except `repo prune` while its own backup
is suspended. A command started from inside a running Elegant
Git process is not blocked by that guard.

`eg repo doctor` prints the owner, source, phase, backup id, and ref. Confirming the repair
clears the journal only. It does not apply the stash, drop it, or retry the command. After you
confirm, prune offers that stash with the ordinary ones and can drop it. Until then a journaled backup, a private ref, or a stash
message still recorded in command memory is hidden, including an older
`On <branch>: eg <command> auto-stash:` subject. A leftover subject with no record can be pruned.
The protected stash stays the same one when other stashes are dropped.

To restore by hand:

```bash
git stash apply --index <backup-oid>
git update-ref -d refs/elegant-git/recovery/<id> <backup-oid>
```

After you restore by hand, or when you mean to discard the changes, drop the stash yourself.
Do not drop it while the changes are not back. An older binary does not understand a new journal.
Finish or inspect that recovery before you switch back to it. Deleting the journal or the ref is
not how you downgrade.

If the temporary stash is already gone, retrying the command removes the private ref and the
journal and does not apply a stash. Other stashes are not dropped.
