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

## When Git is missing or too old

If Git is not on `PATH`, or it is older than 2.40.0, or `git version` cannot be read, the command
prints the required version, what it found, and the executable it selected. It tells you to put a
maintained Git first on `PATH`. It exits with status 1. It does not offer a repair, and it does
not print unrelated findings. Git 2.40.0 is the oldest version that can run Git-backed commands.
It is not a security recommendation.

With a supported Git, diagnosis continues as below.

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
