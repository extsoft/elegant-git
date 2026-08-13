# Interactive questions

## Context

The [Interactive vs Non-Interactive Execution](002-interactive-vs-non-interactive-execution.md)
needs to ask a user a question to collect an argument or confirm a choice. This
document is the design system for those questions.

## Line

Every question is one line:

```text
<prompt>[<suggested>](<action on enter>): <answer>
```

Omit a part when it does not apply. Separate parts with a single space.

- `<prompt>` — what we ask; always present
- `[<suggested>]` — a hint or a closed list of answers; omit when there is nothing to suggest
- `(<action on enter>)` — what empty Enter does; omit when Enter is not allowed
- `<answer>` — what the user types after `:`

## Suggested

`[<suggested>]` is used when there is something to propose.

- One value — `[Alice]`, `[main]` — the only suggestion
- Closed list — whole words split by `/`, for example `[yes/no]`

A closed list is the full set of valid answers. The user types the word or its
first letter (case-insensitive) when that letter is unique.

## Action on enter

- Required with a suggestion: `(press enter to accept)` — empty Enter takes that value.
- Optional with a suggestion: no enter action. Empty Enter leaves the value unset.
- Optional with no suggestion: `(press enter to skip)`.
- Required with no suggestion: no enter action. Empty Enter is invalid; ask again.
- `yes/no` or similar closed choice: always set a default from context and show `(press enter to '<default>')`. Empty Enter takes that word. If there is no default, ask again.

## Required text

No suggestion. The user must type a value.

```text
Profile name: some name
```

## Suggested text

Required field with one value to keep or replace. Enter accepts it.

```text
Git user.name [Alice] (press enter to accept):
Git user.email [alice@example.com] (press enter to accept): Bob@example.com
```

Used today for git identity fields, default branch, protected branches, local
branch name, and profile name when a default exists.

## Optional text

The value may stay empty. A suggestion is shown without an enter action.
Without a suggestion, Enter skips.

```text
Signing key (press enter to skip):
Signing key [ABC123]:
Editor command [vim]: nano
```

Used today for signing key, GPG program, and editor.

## Yes or no

Closed list `[yes/no]`. Always a default, chosen from context (not always `no`).
Enter takes it.

```text
Apply profile "<name>" to current repository? [yes/no] (press enter to 'yes'):
Override existing profile "<name>"? [yes/no] (press enter to 'no'): y
Would you like to apply a global configuration? [yes/no] (press enter to 'no'):
Proceed? [yes/no] (press enter to 'yes'):
```

`yes` or `y` is yes. `no` or `n` is no. Empty Enter takes the default. If the
default is missing, ask again.

## Repeat over many

The same yes/no over a list (repositories). Closed list `[yes/no/all/skip]`.
Always a default from context.

```text
Apply to <repository>? [yes/no/all/skip] (press enter to 'no'):
```

- `yes` / `y` — this one
- `no` / `n` — skip this one
- `all` / `a` — this one and every remaining
- `skip` / `s` — skip this one and every remaining
- Enter — the default from context
- no default — ask again

## Closed list

A small fixed set that is not yes/no. Put the tokens in `[<suggested>]`. If the
field is required, add `(press enter to accept)` only when one token is the
default. If the field is optional, no enter action.

```text
Hook type [ahead/after]: after
Hook location [personal/common]:
Release notes layout [simple/smart]:
Completion shell [bash/zsh/fish/powershell]:
```

Used today for hook type, hook location, release notes layout, and completion
shell. Do not use a numbered menu for these.

## Fuzzy matching

An open list is too large for `[a/b/c]` (branches, profiles, repositories,
refs, hook paths, command ids). The question line is the same. After `:` the
user picks from a filterable list instead of typing a free token.

If the field is required and there is a default:

```text
Start from ref [main] (press enter to accept):
```

If the field is optional and there is a default:

```text
Start from ref [main]:
```

If the answer is required and there is no default:

```text
Remote branch name:
```

The picker then replaces `<answer>`:

```text
Remote branch name:
main
develop
feature/login
```

The user filters by typing; the match is a case-insensitive subsequence of the
value or its description (`mn` matches `main`).

- Type to filter; Backspace to delete
- Up / Down to move; Enter to accept the highlighted line
- Esc, Ctrl+C, or Ctrl+D to cancel

If `fzf` is on `PATH` and stdin is a terminal, that picker may be used. If
stdin is not a terminal, the same question line is used and the user types a
filter, then picks one match as a closed list.
