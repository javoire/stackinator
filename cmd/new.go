package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/javoire/stackinator/internal/ui"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <branch-name> [parent]",
	Short: "Create a new branch in the stack",
	Long: `Create a new branch in the stack, optionally specifying a parent branch.

The new branch will be created from the specified parent (or current branch if not specified),
and the parent relationship will be stored in git config (branch.<name>.stackparent).

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

		gitClient := git.NewGitClient()

		if err := runNew(gitClient, branchName, parent); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runNew(gitClient git.GitClient, branchName string, explicitParent string) error {
	// Check if branch already exists
	if gitClient.BranchExists(branchName) {
		return fmt.Errorf("branch %s already exists", branchName)
	}

	// Determine parent branch
	var parent string
	if explicitParent != "" {
		// Use explicitly provided parent
		parent = explicitParent
		// Verify parent exists
		if !gitClient.BranchExists(parent) {
			return fmt.Errorf("parent branch %s does not exist", parent)
		}
	} else {
		// Get current branch as parent
		currentBranch, err := gitClient.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// If current branch has no parent, check if it's the base branch
		// Otherwise use it as parent
		parent = currentBranch
		currentParent := gitClient.GetConfig(fmt.Sprintf("branch.%s.stackparent", currentBranch))

		// If we're not on a stack branch, use the base branch as parent
		if currentParent == "" && currentBranch != stack.GetBaseBranch(gitClient) {
			// Check if current branch IS the base branch or if we should use base
			parent = currentBranch
		}
	}

	fmt.Printf("Creating new branch %s from %s\n", ui.Branch(branchName), ui.Branch(parent))

	// Create the new branch
	if err := gitClient.CreateBranchAndCheckout(branchName, parent); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Set parent in git config
	configKey := fmt.Sprintf("branch.%s.stackparent", branchName)
	if err := gitClient.SetConfig(configKey, parent); err != nil {
		return fmt.Errorf("failed to set parent config: %w", err)
	}

	if !dryRun {
		fmt.Println(ui.Success(fmt.Sprintf("Created branch %s with parent %s", ui.Branch(branchName), ui.Branch(parent))))
		fmt.Println()

		// Show the local stack (fast, no PR fetching)
		if err := showStack(gitClient); err != nil {
			// Don't fail if we can't show the stack, just warn
			fmt.Fprintf(os.Stderr, "Warning: failed to display stack: %v\n", err)
		}
	}

	return nil
}

// showStack displays the current stack structure (local only, no PR fetching)
func showStack(gitClient git.GitClient) error {
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	tree, err := stack.BuildStackTreeForBranch(gitClient, currentBranch)
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	// Use the same local tree printer as stack show
	printLocalStackTree(tree, currentBranch, false)

	return nil
}
