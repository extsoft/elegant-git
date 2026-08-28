# Interactive questions

## Context

The [Interactive vs Non-Interactive Execution](002-interactive-vs-non-interactive-execution.md)
needs to ask a user a question to collect an argument or confirm a choice. This
document is the design system for those questions.

- **Line** — 0 or 1 option (free text, or a single suggestion)
- **Picker** — 2 or more options (list = single-select, set = multi-select)

## Line

Every question is one line:

```text
<prompt>[<suggested>](<action on enter>): <answer>
```

Omit a part when it does not apply. Separate parts with a single space.

- `<prompt>` — what we ask; always present
- `[<suggested>]` — at most one value (`[Alice]`, `[main]`); omit when there is nothing to suggest
- `(<action on enter>)` — what empty Enter does; omit when Enter is not allowed
- `<answer>` — what the user types after `:`

### Action on enter

- Required with a suggestion: `(press enter to accept)` — empty Enter takes that value.
- Optional with a suggestion: `(press enter to accept)` — empty Enter takes that value.
- Optional with no suggestion: `(press enter to skip)`.
- Required with no suggestion: no enter action. Empty Enter is invalid; ask again.

### Required text

No suggestion. The user must type a value.

```text
Workspace name: some name
```

### Optional text

The value may stay empty when nothing is suggested. A configured value is
shown as a suggestion; empty Enter records it.

```text
Signing key (press enter to skip):
Signing key [ABC123] (press enter to accept):
Editor command [vim] (press enter to accept): nano
```

## Picker

Use when there are **2 or more** choices. Built-in; not an external `fzf`.

```text
<prompt>: <filter>
<filtered> of <total> (<selected> selected) <line>
<mark> <option>[  <description>]
```

Omit a part when it does not apply.

- `<prompt>` — what we ask; always present; bold blue
- `<filter>` — placeholder in italic; disappears when typing
  - `select one option; enter to confirm` for a single value
  - `select one or more; space to select, enter to confirm` for multiple values
- `<filtered>` — number of filtered elements from the list
- `<total>` — number of elements in the list
- `<selected>` — number of selected elements
- `<line>` — a thin horizontal line
- `<mark>` — current and selected rows; see single- and multi-select
- `<option>` — the token we accept; column padded so descriptions align;
  columns separated with spaces (not tabs)
- `<description>` — optional; at most 70 characters

Filter matches a case-insensitive subsequence of the option or the description.
Esc, Ctrl+C, or Ctrl+D cancels.

With one choice left after filtering, accept it as a [Line](#line). With zero,
keep filtering.

### Single-select

One answer. `<mark>` is `>` on the current row and two spaces on the others.

```text
What now:
> start   Creates a new branch.
  save    Commits current modifications.
  list    Prints HEAD state.
  quit    Leave without another action.
```

Type to filter; Backspace; Up / Down; Enter accepts the current row. When a
default exists, that row starts current.

Examples: yes/no, hook type, release layout, completion shell, work “What now”,
branches, workspaces, refs.

### Multi-select

Zero or more answers. `<mark>` is `>` current, `*` selected, `>*` both, two
spaces otherwise.

```text
Protected branches:
* main
> master
  develop
```

Same filter and movement as single-select. Space toggles the current row. Enter
accepts every selected option. Defaults start selected.

### Descriptions by list

| List | Description |
| --- | --- |
| `work` “What now” | Action purpose (e.g. start → “Creates a new branch.”). `quit` → “Leave without another action.” |
| `workspace` “What now” | Action purpose (e.g. link → “Links the current repository to a workspace.”). `quit` → “Leave without another action.” |
| Local branches | Upstream remote-tracking ref when set (`feat` → `origin/feat`); else none |
| Remote-only lists | None |
| Refs / branch union | Locals as above; remotes and tags option-only |
| Workspaces | Identity (`Name <email>`), truncated to 70 |
| Hook paths, command ids, completion shells | None |
| Work start uncommitted (`add`/`reset`/`cancel`) | Short purpose for each |
| yes/no, yes/no/all/skip | None |
