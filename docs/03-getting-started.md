---
layout: default
title: Getting started
nav_order: 3
permalink: /getting-started/
---

# Getting started

Once Elegant Git is [installed](02-00-installation.md), a few actions set up your machine and each
repository you work in.

## Post-installation actions

Configure your Git installation once per machine by running
[`eg git configure`](reference/commands.md#git) and follow the instructions. To find out
more, please read [the configuration approach](reference/configuration.md).

The primary invocation is

```bash
eg <command>
```

where `<command>` is one of the commands described on the
[commands](reference/commands.md) page or printed in a terminal after running `eg`.

`eg git configure` also installs a Git alias `alias.elegant = "!eg"`, so `git elegant <command>`
still reaches Elegant Git after configure. That form runs from the repository top level (not the current
subdirectory) and does not tab-complete — use `eg <TAB>` for completion. It also installs a Git
alias for each of the old flat command names, so `git save-work` still reaches `eg work save`.
Those names are deprecated — please use the object-first form in anything you write down.

Also, please use [`eg repo clone`](reference/commands.md#repo) or
[`eg repo init`](reference/commands.md#repo) instead of regular `clone` or `init` when
starting work with a repository — both of them configure the fresh repository for you. For a
repository you already have, run [`eg repo configure`](reference/commands.md#repo) inside
it.

P.S. Shell completion is generated on demand for `bash`, `zsh`, `fish`, and `powershell`. For
example:

```bash
eg completion bash > ~/.local/share/bash-completion/completions/eg
```

`git elegant <TAB>` does not complete. If you previously installed completion for `git-elegant`,
delete that file so a dead registration does not linger.
