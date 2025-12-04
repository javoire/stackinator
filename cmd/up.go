package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Move to the parent branch in the stack",
	Long: `Checkout the parent branch of the current branch in the stack.

If the current branch has no parent (is at the root of the stack),
an error message will be displayed.`,
	Example: `  # Move to parent branch
  stack up`,
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()

		if err := runUp(gitClient); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runUp(gitClient git.GitClient) error {
	// Get current branch
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get parent from git config
	parent := gitClient.GetConfig(fmt.Sprintf("branch.%s.stackparent", currentBranch))

	if parent == "" {
		return fmt.Errorf("already at stack root (no parent for %s)", currentBranch)
	}

	// Checkout the parent branch
	if err := gitClient.CheckoutBranch(parent); err != nil {
		return fmt.Errorf("failed to checkout parent branch %s: %w", parent, err)
	}

	fmt.Printf("Switched to parent branch: %s\n", parent)
	return nil
}
