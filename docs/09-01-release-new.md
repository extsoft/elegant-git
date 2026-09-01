---
layout: default
title: eg release new
parent: '"release" guide'
nav_order: 1
permalink: /release/new/
---

# eg release new

`release new` releases the default development branch.

```bash
eg release new v2026.8.29
```

The name is the only argument, and it becomes the tag. In interactive mode you are asked for it
when you leave it out; in non-interactive mode a missing name fails the command. From there
Elegant Git

1. checks out the default development branch and runs `git pull --tags`
2. drafts an annotated tag message — a `Release <name>` heading followed by one `- <subject>` line
   per commit since the last tag — and opens it in your editor so you can edit it before it is
   recorded
3. pushes the tags
4. produces the release notes in the `smart` layout and puts them on your clipboard when `pbcopy`
   or `xclip` is available, printing them otherwise

The command runs inside both the [stash pipe and the branch pipe](reference/pipes.md), so
uncommitted changes and the branch you were on are preserved and restored — you end up back where
you started. "The last tag" means the highest tag by version order, not the most recently created
one.
