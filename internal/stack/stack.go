package stack

import (
	"fmt"
	"sort"

	"github.com/javoire/stackinator/internal/git"
)

// StackBranch represents a branch in a stack
type StackBranch struct {
	Name   string
	Parent string
	Exists bool
}

// GetStackBranches returns all branches that are part of a stack
func GetStackBranches(gitClient git.GitClient) ([]StackBranch, error) {
	// Fetch all stack parents in one efficient call
	parents, err := gitClient.GetAllStackParents()
	if err != nil {
		return nil, fmt.Errorf("failed to get stack parents: %w", err)
	}

	var stackBranches []StackBranch
	for branch, parent := range parents {
		stackBranches = append(stackBranches, StackBranch{
			Name:   branch,
			Parent: parent,
			Exists: true,
		})
	}

	return stackBranches, nil
}

// GetChildrenOf returns all direct children of the specified branch
func GetChildrenOf(gitClient git.GitClient, branch string) ([]StackBranch, error) {
	allBranches, err := GetStackBranches(gitClient)
	if err != nil {
		return nil, err
	}

	var children []StackBranch
	for _, b := range allBranches {
		if b.Parent == branch {
			children = append(children, b)
		}
	}

	// Sort children by name for consistent output
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	return children, nil
}

// GetStackChain returns the chain from the base to the specified branch
func GetStackChain(gitClient git.GitClient, branch string) ([]string, error) {
	// Get all parents at once for efficiency
	parents, err := gitClient.GetAllStackParents()
	if err != nil {
		return nil, err
	}

	// If current branch has no stackparent, it's not in a stack
	if parents[branch] == "" {
		return []string{}, nil
	}

	var chain []string
	current := branch
	seen := make(map[string]bool)

	for current != "" {
		if seen[current] {
			return nil, fmt.Errorf("circular dependency detected in stack at %s", current)
		}
		seen[current] = true
		chain = append([]string{current}, chain...)

		parent := parents[current]
		if parent == "" {
			break
		}
		current = parent
	}

	return chain, nil
}

// TopologicalSort returns branches in bottom-to-top order (base to tips)
func TopologicalSort(branches []StackBranch) ([]StackBranch, error) {
	// Build adjacency map
	children := make(map[string][]string)
	inDegree := make(map[string]int)
	branchMap := make(map[string]StackBranch)

	for _, b := range branches {
		branchMap[b.Name] = b
		if _, exists := inDegree[b.Name]; !exists {
			inDegree[b.Name] = 0
		}
		if _, exists := inDegree[b.Parent]; !exists {
			inDegree[b.Parent] = 0
		}
		children[b.Parent] = append(children[b.Parent], b.Name)
		inDegree[b.Name]++
	}

	// Find all base branches (those whose parents are not in the stack)
	var queue []string
	for parent := range children {
		if inDegree[parent] == 0 {
			queue = append(queue, parent)
		}
	}

	var sorted []StackBranch

	// Process queue
	for len(queue) > 0 {
		// Sort queue for deterministic output
		sort.Strings(queue)

		current := queue[0]
		queue = queue[1:]

		// Add to result if it's a stack branch
		if branch, exists := branchMap[current]; exists {
			sorted = append(sorted, branch)
		}

		// Process children
		for _, child := range children[current] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// Check for cycles
	for name, degree := range inDegree {
		if degree > 0 {
			if _, isStack := branchMap[name]; isStack {
				return nil, fmt.Errorf("circular dependency detected involving %s", name)
			}
		}
	}

	return sorted, nil
}

// GetBaseBranch returns the configured base branch or auto-detects it
func GetBaseBranch(gitClient git.GitClient) string {
	base := gitClient.GetConfig("stack.baseBranch")
	if base == "" {
		return gitClient.GetDefaultBranch()
	}
	return base
}

// BuildStackTree builds a tree representation for display
func BuildStackTree(gitClient git.GitClient) (*TreeNode, error) {
	stackBranches, err := GetStackBranches(gitClient)
	if err != nil {
		return nil, err
	}

	// Build parent -> children map
	childrenMap := make(map[string][]StackBranch)
	for _, b := range stackBranches {
		childrenMap[b.Parent] = append(childrenMap[b.Parent], b)
	}

	// Sort children for each parent
	for parent := range childrenMap {
		sort.Slice(childrenMap[parent], func(i, j int) bool {
			return childrenMap[parent][i].Name < childrenMap[parent][j].Name
		})
	}

	// Build tree starting from base branch
	baseBranch := GetBaseBranch(gitClient)
	return buildTreeNode(baseBranch, childrenMap), nil
}

// BuildStackTreeForBranch builds a tree for only the stack containing the specified branch
func BuildStackTreeForBranch(gitClient git.GitClient, branchName string) (*TreeNode, error) {
	// Get the chain from base to current branch
	chain, err := GetStackChain(gitClient, branchName)
	if err != nil {
		return nil, err
	}

	if len(chain) == 0 {
		// Current branch is not in a stack
		return nil, nil
	}

	// Build a set of branches in the chain for quick lookup
	chainSet := make(map[string]bool)
	for _, b := range chain {
		chainSet[b] = true
	}

	// Get all stack branches to find children
	stackBranches, err := GetStackBranches(gitClient)
	if err != nil {
		return nil, err
	}

	// Build parent -> children map, but ONLY include branches that are in the chain
	childrenMap := make(map[string][]StackBranch)
	for _, b := range stackBranches {
		// Only include branches that are actually in the chain
		if chainSet[b.Name] {
			childrenMap[b.Parent] = append(childrenMap[b.Parent], b)
		}
	}

	// Sort children for each parent
	for parent := range childrenMap {
		sort.Slice(childrenMap[parent], func(i, j int) bool {
			return childrenMap[parent][i].Name < childrenMap[parent][j].Name
		})
	}

	// Build tree starting from the root (first element in chain)
	root := chain[0]

	// If the root is not the base branch, we need to include the base branch
	// in the tree as the actual root
	baseBranch := GetBaseBranch(gitClient)
	if root != baseBranch {
		// Check if the root has a parent in childrenMap (meaning there are branches
		// that have root as their parent)
		// We need to insert the base branch as the root
		baseNode := &TreeNode{Name: baseBranch}
		rootNode := buildTreeNode(root, childrenMap)
		baseNode.Children = []*TreeNode{rootNode}
		return baseNode, nil
	}

	return buildTreeNode(root, childrenMap), nil
}

// findStackRoot walks up the parent chain to find the root of the stack
func findStackRoot(branchName string, stackBranches []StackBranch) string {
	// Build a map for quick parent lookup
	parentMap := make(map[string]string)
	for _, b := range stackBranches {
		parentMap[b.Name] = b.Parent
	}

	current := branchName
	visited := make(map[string]bool)

	// Walk up until we find a branch with no stack parent or hit a cycle
	for {
		if visited[current] {
			// Cycle detected, return current as root
			return current
		}
		visited[current] = true

		parent, hasParent := parentMap[current]
		if !hasParent {
			// This branch has no stack parent, so the parent is the root
			// (or this is the root if it has no parent)
			if parentFromMap, exists := parentMap[current]; exists {
				return parentFromMap
			}
			return current
		}
		current = parent
	}
}

// buildConnectedComponent finds all branches in the same stack as root
func buildConnectedComponent(root string, stackBranches []StackBranch) map[string]bool {
	// Build children map
	childrenMap := make(map[string][]string)
	for _, b := range stackBranches {
		childrenMap[b.Parent] = append(childrenMap[b.Parent], b.Name)
	}

	// BFS from root to find all descendants
	component := make(map[string]bool)
	queue := []string{root}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if component[current] {
			continue
		}
		component[current] = true

		// Add all children to queue
		queue = append(queue, childrenMap[current]...)
	}

	return component
}

// TreeNode represents a node in the stack tree
type TreeNode struct {
	Name     string
	Children []*TreeNode
}

func buildTreeNode(name string, childrenMap map[string][]StackBranch) *TreeNode {
	node := &TreeNode{Name: name}

	for _, child := range childrenMap[name] {
		node.Children = append(node.Children, buildTreeNode(child.Name, childrenMap))
	}

	return node
}
