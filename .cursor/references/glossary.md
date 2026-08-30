# Glossary

Canonical domain terms for Elegant Git. Unqualified **repository** means a tracked entry in shared memory; say **git repository** for any clone.

### Argument resolution

Policy that fills required and optional command inputs before business logic runs (`argspec.Resolve`).

Notes: when any required input is missing in interactive mode, optional inputs are reviewed with edit-or-accept even if already passed on the CLI.

### Git

CLI object for installation-wide setup, inspection, and doctor—not “run git” or native `git status`.

### Interactive mode

CLI mode where the prompter accepts input; default when stdin is a TTY and no non-interactive override applies.

Aliases: TTY mode

Avoid: conflating with workflow prompts inside configure/doctor commands

### Non-interactive mode

CLI mode where required missing inputs fail without prompts; auto-detected from non-TTY stdin, `CI`, env, or `--non-interactive`.

Notes: on a TTY, `--interactive` or `ELEGANT_GIT_INTERACTIVE=1` forces prompts; non-TTY stdin is always non-interactive.

### Optional input

Positional argument declared non-required in an arg spec; reviewed in interactive mode only when at least one required input was missing.

### Required input

Positional argument that must be set before the command’s main logic runs; missing values error in non-interactive mode.

### Hook

Ahead/after scripts run around elegant-git commands, unless `--no-workflows` is set.

Notes: repo-tracked under `<repo>/.config/elegant-git/hooks/<command>-<action>-{ahead,after}`; personal under `<repo>/.git/.config/elegant-git/hooks/...`; CLI object actions are `list`, `new`, `edit`.

Avoid: `.git/hooks` (git’s native hook mechanism)

### Memory

CLI object for read-only inspection of shared memory and its contents.

Avoid: conflating with shared/repo memory stores or test types `MemoryRunner` / `MemoryRepo`

### Migrate

Automatic rewrite of Elegant Git-owned state (memory schema, acquired marker, legacy branch keys, dead `elegant …` aliases). Assisted layout changes are doctor findings.

Notes: integer `migrations_version` generation in shared memory gates global Auto steps; local leftovers are scanned whenever cwd is a git repository; hidden `git migrate` / `repo migrate` / `hook migrate` shims remain until DEP-016 `remove_after` (`--dry-run` reports only).

Avoid: adding a user-facing migrate command; using “migrate” for doctor repairs

### Workspace

Reusable git identity bundle stored in shared memory (`user_name`, `user_email`, and optional signing, editor, gpg, and `namespaces` fields).

Notes: stable id is a UUID map key (`workspace_id` in JSON); `name` is the display label; CLI object `workspace` manages list/new/link/edit/delete/fetch/doctor (bare `workspace` detects context then asks); `namespaces` holds confirmed `<domain>/<owner>` values used to suggest a workspace on `repo clone`. `list` treats `current` and `all` as selectors, not display names.

Aliases: profile (deprecated), identity profile

### Namespace

Normalized `<domain>/<owner...>` prefix of a git remote URL, stored on a workspace for clone-time suggestion.

Notes: derived from any origin format (HTTPS, SSH, SCP-like, `git://`); owner is every path segment except the last (the repository name); the same namespace may appear on several workspaces.

Avoid: “source” (collides with `internal/cli/sources` completion providers and repo-memory `BranchSources`); conflating with a git remote named `origin`

### Release

CLI object for tagging and release notes on the default development branch.

### Repo

CLI object for repository lifecycle and maintenance commands (`configure`, `clone`, `init`, `list`, `sync`, `prune`, `doctor`).

Avoid: using “repo” when you mean the **Repository** entity in shared memory

### Repo memory

Per-repository elegant-git store at `<git-dir>/elegant-git/state.json` (`workspace_id`, `default_branch`, `protected_branches`).

Aliases: per-repo memory (deprecated wording)

Notes: override `ELEGANT_GIT_REPO_STATE_FILE`; branch settings live here, not in `git config`

Avoid: “git config” for default or protected branches

### Repository

A git working copy tracked in shared memory (`workspace_id`, `current_path`, `path_history`, optional `origin_url`).

Notes: stable id is UUID (`elegant-git.repo-id` in local git config, `repo_id` in repo memory); `name` is the display label; entries live in the `Repositories` map in shared memory (listed via `memory repositories`).
Avoid: “managed repository”; unqualified “repository” when meaning any clone

### Shared memory

User-level elegant-git store at `$XDG_CONFIG_HOME/elegant-git/state.json` (workspaces and the repository registry).

Aliases: shared state

Notes: override `ELEGANT_GIT_STATE_FILE`

Avoid: “memory” alone; “config file” when meaning git config

### Sync

Action name shared by two CLI objects with different meaning.

Notes: `repo sync` re-applies linked workspace settings to one or all tracked repositories; `work sync` actualizes the current branch with upstream commits.

### Work

CLI object for day-to-day branch workflow (start, save, amend, list, polish, sync, push, track, accept).
