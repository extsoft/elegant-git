---
name: execute
description: >
  Use when a .tasks/<slug>-<id>/ folder exists and the user wants to implement, continue,
  execute, or work through planned tasks in order. Update task status, run verification,
  satisfy acceptance and definition-of-done, commit one task per commit. Do not use when
  no .tasks/ folder exists (use plan first), for brainstorm-only or plan-only edits, or
  for ad-hoc coding without reading plan.md and the active task file.
compatibility: Requires bash and git; use project CI/test commands documented in plan.md or README.
---

# Execute

## Progress (one task at a time)

- [ ] Folder and `plan.md` read
- [ ] Next eligible task selected (`pending`, deps `done`)
- [ ] Loop checklist: [references/loop-checklist.md](references/loop-checklist.md)

## References

| When | Read |
| --- | --- |
| Status changes | [task-schema.md](../../references/task-schema.md) |
| Before `done` | [definition-of-done.md](../../references/definition-of-done.md) |
| Coding discipline | [practices.md](../../references/practices.md) |
| After status change | [adapters.md](../../references/adapters.md) |

## Loop

1. **Locate** `.tasks/<slug>-<id>/` (user path, or single unambiguous folder).
2. **Pick** lowest `NNNN` with `status: pending` and satisfied `depends_on`.
3. **Start** — `status: in_progress`; run:

```bash
.agents/skills/execute/scripts/github-sync.sh .tasks/<folder>/NNNN-<slug>.md
```

4. **Implement** — follow task **Steps**; surgical edits only.
5. **Verify** — use CI/test commands named in `plan.md`, task file, or README (record command + result in task notes).
6. **Complete** — confirm [references/loop-checklist.md](references/loop-checklist.md); set `status: done`; run `github-sync.sh` again.
7. **Commit** (one per task, when user requests commits):

```
<slug>(<task-id>): <imperative summary>

Task: .tasks/<folder>/NNNN-<slug>.md
```

8. **Repeat** until no `pending` tasks remain.

## Blockers

Set `status: blocked`; append `## Blocker` (cause + needed input); run `github-sync.sh`; stop loop.

## Gotchas

- Only one `in_progress` task unless `plan.md` allows parallel work.
- Never mark `done` without evidence for every `acceptance` item.
- Deprecation or contract changes: update the project's canonical deprecation log in the same commit when one exists.
- Do not commit unless the user asked (follow project git rules).

## Handoff

When all tasks are `done`, summarize shipped work and cite the task folder path.

## When to skip

No `.tasks/` folder; user only wants brainstorm or plan updates.
