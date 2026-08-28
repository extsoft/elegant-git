# Approach

Elegant Git aims to standardize how a work environment should be configured. It operates several
levels of configurations (see below) that can be applied to a Git repository (local configuration)
and/or to a Git installation globally (global configuration). So,

- the local configuration applies by running [`git elegant repo configure`](commands.md#repo)
  and configures the current Git repository (workspace linkage, per-repo memory, optional local
  standards and aliases)
- the global configuration applies by running [`git elegant git configure`](commands.md#git)
  and uses `git config --global <key> <value>` for Git installation-wide settings

If you've applied a global configuration (`acquired_version` in shared memory), `repo configure`
does **not** add or rewrite local git aliases or local standards — those come from
`git configure` once per Git installation. It still removes redundant **local**
`elegant …` aliases and a stale local `elegant-git.acquired` marker when present. Run `git elegant git configure` once on each machine where you use Elegant Git globally.

For local-only setups (no global acquired marker), `repo configure` applies the full local
standards and alias set, same as before.

That's why the following markers explain how each particular option will be configured:

- [`b`] - configures for both configurations
- [`l`] - configures only for a local configuration
- [`g`] - configures only for a global configuration
- [`i`] - if a global configuration is applied, this option isn't used in local configuration;
otherwise, uses in local configuration

Also, there are defined [the custom configuration keys](#custom-keys) in addition to
[the standard `git config` options](https://git-scm.com/docs/git-config). These keys are set
automatically during `git configure` or `repo configure`; you do not need to set them manually.

# Level: Basics

The basics configuration sets the mandatory options for the correct user-focused operation of Git and
Elegant Git. During the configuration, you will be asked to provide appropriate values. Furthermore,
if you run `repo configure`, it proposes defaults that are set by `git configure`. The basics includes:

1. setting your full name usign `user.name` [`b`]
2. setting your email usign `user.email` [`b`]
3. setting a default editor using `core.editor` [`b`]
4. setting protected branches (stored in per-repo memory, not `git config`) [`l`]
5. setting a default development branch (stored in per-repo memory, not `git config`) [`l`]

# Level: Standards

The standards configuration adopts the Git setting for painless and user-oriented commands execution
for both Git and Elegant Git. It takes into account OS-specific stuff while configuring specific
options. All Git options in this configuration have the defined values and any changes to them may
affect the designed behavior of the Elegant Git. However, it should not degrade your Git-related
experience. So, the following configuration is applied automatically:

1. `core.commentChar |` [`i`] enables lines in commit messages starting from `#` (`|` prefixes lines that should be ignored)
2. `apply.whitespace fix` [`i`] removes whitespaces when applying a patch
3. `fetch.prune true` [`i`] keeps remote-tracking references up-to-date
4. `fetch.pruneTags false` [`i`] does not remove tags while fetching until you specify it explicitly with
`git fetch --tags`
5. `core.autocrlf input` [`i`] solves issues with line endings on either MacOS/Linux with `input` or
Windows with `true`
6. `pull.rebase true` [`i`] uses `rebase` when `git pull`
7. `rebase.autoStash false` [`i`] don't use `autostash` when `git rebase`
8. `credential.helper osxkeychain` [`i`] configures default credentials storage on MacOS only
9. `acquired_version` in shared memory [`g`] identifies that Elegant Git global configuration is applied (value is the installed version)

# Level: Aliases

In order to make Elegant Git command like a native Git command, each Elegant Git command will have
an appropriate alias like `git elegant save-work` will become `git save-work`. This should
significantly improve user experience.

The configuration is a call of `git config "alias.<command>" "elegant <command>"` [`i`] for each Elegant
Git command. When global configuration is applied, aliases are written globally only;
`repo configure` and `repo migrate` remove redundant local elegant aliases instead.

# Level: Signature

This configuration aims to say Git how to sign commits, tags, and other objects you create. It starts after
all other configurations. In the beginning, all available signing keys will be shown. Then, you need to choose
the key that will be used to make signatures. If the key is provided, the configuration triggers, otherwise,
it does not apply. The signing configuration consists of

1. setting `user.signingkey` [`l`] to a provided value
2. setting `gpg.program` [`l`] to a full path of `gpg` program
3. setting `commit.gpgsign` [`l`] to `true`
4. setting `tag.forceSignAnnotated` [`l`] to `true`
5. setting `tag.gpgSign` [`l`] to `true`

For now, only `gpg` is supported. If you need other tools, please [create a new feature request][https://github.com/bees-hive/elegant-git/issues/new/choose].

# Memory

Elegant Git stores workspaces and repository metadata outside plain `git config`:

- **Shared memory:** `$XDG_CONFIG_HOME/elegant-git/state.json` (override: `ELEGANT_GIT_STATE_FILE`). Holds workspaces (`name`, `user_name`, `user_email`, optional `signing_key`, `editor`, `gpg_program`, `linked_repos`) and a registry of managed repositories (`workspace_id`, `current_path`, `path_history`, `origin_url`). Schema version 2. Older files (v1 `profiles` / `profile_id`) are auto-migrated on load: a `state.json.bak` backup is written first, then the file is rewritten. Restore with `mv state.json.bak state.json` if needed.
- **Per-repo memory:** `<repo>/.git/elegant-git/state.json` (override: `ELEGANT_GIT_REPO_STATE_FILE`). Holds `workspace_id`, `default_branch`, and `protected_branches`.

`repo configure` links the current repository to a workspace, writes `user.name` / `user.email` into `.git/config`, and prompts before applying optional workspace fields (`signing_key`, `editor`, `gpg_program`). Values already matching the workspace are skipped without prompts. Every `git config` set or unset is logged before execution. Elegant-git-specific branch settings live only in per-repo memory; legacy `elegant-git.default-branch` and `elegant-git.protected-branches` keys are removed from `.git/config` after migration (logged unsets).

Workspaces can be created three ways: `git configure` (offer after global setup), `repo configure` (picker: existing workspace, `[Create new]`, or `[Use settings from this repository]` when the repo already has `user.name` and `user.email`), or `workspace create` (manual; suggests from local then global git config).

`workspace edit` is transactional: collect field edits and per-repo apply decisions, show one summary, confirm once, then save shared memory and apply to selected repositories.

# Custom keys

The Elegant Git configuration keys:

- `elegant-git.repo-id` identifies the repository in shared memory (UUIDv7). Set by `repo configure`.
- `acquired_version` in shared memory defines whether global configuration was applied (see
[approach](#approach) for the details). Legacy `elegant-git.acquired` in git config is migrated
by `git configure` / `git migrate`.

Protected branches and the default development branch are read from per-repo memory (legacy values in
`elegant-git.protected-branches` / `elegant-git.default-branch` are migrated by `repo configure` or
`repo migrate`). The "protected" branch rules and default-branch semantics are unchanged; only storage moved.
