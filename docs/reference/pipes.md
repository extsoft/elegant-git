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
because `work accept` deleted it — the stash is left in place instead of being popped onto the
current branch. The branch pipe is used by `work accept` and `release new`, so both of them can
walk away from your branch and bring you back to it. If the saved branch is gone — for example
because `work accept` deleted the local branch you were on — the pipe leaves you on whatever
branch the command checked out.

If a "piped" command is used, each pipe stores the state in per-repo command memory
(`.git/elegant-git/commands.json`), runs the original command, and restores saved state if the
command is successful. If the command is failed and it reruns, the pipes do not preserve the state
again but will restore the initial preserved state if the command is successful.

The file itself is described with the rest of [per-repo memory](memory.md#per-repo-memory).
