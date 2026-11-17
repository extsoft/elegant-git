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

The stash pipe is used by `work sync`, `work polish`, `work push`, `work accept`, and
`release new`. It also wraps `work start` when you ask that command to carry your uncommitted
changes over to the new branch. If the branch the stash was taken from is gone — for example
because `work accept` deleted it — the changes are not restored onto the current branch. The
stash entry stays in the list. The branch pipe is used by `work accept` and `release new`, so both of them can
walk away from your branch and bring you back to it. If the saved branch is gone — for example
because `work accept` deleted the local branch you were on — the pipe leaves you on whatever
branch the command checked out.

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

A stash pipe keeps a backup until restore finishes. The backup is a plain tracked stash (the same
scope these commands use today), a private ref `refs/elegant-git/recovery/<id>`, and a journal at
`<git-common-dir>/elegant-git/recovery.json`. The id in the stash subject is the identity.
`stash@{n}` is not.

Preparation needs Git 2.13 or newer on Linux, macOS, or Windows. Older Git, or another operating
system, refuses before it stashes or writes the journal. A command-state file set with
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
Merge, cherry-pick, revert, and bisect are refused only when there are
changes to stash. A clean tree still runs the command. `applied` only finishes cleanup and does
not run the command again. Cleanup removes the private ref and the journal after the
changes are restored. It leaves the stash entry. Git cannot remove that one entry
without possibly removing a different stash, so the command names the backup and
does not apply it again. If creating the stash fails, or Git saves nothing, the preparing record is removed and the
command runs without a backup.

If the source branch is gone, or a detached HEAD has moved or is no longer detached, the
stash stays in place. The command
tells you to inspect it. Another command that has its own changes will not start while that
backup is pending. A clean command still runs. A command started from inside a running Elegant
Git process is not blocked by that guard.

`eg repo doctor` prints the owner, source, phase, backup id, and ref. Confirming the repair
clears the journal only. It does not apply the stash, drop it, or retry the command. After you
confirm, prune may offer that stash. Until then a journaled backup, a private ref, or a stash
message still recorded in command memory is hidden, including an older
`On <branch>: eg <command> auto-stash:` subject. A leftover subject with no record can be pruned.
The protected stash stays the same one when other stashes are dropped.

To restore by hand:

```bash
git stash apply --index <backup-oid>
git update-ref -d refs/elegant-git/recovery/<id> <backup-oid>
```

Drop the stash yourself only after the changes are back or you mean to discard them. An older
binary does not understand a new journal. Finish or inspect that recovery before you switch back
to it. Deleting the journal or the ref is not how you downgrade.

A recorded backup that is no longer in the stash list is reported and left alone. Other stashes
are not dropped.
