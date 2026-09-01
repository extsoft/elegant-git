---
layout: default
title: eg workspace delete
parent: '"workspace" guide'
nav_order: 5
permalink: /workspace/delete/
---

# eg workspace delete

`workspace delete` deletes a workspace and unlinks its repositories.

```bash
eg workspace delete work
```

The name is required; in interactive mode it is asked if you leave it out.

The command always prints what will happen: the workspace identity, the linked repositories, and
that unlinking leaves their Git config and files untouched. Those repositories stay in the
registry with `workspace_id` cleared. It suggests `eg repo configure` if you want to attach them
to another workspace.

`--yes` skips the confirmation. In non-interactive mode `--yes` is required; without it the
command fails rather than deleting anything. Interactive mode asks `Delete?` — default yes when
nothing is linked, default no when at least one repository is.

On success it prints `Deleted workspace <name>`.
