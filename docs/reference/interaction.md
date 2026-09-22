---
layout: default
title: Interaction
parent: Reference
nav_order: 4
---

# Interaction

Elegant Git runs in two environments, and it behaves differently in each:

- **interactive** — a human is sitting there, typing, and the program can ask questions and wait
  for answers
- **non-interactive** — the program runs start to finish on its own, with everything decided in
  advance through arguments, modifiers, and environment variables. Nobody is there to answer a
  prompt, so a prompt would just hang forever

That's why Elegant Git works out which of the two it is in before it asks you anything.

## How the mode is chosen

The checks run in this order, and Elegant Git stops at the first one that fits:

1. stdin is not a terminal — non-interactive. You cannot override this with a flag.
2. `--interactive`, or `ELEGANT_GIT_INTERACTIVE` set to `1` or `true` — interactive.
3. `--non-interactive`, or `ELEGANT_GIT_NON_INTERACTIVE` set to `1` or `true` — non-interactive.
4. `CI` set to `1` or `true` — non-interactive.
5. otherwise — interactive.

So, on a terminal the modifiers decide: `--non-interactive` runs without prompts, and
`--interactive` forces prompts back on, overriding both `CI` and `--non-interactive`. When stdin is
not a terminal — a pipe, a heredoc, a CI log — the command is always non-interactive and
`--interactive` is ignored.

## How arguments are asked

In interactive mode, a missing required argument is asked as a [question](#questions). Optional
arguments are offered only in that case, and only when you have not already given them; some
optional arguments never ask at all. When an argument has completion options, you get
[a picker](#picker) rather than a blank line to type into.

In non-interactive mode, an action with at least one unspecified required argument fails, and the
error lists what is missing.

Modifiers are different: they keep their default and are never offered as a question.

## When you omit the action

Running `eg work` or `eg workspace` without an action is legitimate in
interactive mode — Elegant Git detects the action from the context, or asks you when more than one
fits. In non-interactive mode there is nothing to detect against, so an action is required and the
command fails without one.

Which checks apply depends on the object: see [work](../08-00-work.md) for `work` and
[workspace](../06-00-workspace.md) for `workspace`.

### How detection is printed

Every check is printed in the order it ran, followed by the decision. The output stops at the first
check that fits, so what you see is exactly what was evaluated:

```text
==>> Detection action...
<check>? <result>
selected: eg <object> <action>
```

When no check fits, the decision line says so and the [ask list](#the-ask-list) follows:

```text
==>> Detection action...
<check>? <result>
selected: ask
```

If detection already ran an action and still needs a choice from you, `selected: ask` is printed
once on its own; the checks are not printed again.

### The ask list

The list holds the actions that still make sense, plus `quit`, and `quit` is the default. With two
or more options you get [the picker](#picker), showing each action and its purpose:

```text
What now:
> quit          Leave without another action.
  work start    Creates a new branch.
  work track    Checks out a remote-tracking branch.
  work list     Prints HEAD state.
  work help     Shows available actions.
```

## Questions

There are two shapes of question. A **line** is used when there are no options or a single
suggestion, and a **picker** is used when there are two or more options to choose from.

### Line

Every line question fits on one line:

```text
<prompt> [<suggested>] (<action on enter>): <answer>
```

A part is omitted when it does not apply to your question, and the parts are separated by a single
space:

- `<prompt>` — what is being asked; always present
- `[<suggested>]` — at most one value, such as `[Alice]` or `[main]`; omitted when there is nothing
  to suggest
- `(<action on enter>)` — what an empty Enter does; omitted when Enter is not allowed
- `<answer>` — what you type after the `:`

What Enter does depends on the combination:

- required with a suggestion — `(press enter to accept)`, and an empty Enter takes that value
- optional with a suggestion — `(press enter to accept)`, same thing
- optional with no suggestion — `(press enter to skip)`
- required with no suggestion — no Enter action at all; an empty Enter is invalid and the question
  is asked again

A **required text** question has no suggestion, so you have to type a value:

```text
Workspace name: some name
```

An **optional text** question may stay empty when nothing is suggested. A value that is already
configured is shown as the suggestion, and an empty Enter records it:

```text
Signing key (press enter to skip):
Signing key [ABC123] (press enter to accept):
Editor command [vim] (press enter to accept): nano
```

A **closed** question lists its options inside the suggestion brackets and names the default:

```text
Apply workspace "work" to current repository? [yes/no] (press enter to 'yes'):
Apply to repo? [yes/no/all/skip] (press enter to 'no'):
```

You can answer a closed question with the whole word or with its first letter, in any case, as long
as that letter is unambiguous.

### Picker

The picker is used when there are **two or more** choices. It is built in — there is no dependency
on an external `fzf`.

```text
<prompt>: <filter>
<filtered> of <total> <line>
<mark> <option>[  <description>]
```

- `<prompt>` — what is being asked; always present, in bold blue
- `<filter>` — what you type; until you do, an italic placeholder sits there instead, reading
  `select one option; enter to confirm`
- `<filtered>` — how many options match what you typed
- `<total>` — how many options there are
- `<line>` — a thin horizontal rule
- `<mark>` — `>` on the current row, two spaces on the others
- `<option>` — the token that gets accepted; padded into a column so the descriptions line up,
  with spaces rather than tabs
- `<description>` — optional, and truncated to 70 characters

Typing filters the list, Backspace un-types, Up and Down move, and Enter accepts the current row.
When a default exists, its row starts out as the current one. Filtering matches a case-insensitive
subsequence of either the option or its description, so `wsp` finds `workspace`. Esc, Ctrl+C, and
Ctrl+D cancel the question. Ten rows are shown at a time; the list scrolls around your cursor when
there are more. Filtering down to zero matches is not an error — keep typing or backspace out of
it.

With fewer than two options you never see the picker; the question is asked as a [line](#line)
instead.

```text
What now:
> work start    Creates a new branch.
  work save     Commits current modifications.
  work list     Prints HEAD state.
  work help     Shows available actions.
  quit          Leave without another action.
```

Descriptions are not free text — each list has its own convention:

| List | Description |
| --- | --- |
| `work`, `workspace`, and `repo` "What now" | Object and action in the option (`work start`, `workspace list`, `repo help`); purpose from the action catalog. `quit` has no object prefix. |
| Local branches | The upstream remote-tracking ref when one is set (`feat` → `origin/feat`), and nothing otherwise |
| Remote-only lists, tags | None |
| Workspaces | The identity, as `Name <email>`, truncated to 70 characters |
| Hook paths, command ids, completion shells | None |
| Uncommitted changes during `work start` | A short purpose for `add`, `reset`, and `cancel` |
