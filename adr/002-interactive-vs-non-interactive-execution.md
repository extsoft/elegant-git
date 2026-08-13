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

1. `--interactive` or env `ELEGANT_GIT_INTERACTIVE` set to `1` or `true` — interactive.
2. `--non-interactive` or env `ELEGANT_GIT_NON_INTERACTIVE` set to `1` or `true` — non-interactive.
3. env `CI` set to `1` or `true` — non-interactive.
4. stdin is not a terminal — non-interactive.
5. otherwise — interactive.

Additionally, a user could force certain behaviours with modifiers:

- `--non-interactive` instructs to execute the command as if it is run in a non-interactive environment.
- `--interactive` instructs to execute the command as if it is run in an interactive environment (preferred).

## Argument Behaviour

### Interactive Mode

Missing required arguments are asked as [interactive questions](003-interactive-questions.md).
Optional arguments are asked only in that case, and only if they were not already given.
Some optional arguments never ask.

If an argument has completion options, a user should be able to pick them using [fuzzy matching](003-interactive-questions.md#fuzzy-matching).

### Non-interactive Mode

If an action has at least one required argument unspecified, the command fails.

### Modifier Behaviour

The modifiers always use default values and are never suggested for user selection.
