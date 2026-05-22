---
name: execute
description: >
  Use when a .tasks/<slug>-<id>/ folder exists and the user wants to implement, continue,
  execute, or work through planned tasks in order. By default complete exactly one task per
  invocation and stop; only chain multiple tasks when the user explicitly asks. Update task
  status, run verification, satisfy acceptance and definition-of-done, commit one task per
  commit. Do not use when no .tasks/ folder exists (use plan first), for brainstorm-only or
  plan-only edits, or for ad-hoc coding without reading plan.md and the active task file.
compatibility: Requires bash and git; use project CI/test commands documented in plan.md or README.
---

# Execute

## Default scope

**One task per run** — unless the user clearly asks otherwise (e.g. “continue”, “next task”, “run all tasks”, “finish the folder”).

After that task is `done`, **stop**: report what shipped, which task is next, and wait for the user before starting another task.

Exceptions:

- **`plan.md` explicitly allows parallel work** — multiple `in_progress` tasks only when the plan says so.
- **User names a specific task** — implement only that `NNNN-*.md` file (still one task).
- **User asks to run the whole folder** — then follow the loop through all remaining `pending` tasks in order.

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
5. **Verify** — from repo root: **`mise run fix`** (lint/format; do not use `mise run check` for agent work), **`mise run test`** when Go code changed, plus any commands named in `plan.md` or the task file (record command + result in task notes).
6. **Complete** — confirm [references/loop-checklist.md](references/loop-checklist.md); set `status: done`; run `github-sync.sh` again.
7. **Commit** (one per task, when user requests commits):

```
<slug>(<task-id>): <imperative summary>

Task: .tasks/<folder>/NNNN-<slug>.md
```

8. **Stop** — end the run after step 6–7 for this task. Do **not** start the next `pending` task in the same run unless the user requested multi-task execution (see **Default scope**).

## Blockers

Set `status: blocked`; append `## Blocker` (cause + needed input); run `github-sync.sh`; stop loop.

## Gotchas

- Default is **one task, then stop** — do not silently continue to 0002 after finishing 0001.
- Only one `in_progress` task unless `plan.md` allows parallel work or the user asked for parallel execution.
- Never mark `done` without evidence for every `acceptance` item.
- Deprecation or contract changes: update the project's canonical deprecation log in the same commit when one exists.
- Do not commit unless the user asked (follow project git rules).

## Handoff

After each single-task run: summarize what changed, cite `.tasks/<folder>/NNNN-<slug>.md`, and name the next eligible task (or “none” if the folder is complete).

When **all** tasks are `done` (only after the user has asked you to work through the full folder, or you finish the last task in such a run): summarize the whole folder and cite its path.

## When to skip

No `.tasks/` folder; user only wants brainstorm or plan updates.
