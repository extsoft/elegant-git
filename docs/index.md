# About
Elegant Git is an assistant who carefully automates routine work with Git.

Main capabilities

- unified local full-cycle work management for any Git repository
- local flexible continuous integration (aka [workflows](#workflows))
- preserve current Git state prior to command execution and restore after (aka [pipes](#pipes))

Used philosophy

- actual work is allowed in custom branches only
- all pushes to `master` are strictly controlled
- all commands, which modify a state of Git, are printed
- declarative interaction matters

<center><iframe width="560" height="315" src="https://www.youtube.com/embed/Py6bpwJw30I"
frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope;
picture-in-picture" allowfullscreen></iframe></center>

> _Looks interesting? Go to [getting started](getting-started.md) guide or take a look for
available [commands](commands.md)._

Elegant Git has a rich test suit that executes on several Git and Bash versions. Please refer to
the table below to see the coverage matrix.

`bash --version`|`git --version`|`git elegant --version`
---|---|---
5.0.17|2.26.2|up to the latest
4.4.23|2.26.2|up to the latest
3.2.57|2.26.2|up to the latest
5.0.17|2.13.7|up to the latest
4.4.23|2.13.7|up to the latest
3.2.57|2.13.7|up to the latest

# Workflows
While developing something, it may be required to format code prior to committing modifications or
to open several URLs to report release notes after a new release. All these and similar actions,
which you're performing in addition to Git actions, are the **_workflows_**. And Elegant Git allows
automating them (it's like [Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks),
but for Elegant Git commands).

There are personal and common actions. The personal actions are stored in `.git/.workflows/`
directory while the common actions are located in `.workflows/` directory within the repository.
That's why there is an ability to split what should be configured for any contributor and what's for
you personally.

An action will be automatically triggered if an executable file is placed in a directory for either
personal or common actions. The executable file name has `<Git Elegant command>-<type>` pattern
where `<Git Elegant command>` is one of the [commands](commands.md) and `<type>` is either `ahead`
(runs prior to the command) or `after` (runs after to the command).

A sample workflow execution:
```bash
==>> git elegant save-work
.git/.workflows/save-work-ahead
.workflows/save-work-ahead
# the command itself
.git/.workflows/save-work-after
.workflows/save-work-after
```

❗Please take into that if an action returns a non-zero exit code, the execution of the workflow
will be interrupted.

If you want to skip workflows for the current command execution, just use `--no-workflows` option
like `git elegant --no-workflows save-work`.

# Pipes
There are a lot of situations when a current Git state needs to be reserved prior to Elegant Git
commands execution. For instance, you are working on something. And now, urgently, you need to
accept someone's critical work - `git-elegant accept-work some-critical-branch`. But there are
uncommitted modifications that need to be stashed prior to accepting work. From the other side, it
will be good to back into the previous working branch and restore modification from the stash when
the needed work is accepted. That's why there are **pipes** which are doing preserve and restore
automatically.

There are the following pipes:

- **branch pipe** which preserves and restores current branch
- **stash pipe** which preserves and restores uncommitted changes

A command can use one or several pipes at the same time (see command help for the details).
And if a "piped" command is used, each pipe stores the state in per-repo command memory (`.git/elegant-git/commands.json`), runs the original command, and restores saved state if the command is
successful. If the command is failed and it reruns, the pipes do not preserve the state again but
will restore the initial preserved state if the command is successful.

# Known limitations
Support only one default remote - `origin`.

# Exit codes
Usually, Elegant Git translates exit codes of original Git commands. However, in some cases,
it may face its own errors and raises them as

- `0` - a successful execution
- `42` - a philosophical constraint
- `43` - a logical constraint
- `45` - a required parameter is empty
- `46` - an unknown command
