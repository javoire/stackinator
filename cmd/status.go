package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/spinner"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/javoire/stackinator/internal/ui"
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

	// Start fetch in parallel with stack tree building (if not --no-pr)
	var fetchWg sync.WaitGroup
	var prCache map[string]*github.PRInfo
	fetchDone := false

	if !noPR {
		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			// Fetch latest changes from origin (needed for sync issue detection)
			_ = gitClient.Fetch()
			fetchDone = true
		}()
	} else {
		prCache = make(map[string]*github.PRInfo)
	}

	// Build the stack tree, then fetch PRs for tree branches only
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

		// If tree is nil, current branch is not in a stack
		if tree == nil {
			return nil // Will be handled after spinner
		}

		// Get ALL branch names in the tree (including intermediate branches without stackparent)
		allTreeBranches = getAllBranchNamesFromTree(tree)

		// Fetch PRs for stack branches only (parallel individual fetches)
		if !noPR {
			prCache, err = githubClient.GetPRsForBranches(allTreeBranches)
			if err != nil {
				return fmt.Errorf("failed to load PRs: %w", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if len(stackBranches) == 0 {
		// Wait for fetch to complete before returning
		fetchWg.Wait()
		fmt.Println("No stack branches found.")
		fmt.Printf("Current branch: %s\n", ui.Branch(currentBranch))
		fmt.Printf("\nUse '%s' to create a new stack branch.\n", ui.Command("stack new <branch-name>"))
		return nil
	}

	// If tree is nil, current branch is not part of any stack
	// Check this BEFORE waiting for PR fetch to avoid long delays
	if tree == nil {
		baseBranch := stack.GetBaseBranch(gitClient)

		// Don't offer to add the base branch to a stack - it can't have a parent
		if currentBranch == baseBranch {
			fmt.Printf("No stack found. You're on the base branch (%s).\n", ui.Branch(currentBranch))
			fmt.Printf("\nUse '%s' to create a new stack branch.\n", ui.Command("stack new <branch-name>"))
			return nil
		}

		fmt.Printf("Current branch '%s' is not part of a stack.\n\n", ui.Branch(currentBranch))
		fmt.Printf("Add to stack with '%s' as parent? [Y/n] ", ui.Branch(baseBranch))

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" || input == "y" || input == "yes" {
			// Set the stackparent config
			configKey := fmt.Sprintf("branch.%s.stackparent", currentBranch)
			if err := gitClient.SetConfig(configKey, baseBranch); err != nil {
				return fmt.Errorf("failed to set stack parent: %w", err)
			}
			fmt.Println(ui.Success(fmt.Sprintf("Added '%s' to stack with parent '%s'", ui.Branch(currentBranch), ui.Branch(baseBranch))))
			fmt.Println()
			// Run status again to show the stack
			return runStatus(gitClient, githubClient)
		}
		return nil
	}

	// Print the tree
	fmt.Println()
	printTree(gitClient, tree, "", true, currentBranch, prCache)

	// Check for sync issues (skip if --no-pr)
	if !noPR {
		// Wait for git fetch to complete (needed for sync issue detection)
		fetchWg.Wait()

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

func printTree(gitClient git.GitClient, node *stack.TreeNode, prefix string, isLast bool, currentBranch string, prCache map[string]*github.PRInfo) {
	if node == nil {
		return
	}

	// Flatten the tree into a vertical list
	printTreeVertical(gitClient, node, currentBranch, prCache, false)
}

func printTreeVertical(gitClient git.GitClient, node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo, isChild bool) {
	if node == nil {
		return
	}

	// Determine the node marker (filled for current branch, hollow for others)
	nodeMarker := ui.TreeNode()
	if node.Name == currentBranch {
		nodeMarker = ui.TreeNodeCurrent()
	}

	// Get PR info from cache
	prInfo := ""
	if node.Name != stack.GetBaseBranch(gitClient) {
		if pr, exists := prCache[node.Name]; exists {
			prInfo = fmt.Sprintf(" %s", ui.PRInfo(pr.URL, pr.State))
		}
	}

	// Print connecting line if this is a child node
	if isChild {
		fmt.Printf("%s\n", ui.TreeLine())
	}

	// Print current node
	fmt.Printf("%s %s%s\n", nodeMarker, ui.Branch(node.Name), prInfo)

	// Print children vertically
	for _, child := range node.Children {
		printTreeVertical(gitClient, child, currentBranch, prCache, true)
	}
}

// syncIssuesResult holds the result of detectSyncIssues
type syncIssuesResult struct {
	issues []string
}

// detectSyncIssues checks if any branches are out of sync and returns the issues (doesn't print)
// If skipFetch is true, assumes git fetch was already called (to avoid redundant network calls)
func detectSyncIssues(gitClient git.GitClient, stackBranches []stack.StackBranch, prCache map[string]*github.PRInfo, progress spinner.ProgressFunc, skipFetch bool) (*syncIssuesResult, error) {
	var issues []string

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

	baseBranch := stack.GetBaseBranch(gitClient)

	// Check each stack branch for sync issues
	for i, branch := range stackBranches {
		progress(fmt.Sprintf("Checking branch %d/%d (%s)...", i+1, len(stackBranches), branch.Name))

		if verbose {
			fmt.Printf("\n[%d/%d] Checking '%s' (parent: %s)\n", i+1, len(stackBranches), branch.Name, branch.Parent)
		}

		// Skip branches with merged PRs - they don't need any sync action
		if pr, exists := prCache[branch.Name]; exists && pr.State == "MERGED" {
			if verbose {
				fmt.Printf("  Skipping (PR is merged)\n")
			}
			continue
		}

		// Check if parent PR is merged (or parent branch merged via git history)
		parentPR := prCache[branch.Parent]
		if parentPR != nil && parentPR.State == "MERGED" {
			if verbose {
				fmt.Printf("  ✗ Parent PR #%d has been merged\n", parentPR.Number)
			}
			issues = append(issues, fmt.Sprintf("  - Branch '%s': parent '%s' has been merged — run '%s' to reparent", ui.Branch(branch.Name), ui.Branch(branch.Parent), ui.Command("stack sync")))
		} else if parentPR == nil && branch.Parent != baseBranch {
			// No PR found for parent - check if parent was merged via git history
			if merged, err := gitClient.IsAncestor(branch.Parent, "origin/"+baseBranch); err == nil && merged {
				if verbose {
					fmt.Printf("  ✗ Parent %s appears merged into %s via git history\n", branch.Parent, baseBranch)
				}
				issues = append(issues, fmt.Sprintf("  - Branch '%s': parent '%s' has been merged — run '%s' to reparent", ui.Branch(branch.Name), ui.Branch(branch.Parent), ui.Command("stack sync")))
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
				issues = append(issues, fmt.Sprintf("  - Branch '%s' PR base (%s) doesn't match parent (%s)", ui.Branch(branch.Name), ui.Branch(pr.Base), ui.Branch(branch.Parent)))
			} else if verbose {
				fmt.Printf("  ✓ PR base matches configured parent\n")
			}
		} else {
			// No PR found — check if branch appears merged via git history
			if verbose {
				fmt.Printf("  No PR found, checking git history for merge...\n")
			}
			merged, err := gitClient.IsAncestor(branch.Name, "origin/"+baseBranch)
			if err == nil && merged {
				if verbose {
					fmt.Printf("  ✓ Branch appears merged into %s via git history\n", baseBranch)
				}
				issues = append(issues, fmt.Sprintf("  - Branch '%s' appears merged into %s (run '%s' to clean up)", ui.Branch(branch.Name), ui.Branch(baseBranch), ui.Command("stack prune")))
				continue // Skip other checks for merged branches
			} else if err != nil && verbose {
				fmt.Printf("  ⚠ Could not check if branch is ancestor: %v\n", err)
			} else if verbose {
				fmt.Printf("  No PR found for this branch\n")
			}
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
			issues = append(issues, fmt.Sprintf("  - Branch '%s' is behind %s (needs rebase)", ui.Branch(branch.Name), ui.Branch(branch.Parent)))
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
				issues = append(issues, fmt.Sprintf("  - Branch '%s' differs from origin (needs push)", ui.Branch(branch.Name)))
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
		issues: issues,
	}, nil
}

// printSyncIssues prints the sync issues result
func printSyncIssues(result *syncIssuesResult) {
	if len(result.issues) > 0 {
		fmt.Println()
		fmt.Println(ui.Warning("Stack out of sync detected:"))
		for _, issue := range result.issues {
			fmt.Println(issue)
		}
		fmt.Println()
		fmt.Printf("Run '%s' to rebase branches and update PR bases.\n", ui.Command("stack sync"))
	} else {
		fmt.Println()
		fmt.Println(ui.Success("Stack is perfectly synced! All branches are up to date."))
	}
}
