package cmd

import (
	"fmt"
	"os"
	"strings"

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
the merged parent's parent.

Uncommitted changes are automatically stashed and reapplied (using --autostash).`,
	Example: `  # Sync all branches and update PRs
  stack sync

  # Preview what would happen
  stack sync --dry-run

  # Show detailed git/gh commands
  stack sync --verbose

  # Common workflow after updating main
  git checkout main && git pull
  stack sync`,
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

	// Check if working tree is clean and stash if needed
	clean, err := git.IsWorkingTreeClean()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}

	stashed := false
	if !clean {
		fmt.Println("Stashing uncommitted changes...")
		if err := git.Stash("stack-sync-autostash"); err != nil {
			return fmt.Errorf("failed to stash changes: %w", err)
		}
		stashed = true
		fmt.Println()
	}

	// Ensure stash is popped at the end
	defer func() {
		if stashed {
			fmt.Println("\nRestoring stashed changes...")
			if err := git.StashPop(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restore stashed changes: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'git stash pop' manually to restore your changes\n")
			}
		}
	}()

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

	// Fetch all PRs upfront for better performance
	prCache, err := github.GetAllPRs()
	if err != nil {
		// If fetching PRs fails, fall back to individual fetches
		prCache = make(map[string]*github.PRInfo)
	}

	// Process each branch
	for i, branch := range sorted {
		progress := fmt.Sprintf("(%d/%d)", i+1, len(sorted))

		// Check if this branch has a merged PR - if so, remove from stack tracking
		if pr, exists := prCache[branch.Name]; exists && pr.State == "MERGED" {
			fmt.Printf("%s Skipping %s (PR #%d is merged)...\n", progress, branch.Name, pr.Number)
			fmt.Printf("  Removing from stack tracking...\n")
			configKey := fmt.Sprintf("branch.%s.stackparent", branch.Name)
			if err := git.UnsetConfig(configKey); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to remove stack config: %v\n", err)
			} else {
				fmt.Printf("  ✓ Removed. You can delete this branch with: git branch -d %s\n", branch.Name)
			}
			fmt.Println()
			continue
		}

		fmt.Printf("%s Processing %s...\n", progress, branch.Name)

		// Check if parent PR is merged
		parentUpdated := false
		parentPR, _ := prCache[branch.Parent]
		if parentPR != nil && parentPR.State == "MERGED" {
			fmt.Printf("  Parent PR #%d has been merged\n", parentPR.Number)

			// Update parent to grandparent
			grandparent := git.GetConfig(fmt.Sprintf("branch.%s.stackparent", branch.Parent))
			if grandparent == "" {
				grandparent = stack.GetBaseBranch()
			}

			fmt.Printf("  Updating parent from %s to %s\n", branch.Parent, grandparent)
			configKey := fmt.Sprintf("branch.%s.stackparent", branch.Name)
			if err := git.SetConfig(configKey, grandparent); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to update parent config: %v\n", err)
			} else {
				branch.Parent = grandparent
				parentUpdated = true
			}
		}

		// Checkout the branch
		if err := git.CheckoutBranch(branch.Name); err != nil {
			return fmt.Errorf("failed to checkout %s: %w", branch.Name, err)
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
		pr, _ := prCache[branch.Name]
		if pr != nil {
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

	fmt.Println()

	// Display the updated stack status
	if err := displayStatusAfterSync(); err != nil {
		// Don't fail if we can't display status, just warn
		fmt.Fprintf(os.Stderr, "Warning: failed to display stack status: %v\n", err)
	}

	fmt.Println()
	fmt.Println("✓ Sync complete!")

	return nil
}

// displayStatusAfterSync shows the stack tree after a successful sync
func displayStatusAfterSync() error {
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	tree, err := stack.BuildStackTree()
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	// Fetch all PRs for display
	prCache, err := github.GetAllPRs()
	if err != nil {
		prCache = make(map[string]*github.PRInfo)
	}

	// Filter out branches with merged PRs (leaf nodes only)
	tree = filterMergedBranchesForSync(tree, prCache)

	// Print the tree
	printTreeForSync(tree, currentBranch, prCache)

	return nil
}

// filterMergedBranchesForSync removes branches with merged PRs from the tree,
// but only if they don't have children (to keep the stack structure visible)
func filterMergedBranchesForSync(node *stack.TreeNode, prCache map[string]*github.PRInfo) *stack.TreeNode {
	if node == nil {
		return nil
	}

	// Filter children recursively first
	var filteredChildren []*stack.TreeNode
	for _, child := range node.Children {
		// Recurse first to process all descendants
		filtered := filterMergedBranchesForSync(child, prCache)

		// Only filter out merged branches if they have no children
		if pr, exists := prCache[child.Name]; exists && pr.State == "MERGED" {
			// If this merged branch still has children after filtering, keep it
			if filtered != nil && len(filtered.Children) > 0 {
				filteredChildren = append(filteredChildren, filtered)
			}
			// Otherwise skip this merged leaf branch
		} else {
			// Not merged, keep it
			if filtered != nil {
				filteredChildren = append(filteredChildren, filtered)
			}
		}
	}

	node.Children = filteredChildren
	return node
}

// printTreeForSync prints the stack tree after sync
func printTreeForSync(node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo) {
	if node == nil {
		return
	}
	printTreeVerticalForSync(node, currentBranch, prCache, false)
}

func printTreeVerticalForSync(node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo, isPipe bool) {
	if node == nil {
		return
	}

	// Determine the current branch marker
	marker := ""
	if node.Name == currentBranch {
		marker = " *"
	}

	// Get PR info from cache
	prInfo := ""
	if node.Name != stack.GetBaseBranch() {
		if pr, exists := prCache[node.Name]; exists {
			prInfo = fmt.Sprintf(" [%s :%s]", pr.URL, strings.ToLower(pr.State))
		}
	}

	// Print pipe if needed
	if isPipe {
		fmt.Println("  |")
	}

	// Print current node
	fmt.Printf(" %s%s%s\n", node.Name, prInfo, marker)

	// Print children vertically
	for _, child := range node.Children {
		printTreeVerticalForSync(child, currentBranch, prCache, true)
	}
}


