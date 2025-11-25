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
	currentParent := git.GetConfig(fmt.Sprintf("branch.%s.stackParent", currentBranch))

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
	configKey := fmt.Sprintf("branch.%s.stackParent", branchName)
	if err := git.SetConfig(configKey, parent); err != nil {
		return fmt.Errorf("failed to set parent config: %w", err)
	}

	if !dryRun {
		fmt.Printf("✓ Created branch %s with parent %s\n", branchName, parent)
	}

	return nil
}

