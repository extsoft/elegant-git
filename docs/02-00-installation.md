---
layout: default
title: Installation
nav_order: 2
has_children: true
has_toc: false
permalink: /installation/
---

# Installation

`git-elegant` is a single executable, and its directory has to be on your `PATH` for Git to resolve
`git elegant`. Pick one way, then continue with [getting started](03-getting-started.md).

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
go install github.com/extsoft/elegant-git/cmd/git-elegant@latest
```

Need to know where the binary lands, or pin a tag? See
[using go install](02-03-installation-go-install.md).
