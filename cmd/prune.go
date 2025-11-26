package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/spinner"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

var (
	pruneForce bool
	pruneAll   bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up branches with merged PRs",
	Long: `Remove branches with merged PRs from stack tracking and delete them locally.

By default, this command only checks branches in the stack (those created with 'stack new').
Use --all to check all local branches.

This command will:
  1. Find all branches with merged PRs
  2. Remove them from stack tracking (if applicable)
  3. Delete the local branches with 'git branch -d'

If a branch has unmerged commits locally, use --force to delete it anyway.`,
	Example: `  # Clean up merged stack branches
  stack prune

  # Clean up all merged branches (including non-stack branches)
  stack prune --all

  # Force delete even if branches have unmerged commits
  stack prune --force

  # Preview what would be deleted
  stack prune --dry-run`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runPrune(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&pruneForce, "force", "f", false, "Force delete branches even if they have unmerged commits")
	pruneCmd.Flags().BoolVarP(&pruneAll, "all", "a", false, "Check all local branches, not just stack branches")
}

func runPrune() error {
	// Get current branch so we don't delete it
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get base branch to exclude it from pruning
	baseBranch := stack.GetBaseBranch()

	// Get branches to check
	var branchNames []string
	if pruneAll {
		// Check all local branches
		if err := spinner.Wrap("Loading all branches...", func() error {
			var err error
			branchNames, err = git.ListBranches()
			return err
		}); err != nil {
			return fmt.Errorf("failed to get branches: %w", err)
		}

		// Filter out base branch and current branch
		var filtered []string
		for _, branch := range branchNames {
			if branch != baseBranch && branch != currentBranch {
				filtered = append(filtered, branch)
			}
		}
		branchNames = filtered
	} else {
		// Check only stack branches
		var stackBranches []stack.StackBranch
		if err := spinner.Wrap("Loading stack branches...", func() error {
			var err error
			stackBranches, err = stack.GetStackBranches()
			return err
		}); err != nil {
			return fmt.Errorf("failed to get stack branches: %w", err)
		}

		for _, sb := range stackBranches {
			branchNames = append(branchNames, sb.Name)
		}
	}

	if len(branchNames) == 0 {
		if pruneAll {
			fmt.Println("No branches found to check.")
		} else {
			fmt.Println("No stack branches found.")
		}
		return nil
	}

	// Fetch all PRs
	var prCache map[string]*github.PRInfo
	if err := spinner.Wrap("Fetching PR information...", func() error {
		var err error
		prCache, err = github.GetAllPRs()
		if err != nil {
			return fmt.Errorf("failed to fetch PRs: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	// Find branches with merged PRs
	var mergedBranches []string
	for _, branchName := range branchNames {
		if pr, exists := prCache[branchName]; exists && pr.State == "MERGED" {
			mergedBranches = append(mergedBranches, branchName)
		}
	}

	if len(mergedBranches) == 0 {
		fmt.Println("\nNo merged branches to prune.")
		return nil
	}

	// Show what will be pruned
	fmt.Println()
	fmt.Printf("Found %d merged branch(es) to prune:\n", len(mergedBranches))
	for _, branch := range mergedBranches {
		pr := prCache[branch]
		fmt.Printf("  - %s (PR #%d)\n", branch, pr.Number)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run - no changes made.")
		return nil
	}

	// Prune each merged branch
	for i, branch := range mergedBranches {
		fmt.Printf("(%d/%d) Pruning %s...\n", i+1, len(mergedBranches), branch)

		// Remove from stack tracking (if in stack)
		configKey := fmt.Sprintf("branch.%s.stackparent", branch)
		if git.GetConfig(configKey) != "" {
			fmt.Println("  Removing from stack tracking...")
			if err := git.UnsetConfig(configKey); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to remove stack config: %v\n", err)
			}
		}

		// Don't delete current branch
		if branch == currentBranch {
			fmt.Println("  ⚠ Skipping deletion (currently checked out)")
			fmt.Println()
			continue
		}

		// Delete the branch
		fmt.Println("  Deleting branch...")
		var deleteErr error
		if pruneForce {
			deleteErr = deleteBranchForce(branch)
		} else {
			deleteErr = deleteBranch(branch)
		}

		if deleteErr != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to delete branch: %v\n", deleteErr)
			if !pruneForce {
				fmt.Fprintf(os.Stderr, "  Use 'stack prune --force' to force delete, or manually delete with: git branch -D %s\n", branch)
			}
		} else {
			fmt.Println("  ✓ Deleted")
		}
		fmt.Println()
	}

	fmt.Println("✓ Prune complete!")

	return nil
}

// deleteBranch deletes a branch using 'git branch -d' (safe delete)
func deleteBranch(name string) error {
	if verbose {
		fmt.Printf("  [git] branch -d %s\n", name)
	}
	return git.DeleteBranch(name)
}

// deleteBranchForce deletes a branch using 'git branch -D' (force delete)
func deleteBranchForce(name string) error {
	if verbose {
		fmt.Printf("  [git] branch -D %s\n", name)
	}
	return git.DeleteBranchForce(name)
}
