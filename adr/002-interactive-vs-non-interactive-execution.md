# Interactive vs Non-Interactive Execution

## Context

Elegant Git could be run in

- Interactive environment - a human is sitting there, typing, and the program can ask questions
  and wait for answers.
- Non-interactive - the program runs start-to-finish on its own, with all input decided in
  advance (arguments, modifiers, pipes, env vars). Nobody is there to answer a prompt — so a prompt
  would just hang forever.

Elegant Git needs to match execution mode to the environment to ensure the best possible experience.

## Environment Detection

Elegant Git picks a mode with this order (first match wins):

1. stdin is not a terminal — non-interactive. Flags cannot override this.
2. `--interactive` or env `ELEGANT_GIT_INTERACTIVE` set to `1` or `true` — interactive.
3. `--non-interactive` or env `ELEGANT_GIT_NON_INTERACTIVE` set to `1` or `true` — non-interactive.
4. env `CI` set to `1` or `true` — non-interactive.
5. otherwise — interactive.

On a TTY, modifiers choose the mode:

- `--non-interactive` runs without prompts.
- `--interactive` forces prompts (overrides `CI` and `--non-interactive`).

On a non-TTY stdin, the command is always non-interactive. `--interactive` is ignored.

## Action Behaviour

If a user runs `[binary] [object]` with no action:

- Interactive mode — detect the action from context, or ask if more than one action fits.
  [How to print and ask](004-object-without-action.md). Detection per object:
  - [work](005-work-actions.md)
  - [workspace](006-workspace-object.md)
- Non-interactive mode — fail; an action is required.

## Argument Behaviour

### Interactive Mode

Missing required arguments are asked as [interactive questions](003-interactive-questions.md).
Optional arguments are asked only in that case, and only if they were not already given.
Some optional arguments never ask.

If an argument has completion options, a user should be able to pick them using the [picker](003-interactive-questions.md#picker).

### Non-interactive Mode

If an action has at least one required argument unspecified, the command fails.

### Modifier Behaviour

The modifiers always use default values and are never suggested for user selection.
