package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/spf13/cobra"
)

var parentCmd = &cobra.Command{
	Use:   "parent",
	Short: "Show the parent of the current branch",
	Long: `Display the parent branch of the current branch in the stack.

If the current branch has no parent set, it will show that the branch
is not part of a stack.`,
	Example: `  # Show parent of current branch
  stack parent`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runParent(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runParent() error {
	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get parent from git config
	parent := git.GetConfig(fmt.Sprintf("branch.%s.stackparent", currentBranch))

	if parent == "" {
		fmt.Printf("%s (not in a stack)\n", currentBranch)
	} else {
		fmt.Println(parent)
	}

	return nil
}

