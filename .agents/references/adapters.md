# Adapters

External trackers sync from `.tasks/` task files. Today only the GitHub stub exists.

## GitHub (stub)

**Script:** `.agents/skills/tasks/execute/scripts/github-sync.sh`

**Input:** Path to a task file (e.g. `.tasks/auth-rewrite-a1b2/0001-scaffold-app.md`).

**Behavior (future):**

1. Read task frontmatter `github.repo` and `github.issue`.
2. If `issue` is null and `status` is `pending` or `in_progress`, create issue via `gh issue create`; write issue number back to frontmatter.
3. If `issue` is set, update title/body/labels from task file; close when `status: done`.
4. Idempotency: same task file + same issue number → update, never duplicate.

**Behavior (today):** Print `github-sync: adapter not configured; skipping` and exit 0.

**Frontmatter contract:**

```yaml
github:
  repo: null    # e.g. "org/repo" when configured
  issue: null   # integer issue number when synced
```

**Environment (future):** `GITHUB_REPO`, `GH_TOKEN` or `gh auth` session.

## Adding adapters

1. Document contract in this file.
2. Add script under `.agents/skills/tasks/execute/scripts/` or a dedicated adapter skill.
3. Call from `execute` skill after each status transition.
4. Keep task files the source of truth; adapters are projections, not masters.
