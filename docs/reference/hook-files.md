---
layout: default
title: Hook files
parent: Reference
nav_order: 6
permalink: /reference/hook-files/
---

# Hook files

While developing something, it may be required to format code prior to committing modifications or
to open several URLs to report release notes after a new release. All these and similar actions,
which you're performing in addition to Git actions, are the **_hooks_**. And Elegant Git allows
automating them — it's like
[Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks), but for Elegant Git
commands. Creating and editing the files is the [`hook`](../10-00-hook.md) object.

## Where hooks live

There are personal and common hooks. The common ones live in `.config/elegant-git/hooks/` inside
the repository, so they are tracked by Git and shared with every contributor. The personal ones
live in `.git/.config/elegant-git/hooks/`, which Git never commits, so they stay yours. That's why
there is an ability to split what should be configured for any contributor and what's for you
personally.

A hook is a single executable file, and its name says when it runs:

```text
<command>-<action>-<type>
```

`<command>` and `<action>` are the object and the action of an Elegant Git command — the same pair
you see in the [CLI anatomy](../04-cli-anatomy.md) — and `<type>` is either `ahead` (runs
prior to the command) or `after` (runs after the command). So a script that formats your code
before every `eg work save` is `work-save-ahead`. Amending through `work save` (picker, `HEAD`, or
the last unique commit hash) also runs `work-amend-ahead` and `work-amend-after` around
`git commit --amend`.

A sample hook execution:

```bash
==>> eg work save
.git/.config/elegant-git/hooks/work-save-ahead
.config/elegant-git/hooks/work-save-ahead
# the command itself
.git/.config/elegant-git/hooks/work-save-after
.config/elegant-git/hooks/work-save-after
```

Every hook that exists runs, personal before common, and Elegant Git prints the path of each one
before executing it. The script itself is executed with `bash` from the repository root, so a
shebang line is decorative and relative paths resolve against the root rather than your current
directory.

❗Please take into account that a non-zero exit code from a hook does **not** stop anything. Elegant
Git reports the hook it is running and moves on regardless of how the script ended, so a hook that
must block the command has to be written to make the command itself fail.

There is one hard limit worth knowing: a hook may call Elegant Git again, but the nesting is
capped at 16 levels. Beyond that the run aborts with a message pointing at recursive hooks, which
is what saves you from a `work-save-ahead` hook that calls `eg work save`.

If you want to skip hooks for the current command execution, just use the `--no-workflows` option
like `eg --no-workflows work save`.

## Coming from the old layout

Earlier versions kept common hooks in `.workflows/` and personal ones in `.git/.workflows/`, with
names like `save-work-ahead`. Those files still run until they are moved: Elegant Git prefers the
new path and falls back to the old one only when the new file is absent, and it warns when it
does. Both old locations are deprecated and will stop working.

[`eg repo doctor`](../07-07-repo-doctor.md) offers to move them after a confirmation. Tracked
hooks need a commit afterwards, so doctor prints the `git add` / `git commit` to use. One old name
is kept on purpose: `repo clone` and `repo init` still run `acquire-repository-{ahead,after}` as
well as their own hooks, so a provisioning script from the previous layout keeps working until it
is renamed.
