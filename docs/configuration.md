# Configuration

Base branch can be configured per-repo:

```bash
git config stack.baseBranch develop  # Default is "main"
```

Worktrees directory can be configured per-repo (or globally):

```bash
git config stack.worktreesDir ~/worktrees
```

View current config:

```bash
stack config
```

Or use the interactive helper:

```bash
stack config set
```
