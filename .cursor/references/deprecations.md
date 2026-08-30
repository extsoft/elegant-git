# Deprecations

When changing a public surface (command name, hook path, config key, alias value), follow this lifecycle.

## Lifecycle

1. **active** — legacy surface still works; using it records a deprecation event and prints a warning after the command finishes.
2. **warned-only** — legacy surface works with warnings only (no removal yet).
3. **removed** — legacy surface no longer works; document in release notes.

Minimum policy: keep **active** for at least two minor releases or six months after `introduced` (see register `remove_after`).

## Surface types

| Type | Example | Repair |
| --- | --- | --- |
| Command name | `start-work` | automatic alias rewrite; use the new name |
| Hook path | `.workflows/start-work-ahead` | `eg repo doctor` |
| Config key | `elegant.start-work-stash` | already removed |
| Alias value | `elegant start-work` | automatic |
| Completion script | hand-written `_git-elegant` | `eg git doctor` / `eg completion <shell>` |
| Binary name | `git-elegant` / `git elegant` dispatch | automatic aliases; `eg git doctor` for leftovers |
| Migrate command | `git migrate`, `repo migrate`, `hook migrate` | automatic + `eg <object> doctor` |

See [docs/reference/migrations.md](../../docs/reference/migrations.md) for the automatic vs assisted split.

## Warning format

After command execution, once per process per deprecation id.

Legacy command names (`DEP-001`):

```text
Warning: the `<legacy>` command is deprecated; please use `<command> <action>`.
```

Other surfaces, when a repair command exists:

```text
warning: <surface> is deprecated; use <replacement>. Run `<migrate>` to migrate.
```

When the rewrite is automatic, the `Run …` clause is omitted.

## Implementation

- Record events in `internal/deprecation` (`Record`, `Flush` from `cli.Execute`).
- Register every id in [deprecations-register.md](deprecations-register.md) with `introduced` and `remove_after` dates.
- Hidden Cobra shims in `internal/cli/legacy` for legacy command names (`DEP-001`).
- Automatic migrations live in `internal/migrate` and run from `PersistentPreRunE`. Assisted repairs are doctor findings.

## Canonical command id

`{command}.{action}[.{condition}][.{extension}]` — e.g. `work.start`. Used for hook paths (`.config/elegant-git/hooks/work-start-ahead`) and per-command state in `.git/elegant-git/commands.json`.
