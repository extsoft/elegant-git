---
description: House style for Elegant Git user docs (just-the-docs site under docs/)
globs: docs/**/*.md
alwaysApply: false
---

# Docs writing

Read [.cursor/references/glossary.md](../references/glossary.md) before naming. `docs/**` is
excluded from prosefmt (`.config/hk.pkl`); wrap prose near 100 columns yourself.

## Site shape

- Numbered chapters: `NN-title.md`; object parents `NN-00-<object>.md`; actions
  `NN-MM-<object>-<action>.md`.
- Front matter: `layout: default`; `title` equals H1; `nav_order`; pretty `permalink`.
  Children set `parent:` to the parent **title** exactly. Parents add `has_children: true` and
  `has_toc: false`.
- Titles: `"self" guide` (not `"eg self" guide`); anatomy `CLI anatomy`; action pages
  `eg self configure`.
- Cross-link with relative `.md` paths.

## Object landing (`xx-00`)

Short lead, then `## Actions` first — unordered list, one-line purposes from `Groups` in
`internal/cli/catalog/catalog.go`. Then concept. No per-action `##` plus fenced `bash`.

```markdown
# BAD
title: self
## configure
```bash
eg self configure
```
See [eg self configure](05-01-self-configure.md).

# GOOD
title: '"self" guide'
## Actions
- [`configure`](05-01-self-configure.md) — Configures Git and the Elegant Git installation.
```

## Action pages

Lead with what the command is for. One short `bash` invocation, explanation after it. `##`
headings are topics (`What it applies`), never `Synopsis` / `Arguments` / `See also`.
Second-person prose; bullets only for real lists; tables stay in `docs/reference/`.
Write from the Go implementation. Skip hidden/deprecated commands.

## Vocabulary

- **Elegant Git installation** — this machine’s `eg` (shared memory, acquired, aliases).
- **Git configuration** — `git config` (identity, standards). Do not say “Git installation” for
  `self`.
- `git elegant` is a feature (git alias after configure), not leftover. Do not lead with the
  `git-elegant` binary or completions unless the page is about upgrading.
