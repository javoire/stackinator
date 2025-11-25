# Stackinator

A minimal CLI tool for managing stacked branches and syncing them to GitHub Pull Requests, inspired by tools like Graphite and Sapling.

## Features

- 🪜 **Stack Management**: Create and manage chains of dependent branches
- 🔄 **One-Command Sync**: Rebase all branches, push changes, and update PR bases automatically
- 📊 **Visual Status**: See your stack structure at a glance
- 🎯 **Minimal State**: Uses git config to track parent relationships - no extra files or databases
- 🔧 **Simple Integration**: Works with standard git and GitHub CLI (`gh`)

## Installation

### Prerequisites

- [Go](https://golang.org/doc/install) 1.21 or later
- [Git](https://git-scm.com/)
- [GitHub CLI (`gh`)](https://cli.github.com/)

### Install from Source

```bash
go install github.com/javoire/stackinator@latest
```

Or clone and build locally:

```bash
git clone https://github.com/javoire/stackinator.git
cd stackinator

# Quick install (builds and symlinks to ~/bin)
./scripts/install

# Or build manually
./scripts/build
```

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
```

Output:

```
 main
  |
 feature-1
  |
 feature-2 *
```

The `*` indicates your current branch.

### 3. Create Pull Requests

Create PRs manually using `gh`:

```bash
# Switch to first branch and create PR
git checkout feature-1
gh pr create --base main --title "Feature 1" --body "Description"

# Switch to second branch and create PR
git checkout feature-2
gh pr create --base feature-1 --title "Feature 2" --body "Description"
```

### 4. Sync Everything

After making changes or when the base branch is updated:

```bash
stack sync
```

This will:

- Fetch latest changes from origin
- Rebase each branch onto its parent (in order)
- Force push all branches
- Update PR base branches to match the stack
- Handle merged parent PRs automatically

## Commands

### `stack new <branch-name>`

Create a new branch in the stack, using the current branch as parent.

```bash
stack new my-feature
```

Options:

- `--dry-run`: Show what would happen without executing
- `--verbose`: Show detailed output

### `stack status`

Display the current stack structure as a tree.

```bash
stack status
```

Shows:

- Branch hierarchy with visual tree
- Current branch (marked with `*`)
- PR information (number and state)

### `stack sync`

Sync all stack branches with their parents and update PRs.

```bash
stack sync
```

Options:

- `--dry-run`: Preview actions without executing
- `--verbose`: Show all git/gh commands being run

## How It Works

### Stack Tracking

Stackinator stores the parent of each branch in git config:

```bash
# View parent of current branch
git config branch.feature-1.stackParent

# Manually set parent (if needed)
git config branch.feature-1.stackParent main
```

This minimal approach means:

- No state files to manage
- No database or JSON files
- Works with standard git workflows
- Easy to inspect and debug

### Sync Algorithm

When you run `stack sync`, Stackinator:

1. Fetches from origin
2. Discovers all stack branches from git config
3. Sorts them in topological order (base to tips)
4. For each branch:
   - Checks if parent PR was merged (updates parent if so)
   - Rebases onto parent
   - Force pushes to origin
   - Updates PR base branch (if PR exists)

### PR Management

**Important**: Stackinator does NOT create PRs for you. You create PRs manually using `gh pr create` or the GitHub web interface.

Stackinator will:

- ✅ Update PR base branches when stack changes
- ✅ Detect when parent PRs are merged
- ✅ Show PR status in `stack status`

## Configuration

### Base Branch

By default, Stackinator uses `main` as the base branch. To change it:

```bash
git config stack.baseBranch develop
```

## Examples

### Example Workflow

```bash
# Start from main
git checkout main
git pull

# Create feature branch
stack new auth-system
# ... make changes, commit ...

# Create sub-feature 1
stack new auth-login
# ... make changes, commit ...

# Create sub-feature 2
git checkout auth-system  # go back to parent
stack new auth-logout
# ... make changes, commit ...

# View structure
stack status
# Output:
#  main
#   |
#  auth-system
#   |
#  auth-login
#   |
#  auth-logout *

# Create PRs
git checkout auth-system
gh pr create --base main --title "Auth System" --body "..."

git checkout auth-login
gh pr create --base auth-system --title "Add login" --body "..."

git checkout auth-logout
gh pr create --base auth-system --title "Add logout" --body "..."

# Later, after making changes or when main updates
stack sync
```

### Dry Run

Preview what sync would do:

```bash
stack sync --dry-run
```

### Verbose Output

See all git/gh commands being executed:

```bash
stack sync --verbose
```

## Troubleshooting

### Rebase Conflicts

If `stack sync` encounters a rebase conflict:

1. Resolve the conflict manually
2. Run `git rebase --continue`
3. Run `stack sync` again to continue with remaining branches

### Orphaned Branches

If you delete a parent branch, child branches become orphaned. To fix:

```bash
# Update the child's parent to point to the grandparent
git config branch.child-branch.stackParent main
```

### Remove from Stack

To remove a branch from the stack (but keep the branch):

```bash
git config --unset branch.my-branch.stackParent
```

## Development

### Scripts

The project includes several convenience scripts in the `scripts/` directory:

```bash
# Build the binary
./scripts/build

# Build and install (symlink to ~/bin)
./scripts/install

# Run tests
./scripts/test

# Clean build artifacts
./scripts/clean
```

### Manual Build

```bash
go build -o stack
```

### Run Tests

```bash
go test ./...
```

## Roadmap

- [x] Core commands: `new`, `status`, `sync`
- [ ] Navigation commands: `up`, `down`
- [ ] Better conflict handling
- [ ] Integration tests
- [ ] Homebrew formula
- [ ] Pre-built binaries via goreleaser

## License

MIT

## Contributing

Contributions welcome! Please open an issue or PR.

## Acknowledgments

Inspired by:

- [Graphite](https://graphite.dev/)
- [Sapling](https://sapling-scm.com/)
- [git-stack](https://github.com/gitext-rs/git-stack)
