# Agent orientation

## Repo map

Discover layout from the repo root (README, CONTRIBUTING, package manifests). Do not assume paths from a prior codebase version—this project may be rewritten.

| Path | Role |
| --- | --- |
| `.agents/` | Agent skills and shared references ([agentskills.io](https://agentskills.io/)) |
| `.tasks/` | Request plans and executable task files |

Record discovered paths (source, tests, CI, docs) in `plan.md` when planning a rewrite or greenfield work.

## Workflow

1. **Brainstorm** — explore options without committing (`.agents/skills/brainstorm/`).
2. **Plan** — pick an approach, write `.tasks/<slug>-<id>/` with `plan.md` and numbered tasks (`.agents/skills/plan/`).
3. **Execute** — implement tasks one at a time, update status, verify DoD (`.agents/skills/execute/`).
4. **GitHub sync** (future) — stub adapter in `execute/scripts/github-sync.sh`.

## Skills index

| Skill | Path | When to use |
| --- | --- | --- |
| brainstorm | [`.agents/skills/brainstorm/SKILL.md`](.agents/skills/brainstorm/SKILL.md) | Ambiguous or multi-approach requests; chat-only exploration |
| plan | [`.agents/skills/plan/SKILL.md`](.agents/skills/plan/SKILL.md) | Scaffold and fill `.tasks/<slug>-<id>/`; run `validate-folder.sh` before handoff |
| execute | [`.agents/skills/execute/SKILL.md`](.agents/skills/execute/SKILL.md) | Implement tasks from an existing `.tasks/` folder |

Skills follow [Agent Skills](https://agentskills.io/) ([spec](https://agentskills.io/specification), [authoring guide](https://agentskills.io/skill-creation/best-practices)). Templates live under each skill's `assets/`; shared contracts under `.agents/references/`.

## Practices index

| Reference | Topic |
| --- | --- |
| [`.agents/references/practices.md`](.agents/references/practices.md) | Engineering practices enforced during plan and execute |
| [`.agents/references/task-schema.md`](.agents/references/task-schema.md) | Task file frontmatter and lifecycle |
| [`.agents/references/definition-of-done.md`](.agents/references/definition-of-done.md) | Definition of Done checklist |
| [`.agents/references/adapters.md`](.agents/references/adapters.md) | GitHub issue adapter contract (stub) |

## Task storage

Active work lives in [`.tasks/`](.tasks/). Each request gets a folder `<slug>-<shorthash>/` containing `plan.md` and `NNNN-<task-slug>.md` files. See [`.tasks/README.md`](.tasks/README.md).
