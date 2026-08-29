---
layout: default
title: Releases
nav_order: 9
---

# Releases

The `release` object has two actions. `eg release new` cuts a release from the default
development branch, and `eg release notes` prints the log between two refs so you can
paste it wherever your release lives.

## Cutting a release

```bash
eg release new v2026.8.29
```

The name is the only argument, and it becomes the tag. In interactive mode you are asked for it
when you leave it out; in non-interactive mode a missing name fails the command. From there
Elegant Git

1. checks out the default development branch and runs `git pull --tags`
2. drafts an annotated tag message — a `Release <name>` heading followed by one `- <subject>` line
   per commit since the last tag — and opens it in your editor so you can edit it before it is
   recorded
3. pushes the tags
4. produces the release notes in the `smart` layout and puts them on your clipboard when `pbcopy`
   or `xclip` is available, printing them otherwise

The command runs inside both the [stash pipe and the branch pipe](pipes.md), so uncommitted
changes and the branch you were on are preserved and restored — you end up back where you
started. "The last tag" means the highest tag by version order, not the most recently created one.

## Shaping the notes

```bash
eg release notes [<layout>] [<from-ref>] [<to-ref>]
```

Every argument is optional. The layout defaults to `simple`, `<from-ref>` defaults to the last tag,
and `<to-ref>` defaults to `HEAD`. Pass `all-commits` as `<from-ref>` when you want the whole
history up to `<to-ref>` instead of a range.

There are two layouts:

- `simple` prints a `Release notes` heading and one `- <subject>` bullet per commit, which is what
  you want for a changelog file or a plain-text announcement
- `smart` prints an HTML list where every subject links to its commit on GitHub, and appends the
  issue references found in the commit body

`smart` needs a GitHub `origin` to build those links. When `origin` points somewhere else, the
command quietly falls back to `simple` rather than emitting broken links. Any other layout name is
an error.

```bash
eg release notes simple v2026.7.1 v2026.8.29
```
