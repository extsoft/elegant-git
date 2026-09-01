---
layout: default
title: eg hook edit
parent: '"hook" guide'
nav_order: 3
permalink: /hook/edit/
---

# eg hook edit

`hook edit` opens a hook file in your editor.

```bash
eg hook edit .git/.config/elegant-git/hooks/work-save-ahead
```

The path is required. In interactive mode it is asked if you leave it out, with completion over
existing hook paths. A missing file is an error.

The editor is `core.editor`. Where hooks live, and how they are named, is on the
[hook files](reference/hook-files.md) page.
