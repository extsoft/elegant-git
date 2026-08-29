---
layout: default
title: Commands
parent: Reference
nav_order: 1
---

# Commands

`git elegant` uses an object-first CLI. Legacy flat names (e.g. `git elegant start-work`) remain available as hidden aliases.

## Argument handling

Every command follows the same policy:

1. When all **required** arguments are present, the command runs with no argument prompts.
2. In **interactive** mode, missing required arguments are asked as one-line questions; then every **optional** argument is offered (including values already passed on the CLI, unless all required arguments were given).
3. In **non-interactive** mode, missing required arguments cause an error listing what is missing.

A question is one line when there are 0–1 options: `<prompt> [<suggested>] (<action on enter>):`. Parts are omitted when they do not apply. With 2 or more options, use the [picker](interaction.md#picker).

- Required with a suggestion: `Git user.name [Alice] (press enter to accept):`
- Optional with a suggestion: `Signing key [ABC123] (press enter to accept):`
- Optional with no suggestion: `Signing key (press enter to skip):`
- Required with no suggestion: `Workspace name:` (empty Enter asks again)

Interactive mode is the default when stdin is a TTY. Non-TTY stdin is always non-interactive. On a TTY, `--non-interactive` / `ELEGANT_GIT_NON_INTERACTIVE=1` or `CI` disable prompts; `--interactive` / `ELEGANT_GIT_INTERACTIVE=1` forces prompts (overrides `CI` and `--non-interactive`).

Workflow prompts inside commands (for example `git configure`, `repo configure`, uncommitted changes during `work start`) use the same question line and honor `--interactive` / `--non-interactive`.

## Objects

### memory

| Command | Description |
| --- | --- |
| `memory status` | Summarizes shared memory paths, workspace/repository counts, and current repository hint. |
| `memory workspaces` | Lists workspaces (`--format=table\|json`). `memory workspaces <name>` shows full details for one workspace. |
| `memory repositories` | Lists managed repositories (`name`, workspace, path). `memory repositories <name-or-path>` shows full details for one. |

### workspace

| Command | Description |
| --- | --- |
| `workspace` | With no action: prints detection checks, then asks What now (interactive). Non-interactive mode requires an action. |
| `workspace list [name]` | Lists workspaces (`--format=table\|json`). With a name, shows full details for that workspace. |
| `workspace new <name> <user-name> <user-email> [<signing-key>] [<gpg-program>] [<editor>]` | Creates a workspace. Required arguments are prompted when missing in interactive mode. Optionally applies to the current repository on confirmation and offers to remember that repository's namespace. |
| `workspace link <name>` | Links the current repository to an existing workspace (prompts for name when omitted in interactive mode). Applies identity and may capture the origin namespace. |
| `workspace edit <name>` | Edits a workspace transactionally: plan changes and repo targets, summary + single confirm, then commit to shared memory and selected repos. |
| `workspace delete <name> [--yes]` | Deletes a workspace after explaining what will happen and asking for confirmation. Linked repositories stay in the registry with `workspace_id` cleared; their git config and files are untouched. `--yes` skips the prompt; non-interactive mode requires `--yes`. |
| `workspace status` | Shows the linked workspace for the current repository (when inside a git work tree). |
| `workspace fetch [name]` | Runs `git fetch --all --tags --prune` in every repository linked to the workspace (prunes stale remote-tracking branches). Exits non-zero if any repository fails. On a TTY: progress bar, processed-repo list, ephemeral logs for the current fetch. When name is omitted, uses the workspace linked to the current repository. |
| `workspace doctor [name]` | Diagnoses shared memory and repository link problems for one workspace and suggests a repair for each. Interactive mode confirms yes/no repairs and asks once for branching repairs; non-interactive mode only reports and exits non-zero when issues are found. When name is omitted, uses the linked workspace or asks interactively. |

### git

| Command | Description |
| --- | --- |
| `git configure` | Configures your Git installation (global); offers to create a workspace from global values. |
| `git status` | Shows global Git installation and shared memory state (not the same as native `git status`). |
| `git migrate` | Migrates global aliases and legacy `elegant-git.acquired` into shared memory. |

### repo

| Command | Description |
| --- | --- |
| `repo configure <workspace>` | Configures the current local repository. Signature setup is skipped whenever a workspace is assigned — including when that workspace has no signing key (change signing via `workspace edit`). |
| `repo clone <repository> [<workspace>] [<directory>]` | Clones a remote repository and configures it. When workspace is omitted, suggests one from the repository namespace (`<domain>/<owner>`). |
| `repo init <workspace>` | Initializes a new repository and configures it. |
| `repo status` | Shows per-repo memory, registry linkage, branch settings, and local git identity for the current repository. |
| `repo sync` | Re-applies workspace settings (`--all` for every managed repo; `[yes/no/all/skip]` per repo). |
| `repo doctor` | Diagnoses registry, identity, and legacy configuration problems for the current repository and suggests a repair for each. Interactive mode confirms yes/no repairs; non-interactive mode only reports and exits non-zero when issues remain. |
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
