# Commands

## `stack new <branch-name> [parent]`

Create a new branch in the stack, optionally specifying a parent branch.

The new branch will be created from the specified parent (or current branch if not specified), and the parent relationship will be stored in git config.

```bash
# Create a stack: main <- A <- B <- C
stack new A main                         # A based on main
stack new B                              # B based on current (A)
stack new C                              # C based on current (B)

# Preview without creating
stack new feature-xyz --dry-run
```

## `stack status`

Display the stack structure as a tree, showing branch hierarchy, current branch (marked with `*`), and PR status.

```bash
stack status

# Show without PR info (faster)
stack status --no-pr
```

Flags:

- `--no-pr` - Skip fetching PR information (faster)

## `stack sync`

Perform a full sync of the stack:

1. Fetch latest changes from origin
2. Rebase each stack branch onto its parent (in bottom-to-top order)
3. Force push each branch to origin
4. Update PR base branches to match the stack (if PRs exist)

```bash
# Sync all branches and update PRs
stack sync

# Preview what would happen
stack sync --dry-run

# Force push even if branches have diverged
stack sync --force
```

Flags:

- `--force`, `-f` - Use `--force` instead of `--force-with-lease` for push (bypasses safety checks)

## `stack parent`

Display the parent branch of the current branch in the stack.

```bash
stack parent
```

## `stack prune`

Remove branches with merged PRs from stack tracking and delete them locally.

```bash
# Clean up merged stack branches
stack prune

# Clean up all merged branches (including non-stack branches)
stack prune --all

# Force delete even if branches have unmerged commits
stack prune --force

# Preview what would be deleted
stack prune --dry-run
```

Flags:

- `--all`, `-a` - Check all local branches, not just stack branches
- `--force`, `-f` - Force delete branches even if they have unmerged commits

## `stack rename <new-name>`

Rename the current branch while preserving all stack relationships.

This command will rename the git branch, update the branch's parent reference, and update all child branches to point to the new name.

```bash
# Rename current branch
stack rename feature-improved-name

# Preview without making changes
stack rename feature-improved-name --dry-run
```

## `stack reparent <new-parent>`

Change the parent branch of the current branch in the stack.

Updates the stack parent relationship and, if a PR exists, automatically updates the PR base to match the new parent.

```bash
# Change current branch to be based on a different parent
stack reparent feature-auth

# Preview what would happen
stack reparent main --dry-run
```

## `stack worktree <branch-name> [base-branch]`

Create a git worktree in the `.worktrees/` directory for the specified branch.

If the branch exists locally or on the remote, it will be used. If the branch doesn't exist, a new branch will be created from the current branch (or from base-branch if specified) and stack tracking will be set up automatically.

```bash
# Create worktree for new branch (from current branch, with stack tracking)
stack worktree my-feature

# Create worktree from a fresh main branch
stack worktree my-feature main

# Create worktree for existing local or remote branch
stack worktree existing-branch

# Clean up worktrees for merged branches
stack worktree --prune
```

Flags:

- `--prune` - Remove worktrees for branches with merged PRs

## `stack version`

Print version information.

```bash
stack version
```

## Global Flags

These flags are available on all commands:

- `--dry-run` - Show what would happen without executing
- `--verbose`, `-v` - Show detailed output
