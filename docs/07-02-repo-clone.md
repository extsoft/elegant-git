---
layout: default
title: eg repo clone
parent: '"repo" guide'
nav_order: 2
permalink: /repo/clone/
---

# eg repo clone

`repo clone` clones a remote repository and configures it.

```bash
eg repo clone git@github.com:acme/app.git
```

The repository URL is required. Workspace and directory are optional. The directory defaults to
the last path segment of the URL, with a trailing `.git` stripped.

If you pass two arguments and the second is not a known workspace name (and not `[Create new]`),
it is treated as the directory, so `eg repo clone <url> <dir>` works as you would expect from
Git. Relative directories honor `GIT_PREFIX` when `git elegant` ran from a subdirectory.

After `git clone` it changes into the new directory and runs the same onboarding as
[`repo configure`](07-01-repo-configure.md). When the workspace argument is omitted, a matching
[namespace](06-00-workspace.md#how-a-namespace-is-remembered) can suggest one.

It prints that the repository was cloned into that directory, then the configure output.
