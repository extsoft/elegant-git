---
name: brainstorm
description: >
  Use when the user has an ambiguous or multi-layer request, several viable approaches,
  or asks to explore, compare, brainstorm, rewrite, or migrate before committing to code.
  Produce 2–4 options with tradeoffs in chat only—no .tasks/ files and no implementation.
  Do not use for obvious single-file fixes, when they already chose an approach, or when
  they ask to create tasks, plan, or execute work under .tasks/.
compatibility: Requires read access to the repo; no network required.
---

# Brainstorm

## Progress

- [ ] Problem and constraints stated
- [ ] 2–4 options with pros, cons, cost/risk, fit for stated constraints
- [ ] Open questions listed (ask user if blocking)
- [ ] Handoff to **plan** skill stated

## Rules

- No writes under `.tasks/`; no code changes.
- Output only in conversation (markdown).

If the request touches **security, data migration, performance, or compliance**, read [practices.md](../../references/practices.md) before options.

## Procedure

1. Read repo orientation from [AGENTS.md](../../../AGENTS.md) and discover current layout from README or root manifests—do not rely on stale path assumptions.
2. Work through sections using [assets/output-template.md](assets/output-template.md) (adapt headings; do not dump the file verbatim).
3. End with: *Next: **plan** skill → `.tasks/<slug>-<id>/`*.

## Gotchas

- For full rewrites, options should include target stack, migration strategy, and what to delete vs keep—grounded in user constraints, not legacy layout.
- Do not assume test runner, CI entrypoint, or directory names until confirmed in the repo or by the user.
- If the user named a preferred option, still sketch one alternative, then focus questions on gaps.

## When to skip

Single obvious fix; user forbade exploration; user only wants **plan** or **execute**.
