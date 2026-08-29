[help-wanted]: https://github.com/extsoft/elegant-git/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22+sort%3Acomments-desc+no%3Aassignee
[by-reactions]: https://github.com/extsoft/elegant-git/issues?q=is%3Aissue+is%3Aopen+sort%3Areactions-%2B1-desc+no%3Aassignee
[dev-notes]: README.md#hands-on-development-notes
[open-issues]: https://github.com/extsoft/elegant-git/issues
[new-issue]: https://github.com/extsoft/elegant-git/issues/new/choose
[commit-sample]: https://github.com/extsoft/elegant-git/commit/1855df5881a4c80f78b0a189342f5003628e0db8
[status-checks]: https://help.github.com/articles/about-status-checks/
[calver]: https://calver.org/

# Contribution guide for Elegant Git
:+1::tada: First off, thanks for taking the time to contribute! :tada::+1:

The content here explains all important aspects of any type of contribution. In addition, there is
["Hands-on development notes" section in README.md][dev-notes] which provides frequently-used
information for maintainers and contributors.

**The two main statements**
1. By contributing, you fully agree with  [the Code of Conduct](CODE_OF_CONDUCT.md) as this project
and everyone participating in it is governed by this document.
2. By contributing, you agree that your contributions will be licensed under
its [MIT License](LICENSE).

**Table of contents**

- [Issue contribution](#issue-contribution)
- [Issue management](#issue-management)
- [Release management](#release-management)
- [Documentation management](#documentation-management)
- [Code contribution](#code-contribution)
  - [Selecting an issue](#selecting-an-issue)
  - [I have some doubts about the issue](#i-have-some-doubts-about-the-issue)
  - [Making development](#making-development)
  - [Committing the changes](#committing-the-changes)
  - [Submitting a pull request](#submitting-a-pull-request)

## Issue contribution
First look at [the open issues][open-issues] as there is a chance that someone has similar problem.

If you've found such issue, please add :+1: reaction. This will up issue's priority in general rank
(see "[Selecting An Issue](#selecting-an-issue)" section for the details).

If you're unable to find an open issue addressing the problem, open [a new one][new-issue]. The
project provides several templates for improving your experience. Please choose one of them and
follow the instruction.

## Issue management
A newly created issue has two automatic labels. The first one identifies the issue type (`bug`,
`chore`, `feature`). The second label is `finalize requirements` - it means the issue requires
review from a maintainer in order to collect any additional information if required. After the
clarifications, this label is removed and the issue is available for the implementation.

Additionally, a maintainer can mark the issue with `help wanted` label which means it's good enough
for the first contribution.

## Release management
The project is released based on a decision of the maintainers or a request from the community (a
comment in a pull request or an issue). And [calendar versioning][calver] is used as a control
mechanism of version numbers: `vYYYY.M.D` for a stable release and `vYYYY.M.D.hhmm-prerelease` for
a preview one. Such release approach requires ready-to-release contributions within a single pull
request. This allows releasing the `main` branch at any point of time.

A maintainer publishes a release locally with `mise run release:github`, which builds the
artifacts, generates the notes from the commits since the last stable tag, and creates the GitHub
release. See ["Releasing" section in README.md](README.md#releasing) for the details.

## Documentation management
All project-related documentation is stored along with source code. It is
- `docs` directory for user documentation, published to <https://elegant-git.extsoft.pro/> and
organized as Home, Installation, Getting started, the guide pages, Reference, and About
- `*.md` files in the root of the repository for the development documentation
- help messages and comments within source code

This means that if you are doing any change, you must update related documentation also. The
changes, which make documentation obsolete, won't be accepted.

## Code contribution
The contributions won't be accepted if they aren't related to previously created issues. This rule
saves you and maintainers a lot of time by rejecting unwanted changes during discussions.

### Selecting an issue
If you are a beginner, you can start by looking through ["Help wanted" issues filter][help-wanted],
otherwise, select from [all issues filter][by-reactions].

> Both filters are sorted by a total number of reactions. While not perfect, the number of reactions
is a reasonable proxy for the impact a given change will have.

Also, in some cases, an issue can be assigned to a maintainer. In this case, it's not available for
external contributions.

Have chosen something? Please leave a comment with an estimate for that issue which indicates your
willingness like
```text
I'm going to solve this by the end of this week.
```
In such a way it's clear who works on the issue (Github allows to assign an issue only for the
maintainers) and when expect some progress.

### I have some doubts about the issue
If yes, please use issue's comments for discussion. It's better having a discussion prior to an
implementation or a pull request review - it saves time for everyone.

### Making development
["Hands-on development notes" section in README.md][dev-notes] should cover any development-related
questions. If no, you are welcome to create [an issue](#issue-contribution) and request what you
need.

### Committing the changes
Please use the following  template when writing a commit message
```text
Describe a change in the subject line (here)

This is the body. Leave here `why` answers about the changes. You can put here
as many worlds as wish.

The final part, please mention the issue number (see the last line). Please
either use
    https://elegant-git.extsoft.pro
or run
    git config --local core.commentChar |
for stating commit message lines with # symbol.

#123
```

Also, don't forget to
* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less

You can use [this commit][commit-sample] as a reference commit message.

### Submitting a pull request
A newly created pull request will be prepopulated from a template. You have to
update issue number (if it affects several issues, please specify all) and
complete checklist. After, feel free to add more comments if you want.

A pull requests must be targeted into the `main` branch (a default one).

Once the pull request is created, please make sure [status checks][status-checks] are passing.

> What if the status checks are failing? If a status check is failing, and you believe that the
failure is unrelated to your change, please leave a comment on the pull request explaining why you
believe the failure is unrelated. A maintainer will re-run the status check for you. If we conclude
that the failure was a false positive, then we will open an issue to track that problem with our
status check suite.
