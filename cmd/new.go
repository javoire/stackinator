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

var newCmd = &cobra.Command{
	Use:   "new <branch-name> [parent]",
	Short: "Create a new branch in the stack",
	Long: `Create a new branch in the stack, optionally specifying a parent branch.

The new branch will be created from the specified parent (or current branch if not specified),
and the parent relationship will be stored in git config (branch.<name>.stackParent).

If no parent is specified and you're not on a stack branch, the base branch (default: main)
will be used as the parent.`,
	Example: `  # Create a stack: main <- A <- B <- C
  stack new A main                         # A based on main
  stack new B                              # B based on current (A)
  stack new C                              # C based on current (B)

  # Preview without creating
  stack new feature-xyz --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]
		var parent string
		if len(args) > 1 {
			parent = args[1]
		}

		if err := runNew(branchName, parent); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runNew(branchName string, explicitParent string) error {
	// Check if working tree is clean
	clean, err := git.IsWorkingTreeClean()
	if err != nil {
		return fmt.Errorf("failed to check working tree status: %w", err)
	}
	if !clean {
		return fmt.Errorf("working tree has uncommitted changes. Please commit or stash them first")
	}

	// Check if branch already exists
	if git.BranchExists(branchName) {
		return fmt.Errorf("branch %s already exists", branchName)
	}

	// Determine parent branch
	var parent string
	if explicitParent != "" {
		// Use explicitly provided parent
		parent = explicitParent
		// Verify parent exists
		if !git.BranchExists(parent) {
			return fmt.Errorf("parent branch %s does not exist", parent)
		}
	} else {
		// Get current branch as parent
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// If current branch has no parent, check if it's the base branch
		// Otherwise use it as parent
		parent = currentBranch
		currentParent := git.GetConfig(fmt.Sprintf("branch.%s.stackparent", currentBranch))

		// If we're not on a stack branch, use the base branch as parent
		if currentParent == "" && currentBranch != stack.GetBaseBranch() {
			// Check if current branch IS the base branch or if we should use base
			parent = currentBranch
		}
	}

	fmt.Printf("Creating new branch %s from %s\n", branchName, parent)

	// Create the new branch
	if err := git.CreateBranch(branchName, parent); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Set parent in git config
	configKey := fmt.Sprintf("branch.%s.stackparent", branchName)
	if err := git.SetConfig(configKey, parent); err != nil {
		return fmt.Errorf("failed to set parent config: %w", err)
	}

	if !dryRun {
		fmt.Printf("✓ Created branch %s with parent %s\n", branchName, parent)
		fmt.Println()

		// Show the full stack
		if err := showStack(); err != nil {
			// Don't fail if we can't show the stack, just warn
			fmt.Fprintf(os.Stderr, "Warning: failed to display stack: %v\n", err)
		}
	}

	return nil
}

// showStack displays the current stack structure
func showStack() error {
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	tree, err := stack.BuildStackTreeForBranch(currentBranch)
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	// Fetch all PRs upfront for better performance
	prCache, err := github.GetAllPRs()
	if err != nil {
		// If fetching PRs fails, just continue without PR info
		prCache = make(map[string]*github.PRInfo)
	}

	// Filter out branches with merged PRs from the tree
	tree = filterMergedBranchesForNew(tree, prCache)

	printStackTree(tree, "", true, currentBranch, prCache)

	return nil
}

// filterMergedBranchesForNew removes branches with merged PRs from the tree,
// but only if they don't have children (to keep the stack structure visible)
func filterMergedBranchesForNew(node *stack.TreeNode, prCache map[string]*github.PRInfo) *stack.TreeNode {
	if node == nil {
		return nil
	}

	// Filter children recursively first
	var filteredChildren []*stack.TreeNode
	for _, child := range node.Children {
		// Recurse first to process all descendants
		filtered := filterMergedBranchesForNew(child, prCache)

		// Only filter out merged branches if they have no children
		// (i.e., they're leaf nodes)
		if pr, exists := prCache[child.Name]; exists && pr.State == "MERGED" {
			// If this merged branch still has children after filtering, keep it
			// so the stack structure remains visible
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

// printStackTree is a simplified version of the status tree printer
func printStackTree(node *stack.TreeNode, prefix string, isLast bool, currentBranch string, prCache map[string]*github.PRInfo) {
	if node == nil {
		return
	}

	// Flatten the tree into a vertical list
	printStackTreeVertical(node, currentBranch, prCache, false)
}

func printStackTreeVertical(node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo, isPipe bool) {
	if node == nil {
		return
	}

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
		printStackTreeVertical(child, currentBranch, prCache, true)
	}
}
