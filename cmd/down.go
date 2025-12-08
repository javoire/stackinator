package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Move to a child branch in the stack",
	Long: `Checkout a child branch of the current branch in the stack.

If the current branch has no children (is at the tip of the stack),
an error message will be displayed.

If there are multiple children, you will be prompted to select one.`,
	Example: `  # Move to child branch
  stack down`,
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()

		if err := runDown(gitClient); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runDown(gitClient git.GitClient) error {
	// Get current branch
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get children of current branch
	children, err := stack.GetChildrenOf(gitClient, currentBranch)
	if err != nil {
		return fmt.Errorf("failed to get children: %w", err)
	}

	if len(children) == 0 {
		return fmt.Errorf("no children (tip of stack)")
	}

	var targetBranch string

	if len(children) == 1 {
		// Only one child, checkout directly
		targetBranch = children[0].Name
	} else {
		// Multiple children, prompt for selection
		fmt.Printf("Multiple children found for %s:\n", currentBranch)
		for i, child := range children {
			fmt.Printf("  %d) %s\n", i+1, child.Name)
		}
		fmt.Print("\nSelect branch (1-" + strconv.Itoa(len(children)) + "): ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		selection, err := strconv.Atoi(input)
		if err != nil || selection < 1 || selection > len(children) {
			return fmt.Errorf("invalid selection: %s", input)
		}

		targetBranch = children[selection-1].Name
	}

	// Checkout the target branch
	if err := gitClient.CheckoutBranch(targetBranch); err != nil {
		return fmt.Errorf("failed to checkout child branch %s: %w", targetBranch, err)
	}

	fmt.Printf("Switched to child branch: %s\n", targetBranch)
	return nil
}
