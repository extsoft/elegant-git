---
layout: default
title: eg hook new
parent: '"hook" guide'
nav_order: 2
permalink: /hook/new/
---

# eg hook new

`hook new` creates a new hook file.

```bash
eg hook new work.start ahead personal
```

All three arguments are required. In interactive mode they are asked if you leave them out.

- `<command-id>` is the canonical dotted form of the command — `work.start`, `repo.clone`,
  `release.new`. Legacy flat names such as `start-work` are still accepted, but they warn and will
  be removed.
- `<ahead|after>` is when the hook runs
- `<personal|common>` is whether the file lives in `.git/.config/elegant-git/hooks/` (yours) or
  `.config/elegant-git/hooks/` (tracked with the repository)

The command creates the file with a small `sh` stub, makes it executable, and opens it in your
editor. If the file already exists, it fails — use [`hook edit`](10-03-hook-edit.md) instead.

When the hook should run, and what a non-zero exit does, is on the
[hook files](reference/hook-files.md) page.
