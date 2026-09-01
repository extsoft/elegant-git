---
layout: default
title: eg workspace new
parent: '"workspace" guide'
nav_order: 2
permalink: /workspace/new/
---

# eg workspace new

`workspace new` creates a workspace.

```bash
eg workspace new work "Alice Doe" alice@acme.com
```

The name, user name, and email are required. Signing key, gpg program, and editor are optional.
In interactive mode, missing required arguments are asked, and the optional ones are offered with
suggestions from the local and then the global `git config`. The name defaults to the local part
of the email. In non-interactive mode a missing required argument fails the command.

After the workspace is saved, if you are inside a git repository, interactive mode asks whether to
apply it to the current repository — default yes. That writes the identity into `.git/config`,
upserts the registry, records per-repo memory, and may ask to remember the origin
[namespace](06-00-workspace.md#how-a-namespace-is-remembered). If the repository is already linked
elsewhere, it asks before overriding. Non-interactive mode creates the workspace and does not
apply it to the current repository.

The command prints `Created workspace <name> (<id>)` when it is done.
