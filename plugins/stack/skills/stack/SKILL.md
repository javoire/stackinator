---
name: stack
description: Manage stacked Git branches with the stack CLI. Use when creating or navigating stack branches and worktrees, changing parent relationships, syncing branches and pull-request bases, recovering an interrupted sync, or pruning merged branches.
---

The `stack` CLI stores branch parent relationships in Git config and syncs them with GitHub pull requests.

## Create branches

- Use `stack new <branch> [parent]` to create and check out a branch in the current worktree. Without an explicit parent, it uses the current stack branch or the configured base branch.
- Use `stack worktree <branch> [base]` to create a separate worktree under `~/.stack/worktrees/<repo>`. For a fresh branch from `main`, run `stack worktree <branch> main`; without `[base]`, a new branch starts from the current branch.
- Prefer a worktree when working in parallel or preserving changes in the current worktree.

## Inspect and navigate

- `stack show` displays the local stack without network access.
- `stack status` includes pull-request state and sync issues; use `--no-pr` for a faster local-only view.
- `stack up` checks out the parent branch. `stack down` checks out a child and prompts when multiple children exist.
- `stack switch [branch]` prints a command for changing to a branch's worktree. Use the installed shell wrapper or `eval "$(stack switch [branch])"` so the current shell changes directory.
- `stack parent` shows the current parent; `stack parent <new-parent>` changes it and updates an existing PR base.
- `stack rename <new-name>` renames the current branch while preserving stack relationships.

## Sync

Run `stack sync [remote]` to fetch the base remote, rebase branches bottom-to-top, force-push them to `origin` with lease protection, and update the bases of existing PRs. It does not create missing PRs; use `gh pr create` for those.

- Use `stack sync --dry-run` to preview mutations.
- Use `stack sync --all` to include descendants below the current branch.
- Use `stack sync --cross-worktree` to include branches checked out in other worktrees.
- Use `stack sync <remote>` in fork workflows; the fetch remote otherwise comes from `stack.fetchRemote`, then `upstream`, then `origin`. Branches are always pushed to `origin`.
- After resolving rebase conflicts, run `stack sync --resume`. Run `stack sync --abort` to abandon an interrupted sync and restore saved state.
- Use `--force` only when intentionally bypassing force-with-lease protection.

## Clean up

- `stack prune` deletes local stack branches whose PRs are merged. Preview with `--dry-run`; `--all` also checks non-stack branches.
- `stack worktree --prune` removes worktrees for merged branches. `stack worktree --list` lists worktrees.

Run `stack <command> --help` for complete, version-specific flags before unusual or destructive operations.
