---
name: tasks-execute
description: >
  Implements one task at a time from an existing .tasks/ request folder. Use when the
  user says /execute, next task, continue the folder, or implement planned work—even
  without saying "execute." One task per run unless they explicitly ask for more. Do
  not use without a .tasks/ folder (tasks-plan first), for exploration-only chat, or
  for ad-hoc coding without reading plan.md and the active task file.
disable-model-invocation: true
compatibility: Requires bash and git; use project commands in plan.md or README.
---

# tasks-execute

## Default scope

**One task per run** unless the user asks otherwise (“continue”, “next task”, “run all”, “finish the folder”).

After `done`, **stop**: report what shipped, name the next task, wait.

Exceptions: parallel work only if `plan.md` allows; user names one `NNNN-*.md`; user asks to run the whole folder.

## References

| When | Read |
| --- | --- |
| Status | [task-schema.md](../../../references/task-schema.md) |
| Before `done` | [definition-of-done.md](../../../references/definition-of-done.md) |
| Coding | [practices.md](../../../references/practices.md) |

## Loop

1. **Locate** `.tasks/<slug>-<id>/` (user path or single unambiguous folder).
2. **Pick** lowest `NNNN` with `status: pending` and satisfied `depends_on`.
3. **Start** — set `in_progress`.
4. **Implement** — task **Steps** only; surgical diff.
5. **Verify** (required):

```bash
.agents/skills/tasks/execute/scripts/verify-task.sh
```

Records `mise run fix` and `mise run test` when Go changed; honor extra commands in plan.md or the task file. Append results under `## Verification` in the task file.
6. **Complete** — every `acceptance` item evidenced; set `done`.
7. **Commit** when the user asks (one commit per task):

```
<slug>(<task-id>): <imperative summary>

Task: .tasks/<folder>/NNNN-<slug>.md
```

8. **Stop** — do not start the next `pending` task unless multi-task was requested.

## Blockers

`status: blocked`; add `## Blocker`; stop.

## Gotchas

- Never mark `done` without verification output and acceptance evidence.
- One `in_progress` unless `plan.md` or the user allows parallel work.
- Deprecation log updates belong in the same commit when the project defines one.
- Do not commit unless the user asked.

## Handoff

Summarize changes, cite the task file, name the next task or “none”.

## When to skip

No `.tasks/` folder; brainstorm or plan-only edits.
