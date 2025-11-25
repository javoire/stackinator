package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <branch-name>",
	Short: "Create a new branch in the stack",
	Long: `Create a new branch in the stack, using the current branch as the parent.

The new branch will be created from the current branch, and the parent relationship
will be stored in git config (branch.<name>.stackParent).

If you're not currently on a stack branch, the base branch (default: main) will be
used as the parent.`,
	Example: `  # Create first branch from main
  git checkout main
  stack new feature-auth

  # Create second branch stacked on feature-auth
  stack new feature-auth-tests

  # Preview without creating
  stack new feature-xyz --dry-run`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		branchName := args[0]

		if err := runNew(branchName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runNew(branchName string) error {
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

	// Get current branch as parent
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// If current branch has no parent, check if it's the base branch
	// Otherwise use it as parent
	parent := currentBranch
	currentParent := git.GetConfig(fmt.Sprintf("branch.%s.stackparent", currentBranch))

	// If we're not on a stack branch, use the base branch as parent
	if currentParent == "" && currentBranch != stack.GetBaseBranch() {
		// Check if current branch IS the base branch or if we should use base
		parent = currentBranch
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

	tree, err := stack.BuildStackTree()
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	printStackTree(tree, "", true, currentBranch)

	return nil
}

// printStackTree is a simplified version of the status tree printer
func printStackTree(node *stack.TreeNode, prefix string, isLast bool, currentBranch string) {
	if node == nil {
		return
	}

	// Flatten the tree into a vertical list
	printStackTreeVertical(node, currentBranch, false)
}

func printStackTreeVertical(node *stack.TreeNode, currentBranch string, isPipe bool) {
	if node == nil {
		return
	}

	marker := ""
	if node.Name == currentBranch {
		marker = " *"
	}

	// Print pipe if needed
	if isPipe {
		fmt.Println("  |")
	}

	// Print current node
	fmt.Printf(" %s%s\n", node.Name, marker)

	// Print children vertically
	for _, child := range node.Children {
		printStackTreeVertical(child, currentBranch, true)
	}
}


