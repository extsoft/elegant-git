---
layout: default
title: eg workspace link
parent: '"workspace" guide'
nav_order: 3
permalink: /workspace/link/
---

# eg workspace link

`workspace link` links the current repository to a workspace.

```bash
eg workspace link work
```

You have to be in a git repository. The name is required; in interactive mode it is asked if you
leave it out, with completion over existing workspaces.

The command writes the identity even if the repository already has values, upserts the registry,
records per-repo memory, and may capture the origin
[namespace](06-00-workspace.md#how-a-namespace-is-remembered). If the repository is already linked
elsewhere, it asks before overriding; declining leaves the previous link in place.

On success it prints `Linked repository to workspace <name>`.
