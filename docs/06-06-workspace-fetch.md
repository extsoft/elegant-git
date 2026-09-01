---
layout: default
title: eg workspace fetch
parent: '"workspace" guide'
nav_order: 6
permalink: /workspace/fetch/
---

# eg workspace fetch

`workspace fetch` fetches remotes for repositories linked to a workspace.

```bash
eg workspace fetch
```

The name is optional and is never asked. When you leave it out, Elegant Git uses the workspace
linked to the current repository. If you are not in a linked repository, name the workspace.

It runs `git fetch --all --tags --prune` in every linked repository, which also prunes stale
remote-tracking branches. A repository with no remotes is skipped. A missing path is reported
and hints at `eg workspace doctor` or `eg repo doctor`.

On a TTY you get a progress bar, a processed-repository list, and ephemeral logs for the current
fetch. Otherwise each repository is a `[i/n] fetching <name>` line.

The command ends with counts of fetched, skipped, and failed. It exits non-zero if any repository
fails. If nothing is linked, it says so and stops.
