package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/spinner"
	"github.com/spf13/cobra"
)

var (
	worktreeStack bool
	worktreePrune bool
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree <branch-name>",
	Short: "Create a worktree in .worktrees/ directory",
	Long: `Create a git worktree in the .worktrees/ directory for the specified branch.

If the branch exists locally or on the remote, it will be used.
If the branch doesn't exist, a new branch will be created from the current branch.
Use --stack to also add stack tracking (set stackparent like 'stack new').
Use --prune to clean up worktrees for branches with merged PRs.`,
	Example: `  # Create worktree for new branch (created from current branch)
  stack worktree my-feature

  # Create worktree for existing local or remote branch
  stack worktree existing-branch

  # Create new branch with stack tracking + worktree
  stack worktree new-feature --stack

  # Clean up worktrees for merged branches
  stack worktree --prune

  # Preview without executing
  stack worktree my-feature --dry-run`,
	Args: func(cmd *cobra.Command, args []string) error {
		if worktreePrune {
			if len(args) > 0 {
				return fmt.Errorf("--prune does not take a branch argument")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 argument: branch name")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if worktreePrune {
			err = runWorktreePrune()
		} else {
			err = runWorktree(args[0])
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	worktreeCmd.Flags().BoolVar(&worktreeStack, "stack", false, "Create new branch with stack tracking (like 'stack new')")
	worktreeCmd.Flags().BoolVar(&worktreePrune, "prune", false, "Remove worktrees for branches with merged PRs")
}

func runWorktree(branchName string) error {
	// Get repo root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repo root: %w", err)
	}

	// Ensure .worktrees is in .gitignore
	if err := ensureWorktreesIgnored(repoRoot); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	// Worktree path
	worktreePath := filepath.Join(repoRoot, ".worktrees", branchName)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree already exists at %s", worktreePath)
	}

	if worktreeStack {
		// Create new branch with stack tracking
		return createStackWorktree(branchName, worktreePath)
	}

	// Check if branch exists locally or on remote
	return createWorktreeForExisting(branchName, worktreePath)
}

func createStackWorktree(branchName, worktreePath string) error {
	// Check if branch already exists
	if git.BranchExists(branchName) {
		return fmt.Errorf("branch %s already exists", branchName)
	}

	// Get current branch as parent
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	parent := currentBranch

	fmt.Printf("Creating new stack branch %s from %s\n", branchName, parent)

	// Create worktree with new branch
	if err := git.AddWorktreeNewBranch(worktreePath, branchName, parent); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Set parent in git config
	configKey := fmt.Sprintf("branch.%s.stackparent", branchName)
	if err := git.SetConfig(configKey, parent); err != nil {
		return fmt.Errorf("failed to set parent config: %w", err)
	}

	if !dryRun {
		fmt.Printf("✓ Created worktree at %s\n", worktreePath)
		fmt.Printf("✓ Branch %s with parent %s\n", branchName, parent)
	}

	return nil
}

func createWorktreeForExisting(branchName, worktreePath string) error {
	// Check if branch exists locally
	if git.BranchExists(branchName) {
		fmt.Printf("Creating worktree for local branch %s\n", branchName)
		if err := git.AddWorktree(worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		if !dryRun {
			fmt.Printf("✓ Created worktree at %s\n", worktreePath)
		}
		return nil
	}

	// Check if branch exists on remote
	if git.RemoteBranchExists(branchName) {
		fmt.Printf("Creating worktree for remote branch %s\n", branchName)
		if err := git.AddWorktreeFromRemote(worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		if !dryRun {
			fmt.Printf("✓ Created worktree at %s (tracking origin/%s)\n", worktreePath, branchName)
		}
		return nil
	}

	// Branch doesn't exist - create new branch from current HEAD
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	fmt.Printf("Creating new branch %s from %s\n", branchName, currentBranch)
	if err := git.AddWorktreeNewBranch(worktreePath, branchName, currentBranch); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	if !dryRun {
		fmt.Printf("✓ Created worktree at %s (new branch from %s)\n", worktreePath, currentBranch)
	}
	return nil
}

func runWorktreePrune() error {
	// Get repo root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repo root: %w", err)
	}

	worktreesDir := filepath.Join(repoRoot, ".worktrees")

	// Check if .worktrees directory exists
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		fmt.Println("No .worktrees directory found.")
		return nil
	}

	// Get all worktrees and their branches
	worktreeBranches, err := git.GetWorktreeBranches()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Filter to only worktrees in .worktrees/ directory
	var worktreesToCheck []struct {
		path   string
		branch string
	}
	for branch, path := range worktreeBranches {
		if strings.HasPrefix(path, worktreesDir) {
			worktreesToCheck = append(worktreesToCheck, struct {
				path   string
				branch string
			}{path: path, branch: branch})
		}
	}

	if len(worktreesToCheck) == 0 {
		fmt.Println("No worktrees found in .worktrees/ directory.")
		return nil
	}

	// Fetch PR info
	var prCache map[string]*github.PRInfo
	if err := spinner.WrapWithSuccess("Fetching PRs...", "Fetched PRs", func() error {
		var prErr error
		prCache, prErr = github.GetAllPRs()
		return prErr
	}); err != nil {
		return fmt.Errorf("failed to fetch PRs: %w", err)
	}

	// Find worktrees with merged PRs
	var mergedWorktrees []struct {
		path   string
		branch string
	}
	for _, wt := range worktreesToCheck {
		if pr, exists := prCache[wt.branch]; exists && pr.State == "MERGED" {
			mergedWorktrees = append(mergedWorktrees, wt)
		}
	}

	if len(mergedWorktrees) == 0 {
		fmt.Println("\nNo worktrees with merged PRs to prune.")
		return nil
	}

	// Show what will be pruned
	fmt.Println()
	fmt.Printf("Found %d worktree(s) with merged PRs:\n", len(mergedWorktrees))
	for _, wt := range mergedWorktrees {
		pr := prCache[wt.branch]
		fmt.Printf("  - %s (%s, PR #%d)\n", wt.branch, wt.path, pr.Number)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run - no changes made.")
		return nil
	}

	// Remove each worktree
	for i, wt := range mergedWorktrees {
		fmt.Printf("(%d/%d) Removing worktree for %s...\n", i+1, len(mergedWorktrees), wt.branch)

		if err := git.RemoveWorktree(wt.path); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to remove worktree: %v\n", err)
		} else {
			fmt.Println("  ✓ Removed")
		}
	}

	fmt.Println("\n✓ Worktree prune complete!")
	fmt.Println("Tip: Run 'stack prune' to also delete the merged branches.")

	return nil
}

func ensureWorktreesIgnored(repoRoot string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	// Check if .worktrees is already in .gitignore
	if _, err := os.Stat(gitignorePath); err == nil {
		file, err := os.Open(gitignorePath)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == ".worktrees" || line == ".worktrees/" {
				return nil // Already ignored
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	if dryRun {
		fmt.Println("  [DRY RUN] Adding .worktrees to .gitignore")
		return nil
	}

	// Append .worktrees to .gitignore
	file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Check if file ends with newline, if not add one
	info, err := file.Stat()
	if err != nil {
		return err
	}

	var prefix string
	if info.Size() > 0 {
		// Read last byte to check for newline
		tempFile, err := os.Open(gitignorePath)
		if err != nil {
			return err
		}
		defer tempFile.Close()

		buf := make([]byte, 1)
		_, err = tempFile.ReadAt(buf, info.Size()-1)
		if err != nil {
			return err
		}
		if buf[0] != '\n' {
			prefix = "\n"
		}
	}

	_, err = file.WriteString(prefix + ".worktrees/\n")
	if err != nil {
		return err
	}

	fmt.Println("Added .worktrees/ to .gitignore")
	return nil
}
