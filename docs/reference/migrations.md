---
layout: default
title: Migrations
parent: Reference
nav_order: 4
---

# Migrations

Elegant Git keeps state in several places — [shared memory](memory.md), per-repo memory,
`git config` keys, and hook files — and those layouts change between versions. There is no
migrate command to run. Rewrites of what Elegant Git owns happen on their own; anything that
needs confirmation is a `doctor` finding.

## Two kinds

Ownership decides which kind a change is, not whether the old layout still works. Almost
every legacy surface still has a read-side fallback.

**Automatic migrations** touch only what Elegant Git owns and carry no user intent. Values
are preserved verbatim, nothing is created that was not already there, and nothing is
asked.

**Assisted migrations** touch user `git config`, the working tree, or files the team
shares — or they have nothing mechanical to fix. Those show up as doctor findings. Repairable
ones wait for a yes; advisory ones are informational only.

If Elegant Git wrote the bytes and the rewrite is determined, it is automatic; otherwise it
belongs to doctor.

## When automatic work runs

Most `eg` invocations apply pending automatic work before the requested command. Nested
hook-spawned processes skip it, as do commands that only inspect the CLI itself, so those
stay cheap and cannot race on the state file.

Global steps run once, then stay out of the way. Local leftovers are still checked whenever
the current directory is a git repository, so configuring Git outside a clone cannot leave
other clones behind.

A failure prints one line to stderr, retries on the next invocation, and does not fail the
requested command. Rewrites of Elegant Git's own files keep a backup next to the original.

## What doctor repairs

`eg self doctor`, `eg repo doctor`, and `eg workspace doctor` each diagnose one scope and
suggest a repair for every finding.

Interactive mode prints each finding and asks `Fix?` before applying a repair. Non-interactive
mode prints the same list and changes nothing, then exits non-zero only when a repairable
finding remains. Advisory findings are printed and never counted toward that exit.

When a repair rewrites files Git is tracking, doctor suggests the `git add` / `git commit`
after the repair is accepted.

The old migrate commands remain as hidden shims for scripts; `doctor` is the supported way
to inspect and confirm what is left.

## Version compatibility

A file whose schema is **newer** than the running binary supports is never rewritten. The
command fails and asks for an Elegant Git upgrade, so an older installation cannot quietly
downgrade state a newer one wrote.
