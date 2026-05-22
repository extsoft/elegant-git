# Tasks

Executable work for agent-driven development. Each **request** gets its own folder.

## Layout

```
.tasks/<slug>-<shorthash>/
├── plan.md                 # goal, ADR-lite, risks, task index
├── 0001-<task-slug>.md
├── 0002-<task-slug>.md
└── ...
```

- **slug** — kebab-case summary (e.g. `auth-rewrite`)
- **shorthash** — 4 hex chars to disambiguate concurrent requests

## Lifecycle

| status | meaning |
| --- | --- |
| `pending` | Not started |
| `in_progress` | Agent is working on it |
| `blocked` | Waiting on external input |
| `done` | Acceptance + DoD satisfied |
| `cancelled` | Will not be done |

Scaffold a new request:

```bash
.agents/skills/plan/scripts/new-request.sh <slug> [task-count]
.agents/skills/plan/scripts/validate-folder.sh .tasks/<slug>-<shorthash>
```

## Workflow

1. **brainstorm** — explore options (no files here).
2. **plan** — fill `plan.md` and task files (see `.agents/skills/plan/`).
3. **execute** — implement tasks in dependency order (see `.agents/skills/execute/`).

Schema: [`.agents/references/task-schema.md`](../.agents/references/task-schema.md)

## Git

Commit task folder updates with implementation work, or keep `.tasks/` local-only — team choice. Folders are safe to gitignore per-request if you prefer ephemeral plans.
