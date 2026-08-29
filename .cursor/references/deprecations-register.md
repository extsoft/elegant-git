# Deprecations register

| id | introduced | surface | replacement | migrate | remove_after | status |
| --- | --- | --- | --- | --- | --- | --- |
| DEP-001 | 2026-05-22 | command names: *-work, *-repository, acquire-git, *-workflow, show-commands | `git elegant <object> <action>` | `git elegant git migrate` / `git elegant repo migrate` | 2027-01-31 | active |
| DEP-002 | 2026-05-22 | personal hooks: `.git/.workflows/<legacy>-{ahead,after}` | `.git/.config/elegant-git/hooks/<command>-<action>-{ahead,after}` | `git elegant repo migrate` | 2027-01-31 | active |
| DEP-003 | 2026-05-22 | common hooks: `.workflows/<legacy>-{ahead,after}` | `.config/elegant-git/hooks/<command>-<action>-{ahead,after}` | `git elegant hook migrate` | 2027-01-31 | active |
| DEP-004 | 2026-05-22 | command: show-commands | `git elegant completion <shell>` | install new completion script | 2027-01-31 | active |
| DEP-005 | 2026-05-22 | files: completions/_git-elegant, completions/git-elegant.bash | `git elegant completion <shell>` | regenerate completion | 2027-01-31 | active |
| DEP-006 | 2026-05-22 | config keys: elegant.<legacy>-stash, elegant.<legacy>-current-branch | per-repo `.git/elegant-git/commands.json` | — | 2027-01-31 | removed |
| DEP-007 | 2026-05-22 | hook new argument: legacy command name | canonical id (e.g. work.start) | use `git elegant hook new work.start ...` | 2027-01-31 | active |
| DEP-008 | 2026-05-22 | hook dispatch: acquire-repository, obtain-work from clone/init/accept | repo.configure, work.track + legacy hooks | migrate hooks | 2027-01-31 | active |
| DEP-009 | 2026-05-22 | config: elegant-git.acquired=true | shared memory `acquired_version` | `git elegant git migrate` / `git elegant repo migrate` | 2027-01-31 | active |
| DEP-010 | 2026-05-24 | config: elegant-git.default-branch, elegant-git.protected-branches in `.git/config` | per-repo memory `.git/elegant-git/state.json` | `git elegant repo configure` / `git elegant repo migrate` | 2027-01-31 | active |
| DEP-011 | 2026-05-24 | commands: `repo list`, `hook list` | `memory repositories`, `hook status` | use replacement commands | 2027-01-31 | active |
| DEP-012 | 2026-08-28 | commands: `profile *`, `memory profiles`; shared/per-repo memory keys `profiles`, `profile_id` | `workspace *`, `memory workspaces`, `workspaces`, `workspace_id` | `git elegant repo migrate` / `git elegant git migrate` | 2027-06-30 | active |
| DEP-013 | 2026-08-28 | command: `workspace create` | `workspace new` | use `git elegant workspace new` | 2027-06-30 | active |
| DEP-014 | 2026-08-28 | Go import path: `github.com/bees-hive/elegant-git/...` | `github.com/extsoft/elegant-git/...` | update imports; reinstall with `go install github.com/extsoft/elegant-git/cmd/git-elegant@latest` | 2027-06-30 | removed |
