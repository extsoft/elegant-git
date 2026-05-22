# Execute loop checklist (judgment only)

Automation handles schema (`validate-folder.sh`), `mise run fix` / `test` (`verify-task.sh`).

Before marking `done`, confirm:

- [ ] `depends_on` tasks are `done`
- [ ] Each `acceptance` item has evidence in the task file or test output
- [ ] [definition-of-done.md](../../../references/definition-of-done.md) satisfied
- [ ] User-visible docs updated when behavior changed
- [ ] Stopped after one task unless multi-task was requested
