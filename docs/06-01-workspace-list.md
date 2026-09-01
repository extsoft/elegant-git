---
layout: default
title: eg workspace list
parent: '"workspace" guide'
nav_order: 1
permalink: /workspace/list/
---

# eg workspace list

`workspace list` lists workspaces or shows the current or named one.

```bash
eg workspace list
```

The name is optional and is never asked. Inside a git work tree, a missing name shows the
workspace linked to the current repository. Outside one, it lists every workspace.
`current` and `all` are selectors; any other name shows that workspace's details.

```bash
eg workspace list all
eg workspace list current
eg workspace list work
```

`--format=table` is the default. `--format=json` prints the same data as JSON, but it is not
allowed with the `current` selector. Passing `--format` without a name lists all, even inside a
repository, so JSON works without typing `all`.

The current and named views show identity fields, namespaces, and the linked-repository catalog.
The all view is a short catalog — identity, repository count, and a command to explore each
workspace. Nothing is asked in either mode.
