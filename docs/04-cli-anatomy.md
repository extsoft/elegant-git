---
layout: default
title: CLI anatomy
nav_order: 4
permalink: /cli-anatomy/
---

# CLI anatomy

Every Elegant Git command line has the same shape:

```text
[binary] [object] [action] [argument]... [modifier]...
```

Knowing the five parts is enough to read any command in these docs, to guess a command you have not
seen yet, and to script the tool without surprises. So, one part at a time.

## Binary

The binary is called `eg`. After [`eg self configure`](05-01-self-configure.md),
you can also run it as `git elegant …`: configure writes the git alias
`alias.elegant = "!eg"`, so Elegant Git sits next to the rest of your Git
commands.

Two commands sit on the binary itself, without an object, because they are about the program
rather than about anything it manages:

```bash
eg version
eg completion bash
```

`version` prints the installed version. `completion` writes a completion script to stdout for
`bash`, `zsh`, `fish`, or `powershell`. The shell name is required; in interactive mode it is
asked if you leave it out.

Install bash completion like this:

```bash
eg completion bash > ~/.local/share/bash-completion/completions/eg
```

`git elegant <TAB>` does not complete (the git alias is a `!` expansion). Use `eg <TAB>`.
If you previously installed a `git-elegant` completion file, delete it.

The hidden legacy flat names — `start-work` and its siblings — also live at the binary level, as
deprecated aliases.

## Object

An object is the thing you are talking to. There are six of them: [`self`](05-00-self.md),
[`workspace`](06-00-workspace.md), [`repo`](07-00-repo.md), [`work`](08-00-work.md),
[`release`](09-00-release.md), and [`hook`](10-00-hook.md).

## Action

An action is what you do to an object. Together they are named `{object}.{action}` — for example
`work.start`. That dotted form is what you pass to [`hook new`](10-02-hook-new.md)
and what appears in per-command state, so it is worth recognizing.

Running `eg <object>` with no action prints the action list, except for `work` and `workspace`:
in interactive mode those two detect the next action from context, or ask you when more than one
fits. In non-interactive mode an action is always required. See
[interaction](reference/interaction.md#when-you-omit-the-action).

## Argument

An argument is a positional input to an action, required or optional. An action can have anywhere
from zero to several of them, and they are all resolved before the action starts to work. If
something is an input to the action, it is positional — you will not see a flag for it. How
missing arguments are asked, or refused, is on the [interaction](reference/interaction.md)
page.

## Modifier

A modifier changes how an action runs without changing what it does. There are three, and they are
global: `--no-workflows`, `--non-interactive`, and `--interactive`. They may appear
anywhere on the command line, so `eg --no-workflows work save` and
`eg work save --no-workflows` are equivalent.

- `--no-workflows` — skip ahead and after [hooks](reference/hook-files.md)
- `--non-interactive` — no prompts; fail when a required input is missing (also
  `ELEGANT_GIT_NON_INTERACTIVE=1`, `CI`, or non-TTY stdin)
- `--interactive` — on a TTY, force prompts (overrides `CI` and `--non-interactive`; also
  `ELEGANT_GIT_INTERACTIVE=1`). Ignored when stdin is not a TTY

`--version` prints the version, same as `eg version`. What each interaction modifier means is on
the [interaction](reference/interaction.md) page.
