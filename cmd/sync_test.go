package cmd

import (
	"fmt"
	"testing"

	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/javoire/stackinator/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRunSyncBasic(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("sync simple 2-branch stack", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: Get current branch
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		// Check working tree
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		// Get base branch
		mockGit.On("GetConfig", "branch.feature-b.stackparent").Return("feature-a")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe() // Called many times in tree printing
		// Get stack chain
		stackParents := map[string]string{
			"feature-a": "main",
			"feature-b": "feature-a",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe() // Called in GetStackChain, TopologicalSort, and displayStatusAfterSync
		// Parallel operations
		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)
		// Check if any branches in the current stack are in worktrees
		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		// Get current worktree path
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		// Get remote branches
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
			"feature-b": true,
		})
		// Process feature-a
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("Push", "feature-a", true).Return(nil)
		// Process feature-b
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		mockGit.On("GetCommitHash", "feature-b").Return("def456", nil)
		mockGit.On("GetCommitHash", "origin/feature-b").Return("def456", nil)
		mockGit.On("Rebase", "feature-a").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("Push", "feature-b", true).Return(nil)
		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncMergedParent(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("rebase when parent PR is merged", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-b.stackparent").Return("feature-a")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe() // Called many times in tree printing

		stackParents := map[string]string{
			"feature-a": "main",
			"feature-b": "feature-a",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe() // Called in GetStackChain, TopologicalSort, and displayStatusAfterSync

		// Parallel operations
		mockGit.On("Fetch").Return(nil)

		// Parent PR is merged
		prCache := map[string]*github.PRInfo{
			"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
		}
		mockGH.On("GetAllPRs").Return(prCache, nil)

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
			"feature-b": true,
		})

		// Process feature-a (merged, skip)
		mockGit.On("UnsetConfig", "branch.feature-a.stackparent").Return(nil)

		// Process feature-b (parent is merged, update parent to grandparent)
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("SetConfig", "branch.feature-b.stackparent", "main").Return(nil)
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		mockGit.On("GetCommitHash", "feature-b").Return("def456", nil)
		mockGit.On("GetCommitHash", "origin/feature-b").Return("def456", nil)
		mockGit.On("RebaseOnto", "origin/main", "feature-a", "feature-b").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("Push", "feature-b", true).Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncUpdatePRBase(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("update PR base when it doesn't match parent", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-b.stackparent").Return("feature-a")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe() // Called many times in tree printing

		stackParents := map[string]string{
			"feature-a": "main",
			"feature-b": "feature-a",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe() // Called in GetStackChain, TopologicalSort, and displayStatusAfterSync

		// Parallel operations
		mockGit.On("Fetch").Return(nil)

		// PRs with mismatched base
		prCache := map[string]*github.PRInfo{
			"feature-a": testutil.NewPRInfo(1, "OPEN", "main", "Feature A", "url"),
			"feature-b": testutil.NewPRInfo(2, "OPEN", "main", "Feature B", "url"), // Wrong base!
		}
		mockGH.On("GetAllPRs").Return(prCache, nil)

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
			"feature-b": true,
		})

		// Process feature-a
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("Push", "feature-a", true).Return(nil)

		// Process feature-b
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		mockGit.On("GetCommitHash", "feature-b").Return("def456", nil)
		mockGit.On("GetCommitHash", "origin/feature-b").Return("def456", nil)
		mockGit.On("Rebase", "feature-a").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("Push", "feature-b", true).Return(nil)
		// Update PR base!
		mockGH.On("UpdatePRBase", 2, "feature-a").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncStashHandling(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("stash and restore uncommitted changes", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup
		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Working tree is dirty
		mockGit.On("IsWorkingTreeClean").Return(false, nil)
		// Stash changes
		mockGit.On("Stash", "stack-sync-autostash").Return(nil)

		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe() // Called in GetStackChain, TopologicalSort, and displayStatusAfterSync

		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})

		// Process feature-a
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("Push", "feature-a", true).Return(nil)

		mockGit.On("CheckoutBranch", "feature-a").Return(nil)

		// Restore stash
		mockGit.On("StashPop").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncErrorHandling(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("rebase conflict", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe() // Called in GetStackChain, TopologicalSort, and displayStatusAfterSync

		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})

		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		// Rebase fails
		mockGit.On("Rebase", "origin/main").Return(fmt.Errorf("rebase conflict"))

		err := runSync(mockGit, mockGH)

		assert.Error(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestFilterMergedBranchesForSync(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tree := &stack.TreeNode{
		Name: "main",
		Children: []*stack.TreeNode{
			{Name: "feature-a", Children: nil},
			{
				Name: "feature-b",
				Children: []*stack.TreeNode{
					{Name: "feature-c", Children: nil},
				},
			},
		},
	}

	prCache := map[string]*github.PRInfo{
		"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
		"feature-b": testutil.NewPRInfo(2, "MERGED", "main", "Feature B", "url"),
		"feature-c": testutil.NewPRInfo(3, "OPEN", "feature-b", "Feature C", "url"),
	}

	filtered := filterMergedBranchesForSync(tree, prCache)

	// feature-a should be filtered out (merged leaf)
	// feature-b should be kept (merged but has children)
	// feature-c should be kept (not merged)

	assert.Equal(t, "main", filtered.Name)
	assert.Len(t, filtered.Children, 1)
	assert.Equal(t, "feature-b", filtered.Children[0].Name)
	assert.Len(t, filtered.Children[0].Children, 1)
	assert.Equal(t, "feature-c", filtered.Children[0].Children[0].Name)
}

func TestRunSyncNoStackBranches(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	mockGit := new(testutil.MockGitClient)
	mockGH := new(testutil.MockGitHubClient)

	mockGit.On("GetCurrentBranch").Return("main", nil)
	mockGit.On("IsWorkingTreeClean").Return(true, nil)
	mockGit.On("GetConfig", "branch.main.stackparent").Return("")
	mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
	mockGit.On("GetDefaultBranch").Return("main").Maybe()

	// Empty stack
	mockGit.On("GetAllStackParents").Return(make(map[string]string), nil)

	mockGit.On("Fetch").Return(nil)
	mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)

	// Even with no stack, we still check worktrees
	mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
	mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
	mockGit.On("GetRemoteBranchesSet").Return(make(map[string]bool))
	mockGit.On("CheckoutBranch", "main").Return(nil) // Return to original branch

	err := runSync(mockGit, mockGH)

	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
	mockGH.AssertExpectations(t)
}

