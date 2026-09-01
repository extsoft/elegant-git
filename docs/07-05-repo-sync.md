---
layout: default
title: eg repo sync
parent: '"repo" guide'
nav_order: 5
permalink: /repo/sync/
---

# eg repo sync

`repo sync` re-applies workspace settings to repositories.

```bash
eg repo sync
```

With no modifier it syncs the current repository, which must already have an
`elegant-git.repo-id` — run [`repo configure`](07-01-repo-configure.md) first if it does not.
`--all` walks every managed repository.

A repository with no workspace, or a missing workspace, is skipped. A missing path is reported
and hints at `eg workspace doctor` or `eg repo doctor`.

In interactive mode each repository is a closed question — yes, no, all, or skip — default no.
`all` applies the rest without asking; `skip` stops further prompts. In non-interactive mode
every target is applied without asking.

On a successful apply it writes `user.name`, `user.email`, and the optional signing key, gpg
program, and editor from the linked workspace, then prints `Synced <name>`.
