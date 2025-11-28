# Contributing to Stackinator

Contributions welcome! Please open an issue or PR.

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

### Run Docs Locally

The docs site uses [Docsify](https://docsify.js.org/). To preview locally:

```bash
# Using npx (no install needed)
npx serve docs

# Or with Python
python -m http.server 8000 --directory docs

# Or with PHP
php -S localhost:8000 -t docs
```

Then open http://localhost:8000 (or the port shown).

## Project Structure

- **`cmd/`**: Cobra CLI commands (root, new, status, sync, prune, etc.)
- **`internal/git/`**: Git operations wrapper with dry-run and verbose support
- **`internal/github/`**: GitHub CLI (`gh`) wrapper for PR operations
- **`internal/stack/`**: Core stack logic including topological sort and tree building
- **`internal/spinner/`**: Loading spinner for slow operations (disabled in verbose mode)

## Architecture

### Core Concepts

1. **Stack Tracking via Git Config**: Parent relationships are stored in git config as `branch.<name>.stackparent`. This is the single source of truth for stack structure.

2. **No External State**: Unlike other stack tools, Stackinator intentionally avoids state files, databases, or JSON files. Everything lives in git config.

3. **Three Main Operations**:
   - `stack new`: Create new branch and record parent in git config
   - `stack status`: Build tree from git config and display
   - `stack sync`: Topological sort, rebase each branch onto parent, update PR bases

### Key Algorithms

**Topological Sort** (`internal/stack/stack.go`):

- Builds dependency graph from parent relationships
- Performs Kahn's algorithm to order branches from base to tips
- Critical for `stack sync` to rebase in correct order

**Merged PR Detection** (`cmd/sync.go`):

- Fetches all PRs upfront for performance (cached in single API call)
- If parent PR is merged, updates child's parent to grandparent
- If branch's own PR is merged, removes from stack tracking

**Tree Building** (`internal/stack/stack.go`):

- Constructs visual tree from parent relationships
- Handles multiple independent stacks in same repo
- Used by `stack status` command

### Global Flags

Both `git` and `github` packages support:

- `DryRun`: Print what would happen without executing mutations
- `Verbose`: Show all git/gh commands being executed

## Testing

When testing git operations (creating branches, stashing, etc.), always use `./tests/test-repo` directory, NOT the main repository. This keeps the main repo clean and prevents pollution from test branches.

## Dependencies

- **cobra**: CLI framework
- **git**: Required in PATH
- **gh** (GitHub CLI): Required in PATH for PR operations
