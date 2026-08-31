---
description: Deprecation policy for CLI and config changes
globs:
  - internal/cli/**
  - internal/config/**
  - internal/workflows/**
  - internal/pipe/**
  - internal/deprecation/**
---

# Deprecations

When you change a public surface, read [.cursor/references/deprecations.md](../references/deprecations.md) and add a row to [.cursor/references/deprecations-register.md](../references/deprecations-register.md) (`id`, `introduced`, `remove_after`, `status`).

- Record usage via `internal/deprecation` (`Record`, constants `DEP-001`…).
- Automatic rewrites of Elegant Git-owned state go in `internal/migrate`. Assisted repairs are doctor findings (`eg self doctor`, `eg repo doctor`). Do not add a migrate command.
- Keep legacy behavior until `remove_after`; hidden Cobra shims live in `internal/cli/legacy`.
