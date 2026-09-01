---
layout: default
title: eg workspace edit
parent: '"workspace" guide'
nav_order: 4
permalink: /workspace/edit/
---

# eg workspace edit

`workspace edit` edits a workspace and optionally applies it to linked repos.

```bash
eg workspace edit work
```

The name is required; in interactive mode it is asked if you leave it out.

The command is transactional. It collects your field edits and your per-repository apply
decisions, shows one summary, confirms once, and only then saves shared memory and touches the
selected repositories.

## What you are asked

In interactive mode you are offered each identity field — `user.name`, `user.email`, signing key,
gpg program, editor — with the current value as the suggestion. Non-interactive mode keeps the
current values and does not prompt.

Then it asks which linked repositories to apply to. The current repository, if linked, is a
yes/no (default yes). The others are a batch choice: yes, apply-all, no, or skip-all. In
non-interactive mode every linked repository is applied.

The summary lists field diffs (`before -> after`, empty as `(unset)`) and the repositories it will
apply, skip, or cannot reach. Interactive mode confirms with `Proceed?` (default yes). Decline
and nothing is written.

Per-repository apply failures are reported; they do not fail the command. A missing path is
listed as missing and suggests `eg repo configure` in that repository.
