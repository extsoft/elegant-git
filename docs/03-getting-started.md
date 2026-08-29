---
layout: default
title: Getting started
nav_order: 3
permalink: /getting-started/
---

# Getting started

Once `git-elegant` is [installed](02-00-installation.md), a few actions set up your machine and each
repository you work in.

## Post-installation actions

Configure your Git installation once per machine by running
[`git elegant git configure`](reference/commands.md#git) and follow the instructions. To find out
more, please read [the configuration approach](reference/configuration.md).

You can access Elegant Git in CLI using either of

```bash
git elegant <command>
git-elegant <command>
```

where `<command>` is one of the commands described on the
[commands](reference/commands.md) page or printed in a terminal after running `git elegant`.

`git configure` also installs a Git alias for each of the old flat command names, so `git save-work`
still reaches `git elegant work save`. Those names are deprecated — please use the object-first form
in anything you write down.

Also, please use [`git elegant repo clone`](reference/commands.md#repo) or
[`git elegant repo init`](reference/commands.md#repo) instead of regular `clone` or `init` when
starting work with a repository — both of them configure the fresh repository for you. For a
repository you already have, run [`git elegant repo configure`](reference/commands.md#repo) inside
it.

P.S. Shell completion is generated on demand for `bash`, `zsh`, `fish`, and `powershell`. For
example:

```bash
git elegant completion bash > ~/.local/share/bash-completion/completions/git-elegant
```
