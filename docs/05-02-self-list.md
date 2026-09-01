---
layout: default
title: eg self list
parent: '"self" guide'
nav_order: 2
permalink: /self/list/
---

# eg self list

`self list` shows Elegant Git installation and Git configuration state. It is read-only.

```bash
eg self list
```

There are no arguments, and nothing is asked. The same output is printed in interactive and
non-interactive mode.

It reports the binary version, the shared-memory path (and whether the file exists), the number of
workspaces and managed repositories, the global Git identity (`user.name`, `user.email`, signing
key, gpg program, editor), and whether this Elegant Git installation is acquired. If `acquired_version` is
missing it suggests `eg self configure`.

When `ELEGANT_GIT_STATE_FILE` overrides the path, that override is printed too.

The footer points at `eg workspace list all` and `eg repo list all` for the contents of shared
memory. The files themselves are described on the [memory](reference/memory.md) page.
