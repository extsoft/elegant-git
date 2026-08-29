---
layout: default
title: Using mise
parent: Installation
nav_order: 2
permalink: /installation/mise/
---

# Using mise

If you already manage your tools with [mise](https://mise.jdx.dev/), let it own the installation
and the updates:

```bash
mise use --global "github:extsoft/elegant-git@latest"
```

The `--global` modifier records Elegant Git in your global mise configuration, so it is available
in every directory. Drop it to pin the tool to the current project instead, which writes the entry
into that project's `mise.toml`.

To pick up a newer release:

```bash
mise upgrade github:extsoft/elegant-git
```

That installs the newest version that still matches the pin in your config — so `@latest`
keeps moving. If you pinned a specific tag, add `--bump` so mise rewrites the pin as well.
