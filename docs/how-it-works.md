# How It Works

## Stack Tracking

Stackinator stores the parent of each branch in git config:

```bash
# View parent of current branch
git config branch.feature-1.stackparent

# Manually set parent (if needed)
git config branch.feature-1.stackparent main
```

This minimal approach means:

- No state files to manage
- No database or JSON files
- Works with standard git workflows
- Easy to inspect and debug

## Sync Algorithm

When you run `stack sync`, Stackinator:

1. Fetches from origin
2. Discovers all stack branches from git config
3. Sorts them in topological order (base to tips)
4. For each branch:
   - Checks if parent PR was merged (updates parent if so)
   - Rebases onto parent
   - Force pushes to origin
   - Updates PR base branch (if PR exists)
