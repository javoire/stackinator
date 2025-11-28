# Stackinator

A minimal CLI tool for managing stacked branches and syncing them to GitHub Pull Requests, inspired by tools like [Charcoal](https://github.com/danerwilliams/charcoal) and [Graphite](https://graphite.dev/).

## Features

- 🪜 **Stack Management**: Create and manage chains of dependent branches
- 🔄 **One-Command Sync**: Rebase all branches, push changes, and update PR bases automatically
- 📊 **Visual Status**: See your stack structure at a glance
- 🎯 **Minimal State**: Uses git config to track parent relationships - no extra files or databases
- 🔧 **Simple Integration**: Works with standard git and GitHub CLI (`gh`)

## Installation

### Prerequisites

- [Git](https://git-scm.com/)
- [GitHub CLI (`gh`)](https://cli.github.com/)

### Homebrew (macOS/Linux)

```bash
brew install javoire/tap/stackinator
```

See [Alternative Installation Methods](docs/alternative-installations.md) for other options (Go install, binary download, build from source).

## Quick Start

### 1. Create a Stack

Start from your base branch (e.g., `main`):

```bash
# Create first branch in stack
stack new feature-1

# Make some changes and commit
git add .
git commit -m "Add feature 1"

# Create second branch stacked on feature-1
stack new feature-2

# Make more changes
git add .
git commit -m "Add feature 2"
```

### 2. View Your Stack

```bash
stack status

 main
  |
 feature-1 [https://github.com/you/repo/pull/1 :open]
  |
 feature-2 [https://github.com/you/repo/pull/2 :open] *
```

The `*` indicates your current branch.

### 3. Sync Everything

After making changes or when the base branch is updated:

```bash
stack sync
```

This will:

- Fetch latest changes from origin
- Rebase each branch onto its parent (in order)
- Force (with lease) push all branches
- Update PR base branches to match the stack
- Handle merged parent PRs automatically

## Commands

See [Commands Reference](docs/commands.md) for full documentation.

- `stack new <branch-name>` - Create a new branch in the stack
- `stack status` - Display the current stack structure
- `stack sync` - Sync all branches and update PRs
- `stack parent` - Show the parent of the current branch
- `stack prune` - Clean up branches with merged PRs
- `stack rename <new-name>` - Rename branch preserving stack relationships
- `stack reparent <new-parent>` - Change the parent of the current branch
- `stack worktree <branch-name>` - Create a worktree for a branch

## Documentation

- [How It Works](docs/how-it-works.md) - Stack tracking and sync algorithm
- [Configuration](docs/configuration.md) - Customizing Stackinator
- [Examples](docs/examples.md) - Workflow examples and tips
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## License

MIT

## Contributing

Contributions welcome! See [Contributing Guide](docs/contributing.md) for development setup and guidelines.

## Acknowledgments

Inspired by:

- [Charcoal](https://github.com/danerwilliams/charcoal)
- [Graphite](https://graphite.dev/)
- [git-stack](https://github.com/gitext-rs/git-stack)
