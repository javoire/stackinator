# Decision Log

Architectural and design decisions for Stackinator.

## 2026-05-20 — Add `--all` flag to `stack sync`

**Decision**: Allow `stack sync --all` to sync the full stack, not just the ancestor chain.

**Context**: The default `stack sync` only processes the ancestor chain from the base branch up to the current branch. Children below the current branch are not synced. Users had to check out a leaf branch or run sync multiple times to propagate changes through the full stack.

**Resolution**: Added `--all` flag that finds the root of the current branch's stack (first stack branch in the ancestor chain), then uses `GetDescendants` (BFS through children map) to collect all branches in the stack. The per-branch processing loop is unchanged — topological sort ensures correct ordering, and each branch rebases onto its configured parent.

## 2026-05-07 — Add `--cross-worktree` flag to `stack sync`

**Decision**: Allow `stack sync` to rebase branches checked out in other worktrees via `git -C <path>`, gated behind `--cross-worktree`.

**Context**: `stack sync` skipped branches checked out in other worktrees because rebasing requires operating on the worktree's working directory. Users with stacks spread across worktrees had to manually sync each worktree.

**Resolution**: Added `WithDir(path) GitClient` to the git client interface, which returns a new client that prepends `git -C <path>` to all commands. When `--cross-worktree` is passed, sync creates a dir-scoped client per cross-worktree branch and uses it for working-tree operations (rebase, cherry-pick, reset). Checkout is skipped since the branch is already checked out in the target worktree. Ref-only operations (push, fetch, commit hash) use the main client. Conflict resolution messages include the worktree path so users know where to `cd`. The flag is opt-in because rebasing in another worktree modifies that worktree's working directory.

## 2026-03-10 — Add `stack switch` command for worktree navigation

**Decision**: Add a `switch` command that outputs `cd <path>` for quick worktree navigation.

**Context**: Users work across multiple worktrees and the main repo. `stack worktree --list` shows paths but requires manual copy-paste. A faster way to jump between worktrees was needed.

**Resolution**: `stack switch <branch>` resolves a branch to its worktree path and prints `cd <path>`. No-args mode shows an interactive picker. `--init` outputs a shell function `ss()` that wraps the command with actual `cd`. `--install` appends the init line to the shell config. The command reuses existing helpers (`getWorktreesBaseDir`, `pathWithinDir`, `GetWorktreeBranches`).

## 2026-03-06 — Add merged-parent detection to `stack status`

**Decision**: Add merged-parent detection to `stack status`.

**Context**: `stack status` reported "perfectly synced" even when a parent branch's PR had been merged. `stack sync` already had this detection logic (checking PR state + git ancestor fallback) but `status` did not, so users had no warning until they ran sync.

**Resolution**: Mirror `sync.go`'s merged-parent detection (PR state check + git ancestor fallback) in `status.go:detectSyncIssues()`. This lets `status` report when a parent branch has been merged and the child needs re-parenting.
