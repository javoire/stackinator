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
	"github.com/javoire/stackinator/internal/ui"
	"github.com/spf13/cobra"
)

var worktreePrune bool

var worktreeCmd = &cobra.Command{
	Use:   "worktree <branch-name> [base-branch]",
	Short: "Create a worktree in .worktrees/ directory",
	Long: `Create a git worktree in the .worktrees/ directory for the specified branch.

If the branch exists locally or on the remote, it will be used.
If the branch doesn't exist, a new branch will be created from the current branch
(or from base-branch if specified) and stack tracking will be set up automatically.
Use --prune to clean up worktrees for branches with merged PRs.`,
	Example: `  # Create worktree for new branch (from current branch, with stack tracking)
  stack worktree my-feature

  # Create worktree from a fresh main branch
  stack worktree my-feature main

  # Create worktree for existing local or remote branch
  stack worktree existing-branch

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
		if len(args) < 1 || len(args) > 2 {
			return fmt.Errorf("requires 1 or 2 arguments: branch name [base-branch]")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()
		repo := github.ParseRepoFromURL(gitClient.GetRemoteURL("origin"))
		githubClient := github.NewGitHubClient(repo)

		var err error
		if worktreePrune {
			err = runWorktreePrune(gitClient, githubClient)
		} else {
			var baseBranch string
			if len(args) > 1 {
				baseBranch = args[1]
			}
			err = runWorktree(gitClient, githubClient, args[0], baseBranch)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	worktreeCmd.Flags().BoolVar(&worktreePrune, "prune", false, "Remove worktrees for branches with merged PRs")
}

func runWorktree(gitClient git.GitClient, githubClient github.GitHubClient, branchName, baseBranch string) error {
	// Get repo root
	repoRoot, err := gitClient.GetRepoRoot()
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

	// If base branch is specified, always create new branch from it
	if baseBranch != "" {
		return createNewBranchWorktree(gitClient, branchName, baseBranch, worktreePath)
	}

	// Check if branch exists locally or on remote
	return createWorktreeForExisting(gitClient, branchName, worktreePath)
}

func createNewBranchWorktree(gitClient git.GitClient, branchName, baseBranch, worktreePath string) error {
	// Check if branch already exists
	if gitClient.BranchExists(branchName) {
		return fmt.Errorf("branch %s already exists", branchName)
	}

	// Verify base branch exists (locally or on remote)
	if !gitClient.BranchExists(baseBranch) && !gitClient.RemoteBranchExists(baseBranch) {
		return fmt.Errorf("base branch %s does not exist locally or on remote", baseBranch)
	}

	// Use origin/baseBranch if it's a remote branch to get fresh copy
	baseRef := baseBranch
	if gitClient.RemoteBranchExists(baseBranch) {
		baseRef = "origin/" + baseBranch
	}

	fmt.Printf("Creating new branch %s from %s\n", ui.Branch(branchName), ui.Branch(baseRef))

	// Create worktree with new branch
	if err := gitClient.AddWorktreeNewBranch(worktreePath, branchName, baseRef); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Set parent in git config for stack tracking
	configKey := fmt.Sprintf("branch.%s.stackparent", branchName)
	if err := gitClient.SetConfig(configKey, baseBranch); err != nil {
		return fmt.Errorf("failed to set parent config: %w", err)
	}

	if !dryRun {
		fmt.Println(ui.Success(fmt.Sprintf("Created worktree at %s", worktreePath)))
		fmt.Println(ui.Success(fmt.Sprintf("Branch %s with parent %s", ui.Branch(branchName), ui.Branch(baseBranch))))
		fmt.Printf("\nTo switch to this worktree, run:\n  %s\n", ui.Command(fmt.Sprintf("cd %s", worktreePath)))
	}

	return nil
}

func createWorktreeForExisting(gitClient git.GitClient, branchName, worktreePath string) error {
	// Check if branch exists locally
	if gitClient.BranchExists(branchName) {
		fmt.Printf("Creating worktree for local branch %s\n", ui.Branch(branchName))
		if err := gitClient.AddWorktree(worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		if !dryRun {
			fmt.Println(ui.Success(fmt.Sprintf("Created worktree at %s", worktreePath)))
			fmt.Printf("\nTo switch to this worktree, run:\n  %s\n", ui.Command(fmt.Sprintf("cd %s", worktreePath)))
		}
		return nil
	}

	// Check if branch exists on remote
	if gitClient.RemoteBranchExists(branchName) {
		fmt.Printf("Creating worktree for remote branch %s\n", ui.Branch(branchName))
		if err := gitClient.AddWorktreeFromRemote(worktreePath, branchName); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		if !dryRun {
			fmt.Println(ui.Success(fmt.Sprintf("Created worktree at %s (tracking origin/%s)", worktreePath, branchName)))
			fmt.Printf("\nTo switch to this worktree, run:\n  %s\n", ui.Command(fmt.Sprintf("cd %s", worktreePath)))
		}
		return nil
	}

	// Branch doesn't exist - create new branch from current branch with stack tracking
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	fmt.Printf("Creating new branch %s from %s\n", ui.Branch(branchName), ui.Branch(currentBranch))
	if err := gitClient.AddWorktreeNewBranch(worktreePath, branchName, currentBranch); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Set parent in git config for stack tracking
	configKey := fmt.Sprintf("branch.%s.stackparent", branchName)
	if err := gitClient.SetConfig(configKey, currentBranch); err != nil {
		return fmt.Errorf("failed to set parent config: %w", err)
	}

	if !dryRun {
		fmt.Println(ui.Success(fmt.Sprintf("Created worktree at %s", worktreePath)))
		fmt.Println(ui.Success(fmt.Sprintf("Branch %s with parent %s", ui.Branch(branchName), ui.Branch(currentBranch))))
		fmt.Printf("\nTo switch to this worktree, run:\n  %s\n", ui.Command(fmt.Sprintf("cd %s", worktreePath)))
	}
	return nil
}

func runWorktreePrune(gitClient git.GitClient, githubClient github.GitHubClient) error {
	// Get repo root
	repoRoot, err := gitClient.GetRepoRoot()
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
	worktreeBranches, err := gitClient.GetWorktreeBranches()
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
		prCache, prErr = githubClient.GetAllPRs()
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
		fmt.Printf("  - %s (%s, PR #%d)\n", ui.Branch(wt.branch), wt.path, pr.Number)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run - no changes made.")
		return nil
	}

	// Remove each worktree
	for i, wt := range mergedWorktrees {
		fmt.Printf("%s Removing worktree for %s...\n", ui.Progress(i+1, len(mergedWorktrees)), ui.Branch(wt.branch))

		if err := gitClient.RemoveWorktree(wt.path); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to remove worktree: %v\n", err)
		} else {
			fmt.Printf("  %s Removed\n", ui.SuccessIcon())
		}
	}

	fmt.Println()
	fmt.Println(ui.Success("Worktree prune complete!"))
	fmt.Printf("Tip: Run '%s' to also delete the merged branches.\n", ui.Command("stack prune"))

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
