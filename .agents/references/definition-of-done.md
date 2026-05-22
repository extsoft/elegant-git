# Definition of Done

Apply to every task unless `plan.md` explicitly narrows the list.

## Required

- [ ] All `acceptance` criteria in the task frontmatter are verified (test, manual check, or command output captured in task notes).
- [ ] Tests added or updated for behavior changes; existing tests still pass.
- [ ] Lint and format pass via **`mise run fix`** (and **`mise run test`** when Go behavior changed); do not use `mise run check` for local/agent verification—CI uses `check` in GitHub Actions.
- [ ] Docs updated when user-visible behavior changes.
- [ ] Deprecation or contract log updated in the same change if the project maintains one and this task changes a public contract.
- [ ] No secrets, tokens, or PII added to source or logs.
- [ ] Self-review: diff is minimal and traces to the task; error paths considered.

## When applicable

- [ ] Rollback or disable path documented in `plan.md` or task notes.
- [ ] Data migration tested on representative data; down/backfill noted if relevant.
- [ ] Threat-model mitigations implemented for security-sensitive tasks.
- [ ] Performance note added if hot path changed.

## Handoff

- Task `status` set to `done` in frontmatter.
- One atomic commit (or PR) per task with message referencing task id and slug.
- Run `scripts/github-sync.sh` after status change (stub until adapter is configured).
