---
layout: default
title: Home
nav_order: 1
permalink: /
---

# Elegant Git

Elegant Git is an assistant that carefully automates routine Git workflows.
Its primary goal is to save mental energy and speed up Git interactions by providing a highly interactive, context-aware CLI with useful features built on top of Git.

Main capabilities

- standardized repository management
  - careful [Git configuration](reference/configuration.md) of this machine and of every
    repository you work in
  - a protected [contribution lifecycle](08-00-work.md) that keeps a change in a branch of
    its own until it is ready for the default development branch
  - many repositories managed as a single project (aka [workspaces](06-00-workspace.md))
- human-focused usability
  - an [interactive mode](reference/interaction.md) that reads the current state and suggests what
    fits it
  - every Git command which modifies a state of Git is printed before it runs
  - unsaved modifications preserved and restored, or carried over to another branch (aka
    [pipes](reference/pipes.md))

Used philosophy

- actual work is allowed in custom branches only
- all pushes to a protected branches are strictly controlled
- nothing is changed or repaired behind your back — you are asked first

## Next steps

- [Installation](02-00-installation.md) — pick a way to install Elegant Git
- [Getting started](03-getting-started.md) — once installed, try it out
