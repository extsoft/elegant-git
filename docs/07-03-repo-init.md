---
layout: default
title: eg repo init
parent: '"repo" guide'
nav_order: 3
permalink: /repo/init/
---

# eg repo init

`repo init` initializes a new repository and configures it.

```bash
eg repo init work
```

The workspace is required, with the same resolver as [`repo configure`](07-01-repo-configure.md)
— an existing name or `[Create new]`.

It runs `git init`, then the same onboarding as configure, then makes an empty initial commit and
shows it. After that the repository is a configured Elegant Git repository on the default
development branch.
