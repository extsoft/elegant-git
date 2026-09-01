---
layout: default
title: eg repo configure
parent: '"repo" guide'
nav_order: 1
permalink: /repo/configure/
---

# eg repo configure

`repo configure` configures the current local Git repository. It links the repository to a
[workspace](06-00-workspace.md), writes user identity into `.git/config`, and records the default
and protected branches in [per-repo memory](reference/memory.md).

```bash
eg repo configure work
```

The workspace name is required. In interactive mode it is asked if you leave it out. The picker
includes existing workspaces, `[Create new]`, and — when the repository already has `user.name`
and `user.email` — `[Use settings from this repository]`. Creating a workspace here asks for the
same fields as [`workspace new`](06-02-workspace-new.md). Non-interactive mode cannot create a
workspace this way; name one that already exists.

## How a workspace is chosen when you omit the name

[`repo clone`](07-02-repo-clone.md) may call configure with no workspace. Then Elegant Git
suggests one from the origin [namespace](06-00-workspace.md#how-a-namespace-is-remembered):

- a unique match asks to use that workspace (default yes); in non-interactive mode it is used
- several matches narrow the picker to those workspaces
- no match, or declining the suggestion, offers the full picker
- in non-interactive mode, anything other than a unique match continues with no workspace
  assigned

## What it writes

If the repository already has an `elegant-git.repo-id` and you have moved it, the registry path is
updated first. Then local leftovers from an older install are cleaned up.

When a workspace is assigned, identity is applied in full — no per-key prompts — the registry is
upserted, and you are asked for the default development branch and the protected branches. A new
origin namespace is offered to remember, silently in non-interactive mode.

Signature setup runs only when **no** workspace is assigned. Once a workspace is linked,
`repo configure` never writes local signing keys, even if the workspace has an empty
`signing_key`. Change signing with [`workspace edit`](06-04-workspace-edit.md). When the
signature step does run, available gpg keys are listed and you choose one, or skip.

If this machine is already acquired by [`self configure`](05-01-self-configure.md), local Git
aliases and standards are not rewritten — those come from the installation. The command still
removes redundant local `elegant …` aliases. Without a global acquired marker it applies the full
local standards and alias set. The exact keys are on the
[configuration](reference/configuration.md) page.

When it finishes it prints that repository configuration is complete.
