# Deprecations

When changing a public surface (command name, hook path, config key, alias value), follow this lifecycle.

## Lifecycle

1. **active** — legacy surface still works; using it records a deprecation event and prints a warning after the command finishes.
2. **warned-only** — legacy surface works with warnings only (no removal yet).
3. **removed** — legacy surface no longer works; document in release notes.

Minimum policy: keep **active** for at least two minor releases or six months after `introduced` (see register `remove_after`).

## Surface types

| Type | Example | Migrate command |
| --- | --- | --- |
| Command name | `start-work` | `git elegant git migrate` / `git elegant repo migrate` |
| Hook path | `.workflows/start-work-ahead` | `git elegant hook migrate` / `git elegant repo migrate` |
| Config key | `elegant.start-work-stash` | `git elegant repo migrate` |
| Alias value | `elegant start-work` | `git elegant git migrate` |
| Completion script | hand-written `_git-elegant` | `git elegant completion <shell>` |

## Warning format

After command execution, once per process per deprecation id.

Legacy command names (`DEP-001`):

```text
Warning: the `<legacy>` command is deprecated; please use `<command> <action>`. Run `git elegant git migrate` to migrate automatically.
```

Other surfaces:

```text
warning: <surface> is deprecated; use <replacement>. Run `<migrate>` to migrate.
```

## Implementation

- Record events in `internal/deprecation` (`Record`, `Flush` from `cli.Execute`).
- Register every id in [deprecations-register.md](deprecations-register.md) with `introduced` and `remove_after` dates.
- Hidden Cobra shims in `internal/cli/legacy` for legacy command names (`DEP-001`).

## Canonical command id

`{command}.{action}[.{condition}][.{extension}]` — e.g. `work.start`. Used for hook paths (`.config/elegant-git/hooks/work-start-ahead`) and pipe keys (`elegant.work-start-stash`).
