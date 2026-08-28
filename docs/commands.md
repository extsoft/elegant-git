# Commands

`git elegant` uses an object-first CLI. Legacy flat names (e.g. `git elegant start-work`) remain available as hidden aliases.

## Argument handling

Every command follows the same policy:

1. When all **required** arguments are present, the command runs with no argument prompts.
2. In **interactive** mode, missing required arguments are asked as one-line questions; then every **optional** argument is offered (including values already passed on the CLI, unless all required arguments were given).
3. In **non-interactive** mode, missing required arguments cause an error listing what is missing.

A question is one line when there are 0–1 options: `<prompt> [<suggested>] (<action on enter>):`. Parts are omitted when they do not apply. With 2 or more options, use the [picker](../adr/003-interactive-questions.md#picker).

- Required with a suggestion: `Git user.name [Alice] (press enter to accept):`
- Optional with a suggestion: `Signing key [ABC123]:` (empty Enter leaves it unset)
- Optional with no suggestion: `Signing key (press enter to skip):`
- Required with no suggestion: `Profile name:` (empty Enter asks again)

Interactive mode is the default when stdin is a TTY. Non-TTY stdin is always non-interactive. On a TTY, `--non-interactive` / `ELEGANT_GIT_NON_INTERACTIVE=1` or `CI` disable prompts; `--interactive` / `ELEGANT_GIT_INTERACTIVE=1` forces prompts (overrides `CI` and `--non-interactive`).

Workflow prompts inside commands (for example `git configure`, `repo configure`, uncommitted changes during `work start`) use the same question line and honor `--interactive` / `--non-interactive`.

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
| `profile create <name> <user-name> <user-email> [<signing-key>] [<gpg-program>] [<editor>]` | Creates a profile. Required arguments are prompted when missing in interactive mode. |
| `profile edit <name>` | Edits a profile transactionally: plan changes and repo targets, summary + single confirm, then commit to shared memory and selected repos. |
| `profile delete [name]` | Deletes a profile only when no repositories are linked. |

### git

| Command | Description |
| --- | --- |
| `git configure` | Configures your Git installation (global); offers to create a profile from global values. |
| `git status` | Shows global Git installation and shared memory state (not the same as native `git status`). |
| `git migrate` | Migrates global aliases and legacy `elegant-git.acquired` into shared memory. |

### repo

| Command | Description |
| --- | --- |
| `repo configure <profile>` | Configures the current local repository. |
| `repo clone <repository> <profile> [<directory>]` | Clones a remote repository and configures it. |
| `repo init <profile>` | Initializes a new repository and configures it. |
| `repo status` | Shows per-repo memory, registry linkage, branch settings, and local git identity for the current repository. |
| `repo sync` | Re-applies profile settings (`--all` for every managed repo; `[yes/no/all/skip]` per repo). |
| `repo prune` | Removes useless local branches. |
| `repo migrate` | Migrates local aliases, hooks, and elegant-git settings into memory. |

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
| `release new <name>` | Tags and releases the default branch. |
| `release notes` | Prints a release log between two refs. |

## Top-level

| Command | Description |
| --- | --- |
| `version` | Prints the program version. |
| `completion` | Generates shell completion (`bash`, `zsh`, `fish`, `powershell`). |

## Flags

- `--no-workflows` — disables ahead/after hooks
- `--non-interactive` — disables argument prompts; fails when required input is missing (also `ELEGANT_GIT_NON_INTERACTIVE=1`, `CI`, or non-TTY stdin)
- `--interactive` — on a TTY, forces prompts (overrides `CI` and `--non-interactive`; also `ELEGANT_GIT_INTERACTIVE=1`). Ignored when stdin is not a TTY.
- `--version` — prints version

## Shell completion

```bash
git elegant completion bash > ~/.local/share/bash-completion/completions/git-elegant
```
