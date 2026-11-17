---
layout: default
title: eg repo init
parent: '"repo" guide'
nav_order: 3
permalink: /repo/init/
---

# eg repo init

`repo init` creates a git repository in the current directory, configures it, and records an
empty initial commit. Files already in the directory stay as they are and are not staged.

```bash
eg repo init work
```

The workspace is required, with the same resolver as [`repo configure`](07-01-repo-configure.md)
— an existing name or `[Create new]`. It records the default and protected branches in
[per-repo memory](reference/memory.md), the same way configure does.

If this directory is already a git repository, or is inside one, init stops before it changes
anything. Hooks do not run, and you are not asked for a workspace. That includes an unborn
repository, a linked or bare worktree, a `.git` file or directory, and a subdirectory of an
enclosing worktree. When Git cannot be checked, the error is reported and the directory is not
initialized.

The initial commit is an empty tree. If the index already has staged entries, or entries appear
before the commit is recorded, init leaves those entries in the index and does not record the
commit. The commit message is written to a temporary file outside this directory. A file named
`a-message-of-initial-commit` is left untouched.

If configuration or the initial commit stops after `git init`, the new repository and its files
stay in place. The command reports the stages that finished.

[`repo clone`](07-02-repo-clone.md) does not use this refusal. It clones into the destination
and leaves the caller's current directory alone.
