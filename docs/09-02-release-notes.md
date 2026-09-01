---
layout: default
title: eg release notes
parent: '"release" guide'
nav_order: 2
permalink: /release/notes/
---

# eg release notes

`release notes` prints a release log between two refs.

```bash
eg release notes
```

Every argument is optional. The layout defaults to `simple`, `<from-ref>` defaults to the last
tag, and `<to-ref>` defaults to `HEAD`. Pass `all-commits` as `<from-ref>` when you want the whole
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
