# Elegant Git Core

## Context

Elegant Git is a CLI tool primarily used interactively by humans. However, it
could be used in scripting or by LLMs. To ensure a clean and consistent design,
the tool defines its behaviours as
`[binary] [object] [action] [argument]... [modifier]...`.

Help and docs invoke it as `git elegant …`. The executable is `git-elegant`; Git
resolves `git elegant` via `git-<subcommand>` on `PATH`.

## Binary

A binary is an executable file that represents compiled Elegant Git.

> In the future, we could symlink it under different names.

## Object

An object represents a logical boundary that we would like to interact with.

Current objects: `memory`, `git`, `workspace`, `repo`, `hook`, `work`, `release`.

`version` and `completion` sit on the binary without an object. Hidden legacy
flat names (for example `start-work`) remain as aliases.

## Action

An action is an operation that you can perform against an object. Its identity
is `{object}.{action}` (for example `work.start`).

## Argument

An argument is an optional or required positional input to an action. An action
could have 0 to N arguments. Arguments are resolved before the action runs.
Flags are not used for arguments.

## Modifier

A modifier alters execution of an action and does not change what the action
does. Global modifiers are `--no-workflows`, `--non-interactive`, and
`--interactive`. They may appear anywhere on the command line.
