---
layout: default
title: eg workspace doctor
parent: '"workspace" guide'
nav_order: 7
permalink: /workspace/doctor/
---

# eg workspace doctor

`workspace doctor` diagnoses and repairs a workspace.

```bash
eg workspace doctor
```

You can name the workspace to examine; if you don't, Elegant Git uses the one linked to the current
repository, or asks you to pick one. In non-interactive mode the name is required unless a
workspace is already linked.

Every problem is printed together with the repair it proposes. A yes/no repair applies only after
you confirm. When there is more than a yes or no, you are asked once — a missing path offers
path / remove / ignore; a path that is not a Git work tree offers remove / ignore; an orphan
workspace reference offers clear / remove, and there is no ignore because shared memory could not
be saved otherwise. If you decline or ignore a repair, the problem stays. Shared memory is written
once, after your last answer, and only when something actually changed.

In non-interactive mode `doctor` changes nothing and fails when at least one problem is found, so
your automation notices an unhealthy workspace. Repairs that need more than a yes or no — a new
path, a replacement name, an identity field — are reported without a question.

If nothing is wrong, it prints that the workspace looks healthy.

## Shared memory problems

- `name`, `user_name`, or `user_email` is empty — asks for the value
- another workspace has the same `name` — asks for a new name
- `linked_repos` holds an unknown repository — drops that entry
- `linked_repos` holds a repository owned by another workspace — drops that entry
- a repository points to this workspace but is missing from `linked_repos` — adds it
- a repository points to a workspace that no longer exists — clears `workspace_id` or removes the
  entry
- `namespaces` holds blank, untrimmed, or duplicated values — normalizes them

## Repository problems

- `current_path` no longer exists or is inaccessible — set a new work tree path, remove the entry,
  or ignore it. Setting a path stamps `elegant-git.repo-id` when it is unset and moves the previous
  path into `path_history`. A path already owned by another registry entry, or already stamped with
  a different `elegant-git.repo-id`, is refused
- `current_path` is not a Git work tree — remove the entry, or ignore it
- `elegant-git.repo-id` is unset or differs from the registry — stamps the registry id
- per-repo memory `workspace_id` differs from the registry — rewrites it
