---
layout: default
title: Exit codes
parent: Reference
nav_order: 6
---

# Exit codes

Usually, Elegant Git translates exit codes of original Git commands, so a failing `git push` fails
the Elegant Git command with whatever Git returned. However, in some cases it faces its own errors
and raises them as

| Code | Meaning |
| --- | --- |
| `0` | A successful execution. |
| `1` | Any other failure, including a required argument left unset. |
| `42` | A protected branch constraint. |
| `43` | A workflow error. |
| `46` | An unknown command. |
| `47` | A usage error. |

`42` is the code you meet when you ask Elegant Git to do something a protected branch does not
allow: commit directly to it, rewrite its history, or push it. The message names the branch and
suggests the way around it — usually `eg work start` or `eg work accept`. Which
branches are protected comes from [per-repo memory](memory.md).

`43` means the command refused to continue because your input or the repository state does not make
sense for it — an unsupported release notes layout, a hook file that already exists, a hook id that
is not a real command. The failure is reported before anything is changed.

`46` means Elegant Git did not recognize the command name at all. It prints the name it could not
resolve and then the usual list of objects, so you can spot the typo.

`47` means the command exists but the command line around it does not: an unknown or malformed
flag, too many positional arguments, an object without an action in
[non-interactive mode](interaction.md#how-the-mode-is-chosen), or an action name that object does
not have. The message is followed by that command's own help.

`1` is everything else, and it is what you get when a required argument is simply missing in
non-interactive mode — the error lists the names it wanted.
