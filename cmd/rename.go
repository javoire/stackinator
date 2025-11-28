package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <new-name>",
	Short: "Rename the current branch while preserving stack relationships",
	Long: `Rename the current branch to a new name while preserving all stack relationships.

This command will:
  - Rename the git branch
  - Update the branch's parent reference in git config
  - Update all child branches to point to the new name

The command must be run while on the branch you want to rename.`,
	Example: `  # Rename current branch
  stack rename feature-improved-name

  # Preview without making changes
  stack rename feature-improved-name --dry-run`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		newName := args[0]

		gitClient := git.NewGitClient()
		githubClient := github.NewGitHubClient()

		if err := runRename(gitClient, githubClient, newName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runRename(gitClient git.GitClient, githubClient github.GitHubClient, newName string) error {
	// Get current branch
	oldName, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Validate old branch is in the stack
	oldParent := gitClient.GetConfig(fmt.Sprintf("branch.%s.stackparent", oldName))
	if oldParent == "" {
		return fmt.Errorf("current branch %s is not part of a stack (no stackparent configured)", oldName)
	}

	// Check if new name already exists
	if gitClient.BranchExists(newName) {
		return fmt.Errorf("branch %s already exists", newName)
	}

	// Get all children of the current branch
	children, err := stack.GetChildrenOf(gitClient, oldName)
	if err != nil {
		return fmt.Errorf("failed to get children: %w", err)
	}

	fmt.Printf("Renaming branch %s -> %s\n", oldName, newName)
	if len(children) > 0 {
		fmt.Printf("  Will update %d child branch(es)\n", len(children))
	}

	// Rename the branch
	if err := gitClient.RenameBranch(oldName, newName); err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}

	// Move the parent config from old to new
	oldConfigKey := fmt.Sprintf("branch.%s.stackparent", oldName)
	newConfigKey := fmt.Sprintf("branch.%s.stackparent", newName)

	if err := gitClient.SetConfig(newConfigKey, oldParent); err != nil {
		return fmt.Errorf("failed to set new parent config: %w", err)
	}

	if err := gitClient.UnsetConfig(oldConfigKey); err != nil {
		// This might fail if the branch was just renamed and git already handled it
		// Don't fail the whole operation
		if verbose {
			fmt.Printf("  Warning: failed to unset old config (may already be removed): %v\n", err)
		}
	}

	// Update all children to point to the new name
	for _, child := range children {
		childConfigKey := fmt.Sprintf("branch.%s.stackparent", child.Name)
		if err := gitClient.SetConfig(childConfigKey, newName); err != nil {
			return fmt.Errorf("failed to update child %s: %w", child.Name, err)
		}
		fmt.Printf("  ✓ Updated child %s to point to %s\n", child.Name, newName)
	}

	if !dryRun {
		fmt.Printf("✓ Successfully renamed branch %s -> %s\n", oldName, newName)
		fmt.Println()

		// Show the updated stack
		if err := showStack(gitClient, githubClient); err != nil {
			// Don't fail if we can't show the stack, just warn
			fmt.Fprintf(os.Stderr, "Warning: failed to display stack: %v\n", err)
		}
	}

	return nil
}

