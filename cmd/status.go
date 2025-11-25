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
		if err := runStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	statusCmd.Flags().BoolVar(&noPR, "no-pr", false, "Skip fetching PR information (faster)")
}

func runStatus() error {
	// Get current branch
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
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

	// Build stack tree for current branch only
	tree, err := stack.BuildStackTreeForBranch(currentBranch)
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	// Get only branches in the current stack for efficient PR fetching
	currentStackBranches := getStackBranchesFromTree(tree, stackBranches)

	// Fetch PRs only for branches in the current stack (much faster than fetching all PRs)
	var prCache map[string]*github.PRInfo
	if noPR {
		prCache = make(map[string]*github.PRInfo)
	} else {
		prCache = fetchPRsForBranches(currentStackBranches)
	}

	// Filter out branches with merged PRs from the tree
	tree = filterMergedBranches(tree, prCache)

	// Print the tree
	printTree(tree, "", true, currentBranch, prCache)

	// Check for sync issues (skip if --no-pr)
	if !noPR {
		if err := detectSyncIssues(currentStackBranches, prCache); err != nil {
			// Don't fail on detection errors, just skip the check
			return nil
		}
	}

	return nil
}

// fetchPRsForBranches fetches PR info for specific branches (faster than fetching all PRs)
func fetchPRsForBranches(branches []stack.StackBranch) map[string]*github.PRInfo {
	prCache := make(map[string]*github.PRInfo)

	for _, branch := range branches {
		pr, err := github.GetPRForBranch(branch.Name)
		if err != nil {
			// Ignore errors, just skip this branch's PR info
			continue
		}
		if pr != nil {
			prCache[branch.Name] = pr
		}
	}

	return prCache
}

// getStackBranchesFromTree extracts all branches from the tree
func getStackBranchesFromTree(node *stack.TreeNode, allBranches []stack.StackBranch) []stack.StackBranch {
	if node == nil {
		return nil
	}

	branchMap := make(map[string]stack.StackBranch)
	for _, b := range allBranches {
		branchMap[b.Name] = b
	}

	var result []stack.StackBranch
	var traverse func(*stack.TreeNode)
	traverse = func(n *stack.TreeNode) {
		if n == nil {
			return
		}
		// Add current branch if it's a stack branch
		if b, exists := branchMap[n.Name]; exists {
			result = append(result, b)
		}
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
func filterMergedBranches(node *stack.TreeNode, prCache map[string]*github.PRInfo) *stack.TreeNode {
	if node == nil {
		return nil
	}

	// Filter children recursively first
	var filteredChildren []*stack.TreeNode
	for _, child := range node.Children {
		// Recurse first to process all descendants
		filtered := filterMergedBranches(child, prCache)

		// Only filter out merged branches if they have no children
		// (i.e., they're leaf nodes)
		if pr, exists := prCache[child.Name]; exists && pr.State == "MERGED" {
			// If this merged branch still has children after filtering, keep it
			// so the stack structure remains visible
			if filtered != nil && len(filtered.Children) > 0 {
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

func printTree(node *stack.TreeNode, prefix string, isLast bool, currentBranch string, prCache map[string]*github.PRInfo) {
	if node == nil {
		return
	}

	// Flatten the tree into a vertical list
	printTreeVertical(node, currentBranch, prCache, false)
}

func printTreeVertical(node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo, isPipe bool) {
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
	if node.Name != stack.GetBaseBranch() {
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
		printTreeVertical(child, currentBranch, prCache, true)
	}
}

// detectSyncIssues checks if any branches are out of sync and suggests running stack sync
func detectSyncIssues(stackBranches []stack.StackBranch, prCache map[string]*github.PRInfo) error {
	var issues []string
	var mergedBranches []string

	// Check each stack branch for sync issues
	for _, branch := range stackBranches {
		// Track branches with merged PRs (for cleanup suggestion, not sync)
		if pr, exists := prCache[branch.Name]; exists && pr.State == "MERGED" {
			mergedBranches = append(mergedBranches, branch.Name)
			continue // Don't check other sync issues for merged branches
		}

		// Check if parent has a merged PR (child needs to be updated)
		if branch.Parent != stack.GetBaseBranch() {
			if parentPR, exists := prCache[branch.Parent]; exists && parentPR.State == "MERGED" {
				issues = append(issues, fmt.Sprintf("  - Branch '%s' parent '%s' has a merged PR", branch.Name, branch.Parent))
			}
		}

		// Check if PR base matches the configured parent
		if pr, exists := prCache[branch.Name]; exists {
			if pr.Base != branch.Parent {
				issues = append(issues, fmt.Sprintf("  - Branch '%s' PR base (%s) doesn't match parent (%s)", branch.Name, pr.Base, branch.Parent))
			}
		}
	}

	// If issues found, print warning
	if len(issues) > 0 {
		fmt.Println()
		fmt.Println("⚠ Stack out of sync detected:")
		for _, issue := range issues {
			fmt.Println(issue)
		}
		fmt.Println()
		fmt.Println("Run 'stack sync' to rebase branches and update PR bases.")
	}

	// Suggest cleanup for merged branches
	if len(mergedBranches) > 0 && len(issues) == 0 {
		fmt.Println()
		fmt.Printf("✓ Stack is synced. Merged branches can be cleaned up: %s\n", strings.Join(mergedBranches, ", "))
	} else if len(mergedBranches) == 0 && len(issues) == 0 {
		// Everything is perfectly synced
		fmt.Println()
		fmt.Println("✓ Stack is perfectly synced! All branches are up to date.")
	}

	return nil
}

// Helper to repeat a string n times
func repeatString(s string, n int) string {
	return strings.Repeat(s, n)
}
