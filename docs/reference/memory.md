---
layout: default
title: Memory
parent: Reference
nav_order: 3
---

# Memory

Not everything Elegant Git knows fits into `git config`. Workspaces are shared between
repositories, the repository registry has to survive a `mv`, and the pipes need a scratch pad. So,
Elegant Git keeps its own state in two JSON files — **shared memory** for everything user-wide and
**per-repo memory** for everything that belongs to one repository. Anything that does land in
`git config` is on the [configuration](configuration.md) page instead.

You never have to open these files. `eg memory status` summarizes them,
`eg memory workspaces` and `eg memory repositories` list their contents, and
`eg repo status` shows what the current repository resolved to.

## Shared memory

The file is `state.json` inside the user configuration directory your OS defines:

| OS | Path |
| --- | --- |
| macOS | `~/Library/Application Support/elegant-git/state.json` |
| Linux | `$XDG_CONFIG_HOME/elegant-git/state.json`, or `~/.config/elegant-git/state.json` when that variable is unset |
| Windows | `%AppData%\elegant-git\state.json` |

Set `ELEGANT_GIT_STATE_FILE` to an absolute path to override it — handy for tests and for keeping
work and personal setups apart.

The document holds four things:

- `schema_version` — currently `2`
- `acquired_version` — the Elegant Git version that applied the global configuration; its presence
  is what [`eg git configure`](configuration.md#approach) checks for
- `workspaces` — each keyed by id, with `name`, `user_name`, `user_email`, and the optional
  `signing_key`, `editor`, `gpg_program`, `namespaces`, and `linked_repos`
- `repositories` — the registry of managed repositories, each keyed by id, with `name`,
  `workspace_id`, `current_path`, and the optional `path_history` and `origin_url`

The id of a repository is also written into its `.git/config` as `elegant-git.repo-id` (a UUIDv7),
which is how a repository recognizes itself after you move it. `path_history` keeps the paths it
used to live at.

## Per-repo memory

The file is `<repo>/.git/elegant-git/state.json`, and `ELEGANT_GIT_REPO_STATE_FILE` overrides it.
It holds `schema_version`, `repo_id`, `workspace_id`, `default_branch`, `protected_branches`, and
`branch_sources` — the last one records which branch each branch was started from, so `work` can
tell your unique commits from the ones you inherited.

When nothing is recorded yet, Elegant Git falls back to `main` for both the default development
branch and the protected branches, and to `origin` for the remote.

Next to it sits `<repo>/.git/elegant-git/commands.json`, the per-repo command memory. That is where
the [pipes](../guides/pipes.md) park the stash message and the branch name they have to
restore, which is why an interrupted command can be rerun without losing your work.

## Schema

Both files carry a `schema_version`. When Elegant Git rewrites its own files, a backup is left
next to the original. How that rewrite is classified — automatic versus a `doctor` finding — is
on the [migrations](migrations.md) page.

## How memory gets filled

`eg repo configure` is what links a repository to a workspace. It writes `user.name` and
`user.email` into `.git/config`, and prompts before applying the optional workspace fields
(`signing_key`, `editor`, `gpg_program`). Values that already match the workspace are skipped
without prompts, and every `git config` set or unset is printed before it runs. Branch settings
that belong to Elegant Git live only in per-repo memory. When the origin URL yields a
[namespace](../guides/workspaces.md#how-a-namespace-is-remembered) not yet on the workspace,
`repo configure` — and `workspace new` when it applies to the current repository — asks to remember
it, recording it silently in non-interactive mode.

Workspaces can be created three ways:

- `git configure` offers to create one from your global values after the global setup
- `repo configure` shows a picker with the existing workspaces, `[Create new]`, and
  `[Use settings from this repository]` when the repository already has `user.name` and `user.email`
- `workspace new` creates one by hand, suggesting values from the local and then the global
  `git config`

`repo clone` may omit the workspace argument altogether and suggest one from a matching namespace.
And `workspace edit` is transactional — it collects the field edits and the per-repository apply
decisions, shows one summary, confirms once, and only then writes shared memory and touches the
selected repositories.
