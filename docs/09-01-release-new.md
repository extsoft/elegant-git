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

A dirty tree uses the same `carry`, `stash`, `discard`, and `cancel` choice as
[eg work sync](08-06-work-sync.md), including `--untracked` on discard. `carry` restores the changes on the branch you started on,
when that branch still exists. If it is gone, the backup stays. Without a terminal, a dirty
tree fails until `--dirty` is set. An in-progress Git operation is refused first. The tag
message is written to a temporary file outside the worktree, so a file named `tag-message` in
your tree is left alone. The editor for that message is still Git's, including when Elegant
Git is non-interactive. A failed pull happens before the tag exists. A failed tag push says
the local tag was created and the remote was not updated. Repeating the command after that
failure does not create or push the tag again. The
[branch pipe](reference/pipes.md) brings you back to the branch you were on when it still
exists. "The last tag" means the highest tag by version order, not the most recently created
one.
