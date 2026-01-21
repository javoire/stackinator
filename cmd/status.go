package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/spinner"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

var (
	noPR bool
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

  # Show without PR info (faster)
  stack status --no-pr

  # Example output:
  #  main
  #   |
  #  feature-auth [PR #123: OPEN]
  #   |
  #  feature-auth-tests *`,
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()
		repo := github.ParseRepoFromURL(gitClient.GetRemoteURL("origin"))
		githubClient := github.NewGitHubClient(repo)

		if err := runStatus(gitClient, githubClient); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	statusCmd.Flags().BoolVar(&noPR, "no-pr", false, "Skip fetching PR information (faster)")
}

func runStatus(gitClient git.GitClient, githubClient github.GitHubClient) error {
	var currentBranch string
	var stackBranches []stack.StackBranch
	var tree *stack.TreeNode
	var allTreeBranches []string

	// Start fetch and PR loading in parallel with stack tree building (if not --no-pr)
	// These are the slowest operations and can run while we build the tree
	var wg sync.WaitGroup
	var prCache map[string]*github.PRInfo
	var prErr error
	fetchDone := false

	if !noPR {
		wg.Add(2)
		go func() {
			defer wg.Done()
			prCache, prErr = githubClient.GetAllPRs()
			if prErr != nil {
				// If fetching fails, fall back to empty cache
				prCache = make(map[string]*github.PRInfo)
			}
		}()
		go func() {
			defer wg.Done()
			// Fetch latest changes from origin (needed for sync issue detection)
			_ = gitClient.Fetch()
			fetchDone = true
		}()
	} else {
		prCache = make(map[string]*github.PRInfo)
	}

	// Build the stack tree (runs in parallel with PR fetch)
	if err := spinner.WrapWithAutoDelay("Loading stack...", 300*time.Millisecond, func() error {
		// Get current branch
		var err error
		currentBranch, err = gitClient.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// Check if there are any stack branches
		stackBranches, err = stack.GetStackBranches(gitClient)
		if err != nil {
			return fmt.Errorf("failed to get stack branches: %w", err)
		}

		if len(stackBranches) == 0 {
			return nil // Handle this after the spinner
		}

		// Build stack tree for current branch only
		tree, err = stack.BuildStackTreeForBranch(gitClient, currentBranch)
		if err != nil {
			return fmt.Errorf("failed to build stack tree: %w", err)
		}

		// Get ALL branch names in the tree (including intermediate branches without stackparent)
		allTreeBranches = getAllBranchNamesFromTree(tree)
		return nil
	}); err != nil {
		return err
	}

	if len(stackBranches) == 0 {
		// Wait for PR fetch to complete before returning
		wg.Wait()
		fmt.Println("No stack branches found.")
		fmt.Printf("Current branch: %s\n", currentBranch)
		fmt.Println("\nUse 'stack new <branch-name>' to create a new stack branch.")
		return nil
	}

	// Wait for PR fetch to complete (if running)
	if !noPR {
		wg.Wait()
	}

	// Filter out branches with merged PRs from the tree (but keep current branch)
	tree = filterMergedBranches(tree, prCache, currentBranch)

	// Print the tree
	fmt.Println()
	printTree(gitClient, tree, "", true, currentBranch, prCache)

	// Check for sync issues (skip if --no-pr)
	if !noPR {
		// Filter stackBranches to only include branches in the current tree
		branchSet := make(map[string]bool)
		for _, name := range allTreeBranches {
			branchSet[name] = true
		}
		var treeBranches []stack.StackBranch
		for _, branch := range stackBranches {
			if branchSet[branch.Name] {
				treeBranches = append(treeBranches, branch)
			}
		}

		var syncResult *syncIssuesResult
		if err := spinner.WrapWithAutoDelayAndProgress("Checking for sync issues...", 300*time.Millisecond, func(progress spinner.ProgressFunc) error {
			var err error
			syncResult, err = detectSyncIssues(gitClient, treeBranches, prCache, progress, fetchDone)
			return err
		}); err != nil {
			// Don't fail on detection errors, just skip the check
			return nil
		}
		// Print the result after spinner is stopped
		if syncResult != nil {
			printSyncIssues(syncResult)
		}
	}

	return nil
}

// getAllBranchNamesFromTree extracts all branch names from the tree
// (including intermediate branches that may not have stackparent config)
func getAllBranchNamesFromTree(node *stack.TreeNode) []string {
	if node == nil {
		return nil
	}

	var result []string
	var traverse func(*stack.TreeNode)
	traverse = func(n *stack.TreeNode) {
		if n == nil {
			return
		}
		// Add current branch name
		result = append(result, n.Name)
		// Traverse children
		for _, child := range n.Children {
			traverse(child)
		}
	}

	traverse(node)
	return result
}

// filterMergedBranches removes branches with merged PRs from the tree,
// but only if they don't have children (to keep the stack structure visible)
// and they are not the current branch (always show where user is)
func filterMergedBranches(node *stack.TreeNode, prCache map[string]*github.PRInfo, currentBranch string) *stack.TreeNode {
	if node == nil {
		return nil
	}

	// Filter children recursively first
	var filteredChildren []*stack.TreeNode
	for _, child := range node.Children {
		// Recurse first to process all descendants
		filtered := filterMergedBranches(child, prCache, currentBranch)

		// Only filter out merged branches if they have no children
		// (i.e., they're leaf nodes) AND they're not the current branch
		if pr, exists := prCache[child.Name]; exists && pr.State == "MERGED" {
			// Always keep the current branch, even if merged
			if child.Name == currentBranch {
				filteredChildren = append(filteredChildren, filtered)
			} else if filtered != nil && len(filtered.Children) > 0 {
				// If this merged branch still has children after filtering, keep it
				// so the stack structure remains visible
				filteredChildren = append(filteredChildren, filtered)
			}
			// Otherwise skip this merged leaf branch
		} else {
			// Not merged, keep it
			if filtered != nil {
				filteredChildren = append(filteredChildren, filtered)
			}
		}
	}

	node.Children = filteredChildren
	return node
}

func printTree(gitClient git.GitClient, node *stack.TreeNode, prefix string, isLast bool, currentBranch string, prCache map[string]*github.PRInfo) {
	if node == nil {
		return
	}

	// Flatten the tree into a vertical list
	printTreeVertical(gitClient, node, currentBranch, prCache, false)
}

func printTreeVertical(gitClient git.GitClient, node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo, isPipe bool) {
	if node == nil {
		return
	}

	// Determine the current branch marker
	marker := ""
	if node.Name == currentBranch {
		marker = " *"
	}

	// Get PR info from cache
	prInfo := ""
	if node.Name != stack.GetBaseBranch(gitClient) {
		if pr, exists := prCache[node.Name]; exists {
			prInfo = fmt.Sprintf(" [%s :%s]", pr.URL, strings.ToLower(pr.State))
		}
	}

	// Print pipe if needed
	if isPipe {
		fmt.Println("  |")
	}

	// Print current node
	fmt.Printf(" %s%s%s\n", node.Name, prInfo, marker)

	// Print children vertically
	for _, child := range node.Children {
		printTreeVertical(gitClient, child, currentBranch, prCache, true)
	}
}

// syncIssuesResult holds the result of detectSyncIssues
type syncIssuesResult struct {
	issues         []string
	mergedBranches []string
}

// detectSyncIssues checks if any branches are out of sync and returns the issues (doesn't print)
// If skipFetch is true, assumes git fetch was already called (to avoid redundant network calls)
func detectSyncIssues(gitClient git.GitClient, stackBranches []stack.StackBranch, prCache map[string]*github.PRInfo, progress spinner.ProgressFunc, skipFetch bool) (*syncIssuesResult, error) {
	var issues []string
	var mergedBranches []string

	// Fetch once upfront to ensure we have latest remote refs (unless already done)
	if !skipFetch {
		progress("Fetching latest changes...")
		if verbose {
			fmt.Println("Fetching latest changes from origin...")
		}
		_ = gitClient.Fetch()
	}

	if verbose {
		fmt.Printf("Checking %d branch(es) for sync issues...\n", len(stackBranches))
	}

	// Check each stack branch for sync issues
	for i, branch := range stackBranches {
		progress(fmt.Sprintf("Checking branch %d/%d (%s)...", i+1, len(stackBranches), branch.Name))

		if verbose {
			fmt.Printf("\n[%d/%d] Checking '%s' (parent: %s)\n", i+1, len(stackBranches), branch.Name, branch.Parent)
		}

		// Track branches with merged PRs (for cleanup suggestion, not sync)
		if pr, exists := prCache[branch.Name]; exists && pr.State == "MERGED" {
			if verbose {
				fmt.Printf("  ✓ Branch has merged PR #%d - marking for cleanup\n", pr.Number)
			}
			mergedBranches = append(mergedBranches, branch.Name)
			continue // Don't check other sync issues for merged branches
		}

		// Check if parent has a merged PR (child needs to be updated)
		if branch.Parent != stack.GetBaseBranch(gitClient) {
			if parentPR, exists := prCache[branch.Parent]; exists && parentPR.State == "MERGED" {
				if verbose {
					fmt.Printf("  ✗ Parent '%s' has merged PR #%d\n", branch.Parent, parentPR.Number)
				}
				issues = append(issues, fmt.Sprintf("  - Branch '%s' parent '%s' has a merged PR", branch.Name, branch.Parent))
			} else if verbose {
				fmt.Printf("  ✓ Parent '%s' is not merged\n", branch.Parent)
			}
		}

		// Check if PR base matches the configured parent (if PR exists)
		if pr, exists := prCache[branch.Name]; exists {
			if verbose {
				fmt.Printf("  Found PR #%d (base: %s, state: %s)\n", pr.Number, pr.Base, pr.State)
			}

			if pr.Base != branch.Parent {
				if verbose {
					fmt.Printf("  ✗ PR base (%s) doesn't match configured parent (%s)\n", pr.Base, branch.Parent)
				}
				issues = append(issues, fmt.Sprintf("  - Branch '%s' PR base (%s) doesn't match parent (%s)", branch.Name, pr.Base, branch.Parent))
			} else if verbose {
				fmt.Printf("  ✓ PR base matches configured parent\n")
			}
		} else if verbose {
			fmt.Printf("  No PR found for this branch\n")
		}

		// Check if branch is behind its parent (needs rebase) - always check this regardless of PR
		if verbose {
			fmt.Printf("  Checking if branch is behind parent %s...\n", branch.Parent)
		}
		behind, err := gitClient.IsCommitsBehind(branch.Name, branch.Parent)
		if err == nil && behind {
			if verbose {
				fmt.Printf("  ✗ Branch is behind %s (needs rebase)\n", branch.Parent)
			}
			issues = append(issues, fmt.Sprintf("  - Branch '%s' is behind %s (needs rebase)", branch.Name, branch.Parent))
		} else if err == nil && verbose {
			fmt.Printf("  ✓ Branch is up to date with %s\n", branch.Parent)
		} else if err != nil && verbose {
			fmt.Printf("  ⚠ Could not check if branch is behind: %v\n", err)
		}

		// Check if local branch differs from remote (needs push)
		if gitClient.RemoteBranchExists(branch.Name) {
			if verbose {
				fmt.Printf("  Checking if local branch differs from origin/%s...\n", branch.Name)
			}
			localHash, localErr := gitClient.GetCommitHash(branch.Name)
			remoteHash, remoteErr := gitClient.GetCommitHash("origin/" + branch.Name)
			if localErr == nil && remoteErr == nil && localHash != remoteHash {
				if verbose {
					fmt.Printf("  ✗ Local branch differs from origin/%s (needs push)\n", branch.Name)
				}
				issues = append(issues, fmt.Sprintf("  - Branch '%s' differs from origin (needs push)", branch.Name))
			} else if localErr == nil && remoteErr == nil && verbose {
				fmt.Printf("  ✓ Local branch matches origin/%s\n", branch.Name)
			} else if verbose {
				if localErr != nil {
					fmt.Printf("  ⚠ Could not get local commit hash: %v\n", localErr)
				}
				if remoteErr != nil {
					fmt.Printf("  ⚠ Could not get remote commit hash: %v\n", remoteErr)
				}
			}
		} else if verbose {
			fmt.Printf("  ℹ No remote branch origin/%s found\n", branch.Name)
		}
	}

	return &syncIssuesResult{
		issues:         issues,
		mergedBranches: mergedBranches,
	}, nil
}

// printSyncIssues prints the sync issues result
func printSyncIssues(result *syncIssuesResult) {
	// If issues found, print warning
	if len(result.issues) > 0 {
		fmt.Println()
		fmt.Println("⚠ Stack out of sync detected:")
		for _, issue := range result.issues {
			fmt.Println(issue)
		}
		fmt.Println()
		fmt.Println("Run 'stack sync' to rebase branches and update PR bases.")

		// Also mention merged branches if any
		if len(result.mergedBranches) > 0 {
			fmt.Println()
			fmt.Printf("After syncing, clean up merged branches with 'stack prune': %s\n", strings.Join(result.mergedBranches, ", "))
		}
	} else if len(result.mergedBranches) > 0 {
		// Merged branches need cleanup via prune
		fmt.Println()
		fmt.Printf("⚠ Merged branches need cleanup: %s\n", strings.Join(result.mergedBranches, ", "))
		fmt.Println()
		fmt.Println("Run 'stack prune' to remove merged branches.")
	} else {
		// Everything is perfectly synced
		fmt.Println()
		fmt.Println("✓ Stack is perfectly synced! All branches are up to date.")
	}
}

// Helper to repeat a string n times
func repeatString(s string, n int) string {
	return strings.Repeat(s, n)
}
