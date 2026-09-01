---
layout: default
title: eg repo doctor
parent: '"repo" guide'
nav_order: 7
permalink: /repo/doctor/
---

# eg repo doctor

`repo doctor` diagnoses and repairs the current repository. You have to be in a git repository.

```bash
eg repo doctor
```

There are no arguments.

Every problem is printed together with the repair it proposes. Interactive mode confirms each
repair. In non-interactive mode the command changes nothing and fails when at least one problem
is found, so automation notices an unhealthy repository.

If nothing is wrong, it prints that the repository looks healthy. After a repair actually ran, it
prints that the current repository was repaired.

## What it looks for

- `elegant-git.repo-id` is unset — stamps a matching registry entry, or tells you to run
  `eg repo configure` if there is none
- the id has no registry entry — re-adds this repository to shared memory
- the registry path differs from the current directory — updates `current_path` and keeps the old
  one in `path_history`
- `workspace_id` is empty or points at a missing workspace — pick a workspace and link
- per-repo memory `repo_id` or `workspace_id` differs from the registry — rewrites it
- local git identity differs from the linked workspace — re-applies the workspace with force
- legacy `elegant-git.default-branch` / `protected-branches` still in local config — moves them
  into per-repo memory
- global Elegant Git is acquired but local elegant aliases or an acquired marker remain — removes
  them
- personal or repo-tracked hooks still live under `.workflows/` — moves them; tracked hooks need
  a commit afterwards, so doctor prints the `git add` / `git commit` to use
- hook files still use the `git-` command prefix — renames them to `self-`

The hook layout itself is on the [hook files](reference/hook-files.md) page.
