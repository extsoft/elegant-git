---
layout: default
title: eg self configure
parent: '"self" guide'
nav_order: 1
permalink: /self/configure/
---

# eg self configure

`self configure` applies Git configuration globally and records this Elegant Git installation as
acquired. It writes global Git identity, standards, and aliases, then stores that marker in
[shared memory](reference/memory.md).

```bash
eg self configure
```

There are no arguments. The first interactive `eg` command runs the same flow automatically when
the Elegant Git installation is not yet acquired. `--non-interactive`, `CI`, and a non-TTY skip
that automatic start; `eg self configure` itself still runs at any time.

## What it applies

The command prints a plan — Git basics, standards, and aliases — points at
[configuration](reference/configuration.md), and in interactive mode waits for Enter. Then it
walks those steps in order, using `git config --global`.

Basics fill only keys that are still unset: `user.name` and `user.email` are required,
`core.editor` is optional and defaults to `vim` when nothing is set. Keys that already have a
value are printed and left alone. If name or email is still empty after this step, the command
stops.

After basics, in interactive mode, it offers to create a [workspace](06-00-workspace.md) from the
global values. The name defaults to the local part of the email. If a workspace already matches
those values, it says so and does not ask. Non-interactive mode skips the offer.

Standards and aliases follow. The exact keys, the `acquired_version` marker, and how
`alias.elegant = "!eg"` is written are on the [configuration](reference/configuration.md) page.
The acquired marker is recorded last, so an interrupted run still auto-starts next time.

When the run finishes it prints that global Git configuration is complete.
