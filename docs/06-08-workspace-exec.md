---
layout: default
title: eg workspace exec
parent: '"workspace" guide'
nav_order: 8
permalink: /workspace/exec/
---

# eg workspace exec

`workspace exec` runs a command in every repository linked to a workspace.

```bash
eg workspace exec [name] -- <command> [args...]
```

The workspace name is optional and is never asked. When you leave it out, Elegant Git uses the
workspace linked to the current repository. If you are not in a linked repository, name the
workspace. From bare `eg workspace`, choosing `exec` asks for the command on the next line.

Everything after `--` is passed to the child as an argument vector. Elegant Git does not invoke a
shell, so pipes, redirections, and shell expansion require an explicit shell:

```bash
eg workspace exec -- sh -c 'git log -1 --format=%h | tr a-z A-Z'
```

## Standard streams

Each child inherits stdin, stdout, and stderr directly. Elegant Git does not read or buffer stdin
before starting the child. When stdin is a pipe, the first repository that reads it consumes the
input; later repositories see an empty stream.

Stdout carries only the child's output. Stderr carries the child's stderr and one attribution line
per repository:

```text
==> repo-name (/path/to/repo)
```

Wrapper errors also go to stderr.

## Working directory and environment

Each child runs with its working directory set to the repository path. The parent environment is
inherited, with these variables added:

| Variable | Value |
| --- | --- |
| `EG_WORKSPACE` | workspace display name |
| `EG_REPO_NAME` | repository name in the registry |
| `EG_REPO_PATH` | repository path |

Executable lookup uses the caller's `PATH`. A relative path such as `./tool` resolves against the
repository directory.

## Exit status

The command stops at the first non-zero child exit and returns that exit code without printing an
extra message. A missing executable exits 127; a non-executable file exits 126. A missing
repository path or registry entry exits 1 and hints at `eg workspace doctor` or `eg repo doctor`.
When nothing is linked, a message is printed to stderr and the exit code is 0.
