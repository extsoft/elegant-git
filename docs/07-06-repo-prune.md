---
layout: default
title: eg repo prune
parent: '"repo" guide'
nav_order: 6
permalink: /repo/prune/
---

# eg repo prune

`repo prune` removes useless local branches.

```bash
eg repo prune
```

There are no arguments. The command checks out the default development branch, fetches if an
upstream exists (and says so if fetch fails, then continues with the local version), and rebases
onto that upstream when it can.

Then each local branch that is not protected is deleted when either its upstream is gone, or it
has no upstream and its tip is already on the default development branch. The corresponding
branch source is cleared from per-repo memory.

Protected branches and the default development branch are never removed.
