# Elegant Git

[sb]: https://img.shields.io/badge/Choose%20issue-simple-green
[sl]: https://github.com/extsoft/elegant-git/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22+sort%3Acomments-desc+no%3Aassignee
[ab]: https://img.shields.io/badge/Choose%20issue-any-blue
[al]: https://github.com/extsoft/elegant-git/issues?q=is%3Aissue+is%3Aopen+sort%3Areactions-%2B1-desc+no%3Aassignee
[cb]: https://img.shields.io/github/commits-since/extsoft/elegant-git/latest?label=Commits%20for%20next%20release
[cl]: https://github.com/extsoft/elegant-git/commits/main
[vb]: https://img.shields.io/github/v/tag/extsoft/elegant-git?label=Last%20release
[vl]: https://github.com/extsoft/elegant-git/releases/latest
[bb]: https://github.com/extsoft/elegant-git/actions/workflows/mise-checks.yaml/badge.svg
[bl]: https://github.com/extsoft/elegant-git/actions/workflows/mise-checks.yaml
[db]: https://img.shields.io/badge/docs-elegant--git.extsoft.pro-blue
[dl]: https://elegant-git.extsoft.pro/
[eb]: https://img.shields.io/badge/assistant-Elegant%20Git-000000.svg
[el]: https://github.com/extsoft/elegant-git

Elegant Git is an assistant who carefully automates routine work with Git.

Please visit <https://elegant-git.extsoft.pro/> to get started with user documentation or
click on the picture :point_down::point_down::point_down: to see a demo.

[![Elegant Git Demo](docs/git-elegant-demo.png)](http://www.youtube.com/watch?v=Py6bpwJw30I)

---

[![][vb]][vl] [![][cb]][cl]

[![][sb]][sl] [![][ab]][al]

[![][bb]][bl] [![][db]][dl] [![][eb]][el]

---

## The contribution process at a glance

1. Everyone contributing is governed by the [Code of Conduct](CODE_OF_CONDUCT.md) - please read it
2. After, please get familiar with project rules described in [CONTRIBUTING.md](CONTRIBUTING.md).
3. Make a contribution

:tada::tada::tada: That's all! :tada::tada::tada:

## Hands-on development notes

The information below guides you on different aspects of the development process. If you have
something which should be quickly available, please propose changes here.

**Table of contents**

- [Architecture](#architecture)
- [Development environment](#development-environment)
- [Checks and tests](#checks-and-tests)
- [Updating documentation](#updating-documentation)
- [Releasing](#releasing)

### Architecture

The structure of directories:

```text
.
├── .config/elegant-git/hooks/  <- the repo's own Elegant Git hooks
├── .config/mise/               <- tool versions and project tasks
├── cmd/                        <- Go CLI entrypoint
├── docs/                       <- user documentation (Jekyll + just-the-docs)
├── internal/                   <- Go packages (CLI commands, git, memory, …)
└── install.sh                  <- the installer published with every release
```

`eg …` runs the Go binary built from `cmd/eg`, which is part of the
`github.com/extsoft/elegant-git` module. Commands live under `internal/cli/` as an object-first
Cobra tree (for example `work start`, `repo clone`).

### Development environment

1. Install [mise](https://mise.jdx.dev/)
2. Run `mise run init` once after clone
3. Use `mise run build` to produce `dist/eg`. `mise run eg-preview` cross-compiles a Linux
   binary and opens a fresh Docker shell with `eg` on `PATH` (Git installed, empty `HOME`).

### Checks and tests

- Format and lint: `mise run fix` (CI runs `mise run check`)
- Go tests: `mise run test`

### Updating documentation

Edit Markdown files in [`docs/`](docs/). The site is organized as Home, Installation, Getting
started, the guide pages, Reference, and About. Installation, Reference, and About nest via
`parent` front matter of
[Just the Docs](https://github.com/just-the-docs/just-the-docs). Preview locally with
`mise run docs-preview` (serves at <http://localhost:4000>; override with `DOCS_PORT`). The site
deploys to GitHub Pages on pushes to `main`.

### Releasing

Releases are cut locally by a maintainer with [gh](https://cli.github.com/) installed. Versions
follow CalVer — `vYYYY.M.D` for a stable release and `vYYYY.M.D.hhmm-prerelease` for a preview.

- `mise run release:artifacts <version>` cross-compiles the binaries, archives them with `LICENSE`
  and `README.md`, writes a `.sha256` next to every archive, and copies `install.sh` into
  `dist/<version>/`
- `mise run release:github` asks for the release type, builds the tag, generates notes from the
  commits since the last stable tag, and publishes everything with `gh release create`

After a release, bump the Homebrew formula from `homebrew-tools` with
`mise run bump-elegant-git -- <tag>`.
