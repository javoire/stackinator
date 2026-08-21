# Decision Log

Architectural and design decisions for Stackinator.

## 2026-08-17 — Generate a randomized worktree when no branch is provided

**Decision**: Allow `stack worktree` with no arguments to create a new branch and worktree using a randomized `worktree-<16 hex characters>` name.

**Context**: Creating a disposable worktree required inventing and typing a branch name even when the name itself was unimportant.

**Resolution**: Made the branch-name argument optional. No-argument invocation generates the name with the operating system's cryptographic random source, then follows the existing new-branch worktree flow so the current branch is recorded as its stack parent. Explicit branch names and worktree management flags retain their existing behavior.

## 2026-08-12 — Bound and separate sync network operations

**Decision**: Give git fetches a five-minute timeout and GitHub CLI operations a 30-second timeout. Display fetch and PR loading as separate sync progress steps, and propagate PR lookup failures.

**Context**: `stack sync` grouped an unbounded background `git fetch` and unbounded parallel `gh pr view` calls under one spinner. A stalled remote, credential helper, or GHE request could therefore hang forever at `Fetching from origin and loading PRs...`, and GitHub errors were silently interpreted as missing PRs.

**Resolution**: Run fetch and GitHub subprocesses with context deadlines and a bounded pipe wait. Wait for fetch and PR loading in distinct spinner steps so the active dependency is visible. Return PR lookup failures to callers while preserving the normal no-PR result.

## 2026-08-04 — Distribute the Codex skill through a plugin marketplace

**Decision**: Publish the existing Stackinator skill as a Codex plugin from the Stackinator repository.

**Context**: `stack skill install` copied `SKILL.md` directly into `~/.agents/skills/stack`, while Claude Code installed the same skill from the repository marketplace. Direct copies could drift from the CLI and lacked Codex plugin lifecycle support.

**Resolution**: Add a repository-local Codex marketplace and plugin manifest around the shared `plugins/stack/skills/stack/SKILL.md`. Change the Codex installer to register `javoire/stackinator` and install `stack@stackinator`; keep the existing Claude marketplace and Cursor rule installation.

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
