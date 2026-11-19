---
layout: default
title: eg work polish
parent: '"work" guide'
nav_order: 5
permalink: /work/polish/
---

# eg work polish

`work polish` rebases HEAD interactively.

```bash
eg work polish
```

There are no arguments. A new rebase on a protected branch is refused. A rebase already in
progress is continued first, including when that branch is protected, and conflict resolutions
are not stashed or discarded.

If there are no unique commits against the freshest source, it says so and stops. Otherwise it
rebases those unique commits. A dirty tree uses the same `carry`, `stash`, `discard`, and
`cancel` choice as [eg work sync](08-06-work-sync.md), including `--untracked=keep|delete` on discard. `carry` restores the changes on this
branch. Without a terminal, a dirty tree fails until `--dirty` is set. Non-interactive mode
does not replace Git's rebase editor. Ignored files that a replay would overwrite are left in
place, as described on that page.

When you started this rebase from `accept`, the helper branch is `__eg` and bare `eg work`
continues with `accept` instead of `polish`. See [the branch lifecycle](08-00-work.md#the-branch-lifecycle).
