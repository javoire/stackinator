# Go Stacked-PR CLI – High-Level Implementation Plan

Goal: Implement a Go CLI that manages **stacks of branches** and **syncs them to GitHub PRs**, similar to Graphite/Charcoal, with a focus on:

- **Minimal stack management** (derive stack from git config parent tracking)
- **One-command sync** (rebase cascading stacks, push branches, update PR bases)
- **Humans create PRs** - tool only keeps existing PRs in sync

---

## 0. Tech Choices & Overall Architecture

- **Language**: Go
- **CLI framework**: `spf13/cobra` (provides automatic --help generation)
- **Process execution**: call `git` and `gh` (GitHub CLI) via `os/exec`.
- **State storage**:
  - **Extremely minimal**: store parent branch in git config per branch
  - Example: `git config branch.st/feature-1.stackParent main`
  - No state files, no JSON, no complex persistence
  - Stack structure is derived on-demand from git config + git branch list
- **GitHub integration**:
  - Always shell out to `gh` CLI (re-use user auth, no REST APIs or Go libraries).

---

## 1. Core Domain Concepts & Invariants

Define these core types / concepts:

- **Stack**: A chain of branches where each branch depends on one parent branch.
  - Derived dynamically from git config, not stored explicitly.
- **Stack Branch**:
  - Any branch with `branch.<name>.stackParent` set in git config
  - Parent can be `main` or another stack branch
  - No stored metadata beyond parent relationship
- **Base branch**:
  - Typically `main` or whichever branch the stack ultimately targets.
- **Invariant assumptions**:
  - Parent relationship is the only tracked state
  - Branch names are unique within repo
  - Stack is reconstructed by walking parent relationships

---

## 2. Data Model (Minimal)

No persistent state files. All state is derived from:

1. **Git config**: `git config branch.<name>.stackParent <parent>`
2. **Git branches**: `git branch --list`
3. **GitHub state**: PR numbers/status fetched via `gh` when needed

Runtime representation:

```go
type StackBranch struct {
    Name   string
    Parent string  // from git config
    Exists bool    // from git branch list
}
```

Stack is built by:

- List all branches
- For each branch, check if `branch.<name>.stackParent` exists
- Build tree from parent relationships

---

## 3. Git Integration Layer

Wrap git commands using `os/exec`.

### Key Helpers:

- `GetRepoRoot()` → `git rev-parse --show-toplevel`
- `GetCurrentBranch()` → `git branch --show-current`
- `ListBranches()` → `git branch --list`
- `GetConfig(key)` → `git config --get <key>`
- `SetConfig(key, value)` → `git config <key> <value>`
- `CreateBranch(name, from)` → `git checkout -b <name> <from>`
- `CheckoutBranch(name)` → `git checkout <name>`
- `Rebase(onto)` → `git rebase <onto>`

---

## 4. GitHub Integration Layer

Simple wrapper for `gh` CLI operations:

```go
// All GitHub operations shell out to `gh` CLI
func GetPRForBranch(branch string) (prNumber int, state string, err error)
func UpdatePRBase(prNumber int, base string) error
```

Implementation always shells out to `gh`:

- `gh pr view <branch> --json number,state`
- `gh pr edit <number> --base <base>`

---

## 5. CLI Surface (Cobra)

Commands:

- `stack new <branch-name>` - Create new branch in stack
- `stack status` - Show current stack structure
- `stack up` - Move to parent branch
- `stack down` - Move to child branch (if exists)
- `stack sync` - Full sync: rebase all stacks cascading, push branches, update PR bases

Note: Humans create PRs manually (e.g., via `gh pr create`). The tool only updates bases of existing PRs.

### Global Flags

- `--help` / `-h` - All commands have help text
- `--dry-run` - All mutation commands (`new`, `sync`, `up`, `down`) show what would happen without executing
- `--verbose` / `-v` - Detailed output for debugging

---

## 6. Stack Discovery Logic

Build stack structure on-demand from git state:

```go
func GetStackBranches() []StackBranch {
    branches := git.ListBranches()
    var stackBranches []StackBranch

    for _, branch := range branches {
        parent := git.GetConfig("branch." + branch + ".stackParent")
        if parent != "" {
            stackBranches = append(stackBranches, StackBranch{
                Name: branch,
                Parent: parent,
            })
        }
    }
    return stackBranches
}
```

Walk the tree to find children, ancestors, etc.

---

## 7. Core Algorithms

All mutation commands respect `--dry-run` flag - print planned actions without executing.

### Creating new branch (`stack new <name>`)

1. Check working tree is clean
2. Determine parent = current branch (or `main` if not in stack)
3. `git checkout -b <name> <parent>`
4. `git config branch.<name>.stackParent <parent>`

### Full sync (`stack sync`)

Does everything: rebase cascading stacks, push branches, update PR bases

1. `git fetch origin`
2. Build stack structure from git config
3. Find all stacks from base branch (e.g., `main`)
4. For each branch in bottom→top order:
   - Check if parent PR is merged (via `gh pr view`)
   - If merged: update git config parent to the next level up (e.g., `main`)
   - Rebase branch onto its (possibly updated) parent: `git rebase <parent>` using --onto, account for e.g. squash merged commits.
   - Force push: `git push --force-with-lease origin <branch>`
   - Check if PR exists for this branch (via `gh pr view <branch>`)
   - If PR exists: update base with `gh pr edit <number> --base <parent>`
   - If no PR: skip (human will create PR when ready)

---

## 8. Config (Optional, Minimal)

Store repo-level config in git config if needed:

- `stack.baseBranch` (default: `main`)

Access via `git config stack.baseBranch` or use sensible defaults.

No config files needed.

---

## 9. Error Handling

- Surface git/gh errors clearly to user
- Exit codes: 0 = success, 1 = user error, 2 = system error
- `--dry-run` flag for all mutation commands - show planned actions without executing
  - Print what git commands would be run
  - Show what PRs would be updated
  - Prefix output with `[DRY RUN]`
- `--verbose` flag for debug output (show all git/gh commands being run)
- Validate clean working tree before destructive operations

---

## 10. Testing Strategy

- Unit tests for stack discovery logic
- Integration tests with temporary git repos
- Test key workflows: new → sync → restack

---

## 11. Packaging

Use `goreleaser`:

- Ship macOS/Linux/Windows binaries
- Provide Homebrew tap for easy install

---

## 12. Roadmap

**MVP**: `new`, `status`, `sync`
**v0.2**: `up`/`down` navigation
