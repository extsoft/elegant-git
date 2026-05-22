# Task request folders

Executable work for agent-driven development. Each **request** gets a folder under
`.tasks/` (typically gitignored; contract lives here in `.agents/skills/tasks/`).

## Layout

```
.tasks/<slug>-<shorthash>/
├── plan.md
├── 0001-<task-slug>.md
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

## Scaffold and validate

```bash
.agents/skills/tasks/plan/scripts/new-request.sh <slug> [task-count]
.agents/skills/tasks/plan/scripts/validate-folder.sh .tasks/<slug>-<shorthash>
mise run tasks-validate .tasks/<slug>-<shorthash>   # optional wrapper
```

## Workflow skills

| Skill | Path |
| --- | --- |
| tasks-brainstorm | [brainstorm/SKILL.md](brainstorm/SKILL.md) |
| tasks-plan | [plan/SKILL.md](plan/SKILL.md) |
| tasks-execute | [execute/SKILL.md](execute/SKILL.md) |

Schema: [`.agents/references/task-schema.md`](../../references/task-schema.md)

## Git

`.tasks/**/*.md` is gitignored by default. Request folders are ephemeral; only
change tracked repo files when a task requires it.
