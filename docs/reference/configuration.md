---
layout: default
title: Configuration
parent: Reference
nav_order: 2
permalink: /reference/configuration/
---

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
`elegant …` aliases and a stale local `elegant-git.acquired` marker when present. Run
`git elegant git configure` once on each machine where you use Elegant Git globally.

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
Everything that does **not** live in `git config` — workspaces, the repository registry, protected
branches, the default development branch — is described on the [memory](memory.md) page.

## Level: Basics

The basics configuration sets the mandatory options for the correct user-focused operation of Git and
Elegant Git. During the configuration, you will be asked to provide appropriate values. Furthermore,
if you run `repo configure`, it proposes defaults that are set by `git configure`. The basics includes:

1. setting your full name usign `user.name` [`b`]
2. setting your email usign `user.email` [`b`]
3. setting a default editor using `core.editor` [`b`]
4. setting protected branches (stored in [per-repo memory](memory.md), not `git config`) [`l`]
5. setting a default development branch (stored in [per-repo memory](memory.md), not `git config`)
[`l`]

## Level: Standards

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

## Level: Aliases

In order to make an Elegant Git command look like a native Git command, the historical flat command
names are registered as Git aliases — `git save-work` reaches `git elegant work save`. This should
significantly improve user experience.

The configuration is a call of `git config "alias.<flat-name>" "elegant <object> <action>"` [`i`]
for each of those names, `show-commands` excepted. Note that only the flat names get an alias; the
object-first form is always spelled in full as `git elegant work save`. The flat names themselves
are deprecated, so treat the aliases as compatibility rather than as the way to drive the tool.

When global configuration is applied, aliases are written globally only; `repo configure` and
`repo migrate` remove redundant local elegant aliases instead.

## Level: Signature

This configuration aims to say Git how to sign commits, tags, and other objects you create. It runs
during `repo configure` **only when no workspace is assigned**. Once a workspace is assigned,
`repo configure` never runs local signature setup — even if the workspace has an empty
`signing_key`. Signature on a workspace (`signing_key`, `gpg_program`) is applied from the
workspace instead; set or change it with `workspace new` / `workspace edit`. When the signature
step runs (no workspace), all available signing keys are shown. Then you choose the key that will
be used to make signatures. If the key is provided, the configuration triggers, otherwise it does
not apply. The signing configuration consists of

1. setting `user.signingkey` [`l`] to a provided value
2. setting `gpg.program` [`l`] to a full path of `gpg` program
3. setting `commit.gpgsign` [`l`] to `true`
4. setting `tag.forceSignAnnotated` [`l`] to `true`
5. setting `tag.gpgSign` [`l`] to `true`

For now, only `gpg` is supported. If you need other tools, please
[create a new feature request](https://github.com/extsoft/elegant-git/issues/new/choose).

## Custom keys

The Elegant Git configuration keys:

- `elegant-git.repo-id` identifies the repository in shared memory (UUIDv7). Set by `repo configure`.
- `acquired_version` in shared memory defines whether global configuration was applied (see
[approach](#approach) for the details). Legacy `elegant-git.acquired` in git config is migrated
by `git configure` / `git migrate`.

Protected branches and the default development branch are read from [per-repo memory](memory.md)
(legacy values in `elegant-git.protected-branches` / `elegant-git.default-branch` are migrated by
`repo configure` or `repo migrate`). The "protected" branch rules and default-branch semantics are
unchanged; only storage moved.
