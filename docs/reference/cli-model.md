---
layout: default
title: CLI model
parent: Reference
nav_order: 5
---

# CLI model

Every Elegant Git command line has the same shape:

```text
[binary] [object] [action] [argument]... [modifier]...
```

Knowing the five parts is enough to read any command in these docs, to guess a command you have not
seen yet, and to script the tool without surprises. So, one part at a time.

## Binary

The binary is called `eg`. After `eg git configure`,
`git elegant …` still works via the git alias `alias.elegant = "!eg"`. That form
runs from the repository top level (Git sets `GIT_PREFIX` to the subdirectory you
started in) and does not tab-complete. Treat `eg …` as the primary form.

## Object

An object is the thing you are talking to. There are seven of them: `memory`, `git`, `workspace`,
`repo`, `hook`, `work`, and `release`.

`version` and `completion` sit on the binary itself, without an object, because they are about the
program rather than about anything it manages. The hidden legacy flat names — `start-work` and its
siblings — also live at that level, as deprecated aliases.

## Action

An action is what you do to an object. Together they are named `{object}.{action}` — for example
`work.start`. That dotted form is what you pass to [`hook new`](../guides/hooks.md#managing-hooks)
and what appears in per-command state, so it is worth recognizing.

## Argument

An argument is a positional input to an action, required or optional. An action can have anywhere
from zero to several of them, and they are all resolved before the action starts to work. If
something is an input to the action, it is positional — you will not see a flag for it.

## Modifier

A modifier changes how an action runs without changing what it does. There are three, and they are
global: `--no-workflows`, `--non-interactive`, and `--interactive`. They may appear
anywhere on the command line, so `eg --no-workflows work save` and
`eg work save --no-workflows` are equivalent. What each one means is on the
[interaction](interaction.md) page, and the full list is in the
[commands](commands.md#flags) reference.
