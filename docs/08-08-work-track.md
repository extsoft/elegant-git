---
layout: default
title: eg work track
parent: '"work" guide'
nav_order: 8
permalink: /work/track/
---

# eg work track

`work track` checks out a remote-tracking branch.

```bash
eg work track origin/feat
```

The remote name is required. It can be a full remote-tracking ref or a substring that matches
one. In interactive mode, leaving it out offers a picker of remote branches. The local branch
name is optional; interactive mode offers the branch part of the remote as a suggestion.

The command fetches all remotes, then matches. Zero matches or more than one is an error — make
the pattern unambiguous. Then it checks out `-B` the local branch onto that remote so it tracks
it.

There are no pipes and no protected-branch check. Without remotes, bare `eg work` does not offer
`track`.
