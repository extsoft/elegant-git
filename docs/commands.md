# Commands

`git elegant` uses an object-first CLI. Legacy flat names (e.g. `git elegant start-work`) remain available as hidden aliases.

## Objects

### git

| Command | Description |
| --- | --- |
| `git configure` | Configures your Git installation (global). |
| `git migrate` | Migrates global aliases and `elegant-git.acquired`. |

### repo

| Command | Description |
| --- | --- |
| `repo configure` | Configures the current local repository. |
| `repo clone` | Clones a remote repository and configures it. |
| `repo init` | Initializes a new repository and configures it. |
| `repo prune` | Removes useless local branches. |
| `repo migrate` | Migrates local aliases, personal hooks, and pipe keys. |

### hook

Hooks live under `.config/elegant-git/hooks/<command>-<action>-{ahead,after}` (repo) and `.git/.config/elegant-git/hooks/...` (personal).

| Command | Description |
| --- | --- |
| `hook list` | Lists hook file paths. |
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
- `--version` — prints version

## Shell completion

```bash
git elegant completion bash > ~/.local/share/bash-completion/completions/git-elegant
```
