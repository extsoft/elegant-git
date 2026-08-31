# Deprecations register

| id | introduced | surface | replacement | migrate | remove_after | status |
| --- | --- | --- | --- | --- | --- | --- |
| DEP-001 | 2026-05-22 | command names: *-work, *-repository, acquire-git, *-workflow, show-commands | `eg <object> <action>` | automatic alias rewrite | 2027-01-31 | active |
| DEP-002 | 2026-05-22 | personal hooks: `.git/.workflows/<legacy>-{ahead,after}` | `.git/.config/elegant-git/hooks/<command>-<action>-{ahead,after}` | `eg repo doctor` | 2027-01-31 | active |
| DEP-003 | 2026-05-22 | common hooks: `.workflows/<legacy>-{ahead,after}` | `.config/elegant-git/hooks/<command>-<action>-{ahead,after}` | `eg repo doctor` | 2027-01-31 | active |
| DEP-004 | 2026-05-22 | command: show-commands | `eg completion <shell>` | install new completion script | 2027-01-31 | active |
| DEP-005 | 2026-05-22 | files: completions/_git-elegant, completions/git-elegant.bash | `eg completion <shell>` | `eg self doctor` / regenerate completion | 2027-01-31 | active |
| DEP-006 | 2026-05-22 | config keys: elegant.<legacy>-stash, elegant.<legacy>-current-branch | per-repo `.git/elegant-git/commands.json` | — | 2027-01-31 | removed |
| DEP-007 | 2026-05-22 | hook new argument: legacy command name | canonical id (e.g. work.start) | use `eg hook new work.start ...` | 2027-01-31 | active |
| DEP-008 | 2026-05-22 | hook dispatch: acquire-repository, obtain-work from clone/init/accept | repo.configure, work.track + legacy hooks | `eg repo doctor` | 2027-01-31 | active |
| DEP-009 | 2026-05-22 | config: elegant-git.acquired=true | shared memory `acquired_version` | automatic | 2027-01-31 | active |
| DEP-010 | 2026-05-24 | config: elegant-git.default-branch, elegant-git.protected-branches in `.git/config` | per-repo memory `.git/elegant-git/state.json` | automatic | 2027-01-31 | active |
| DEP-011 | 2026-05-24 | former catalog command `repo list` | `repo list all` (was `memory repositories`) | use `eg repo list all` | 2027-01-31 | active |
| DEP-012 | 2026-08-28 | commands: `profile *`, `memory profiles`; shared/per-repo memory keys `profiles`, `profile_id` | `workspace *`, `workspace list all`, `workspaces`, `workspace_id` | automatic (memory keys); use `workspace` (commands) | 2027-06-30 | active |
| DEP-013 | 2026-08-28 | command: `workspace create` | `workspace new` | use `eg workspace new` | 2027-06-30 | active |
| DEP-014 | 2026-08-28 | Go import path: `github.com/bees-hive/elegant-git/...` | `github.com/extsoft/elegant-git/...` | update imports; reinstall with `go install github.com/extsoft/elegant-git/cmd/eg@latest` | 2027-06-30 | removed |
| DEP-015 | 2026-08-29 | binary name `git-elegant` (`git-<subcommand>` dispatch), alias values `elegant …`, go install `.../cmd/git-elegant` | binary `eg`; `git elegant …` via `alias.elegant = "!eg"`; alias values `!eg …`; go install `.../cmd/eg` | automatic (aliases); `eg self doctor` (leftover binary / completions) | 2027-06-30 | active |
| DEP-016 | 2026-08-29 | commands: `git migrate`, `repo migrate`, `hook migrate` | automatic migrations + `eg self doctor` / `eg repo doctor` | hidden shims remain | 2027-06-30 | active |
| DEP-017 | 2026-08-30 | command: `workspace status` | `workspace list current` | use `eg workspace list current` | 2027-06-30 | active |
| DEP-018 | 2026-08-30 | commands: `memory status`, `git status`, `repo status`, `hook status` | `memory list`, `git list`, `repo list`, `hook list` | use replacement commands | 2027-06-30 | active |
| DEP-019 | 2026-08-30 | objects: `git *`, `memory *` | `self *`, `workspace list all`, `repo list all` | `eg repo doctor` (hook files) | 2027-06-30 | active |
