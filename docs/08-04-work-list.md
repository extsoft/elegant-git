---
layout: default
title: eg work list
parent: '"work" guide'
nav_order: 4
permalink: /work/list/
---

# eg work list

`work list` prints HEAD state. It is read-only.

```bash
eg work list
```

There are no arguments. It reports the local branch, its upstream, and the source branch it was
started from, a short status (and suggests `save` when the work tree is dirty), the unique
commits against the freshest source (and suggests `polish` when there are some), and the stash
list.

A footer points at further steps. Nothing is asked, and protected branches are not special —
listing is always allowed.
