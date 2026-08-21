package cmd

import (
	"crypto/rand"
	"encoding/hex"
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
var worktreePruneAll bool
var worktreeList bool

var worktreeCmd = &cobra.Command{
	Use:   "worktree [branch-name] [base-branch]",
	Short: "Create a worktree in the configured worktrees directory",
	Long: `Create a git worktree in the configured worktrees directory.

If the branch exists locally or on the remote, it will be used.
If the branch doesn't exist, a new branch will be created from the current branch
(or from base-branch if specified) and stack tracking will be set up automatically.
If no branch name is specified, a randomized branch name will be generated.
Use --list to show worktrees for this repository, or --list --all for all repos.
Use --prune to clean up worktrees for branches with merged PRs.
Use --prune --all to remove all worktrees for this repository.

By default, worktrees are created under ~/.stack/worktrees/<reponame>.
You can change this with: git config stack.worktreesDir <path> (or use 'stack config set')`,
	Example: `  # Create a worktree with a randomized branch name
  stack worktree

  # Create worktree for new branch (from current branch, with stack tracking)
  stack worktree my-feature

  # Create worktree from a fresh main branch
  stack worktree my-feature main

  # Create worktree for existing local or remote branch
  stack worktree existing-branch

  # List worktrees for this repository
  stack worktree --list

  # List worktrees for all repositories
  stack worktree --list --all

  # Clean up worktrees for merged branches
  stack worktree --prune

  # Remove all worktrees for this repository
  stack worktree --prune --all

  # Preview without executing
  stack worktree my-feature --dry-run`,
	Args: func(cmd *cobra.Command, args []string) error {
		if worktreePruneAll && !worktreeList && !worktreePrune {
			return fmt.Errorf("--all requires --list or --prune")
		}
		if worktreeList {
			if len(args) > 0 {
				return fmt.Errorf("--list does not take arguments")
			}
			return nil
		}
		if worktreePrune {
			if len(args) > 0 {
				return fmt.Errorf("--prune does not take a branch argument")
			}
			return nil
		}
		if len(args) > 2 {
			return fmt.Errorf("accepts at most 2 arguments: branch name [base-branch]")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()
		repo := github.ParseRepoFromURL(gitClient.GetRemoteURL("origin"))
		githubClient := github.NewGitHubClient(repo)

		var err error
		if worktreeList {
			err = runWorktreeList(gitClient)
		} else if worktreePrune {
			err = runWorktreePrune(gitClient, githubClient)
		} else {
			branchName := ""
			if len(args) > 0 {
				branchName = args[0]
			}
			var baseBranch string
			if len(args) > 1 {
				baseBranch = args[1]
			}
			if branchName == "" {
				branchName, err = generateRandomWorktreeName()
				if err == nil {
					fmt.Printf("Generated branch name %s\n", ui.Branch(branchName))
				}
			}
			if err == nil {
				err = runWorktree(gitClient, githubClient, branchName, baseBranch)
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func generateRandomWorktreeName() (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random worktree name: %w", err)
	}
	return "worktree-" + hex.EncodeToString(randomBytes), nil
}

func init() {
	worktreeCmd.Flags().BoolVarP(&worktreeList, "list", "l", false, "List all worktrees for this repository")
	worktreeCmd.Flags().BoolVar(&worktreePrune, "prune", false, "Remove worktrees for branches with merged PRs")
	worktreeCmd.Flags().BoolVarP(&worktreePruneAll, "all", "a", false, "With --list: show all repos. With --prune: remove all worktrees")
}

func runWorktree(gitClient git.GitClient, githubClient github.GitHubClient, branchName, baseBranch string) error {
	worktreesBaseDir, err := getWorktreesBaseDir(gitClient)
	if err != nil {
		return err
	}

	// Get repository name
	repoName, err := gitClient.GetRepoName()
	if err != nil {
		return fmt.Errorf("failed to get repo name: %w", err)
	}

	// Worktree path: <worktreesBaseDir>/<reponame>/<branchname>
	worktreePath := filepath.Join(worktreesBaseDir, repoName, branchName)

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

func runWorktreeList(gitClient git.GitClient) error {
	worktreesBaseDir, err := getWorktreesBaseDir(gitClient)
	if err != nil {
		return err
	}

	// Check if ~/.stack/worktrees directory exists
	if _, err := os.Stat(worktreesBaseDir); os.IsNotExist(err) {
		fmt.Printf("No worktrees found in %s\n", worktreesBaseDir)
		return nil
	}

	if worktreePruneAll {
		// List worktrees for all repositories
		return listAllWorktrees(worktreesBaseDir)
	}

	// List worktrees for current repository only
	repoName, err := gitClient.GetRepoName()
	if err != nil {
		return fmt.Errorf("failed to get repo name: %w", err)
	}

	worktreesDir := filepath.Join(worktreesBaseDir, repoName)

	// Check if ~/.stack/worktrees/<reponame> directory exists
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		fmt.Printf("No worktrees found in %s\n", worktreesDir)
		return nil
	}

	// Get all worktrees and their branches
	worktreeBranches, err := gitClient.GetWorktreeBranches()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Filter to only worktrees in ~/.stack/worktrees/<reponame> directory
	var worktrees []struct {
		path   string
		branch string
	}
	for branch, path := range worktreeBranches {
		if pathWithinDir(path, worktreesDir) {
			worktrees = append(worktrees, struct {
				path   string
				branch string
			}{path: path, branch: branch})
		}
	}

	if len(worktrees) == 0 {
		fmt.Printf("No worktrees found in %s\n", worktreesDir)
		return nil
	}

	fmt.Printf("Worktrees in %s:\n\n", worktreesDir)
	for _, wt := range worktrees {
		fmt.Printf("  %s\n    %s\n\n", ui.Branch(wt.branch), wt.path)
	}

	return nil
}

func listAllWorktrees(worktreesBaseDir string) error {
	// Read all repo directories
	entries, err := os.ReadDir(worktreesBaseDir)
	if err != nil {
		return fmt.Errorf("failed to read worktrees directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}

	totalCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoName := entry.Name()
		repoWorktreesDir := filepath.Join(worktreesBaseDir, repoName)

		// Read worktree directories for this repo
		worktreeEntries, err := os.ReadDir(repoWorktreesDir)
		if err != nil {
			continue
		}

		var branches []string
		for _, wt := range worktreeEntries {
			if wt.IsDir() {
				branches = append(branches, wt.Name())
			}
		}

		if len(branches) == 0 {
			continue
		}

		fmt.Printf("%s:\n", repoName)
		for _, branch := range branches {
			path := filepath.Join(repoWorktreesDir, branch)
			fmt.Printf("  %s\n    %s\n\n", ui.Branch(branch), path)
		}
		totalCount += len(branches)
	}

	if totalCount == 0 {
		fmt.Printf("No worktrees found in %s\n", worktreesBaseDir)
	}

	return nil
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
		if existingPath := parseWorktreeInUseError(err); existingPath != "" {
			return fmt.Errorf("branch %s is already checked out in another worktree\n\n"+
				"Existing worktree: %s\n\n"+
				"Options:\n"+
				"  1. Navigate to existing worktree:\n"+
				"     %s\n\n"+
				"  2. Remove existing worktree and retry:\n"+
				"     %s",
				ui.Branch(branchName),
				existingPath,
				ui.Command(fmt.Sprintf("cd %s", existingPath)),
				ui.Command(fmt.Sprintf("git worktree remove %s", existingPath)))
		}
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
		symlinkEnvFile(gitClient, worktreePath)
		fmt.Printf("\nTo switch to this worktree, run:\n  %s\n", ui.Command(fmt.Sprintf("cd %s", worktreePath)))
	}

	return nil
}

func createWorktreeForExisting(gitClient git.GitClient, branchName, worktreePath string) error {
	// Check if branch exists locally
	if gitClient.BranchExists(branchName) {
		fmt.Printf("Creating worktree for local branch %s\n", ui.Branch(branchName))
		if err := gitClient.AddWorktree(worktreePath, branchName); err != nil {
			if existingPath := parseWorktreeInUseError(err); existingPath != "" {
				return fmt.Errorf("branch %s is already checked out in another worktree\n\n"+
					"Existing worktree: %s\n\n"+
					"Options:\n"+
					"  1. Navigate to existing worktree:\n"+
					"     %s\n\n"+
					"  2. Remove existing worktree and retry:\n"+
					"     %s",
					ui.Branch(branchName),
					existingPath,
					ui.Command(fmt.Sprintf("cd %s", existingPath)),
					ui.Command(fmt.Sprintf("git worktree remove %s", existingPath)))
			}
			return fmt.Errorf("failed to create worktree: %w", err)
		}
		if !dryRun {
			fmt.Println(ui.Success(fmt.Sprintf("Created worktree at %s", worktreePath)))
			symlinkEnvFile(gitClient, worktreePath)
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
			symlinkEnvFile(gitClient, worktreePath)
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
		symlinkEnvFile(gitClient, worktreePath)
		fmt.Printf("\nTo switch to this worktree, run:\n  %s\n", ui.Command(fmt.Sprintf("cd %s", worktreePath)))
	}
	return nil
}

func runWorktreePrune(gitClient git.GitClient, githubClient github.GitHubClient) error {
	worktreesBaseDir, err := getWorktreesBaseDir(gitClient)
	if err != nil {
		return err
	}

	// Get repository name
	repoName, err := gitClient.GetRepoName()
	if err != nil {
		return fmt.Errorf("failed to get repo name: %w", err)
	}

	worktreesDir := filepath.Join(worktreesBaseDir, repoName)

	// Check if ~/.stack/worktrees/<reponame> directory exists
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		fmt.Printf("No ~/.stack/worktrees/%s directory found.\n", repoName)
		return nil
	}

	// Get all worktrees and their branches
	worktreeBranches, err := gitClient.GetWorktreeBranches()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Filter to only worktrees in ~/.stack/worktrees/<reponame> directory
	var worktreesToCheck []struct {
		path   string
		branch string
	}
	for branch, path := range worktreeBranches {
		if pathWithinDir(path, worktreesDir) {
			worktreesToCheck = append(worktreesToCheck, struct {
				path   string
				branch string
			}{path: path, branch: branch})
		}
	}

	if len(worktreesToCheck) == 0 {
		fmt.Printf("No worktrees found in ~/.stack/worktrees/%s directory.\n", repoName)
		return nil
	}

	// Determine which worktrees to prune
	var worktreesToPrune []struct {
		path   string
		branch string
	}

	if worktreePruneAll {
		// Prune all worktrees
		worktreesToPrune = worktreesToCheck

		fmt.Println()
		fmt.Printf("Found %d worktree(s) to remove:\n", len(worktreesToPrune))
		for _, wt := range worktreesToPrune {
			fmt.Printf("  - %s (%s)\n", wt.branch, wt.path)
		}
		fmt.Println()
	} else {
		// Prune only worktrees with merged PRs
		var wtBranches []string
		for _, wt := range worktreesToCheck {
			wtBranches = append(wtBranches, wt.branch)
		}
		var prCache map[string]*github.PRInfo
		if err := spinner.WrapWithSuccess("Fetching PRs...", "Fetched PRs", func() error {
			var err error
			prCache, err = githubClient.GetPRsForBranches(wtBranches)
			return err
		}); err != nil {
			return fmt.Errorf("failed to fetch PRs: %w", err)
		}

		for _, wt := range worktreesToCheck {
			if pr, exists := prCache[wt.branch]; exists && pr.State == "MERGED" {
				worktreesToPrune = append(worktreesToPrune, wt)
			}
		}

		if len(worktreesToPrune) == 0 {
			fmt.Println("\nNo worktrees with merged PRs to prune.")
			return nil
		}

		fmt.Println()
		fmt.Printf("Found %d worktree(s) with merged PRs:\n", len(worktreesToPrune))
		for _, wt := range worktreesToPrune {
			pr := prCache[wt.branch]
			fmt.Printf("  - %s (%s, PR #%d)\n", ui.Branch(wt.branch), wt.path, pr.Number)
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Println("Dry run - no changes made.")
		return nil
	}

	// Remove each worktree
	for i, wt := range worktreesToPrune {
		fmt.Printf("%s Removing worktree for %s...\n", ui.Progress(i+1, len(worktreesToPrune)), ui.Branch(wt.branch))

		if err := gitClient.RemoveWorktree(wt.path); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to remove worktree: %v\n", err)
		} else {
			fmt.Printf("  %s Removed\n", ui.SuccessIcon())
		}
	}

	fmt.Println()
	fmt.Println(ui.Success("Worktree prune complete!"))
	if !worktreePruneAll {
		fmt.Printf("Tip: Run '%s' to also delete the merged branches.\n", ui.Command("stack prune"))
	}

	return nil
}

func getHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return homeDir, nil
}

func getDefaultWorktreesBaseDir() (string, error) {
	homeDir, err := getHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".stack", "worktrees"), nil
}

func getWorktreesBaseDir(gitClient git.GitClient) (string, error) {
	defaultDir, err := getDefaultWorktreesBaseDir()
	if err != nil {
		return "", err
	}
	homeDir, err := getHomeDir()
	if err != nil {
		return "", err
	}
	repoRoot, repoErr := gitClient.GetRepoRoot()

	configured := strings.TrimSpace(gitClient.GetConfig("stack.worktreesDir"))
	if configured == "" {
		return defaultDir, nil
	}

	expanded := os.ExpandEnv(configured)
	if trimmed, found := strings.CutPrefix(expanded, "~"); found {
		trimmed = strings.TrimPrefix(trimmed, string(os.PathSeparator))
		expanded = filepath.Join(homeDir, trimmed)
	}
	if !filepath.IsAbs(expanded) {
		if repoErr == nil {
			expanded = filepath.Join(repoRoot, expanded)
		} else {
			expanded = filepath.Join(homeDir, expanded)
		}
	}

	return filepath.Clean(expanded), nil
}

func pathWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func symlinkEnvFile(gitClient git.GitClient, worktreePath string) {
	repoRoot, err := gitClient.GetRepoRoot()
	if err != nil {
		return
	}

	envPath := filepath.Join(repoRoot, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return // No .env file, nothing to do
	}

	targetPath := filepath.Join(worktreePath, ".env")

	if dryRun {
		fmt.Printf("Would symlink .env from %s\n", envPath)
		return
	}

	if err := os.Symlink(envPath, targetPath); err != nil {
		// Silently ignore errors (e.g., symlink already exists)
		return
	}

	fmt.Println(ui.Success("Symlinked .env file"))
}

// parseWorktreeInUseError extracts the existing worktree path from a git error.
// Returns the path if error matches "is already used by worktree at", empty string otherwise.
func parseWorktreeInUseError(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()
	// Git error format: fatal: 'branch' is already used by worktree at 'path'
	marker := "is already used by worktree at '"
	idx := strings.Index(errStr, marker)
	if idx == -1 {
		return ""
	}
	pathStart := idx + len(marker)
	pathEnd := strings.LastIndex(errStr, "'")
	if pathEnd <= pathStart {
		return ""
	}
	return errStr[pathStart:pathEnd]
}
