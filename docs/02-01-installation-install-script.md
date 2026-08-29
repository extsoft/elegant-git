---
layout: default
title: Using install script
parent: Installation
nav_order: 1
permalink: /installation/install-script/
---

# Using install script

The installer downloads the latest release for your OS and architecture, verifies its `.sha256`
checksum, and copies the executable into the first directory that already exists and is writable —
`~/.local/bin`, then `~/bin`, then the system ones. So, on most machines the one-liner is all you
need:

```bash
curl -fsSL https://github.com/extsoft/elegant-git/releases/latest/download/install.sh | sh
```

If you want to decide the details yourself, download the script and pass it options:

- `-d <dir>` installs into `<dir>` instead of the auto-detected one — the `BINDIR` environment
  variable does the same
- `-v <tag>` installs a specific release instead of the latest one
- `-x` turns on debug logging
- `-h` prints the usage

```bash
curl -fsSLO https://github.com/extsoft/elegant-git/releases/latest/download/install.sh
sh install.sh -d ~/.local/bin
```

The installer prints an `export PATH=…` line when the chosen directory is not on your `PATH` yet.
It does **not** create the directory for you, so `mkdir -p ~/.local/bin` first if you have never
used it. When none of the candidates exists it asks where to install; with no terminal to ask on it
stops and tells you to pass `-d`.
