# Task schema

Contract overview: [`.agents/skills/tasks/README.md`](../skills/tasks/README.md).

## Folder layout

```
.tasks/<slug>-<shorthash>/
├── plan.md
├── 0001-<task-slug>.md
├── 0002-<task-slug>.md
└── ...
```

- **slug:** kebab-case summary of the request (e.g. `auth-rewrite`).
- **shorthash:** 4 hex chars from `openssl rand -hex 2` or similar; disambiguates concurrent requests.
- **Task files:** `NNNN-<task-slug>.md` where `NNNN` is zero-padded sequence starting at `0001`.

## Task frontmatter

| Field | Required | Values / notes |
| --- | --- | --- |
| `id` | yes | `"0001"` — matches filename prefix |
| `title` | yes | Short imperative title |
| `status` | yes | `pending` \| `in_progress` \| `blocked` \| `done` \| `cancelled` |
| `depends_on` | yes | List of task ids, e.g. `["0001"]` or `[]` |
| `acceptance` | yes | Bullet list of verifiable outcomes |
| `github` | yes | `repo: null`, `issue: null` until adapter populates |

## Status transitions

```
pending → in_progress → done
pending → in_progress → blocked → pending | in_progress
any → cancelled
```

- Only one task should be `in_progress` at a time unless explicitly parallelized in `plan.md`.
- Set `blocked` with a **Blocker** section in the task body; do not proceed the execute loop until resolved.
- `done` requires all `acceptance` items verified and DoD satisfied.

## plan.md sections

1. **Goal** — what done means (1–2 sentences).
2. **Non-goals** — explicit out-of-scope items.
3. **Decision (ADR-lite)** — chosen approach, rejected alternatives, consequences.
4. **Risks and rollback** — failure modes and revert path.
5. **Tasks overview** — ordered list linking to `NNNN-*.md` files.
6. Optional: **Threat model**, **Data migration**, **Deprecation** — when applicable per [practices.md](practices.md).

## Naming rules

- Slugs: lowercase letters, numbers, hyphens only.
- Task slugs: verb-led when possible (`add-auth-module`, `integration-tests`).
