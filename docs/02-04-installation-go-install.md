---
layout: default
title: Using go install
parent: Installation
nav_order: 4
permalink: /installation/go-install/
---

# Using go install

With a Go toolchain available, you can build from source instead of downloading a release:

```bash
go install github.com/extsoft/elegant-git/cmd/eg@latest
```

This puts the `eg` binary into `$(go env GOPATH)/bin`, which is `~/go/bin` unless you changed it.
That directory has to be on your `PATH`.

Replace `@latest` with a tag — for example `@v2026.8.29` — to build a specific release. Note that a
binary built this way reports the version baked in at build time, so `eg --version` may
read `dev` for a source build outside a release tag.
