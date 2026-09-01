---
layout: default
title: eg repo list
parent: '"repo" guide'
nav_order: 4
permalink: /repo/list/
---

# eg repo list

`repo list` lists repositories or shows the current or named one.

```bash
eg repo list
```

The argument is optional and is never asked. Inside a git work tree, a missing argument shows the
current repository. Outside one, it lists every managed repository. `current` and `all` are
selectors; any other name or path shows that repository's details.

```bash
eg repo list all
eg repo list current
eg repo list app
```

`--format=table` is the default. `--format=json` prints the same data as JSON, but it is not
allowed with the `current` selector. Passing `--format` without a name lists all, even inside a
repository, so JSON works without typing `all`.

The current view is per-repo memory, registry linkage, branch settings, and the local git
identity. The all view is a catalog of name, workspace, and path. The named view is registry
fields, previous paths, workspace, and per-repo memory. Nothing is asked in either mode.
