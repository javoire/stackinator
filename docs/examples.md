# Examples

## Example Workflow

```bash
# Start from main
git checkout main
git pull

# Create feature branch
stack new auth-system
# ... make changes, commit, create PR

# Create sub-feature 1
stack new auth-login
# ... make changes, commit, create PR

# Create sub-feature 2
git checkout auth-system  # go back to parent
stack new auth-logout
# ... make changes, commit, create PR

# View structure
stack status
# Output:
#  main
#   |
#  auth-system [https://github.com/you/repo/pull/10 :merged]
#   |
#  auth-login [https://github.com/you/repo/pull/11 :open]
#   |
#  auth-logout [https://github.com/you/repo/pull/12 :open] *

# Later, after making changes or when main updates
stack sync
```

## Dry Run

Preview what sync would do:

```bash
stack sync --dry-run
```

## Verbose Output

See all git/gh commands being executed:

```bash
stack sync --verbose
```

