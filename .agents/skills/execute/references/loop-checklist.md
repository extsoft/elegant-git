# Execute loop checklist

**Scope:** one task per agent run unless the user asked to continue through multiple tasks.

Copy into your working notes; check before marking `done`.

## Per task

- [ ] `depends_on` tasks are `done`
- [ ] `status` set to `in_progress`; `github-sync.sh` run
- [ ] **Steps** followed; diff is surgical (only task scope)
- [ ] Each `acceptance` item verified (test output or explicit note in task file)
- [ ] [definition-of-done.md](../../../references/definition-of-done.md) satisfied
- [ ] `status` set to `done`; `github-sync.sh` run
- [ ] One atomic commit with task id in message (when user requested a commit)

## Verification

- [ ] **`mise run fix`** (and **`mise run test`** when applicable) from repo root — not `mise run check` unless verifying CI parity explicitly
- [ ] Commands from `plan.md` or task **Steps** run successfully (record command + outcome)
- [ ] Project CI or test suite green per DoD
- [ ] User-visible docs updated when behavior changed

## After this task

- [ ] Stopped after one task (did not start the next `pending` task unless user requested multi-task run)
- [ ] Handoff names next task id or states the folder is complete
