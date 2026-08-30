---
layout: default
title: Installation
nav_order: 2
has_children: true
has_toc: false
permalink: /installation/
---

# Installation

Pick one way to install Elegant Git so the `eg` command is on your `PATH`, then continue with
[getting started](03-getting-started.md).

## Using install script

```bash
curl -fsSL https://github.com/extsoft/elegant-git/releases/latest/download/install.sh | sh
```

Need a different directory, a specific release, or a `PATH` hint? See [using the install
script](02-01-installation-install-script.md).

## Using mise

```bash
mise use --global "github:extsoft/elegant-git@latest"
```

Want it only in this project, or to update later? See [using mise](02-02-installation-mise.md).

## Using go install

```bash
go install github.com/extsoft/elegant-git/cmd/eg@latest
```

Need to know where the binary lands, or pin a tag? See
[using go install](02-03-installation-go-install.md).

## Upgrading from `git-elegant`

The binary was renamed from `git-elegant` to `eg`. Git no longer finds it as a
`git-<subcommand>` on `PATH`.

1. Remove every `git-elegant` binary from `PATH`. A leftover copy **shadows** the
   `alias.elegant` git alias, so `git elegant …` keeps invoking the old program.
2. Delete stale completion files (`~/.local/share/bash-completion/completions/git-elegant`
   and the zsh equivalent).
3. Install Elegant Git. The first interactive `eg` command (or `eg git configure`) writes
   `alias.elegant = "!eg"`. Leftover aliases from the previous binary are rewritten on their
   own; `eg git doctor` shows anything that still needs a decision.
