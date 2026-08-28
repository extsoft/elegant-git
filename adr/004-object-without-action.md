# Object without an action

## Context

[Object without an action](002-interactive-vs-non-interactive-execution.md) in
interactive mode: detect from context, or ask. How that looks is here. Rules
are per object ([work](005-work-actions.md)). Questions use
[interactive questions](003-interactive-questions.md).

## Detection

Print each check in order, then the decision. Stop after the first matching
rule.

```text
==>> Detection action...
<check>? <result>
selected: git elegant <object> <action>
```

```text
==>> Detection action...
<check>? <result>
selected: ask
```

If detection already ran an action and still needs a choice, print
`selected: ask` once; do not re-print the checks.

## Ask

Actions that still make sense, plus `quit`. Default is `quit`. With two or more
options the [picker](003-interactive-questions.md#picker) shows each action and
its purpose.

```text
What now:
> quit    Leave without another action.
  start   Creates a new branch.
  track   Checks out a remote-tracking branch.
  list    Prints HEAD state.
```
