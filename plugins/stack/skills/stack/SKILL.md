---
name: stack
description: Manage stacked branches with the stack CLI. Covers branch creation, navigation, syncing, and PR management.
---

The `stack` CLI manages stacked branches and syncs them to GitHub PRs. Use this for working with dependent branches.

## Choosing Between `stack worktree` vs `stack new`

- **`stack worktree <branch>`** - Creates a separate git worktree directory at `~/.stack/worktrees/<repo>/<branch>`. Use when:
  - Starting a new feature from master
  - Want to work on multiple things in parallel without stashing
  - The current worktree has uncommitted changes you want to keep

- **`stack new <branch>`** - Creates a branch in the current worktree. Use when:
  - Building on top of the current feature branch (stacked PRs)
  - Already in a worktree and want to add dependent branches

## Common Commands

- `stack new <branch>` - Create new branch in the stack (use instead of `git checkout -b`)
- `stack status` - Show stack structure with PR status (fetches from GitHub)
- `stack show` - Show local stack structure (fast, no network)
- `stack sync` - Sync branches and update PRs on GitHub
- `stack up` / `stack down` - Navigate up/down the stack
- `stack prune` - Clean up merged branches
- `stack reparent <parent>` - Change the parent branch
- `stack rename <name>` - Rename the current branch
- `stack worktree` - Create a git worktree for parallel work

## Workflow

1. Start a new feature from master: `stack worktree feature-name`
   - Or use `stack new feature-name` if building stacked PRs on a feature branch
2. Make changes and commit
3. Sync to GitHub: `stack sync` (creates/updates PR)
4. Check status: `stack status`
5. After merge: `stack prune` to clean up

Run `stack --help` for full documentation.
