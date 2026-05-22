---
name: plan
description: >
  Use when the user wants a plan, task breakdown, ADR, or executable work under
  .tasks/<slug>-<id>/—after brainstorm or when they ask to implement but no task folder
  exists yet. Scaffold with scripts/new-request.sh, fill plan.md and NNNN-*.md files,
  validate with scripts/validate-folder.sh. Do not use for trivial one-line fixes, when
  they only want to run existing .tasks/ (use execute), or when they forbade planning.
compatibility: Requires bash, git, and write access under .tasks/ from repo root.
---

# Plan

## Progress

- [ ] Goal, assumptions, non-goals written
- [ ] ADR-lite (chosen, rejected, consequences)
- [ ] Risks and rollback documented
- [ ] Target repo layout noted in plan.md if rewrite or greenfield
- [ ] `scripts/new-request.sh <slug> [count]` run once
- [ ] `plan.md` and tasks filled from templates
- [ ] `scripts/validate-folder.sh` passes
- [ ] User told folder path; handoff to **execute**

## References (read before writing)

| When | Read |
| --- | --- |
| Always | [task-schema.md](../../references/task-schema.md) |
| Always | [definition-of-done.md](../../references/definition-of-done.md) |
| ADR, PR size, security | [practices.md](../../references/practices.md) |

## Procedure

1. Restate **goal** (1–2 sentences) and **non-goals**.
2. Record **ADR-lite** and **risks/rollback**; add Threat model / Data migration / Deprecation sections only when [practices.md](../../references/practices.md) applies.
3. For rewrites: document intended **target structure** and **legacy teardown** in `plan.md` (paths discovered from repo, not assumed).
4. Break work into tasks: each ≤1 PR, ordered `depends_on`.
5. Scaffold (repo root):

```bash
.agents/skills/plan/scripts/new-request.sh <slug> [task-count]
```

Default `task-count` is `3`. Script exits with error if the folder already exists—reuse that folder instead of re-running.

6. Fill `plan.md` from [assets/plan.template.md](assets/plan.template.md); rename skeleton `NNNN-task-N.md` files to `NNNN-<verb-slug>.md` and fill from [assets/task.template.md](assets/task.template.md).
7. Validate:

```bash
.agents/skills/plan/scripts/validate-folder.sh .tasks/<slug>-<shorthash>
```

Fix failures before handoff.

## Gotchas

- Split tasks by concern (e.g. core behavior, tests, docs, tooling)—not one mega-task.
- Task **Steps** must use paths and tools that exist in the plan or repo discovery—not legacy layout.
- Link each task **Context** to `plan.md`.
- Do not implement code in this skill—only `.tasks/` artifacts.

## Handoff

*Next: **execute** skill on `.tasks/<slug>-<shorthash>/`.*

## When to skip

Trivial fix; user forbade planning; `.tasks/` exists and they only want implementation.
