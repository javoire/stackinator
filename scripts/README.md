# Development Scripts

Convenience scripts for building, testing, and installing stackinator.

## Scripts

### `./scripts/build`

Builds the `stack` binary in the project root.

```bash
./scripts/build
```

### `./scripts/install`

Builds the binary and creates a symlink in `~/bin/stack`. This allows you to run `stack` from anywhere.

```bash
./scripts/install
```

**What it does:**
- Runs `./scripts/build`
- Creates `~/bin` if it doesn't exist
- Symlinks `~/bin/stack` → `<project>/stack`
- Warns if `~/bin` is not in your PATH

### `./scripts/test`

Runs all Go tests with verbose output.

```bash
./scripts/test
```

### `./scripts/clean`

Removes build artifacts (the `stack` binary and Go build cache).

```bash
./scripts/clean
```

## Usage

All scripts are designed to be run from any directory:

```bash
# From project root
./scripts/build

# From anywhere (if in git repo)
git rev-parse --show-toplevel | xargs -I {} {}/scripts/build
```

## Development Workflow

```bash
# One-time setup
./scripts/install

# After making changes
./scripts/build
# Changes are immediately available via the symlink

# Run tests
./scripts/test

# Clean up
./scripts/clean
```

