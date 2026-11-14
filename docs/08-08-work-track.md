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
name is optional. In a terminal, interactive mode opens the validated name prompt with the
branch part of the remote already filled in, or with the local name you passed. That value is
checked immediately. Accepting a filled-in name that already tracks this remote asks what to
do next.

The command fetches all remotes, then matches. Zero matches or more than one is an error — make
the pattern unambiguous.

A free local name is created with `checkout -b` so the new branch tracks that remote. An
existing local branch is not moved unless you ask for that.

When the local name already exists and tracks a different remote, or tracks nothing, the command
stops. A name that conflicts across a slash with an existing branch is the same kind of error.
Non-interactive mode exits with a [workflow error](reference/exit-codes.md). In a terminal,
interactive mode shows that problem under the filled-in name and ignores Enter until the name
is usable. A name you pass on the command line that is rejected is kept in that prompt, not
cleared.

When the local branch already tracks the matched remote, non-interactive mode exits with that
workflow error and tells you to run `git switch` on the branch. Interactive mode asks what to
do:

- check the branch out without moving its tip
- fast-forward it, when it is an ancestor of the remote
- reset it to the remote; the choice warns when the local branch has commits the remote does not
- pick a new local name, with that same prompt, and create that branch

The fast-forward choice is the default when it is offered. Otherwise the default is to check the
branch out without moving its tip.

There are no pipes and no protected-branch check. Without remotes, bare `eg work` does not offer
`track`.
