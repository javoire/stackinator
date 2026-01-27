package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/javoire/stackinator/internal/ui"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the local stack structure (fast)",
	Long: `Display the local stack structure as a tree without fetching remote PR info.

This is a fast version of 'stack status' that only reads local git config.
Use 'stack status' to see PR information and sync issues.`,
	Example: `  # Show local stack structure
  stack show

  # Example output:
  #  main
  #   |
  #  feature-auth
  #   |
  #  feature-auth-tests *`,
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()

		if err := runShow(gitClient); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runShow(gitClient git.GitClient) error {
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if there are any stack branches
	stackBranches, err := stack.GetStackBranches(gitClient)
	if err != nil {
		return fmt.Errorf("failed to get stack branches: %w", err)
	}

	if len(stackBranches) == 0 {
		fmt.Println("No stack branches found.")
		fmt.Printf("Current branch: %s\n", ui.Branch(currentBranch))
		fmt.Printf("\nUse '%s' to create a new stack branch.\n", ui.Command("stack new <branch-name>"))
		return nil
	}

	// Build stack tree for current branch only
	tree, err := stack.BuildStackTreeForBranch(gitClient, currentBranch)
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	// Print the tree
	fmt.Println()
	printLocalStackTree(tree, currentBranch, false)

	return nil
}

// printLocalStackTree prints the stack tree without PR info (local-only, fast)
func printLocalStackTree(node *stack.TreeNode, currentBranch string, isPipe bool) {
	if node == nil {
		return
	}

	marker := ""
	if node.Name == currentBranch {
		marker = ui.CurrentBranchMarker()
	}

	// Print pipe if needed
	if isPipe {
		fmt.Printf("  %s\n", ui.Pipe())
	}

	// Print current node (no PR info)
	fmt.Printf(" %s%s\n", ui.Branch(node.Name), marker)

	// Print children vertically
	for _, child := range node.Children {
		printLocalStackTree(child, currentBranch, true)
	}
}
