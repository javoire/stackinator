# Troubleshooting

## Rebase Conflicts

If `stack sync` encounters a rebase conflict:

1. Resolve the conflict manually
2. Run `git rebase --continue`
3. Run `stack sync` again to continue with remaining branches

## Orphaned Branches

If you delete a parent branch, child branches become orphaned. To fix:

```bash
# Update the child's parent to point to the grandparent
git config branch.child-branch.stackparent main
```

## Remove from Stack

To remove a branch from the stack (but keep the branch):

```bash
git config --unset branch.my-branch.stackparent
```
