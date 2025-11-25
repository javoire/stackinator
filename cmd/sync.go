package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync all stack branches with their parents and update PRs",
	Long: `Perform a full sync of the stack:
  1. Fetch latest changes from origin
  2. Rebase each stack branch onto its parent (in bottom-to-top order)
  3. Force push each branch to origin
  4. Update PR base branches to match the stack (if PRs exist)

This ensures your stack is up-to-date and all PRs have the correct base branches.

If a parent PR has been merged, the child branches will be rebased to point to
the merged parent's parent.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSync(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runSync() error {
	// Get current branch so we can return to it
	originalBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if working tree is clean
	clean, err := git.IsWorkingTreeClean()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}
	if !clean {
		return fmt.Errorf("working tree has uncommitted changes. Please commit or stash them first")
	}

	fmt.Println("Syncing stack...")
	fmt.Println()

	// Fetch from origin
	fmt.Println("Fetching from origin...")
	if err := git.Fetch(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Get all stack branches
	stackBranches, err := stack.GetStackBranches()
	if err != nil {
		return fmt.Errorf("failed to get stack branches: %w", err)
	}

	if len(stackBranches) == 0 {
		fmt.Println("No stack branches found.")
		return nil
	}

	// Sort branches in topological order (bottom to top)
	sorted, err := stack.TopologicalSort(stackBranches)
	if err != nil {
		return fmt.Errorf("failed to sort branches: %w", err)
	}

	fmt.Printf("Processing %d branch(es)...\n\n", len(sorted))

	// Process each branch
	for _, branch := range sorted {
		fmt.Printf("Processing %s...\n", branch.Name)

		// Check if parent PR is merged
		parentUpdated := false
		parentPR, err := github.GetPRForBranch(branch.Parent)
		if err == nil && parentPR != nil {
			merged, err := github.IsPRMerged(parentPR.Number)
			if err == nil && merged {
				fmt.Printf("  Parent PR #%d has been merged\n", parentPR.Number)

				// Update parent to grandparent
				grandparent := git.GetConfig(fmt.Sprintf("branch.%s.stackParent", branch.Parent))
				if grandparent == "" {
					grandparent = stack.GetBaseBranch()
				}

				fmt.Printf("  Updating parent from %s to %s\n", branch.Parent, grandparent)
				configKey := fmt.Sprintf("branch.%s.stackParent", branch.Name)
				if err := git.SetConfig(configKey, grandparent); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to update parent config: %v\n", err)
				} else {
					branch.Parent = grandparent
					parentUpdated = true
				}
			}
		}

		// Checkout the branch
		if err := git.CheckoutBranch(branch.Name); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to checkout %s: %v\n", branch.Name, err)
			continue
		}

		// Rebase onto parent
		fmt.Printf("  Rebasing onto %s...\n", branch.Parent)
		if err := git.Rebase(branch.Parent); err != nil {
			fmt.Fprintf(os.Stderr, "  Error: failed to rebase: %v\n", err)
			fmt.Fprintf(os.Stderr, "  Please resolve conflicts and run 'git rebase --continue', then run 'stack sync' again\n")
			return err
		}

		// Push to origin
		fmt.Printf("  Pushing to origin...\n")
		if err := git.Push(branch.Name, true); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to push: %v\n", err)
		}

		// Check if PR exists and update base if needed
		pr, err := github.GetPRForBranch(branch.Name)
		if err == nil && pr != nil {
			if pr.Base != branch.Parent || parentUpdated {
				fmt.Printf("  Updating PR #%d base from %s to %s...\n", pr.Number, pr.Base, branch.Parent)
				if err := github.UpdatePRBase(pr.Number, branch.Parent); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to update PR base: %v\n", err)
				} else {
					fmt.Printf("  ✓ PR #%d updated\n", pr.Number)
				}
			} else {
				fmt.Printf("  PR #%d base is already correct (%s)\n", pr.Number, pr.Base)
			}
		} else {
			fmt.Printf("  No PR found (create one with 'gh pr create')\n")
		}

		fmt.Println()
	}

	// Return to original branch
	fmt.Printf("Returning to %s...\n", originalBranch)
	if err := git.CheckoutBranch(originalBranch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to return to original branch: %v\n", err)
	}

	fmt.Println("✓ Sync complete!")

	return nil
}


