---
name: tasks-plan
description: >
  Creates plan.md and numbered task files under .tasks/<slug>-<id>/. Use when the
  user wants a plan, task breakdown, ADR, /plan, or to implement before a request
  folder exists—even if they say "break this into steps" without naming plan. Do not
  use for trivial one-line fixes, when only executing an existing folder
  (tasks-execute), or when planning is forbidden.
disable-model-invocation: true
compatibility: Requires bash, git, and write access under .tasks/ from repo root.
---

# tasks-plan

## References

| When | Read |
| --- | --- |
| Always | [task-schema.md](../../../references/task-schema.md) |
| Always | [definition-of-done.md](../../../references/definition-of-done.md) |
| Contract | [../README.md](../README.md) |
| ADR, security | [practices.md](../../../references/practices.md) |

## Procedure

1. Restate **goal** and **non-goals**.
2. Record **ADR-lite** and **risks/rollback**; add Threat model / Data migration / Deprecation only when [practices.md](../../../references/practices.md) applies.
3. For rewrites: **target structure** and **legacy teardown** in `plan.md` (discovered paths only).
4. Break work into tasks (≤1 PR each), ordered `depends_on`.
5. Scaffold once (reuse folder if it exists):

```bash
.agents/skills/tasks/plan/scripts/new-request.sh <slug> [task-count]
```

6. Fill `plan.md` from [assets/plan.template.md](assets/plan.template.md); rename `NNNN-task-N.md` → `NNNN-<verb-slug>.md` using [assets/task.template.md](assets/task.template.md).
7. **Validate before handoff** (required):

```bash
.agents/skills/tasks/plan/scripts/validate-folder.sh .tasks/<slug>-<shorthash>
```

8. Tell the user the folder path; hand off to **tasks-execute**.

## Gotchas

- Split by concern—not one mega-task.
- **Steps** use paths from repo discovery, not legacy layout.
- Link each task **Context** to `plan.md`.
- No product code in this skill—only `.tasks/` artifacts.

## When to skip

Trivial fix; user forbade planning; `.tasks/` exists and they only want implementation.
