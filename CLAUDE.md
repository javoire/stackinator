# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Stackinator is a minimal CLI tool for managing stacked branches and syncing them to GitHub Pull Requests. It uses git config to track parent-child relationships between branches, with no external state files.

## Build and Development Commands

```bash
# Build binary
go build -o stack

# Or use convenience scripts
./scripts/build          # Build binary
./scripts/install        # Build and symlink to ~/bin
./scripts/test           # Run tests
./scripts/clean          # Clean build artifacts

# Run tests
go test ./...

# Run the tool locally
./stack <command>
```

## Architecture

### Core Concepts

1. **Stack Tracking via Git Config**: Parent relationships are stored in git config as `branch.<name>.stackparent`. This is the single source of truth for stack structure.

2. **No External State**: Unlike other stack tools, Stackinator intentionally avoids state files, databases, or JSON files. Everything lives in git config.

3. **Three Main Operations**:
   - `stack new`: Create new branch and record parent in git config
   - `stack status`: Build tree from git config and display
   - `stack sync`: Topological sort, rebase each branch onto parent, update PR bases

### Package Structure

- **`cmd/`**: Cobra CLI commands (root, new, status, sync)
- **`internal/git/`**: Git operations wrapper with dry-run and verbose support
- **`internal/github/`**: GitHub CLI (`gh`) wrapper for PR operations
- **`internal/stack/`**: Core stack logic including topological sort and tree building
- **`internal/spinner/`**: Loading spinner for slow operations (disabled in verbose mode)

### Key Algorithms

**Topological Sort** (`internal/stack/stack.go:TopologicalSort`):

- Builds dependency graph from parent relationships
- Performs Kahn's algorithm to order branches from base to tips
- Critical for `stack sync` to rebase in correct order

**Merged PR Detection** (`cmd/sync.go:runSync`):

- Fetches all PRs upfront for performance (cached in single API call)
- If parent PR is merged, updates child's parent to grandparent
- If branch's own PR is merged, removes from stack tracking

**Tree Building** (`internal/stack/stack.go:BuildStackTree`):

- Constructs visual tree from parent relationships
- Handles multiple independent stacks in same repo
- Used by `stack status` command

### Global Flags

Both `git` and `github` packages support:

- `DryRun`: Print what would happen without executing mutations
- `Verbose`: Show all git/gh commands being executed

The `spinner` package also respects the verbose flag:

- `Enabled`: When false (verbose mode), spinners are hidden to avoid visual conflicts with command output

These are set via persistent flags on root command.

## Dependencies

- **cobra**: CLI framework
- **git**: Required in PATH
- **gh** (GitHub CLI): Required in PATH for PR operations

## Configuration

Base branch can be configured per-repo:

```bash
git config stack.baseBranch develop  # Default is "main"
```

## Testing

Currently no test files exist. When adding tests:

- Use table-driven tests for topological sort and tree building
- Mock git/gh command execution for unit tests
- Consider integration tests that use temporary git repos

**IMPORTANT**: When testing git operations (creating branches, stashing, etc.), always use `./tests/test-repo` directory, NOT the main repository. This keeps the main repo clean and prevents pollution from test branches.

## Key Implementation Details

1. **Stash Handling**: `stack sync` auto-stashes uncommitted changes using `git rebase --autostash` and manual stash/pop for safety.

2. **Force Push Safety**: Uses `--force-with-lease` to prevent overwriting remote changes from other sources.

3. **PR Base Updates**: When parent relationships change (due to merged PRs), PR bases are automatically updated via `gh pr edit --base`.

4. **Error Handling**: Commands return early on git errors. For rebase conflicts, user must resolve manually and re-run `stack sync`.

5. **Branch Existence**: Parent branches may not exist locally if they were merged and deleted. Code handles missing parents gracefully.
