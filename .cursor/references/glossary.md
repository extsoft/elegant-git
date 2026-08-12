# Glossary

Canonical domain terms for Elegant Git. Unqualified **repository** means a tracked entry in shared memory; say **git repository** for any clone.

### Argument resolution

Policy that fills required and optional command inputs before business logic runs (`argspec.Resolve`).

Notes: when any required input is missing in interactive mode, optional inputs are reviewed with edit-or-accept even if already passed on the CLI.

### Git

CLI object for installation-wide setup, inspection, and migration—not “run git” or native `git status`.

### Interactive mode

CLI mode where the prompter accepts input; default when stdin is a TTY and no non-interactive override applies.

Aliases: TTY mode

Avoid: conflating with workflow prompts inside configure/migrate commands

### Non-interactive mode

CLI mode where required missing inputs fail without prompts; auto-detected from non-TTY stdin, `CI`, env, or `--non-interactive`.

Notes: override with `--interactive` or `ELEGANT_GIT_INTERACTIVE=1`

### Optional input

Command argument or flag declared non-required in an arg spec; reviewed in interactive mode only when at least one required input was missing.

### Required input

Command argument or flag that must be set before the command’s main logic runs; missing values error in non-interactive mode.

### Hook

Ahead/after scripts run around elegant-git commands, unless `--no-workflows` is set.

Notes: repo-tracked under `<repo>/.config/elegant-git/hooks/<command>-<action>-{ahead,after}`; personal under `<repo>/.git/.config/elegant-git/hooks/...`; CLI object actions are `status`, `new`, `edit`, `migrate`.

Avoid: `.git/hooks` (git’s native hook mechanism)

### Memory

CLI object for read-only inspection of shared memory and its contents.

Avoid: conflating with shared/repo memory stores or test types `MemoryRunner` / `MemoryRepo`

### Migrate

Imports legacy elegant-git data; scope depends on the CLI object prefix.

### Profile

Reusable git identity bundle stored in shared memory (`user_name`, `user_email`, and optional signing, editor, gpg fields).

Notes: stable id is a UUID map key (`profile_id` in JSON); `name` is the display label; CLI object `profile` manages create/edit/delete/status.

Aliases: identity profile

### Release

CLI object for tagging and release notes on the default development branch.

### Repo

CLI object for repository lifecycle and maintenance commands (`configure`, `clone`, `init`, `status`, `sync`, `prune`, `migrate`).

Avoid: using “repo” when you mean the **Repository** entity in shared memory

### Repo memory

Per-repository elegant-git store at `<git-dir>/elegant-git/state.json` (`profile_id`, `default_branch`, `protected_branches`).

Aliases: per-repo memory (deprecated wording)

Notes: override `ELEGANT_GIT_REPO_STATE_FILE`; branch settings live here, not in `git config`

Avoid: “git config” for default or protected branches

### Repository

A git working copy tracked in shared memory (`profile_id`, `current_path`, `path_history`, optional `origin_url`).

Notes: stable id is UUID (`elegant-git.repo-id` in local git config, `repo_id` in repo memory); `name` is the display label; entries live in the `Repositories` map in shared memory (listed via `memory repositories`).
Avoid: “managed repository”; unqualified “repository” when meaning any clone

### Shared memory

User-level elegant-git store at `$XDG_CONFIG_HOME/elegant-git/state.json` (profiles and the repository registry).

Aliases: shared state

Notes: override `ELEGANT_GIT_STATE_FILE`

Avoid: “memory” alone; “config file” when meaning git config

### Sync

Action name shared by two CLI objects with different meaning.

Notes: `repo sync` re-applies linked profile settings to one or all tracked repositories; `work sync` actualizes the current branch with upstream commits.

### Work

CLI object for day-to-day branch workflow (start, save, amend, list, polish, sync, push, track, accept).
