# Work Actions

## Context

[Object without an action](002-interactive-vs-non-interactive-execution.md) for
`git elegant work`. The work object is day-to-day branch workflow: `start`,
`save`, `amend`, `list`, `polish`, `sync`, `push`, `track`, `accept`.

When the action is omitted, pick one from git state, or
[ask](004-object-without-action.md#ask). Non-interactive mode always requires
an action.

## Lifecycle

```mermaid
stateDiagram-v2
  direction LR
  [*] --> OnProtected
  OnProtected --> OnFeature: start
  OnProtected --> OnFeature: track
  OnFeature --> OnFeature: save
  OnFeature --> OnFeature: amend
  OnFeature --> OnFeature: polish
  OnFeature --> OnFeature: sync
  OnFeature --> OnFeature: push
  OnFeature --> OnProtected: accept
  OnFeature --> Rebasing: polish_starts
  Rebasing --> OnFeature: polish
  OnProtected --> RebasingAccept: accept_starts
  RebasingAccept --> OnProtected: accept
```

`OnProtected` is the default development branch or any other protected branch.
`OnFeature` is any other local branch. `list` does not change state.

## Detection

First matching rule wins. “Dirty” means uncommitted changes. “Unique commits”
means commits on HEAD that are not on the source branch from `start`.

```mermaid
stateDiagram-v2
  [*] --> if_rebase
  state if_rebase <<choice>>
  if_rebase --> if_acceptRebase: rebase
  if_rebase --> if_protected: no rebase

  state if_acceptRebase <<choice>>
  if_acceptRebase --> accept: accept helper
  if_acceptRebase --> polish: feature

  state if_protected <<choice>>
  if_protected --> if_dirtyProtected: protected
  if_protected --> if_dirtyFeature: feature

  state if_dirtyProtected <<choice>>
  if_dirtyProtected --> start: dirty
  if_dirtyProtected --> ask: clean

  state if_dirtyFeature <<choice>>
  if_dirtyFeature --> save: dirty
  if_dirtyFeature --> if_behind: clean

  state if_behind <<choice>>
  if_behind --> sync: behind only
  sync --> ask
  if_behind --> if_idle: else

  state if_idle <<choice>>
  if_idle --> list: even, no unique commits
  list --> ask
  if_idle --> ask: unique commits
```

1. Rebase in progress — `polish` (continues the rebase). If the branch being
   rebased is `__eg` (accept’s helper checkout), `accept` instead. Git records
   that name in rebase metadata (`head-name`); it does not record which command
   started the rebase.
2. Dirty and on a protected branch — `start` (no commits on protected branches).
3. Dirty and on a feature branch — `save`.
4. Clean and behind upstream only — `sync`, then ask.
5. Clean, even with upstream, and no unique commits — `list`, then ask.
6. Otherwise ask.

Print and ask: [Detection](004-object-without-action.md#detection),
[Ask](004-object-without-action.md#ask).

## Ask

Typical sets:

- On protected, clean — `start` / `track` / `accept` / `list` / `quit`
- On feature, unique commits, no upstream or ahead of upstream —
  `polish` / `push` / `accept` / `list` / `quit`
- On feature, diverged from upstream — `sync` / `push` / `accept` / `list` / `quit`
- On feature, idle — `start` / `accept` / `list` / `track` / `quit`
- Detached HEAD — `start` / `track` / `quit`

On a feature branch, ask `accept` uses HEAD (does not pick a branch). On a
protected branch, `accept` still asks which branch to take.

`amend` is never auto-detected; it is always an explicit action.
