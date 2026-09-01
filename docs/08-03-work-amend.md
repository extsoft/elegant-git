---
layout: default
title: eg work amend
parent: '"work" guide'
nav_order: 3
permalink: /work/amend/
---

# eg work amend

`work amend` amends the last commit.

```bash
eg work amend
```

There are no arguments. It runs `git add --interactive` and then `git commit --amend`.

On a protected branch it always refuses — rewriting history there is not allowed, on a TTY or
off one. `amend` is never detected and never offered by bare `eg work`; you have to ask for it.

This command does not use the stash pipe.
