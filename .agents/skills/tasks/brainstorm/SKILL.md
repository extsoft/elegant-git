---
name: tasks-brainstorm
description: >
  Explores ambiguous requests and compares approaches before code or task files.
  Use when the user wants to brainstorm, rewrite, migrate, compare stacks, or choose
  architecture—even without saying "brainstorm." Chat-only output. Do not use for
  obvious one-line fixes, when they already chose an approach, or when they want
  /plan or tasks-execute.
disable-model-invocation: true
compatibility: Requires read access to the repo; no network required.
---

# tasks-brainstorm

## Rules

- No writes under `.tasks/`; no code changes.
- Output only in conversation.

If the request touches **security, data migration, performance, or compliance**, read [practices.md](../../../references/practices.md) first.

## Procedure

1. Read [AGENTS.md](../../../AGENTS.md); discover layout from README or manifests—do not assume stale paths.
2. Structure output using [assets/output-template.md](assets/output-template.md) (adapt; do not dump verbatim).
3. Hand off: *Next: **tasks-plan** → `.tasks/<slug>-<id>/`*.

## Gotchas

- Rewrites: cover stack, migration, and teardown—grounded in user constraints.
- Confirm test runner and CI entrypoints from the repo before recommending them.
- If the user named a preferred option, still sketch one alternative, then ask blocking questions.

## When to skip

Single obvious fix; user forbade exploration; user only wants **tasks-plan** or **tasks-execute**.
