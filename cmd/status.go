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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current stack structure",
	Long: `Display the stack structure as a tree, showing:
  - Branch hierarchy (parent → child relationships)
  - Current branch (highlighted with *)
  - PR status for each branch (if available)

This helps you visualize your stack and see which branches have PRs.`,
	Example: `  # Show stack structure
  stack status

  # Example output:
  #   main
  #   └─ feature-auth [PR #123: OPEN]
  #      ├─ feature-auth-tests *
  #      └─ feature-auth-docs`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runStatus() error {
	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Build stack tree
	tree, err := stack.BuildStackTree()
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	// Check if there are any stack branches
	stackBranches, err := stack.GetStackBranches()
	if err != nil {
		return fmt.Errorf("failed to get stack branches: %w", err)
	}

	if len(stackBranches) == 0 {
		fmt.Println("No stack branches found.")
		fmt.Printf("Current branch: %s\n", currentBranch)
		fmt.Println("\nUse 'stack new <branch-name>' to create a new stack branch.")
		return nil
	}

	fmt.Println("Stack structure:")
	fmt.Println()

	// Fetch all PRs upfront for better performance
	prCache, err := github.GetAllPRs()
	if err != nil {
		// If fetching PRs fails, just continue without PR info
		prCache = make(map[string]*github.PRInfo)
	}

	// Print the tree
	printTree(tree, "", true, currentBranch, prCache)

	return nil
}

func printTree(node *stack.TreeNode, prefix string, isLast bool, currentBranch string, prCache map[string]*github.PRInfo) {
	if node == nil {
		return
	}

	// Determine the current branch marker
	marker := " "
	if node.Name == currentBranch {
		marker = "*"
	}

	// Print current node
	branch := prefix
	if prefix != "" {
		if isLast {
			branch += "└─ "
		} else {
			branch += "├─ "
		}
	}

	// Get PR info from cache
	prInfo := ""
	if node.Name != stack.GetBaseBranch() {
		if pr, exists := prCache[node.Name]; exists {
			prInfo = fmt.Sprintf(" [%s :%s]", pr.URL, strings.ToLower(pr.State))
		}
	}

	fmt.Printf("%s%s%s%s\n", marker, branch, node.Name, prInfo)

	// Print children
	childPrefix := prefix
	if isLast {
		childPrefix += "   "
	} else {
		childPrefix += "│  "
	}

	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		printTree(child, childPrefix, isLastChild, currentBranch, prCache)
	}
}

// Helper to repeat a string n times
func repeatString(s string, n int) string {
	return strings.Repeat(s, n)
}


