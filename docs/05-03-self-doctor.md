---
layout: default
title: eg self doctor
parent: '"self" guide'
nav_order: 3
permalink: /self/doctor/
---

# eg self doctor

`self doctor` diagnoses and repairs the Elegant Git installation and Git configuration. It looks
at global aliases. It does not inspect a repository — that is
[`eg repo doctor`](07-07-repo-doctor.md).

```bash
eg self doctor
```

There are no arguments.

## What it looks for

Every finding is printed as a problem and a proposed repair.

- Global Elegant aliases have drifted — `alias.elegant` is not `!eg`, or a leftover flat alias
  points at the wrong command. The repair rewrites them.

If nothing is wrong, it prints that the Elegant Git installation looks healthy.

## Repairing

In interactive mode, a repairable finding is applied only after you confirm.

In non-interactive mode the command changes nothing. It prints every finding, then fails if at
least one of them is repairable, so automation notices an unhealthy Elegant Git installation.

When a repair actually ran, it prints that the Elegant Git installation was repaired.
