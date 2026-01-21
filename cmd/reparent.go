package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/spf13/cobra"
)

var reparentCmd = &cobra.Command{
	Use:   "reparent <new-parent>",
	Short: "Change the parent of the current branch",
	Long: `Change or set the parent branch of the current branch.

This command updates the stack parent relationship in git config and, if a PR
exists for the current branch, automatically updates the PR base to match the
new parent.

This is useful for:
- Adding an existing branch to a stack (when no parent is currently set)
- Reorganizing your stack when you want to change which branch a feature is based on`,
	Example: `  # Change current branch to be based on a different parent
  stack reparent feature-auth

  # Preview what would happen
  stack reparent main --dry-run

  # See all git/gh commands
  stack reparent feature-base --verbose`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		newParent := args[0]

		gitClient := git.NewGitClient()
		repo := github.ParseRepoFromURL(gitClient.GetRemoteURL("origin"))
		githubClient := github.NewGitHubClient(repo)

		if err := runReparent(gitClient, githubClient, newParent); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runReparent(gitClient git.GitClient, githubClient github.GitHubClient, newParent string) error {
	// Get current branch
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get current parent (may be empty if not in a stack)
	currentParent := gitClient.GetConfig(fmt.Sprintf("branch.%s.stackparent", currentBranch))

	// Check if new parent is the same as current parent
	if currentParent != "" && newParent == currentParent {
		fmt.Printf("Branch %s is already parented to %s\n", currentBranch, newParent)
		return nil
	}

	// Verify new parent branch exists
	if !gitClient.BranchExists(newParent) {
		return fmt.Errorf("new parent branch %s does not exist", newParent)
	}

	// Check if this would create a cycle
	if newParent == currentBranch {
		return fmt.Errorf("cannot set branch as its own parent")
	}

	// Check if new parent is a descendant of current branch (would create cycle)
	if isDescendant(gitClient, currentBranch, newParent) {
		return fmt.Errorf("cannot reparent to %s: it is a descendant of %s (would create a cycle)", newParent, currentBranch)
	}

	// Print appropriate message based on whether we're adding to stack or reparenting
	if currentParent == "" {
		fmt.Printf("Adding %s to stack with parent %s\n", currentBranch, newParent)
	} else {
		fmt.Printf("Reparenting %s: %s -> %s\n", currentBranch, currentParent, newParent)
	}

	// Update git config
	configKey := fmt.Sprintf("branch.%s.stackparent", currentBranch)
	if err := gitClient.SetConfig(configKey, newParent); err != nil {
		return fmt.Errorf("failed to update parent config: %w", err)
	}

	// Check if there's a PR for this branch
	pr, err := githubClient.GetPRForBranch(currentBranch)
	if err != nil {
		// Error fetching PR info, but config was updated successfully
		fmt.Printf("✓ Updated parent to %s\n", newParent)
		fmt.Printf("Warning: failed to check for PR: %v\n", err)
		return nil
	}

	if pr != nil {
		// PR exists, update its base
		fmt.Printf("Updating PR #%d base: %s -> %s\n", pr.Number, pr.Base, newParent)

		if err := githubClient.UpdatePRBase(pr.Number, newParent); err != nil {
			// Config was updated but PR base update failed
			fmt.Printf("✓ Updated parent to %s\n", newParent)
			return fmt.Errorf("failed to update PR base: %w", err)
		}

		if !dryRun {
			fmt.Printf("✓ Updated parent to %s\n", newParent)
			fmt.Printf("✓ Updated PR #%d base to %s\n", pr.Number, newParent)
		}
	} else {
		// No PR exists
		if !dryRun {
			fmt.Printf("✓ Updated parent to %s\n", newParent)
			fmt.Println("  (no PR found for this branch)")
		}
	}

	return nil
}

// isDescendant checks if possibleDescendant is a descendant of ancestor in the stack
func isDescendant(gitClient git.GitClient, ancestor, possibleDescendant string) bool {
	// Walk up from possibleDescendant to see if we reach ancestor
	current := possibleDescendant
	visited := make(map[string]bool)

	for current != "" {
		// Prevent infinite loops
		if visited[current] {
			return false
		}
		visited[current] = true

		// Get parent of current
		parent := gitClient.GetConfig(fmt.Sprintf("branch.%s.stackparent", current))
		if parent == "" {
			// Reached the top of the stack without finding ancestor
			return false
		}

		if parent == ancestor {
			// Found ancestor in the chain
			return true
		}

		current = parent
	}

	return false
}
