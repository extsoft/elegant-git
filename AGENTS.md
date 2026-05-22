# Agent orientation

## Repo map

Discover layout from the repo root (README, CONTRIBUTING, package manifests). Do not assume paths from a prior codebase version—this project may be rewritten.

| Path | Role |
| --- | --- |
| `.agents/` | Agent skills and shared references ([agentskills.io](https://agentskills.io/)) |
| `.agents/skills/tasks/` | Brainstorm / plan / execute workflow and task-folder contract |
| `.config/mise/` | Dev tools, `mise run fix` / `test` / `build` tasks ([mise](https://mise.jdx.dev/)) |
| `.cursor/rules/mise.mdc` | Agents: use `mise run fix` for lint/format (not `check`) |
| `.tasks/` | Ephemeral request folders (gitignored `*.md`; see tasks README in skills) |

Record discovered paths (source, tests, CI, docs) in `plan.md` when planning a rewrite or greenfield work.

Run `mise run init` once after clone. For lint/format during agent work, use **`mise run fix`** (not `mise run check`; CI runs `check` in Actions).

## Workflow

1. **tasks-brainstorm** — explore options without committing ([`.agents/skills/tasks/brainstorm/`](.agents/skills/tasks/brainstorm/)).
2. **tasks-plan** — write `.tasks/<slug>-<id>/` with `plan.md` and numbered tasks ([`.agents/skills/tasks/plan/`](.agents/skills/tasks/plan/)).
3. **tasks-execute** — implement one task at a time ([`.agents/skills/tasks/execute/`](.agents/skills/tasks/execute/)).
4. **GitHub sync** (future) — stub in `tasks/execute/scripts/github-sync.sh`.

## Skills index

| Skill | Path | When to use |
| --- | --- | --- |
| tasks-brainstorm | [`.agents/skills/tasks/brainstorm/SKILL.md`](.agents/skills/tasks/brainstorm/SKILL.md) | Ambiguous or multi-approach requests; chat-only |
| tasks-plan | [`.agents/skills/tasks/plan/SKILL.md`](.agents/skills/tasks/plan/SKILL.md) | Scaffold and fill `.tasks/<slug>-<id>/`; validate before handoff |
| tasks-execute | [`.agents/skills/tasks/execute/SKILL.md`](.agents/skills/tasks/execute/SKILL.md) | Implement from an existing `.tasks/` folder |

Skills follow [Agent Skills](https://agentskills.io/). Templates under each skill's `assets/`; contracts under [`.agents/skills/tasks/README.md`](.agents/skills/tasks/README.md) and [`.agents/references/`](.agents/references/).

## Practices index

| Reference | Topic |
| --- | --- |
| [`.agents/references/practices.md`](.agents/references/practices.md) | Engineering practices |
| [`.agents/references/task-schema.md`](.agents/references/task-schema.md) | Task file frontmatter and lifecycle |
| [`.agents/references/definition-of-done.md`](.agents/references/definition-of-done.md) | Definition of Done |
| [`.agents/references/adapters.md`](.agents/references/adapters.md) | GitHub issue adapter (stub) |

## Task storage

Ephemeral instances: `.tasks/<slug>-<shorthash>/` with `plan.md` and `NNNN-<task-slug>.md`. Full contract: [`.agents/skills/tasks/README.md`](.agents/skills/tasks/README.md).
