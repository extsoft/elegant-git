---
layout: default
title: '"release" guide'
nav_order: 9
has_children: true
has_toc: false
permalink: /release/
---

# "release" guide

The `release` object has two actions. `eg release new` cuts a release from the default
development branch, and `eg release notes` prints the log between two refs so you can
paste it wherever your release lives.

`eg release` with no action prints the action list. There is nothing to detect from context, so
an action is always required.

## Actions

- [`new`](09-01-release-new.md) — Releases the default development branch.
- [`notes`](09-02-release-notes.md) — Prints a release log between two refs.
