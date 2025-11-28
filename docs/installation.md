# Installation

## Prerequisites

- [Git](https://git-scm.com/)
- [GitHub CLI (`gh`)](https://cli.github.com/) — required for PR operations

## Homebrew (macOS/Linux)

The recommended installation method:

```bash
brew install javoire/tap/stackinator
```

This installs the `stack` command and keeps it updated with `brew upgrade`.

## Go Install

If you have Go installed:

```bash
go install github.com/javoire/stackinator@latest
```

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your PATH.

## Download Binary

Download pre-built binaries from the [releases page](https://github.com/javoire/stackinator/releases).

Available for:
- macOS (Apple Silicon & Intel)
- Linux (arm64 & x86_64)
- Windows (x86_64)

After downloading, extract and move to a directory in your PATH:

```bash
# Example for macOS ARM
tar -xzf stackinator_Darwin_arm64.tar.gz
sudo mv stack /usr/local/bin/
```

## Build from Source

```bash
git clone https://github.com/javoire/stackinator.git
cd stackinator

# Quick install (builds and symlinks to ~/bin)
./scripts/install

# Or build manually
go build -o stack
```

## Verify Installation

```bash
stack version
```

