# Commands

`git elegant` uses an object-first CLI. Legacy flat names (e.g. `git elegant start-work`) remain available as hidden aliases.

## Objects

### memory

| Command | Description |
| --- | --- |
| `memory status` | Summarizes shared memory paths, profile/repository counts, and current repository hint. |
| `memory profiles` | Lists profiles (`--format=table\|json`). `memory profiles <name>` shows full details for one profile. |
| `memory repositories` | Lists managed repositories (`name`, profile, path). `memory repositories <name-or-path>` shows full details for one. |

### profile

| Command | Description |
| --- | --- |
| `profile status` | Shows the linked profile for the current repository (when inside a git work tree). |
| `profile create` | Creates a profile (edit-or-accept for all fields; suggests from local/global git config; offers apply to current repo). |
| `profile edit <name>` | Edits a profile transactionally: plan changes and repo targets, summary + single confirm, then commit to shared memory and selected repos. |
| `profile delete <name>` | Deletes a profile only when no repositories are linked. |

### git

| Command | Description |
| --- | --- |
| `git configure` | Configures your Git installation (global); offers to create a profile from global values. |
| `git status` | Shows global Git installation and shared memory state (not the same as native `git status`). |
| `git migrate` | Migrates global aliases and `elegant-git.acquired`. |

### repo

| Command | Description |
| --- | --- |
| `repo configure` | Configures the current local repository (`--profile <name>`). |
| `repo clone` | Clones a remote repository and configures it. |
| `repo init` | Initializes a new repository and configures it. |
| `repo status` | Shows per-repo memory, registry linkage, branch settings, and local git identity for the current repository. |
| `repo sync` | Re-applies profile settings (`--all` for every managed repo; `[y/n/A/S]` per repo). |
| `repo relocate <path>` | Updates the managed path for the current repository. |
| `repo prune` | Removes useless local branches. |
| `repo migrate` | Migrates local aliases, hooks, pipe keys, and elegant-git settings into memory. |

### hook

Hooks live under `.config/elegant-git/hooks/<command>-<action>-{ahead,after}` (repo) and `.git/.config/elegant-git/hooks/...` (personal).

| Command | Description |
| --- | --- |
| `hook status` | Lists hook file paths. |
| `hook new` | Creates a new hook file. |
| `hook edit` | Opens a hook file in your editor. |
| `hook migrate` | Moves `.workflows/*` to the new layout. |

### work

| Command | Description |
| --- | --- |
| `work start` | Creates a new branch. |
| `work save` | Commits current modifications. |
| `work amend` | Amends the last commit. |
| `work list` | Prints HEAD state. |
| `work polish` | Rebases HEAD interactively. |
| `work sync` | Actualizes the branch with upstream commits. |
| `work push` | Publishes HEAD to a remote repository. |
| `work track` | Checks out a remote-tracking branch. |
| `work accept` | Merges a branch into the default development branch. |

### release

| Command | Description |
| --- | --- |
| `release new` | Tags and releases the default branch. |
| `release notes` | Prints a release log between two refs. |

## Top-level

| Command | Description |
| --- | --- |
| `version` | Prints the program version. |
| `completion` | Generates shell completion (`bash`, `zsh`, `fish`, `powershell`). |

## Flags

- `--no-workflows` — disables ahead/after hooks
- `--non-interactive` — disables prompts; fails when required input is missing (also `ELEGANT_GIT_NON_INTERACTIVE=1`)
- `--version` — prints version

## Shell completion

```bash
git elegant completion bash > ~/.local/share/bash-completion/completions/git-elegant
```
