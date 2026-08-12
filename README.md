[sb]: https://img.shields.io/badge/Choose%20issue-simple-green
[sl]: https://github.com/bees-hive/elegant-git/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22+sort%3Acomments-desc+no%3Aassignee
[ab]: https://img.shields.io/badge/Choose%20issue-any-blue
[al]: https://github.com/bees-hive/elegant-git/issues?q=is%3Aissue+is%3Aopen+sort%3Areactions-%2B1-desc+no%3Aassignee
[cb]: https://img.shields.io/github/commits-since/bees-hive/elegant-git/latest?label=Commits%20for%20next%20release
[cl]: https://github.com/bees-hive/elegant-git/commits/master
[vb]: https://img.shields.io/github/v/tag/bees-hive/elegant-git?label=Last%20release
[vl]: https://github.com/bees-hive/elegant-git/releases/latest
[lb]: https://img.shields.io/github/license/bees-hive/elegant-git
[bb]: https://github.com/bees-hive/elegant-git/workflows/Quality%20pipeline/badge.svg
[bl]: https://github.com/bees-hive/elegant-git/actions?workflow=Quality+pipeline
[db]: https://readthedocs.org/projects/elegant-git/badge/?version=latest
[dl]: https://elegant-git.bees-hive.org/en/latest/?badge=latest
[eb]: https://img.shields.io/badge/assistant-Elegant%20Git-000000.svg
[el]: https://github.com/bees-hive/elegant-git

# Elegant Git
Elegant Git is an assistant who carefully automates routine work with Git.

Please visit <https://elegant-git.bees-hive.org/> to get started with user documentation or
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

### Architecture

The structure of directories:
```text
.
├── .config/mise/  <- tool versions and project tasks
├── .workflows/    <- docs helpers and repo hook stubs
├── cmd/           <- Go CLI entrypoint
├── docs/          <- user documentation
└── internal/      <- Go packages (CLI commands, git, memory, …)
```

`git elegant …` runs the Go binary built from `cmd/git-elegant`. Commands live under
`internal/cli/` as an object-first Cobra tree (for example `work start`, `repo clone`).

### Development environment
1. Install [mise](https://mise.jdx.dev/)
2. Run `mise run init` once after clone
3. Use `mise run build` to produce `dist/git-elegant`

### Checks and tests
- Format and lint: `mise run fix` (CI runs `mise run check`)
- Go tests: `mise run test`

### Updating documentation
Build the binary (`mise run build`), then generate command docs with
`.workflows/docs/docs-workflows.bash generate` (or `python .workflows/docs/docs.py`).
Preview the site with `python -m mkdocs serve`.
All other files in ["docs" directory](docs/) require manual corrections.
