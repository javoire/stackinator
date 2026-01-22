package cmd

import (
	"fmt"
	"os"
	"strings"
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

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		// Setup: Get current branch
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-b").Return(nil)
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
		// GetPRForBranch is called for branches not in the cache (to detect merged PRs)
		mockGH.On("GetPRForBranch", "feature-a").Return(nil, nil).Maybe()
		mockGH.On("GetPRForBranch", "feature-b").Return(nil, nil).Maybe()
		mockGH.On("GetPRForBranch", "main").Return(nil, nil).Maybe()
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
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		// Patch-based unique commit detection
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		// Falls through to regular rebase since merge-base == parent
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil)
		// Process feature-b
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		mockGit.On("GetCommitHash", "feature-b").Return("def456", nil)
		mockGit.On("GetCommitHash", "origin/feature-b").Return("def456", nil)
		// Patch-based unique commit detection
		mockGit.On("GetUniqueCommitsByPatch", "feature-a", "feature-b").Return([]string{"def456"}, nil)
		mockGit.On("GetMergeBase", "feature-b", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		// Falls through to regular rebase since merge-base == parent
		mockGit.On("Rebase", "feature-a").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-b", "def456").Return(nil)
		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

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

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		// Setup
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-b").Return(nil)
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
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("RebaseOnto", "origin/main", "feature-a", "feature-b").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-b", "def456").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

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

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		// Setup
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-b").Return(nil)
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
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil)

		// Process feature-b
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		mockGit.On("GetCommitHash", "feature-b").Return("def456", nil)
		mockGit.On("GetCommitHash", "origin/feature-b").Return("def456", nil)
		mockGit.On("GetUniqueCommitsByPatch", "feature-a", "feature-b").Return([]string{"def456"}, nil)
		mockGit.On("GetMergeBase", "feature-b", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("Rebase", "feature-a").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-b", "def456").Return(nil)

		// Update PR base
		mockGH.On("UpdatePRBase", 2, "feature-a").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

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

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		// Setup
		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-a").Return(nil)
		// Working tree is dirty
		mockGit.On("IsWorkingTreeClean").Return(false, nil)
		// Stash changes
		mockGit.On("Stash", "stack-sync-autostash").Return(nil)
		// Save stash state
		mockGit.On("SetConfig", "stack.sync.stashed", "true").Return(nil)

		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)
		// GetPRForBranch is called for branches not in the cache (to detect merged PRs)
		mockGH.On("GetPRForBranch", "feature-a").Return(nil, nil).Maybe()
		mockGH.On("GetPRForBranch", "main").Return(nil, nil).Maybe()

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
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil)

		mockGit.On("CheckoutBranch", "feature-a").Return(nil)

		// Restore stash and clean up sync state
		mockGit.On("StashPop").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncErrorHandling(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("rebase conflict without stash", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")

		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-a").Return(nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)
		// GetPRForBranch is called for branches not in the cache (to detect merged PRs)
		mockGH.On("GetPRForBranch", "feature-a").Return(nil, nil).Maybe()
		mockGH.On("GetPRForBranch", "main").Return(nil, nil).Maybe()

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})

		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		// Rebase fails
		mockGit.On("Rebase", "origin/main").Return(fmt.Errorf("rebase conflict"))
		// Note: StashPop is NOT called because rebaseConflict=true

		err := runSync(mockGit, mockGH)

		assert.Error(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})

	t.Run("rebase conflict with stash preserves stash for --resume", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")

		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-a").Return(nil)
		// Working tree is dirty - will stash
		mockGit.On("IsWorkingTreeClean").Return(false, nil)
		mockGit.On("Stash", "stack-sync-autostash").Return(nil)
		// Save sync state
		mockGit.On("SetConfig", "stack.sync.stashed", "true").Return(nil)

		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)
		// GetPRForBranch is called for branches not in the cache (to detect merged PRs)
		mockGH.On("GetPRForBranch", "feature-a").Return(nil, nil).Maybe()
		mockGH.On("GetPRForBranch", "main").Return(nil, nil).Maybe()

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})

		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		// Rebase fails - stash should NOT be popped (preserved for --resume)
		mockGit.On("Rebase", "origin/main").Return(fmt.Errorf("rebase conflict"))
		// Note: StashPop is NOT called because rebaseConflict=true

		err := runSync(mockGit, mockGH)

		assert.Error(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestFilterMergedBranchesForSync(t *testing.T) {
	// Test the filterMergedBranchesForSync function
	// This is a simple unit test for the tree filtering logic
	prCache := map[string]*github.PRInfo{
		"merged-leaf":   testutil.NewPRInfo(1, "MERGED", "main", "Merged Leaf", "url"),
		"merged-parent": testutil.NewPRInfo(2, "MERGED", "main", "Merged Parent", "url"),
	}

	tree := &stack.TreeNode{
		Name: "main",
		Children: []*stack.TreeNode{
			{
				Name: "merged-parent",
				Children: []*stack.TreeNode{
					{Name: "child-of-merged", Children: nil},
				},
			},
			{Name: "merged-leaf", Children: nil},
			{Name: "open-branch", Children: nil},
		},
	}

	filtered := filterMergedBranchesForSync(tree, prCache)

	// merged-parent should be kept because it has children
	// merged-leaf should be filtered out
	// open-branch should be kept
	assert.Equal(t, 2, len(filtered.Children))
}

func TestRunSyncNoStackBranches(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	mockGit := new(testutil.MockGitClient)
	mockGH := new(testutil.MockGitHubClient)

	// Setup: Check for existing sync state (none)
	mockGit.On("GetConfig", "stack.sync.stashed").Return("")
	mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")

	mockGit.On("GetCurrentBranch").Return("main", nil)
	// Save original branch state
	mockGit.On("SetConfig", "stack.sync.originalBranch", "main").Return(nil)
	mockGit.On("IsWorkingTreeClean").Return(true, nil)
	mockGit.On("GetConfig", "branch.main.stackparent").Return("")
	mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
	mockGit.On("GetDefaultBranch").Return("main").Maybe()

	stackParents := map[string]string{}
	mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

	mockGit.On("Fetch").Return(nil)
	mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)

	mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
	mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
	mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
		"main": true,
	})

	mockGit.On("CheckoutBranch", "main").Return(nil) // Return to original branch
	// Clean up sync state
	mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
	mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

	err := runSync(mockGit, mockGH)

	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
	mockGH.AssertExpectations(t)
}

func TestRunSyncResume(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("resume fails when no saved state", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// No saved state
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")

		// Set resume flag
		syncResume = true
		defer func() { syncResume = false }()

		err := runSync(mockGit, mockGH)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no interrupted sync to resume")
	})

	t.Run("resume succeeds with saved state", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Saved state exists
		mockGit.On("GetConfig", "stack.sync.stashed").Return("true")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("feature-a")

		// Set resume flag
		syncResume = true
		defer func() { syncResume = false }()

		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)
		// GetPRForBranch is called for branches not in the cache (to detect merged PRs)
		mockGH.On("GetPRForBranch", "feature-a").Return(nil, nil).Maybe()
		mockGH.On("GetPRForBranch", "main").Return(nil, nil).Maybe()

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
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil)

		// Return to original branch (called twice: once for return, once for displayStatusAfterSync)
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		// For displayStatusAfterSync
		mockGit.On("GetCurrentBranch").Return("feature-a", nil)

		// Restore stash and clean up state
		mockGit.On("StashPop").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})

	t.Run("stale state cleaned up when user confirms", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Inject "y" input for the prompt
		stdinReader = strings.NewReader("y\n")
		defer func() { stdinReader = os.Stdin }()

		// Orphaned state exists but --resume not passed
		mockGit.On("GetConfig", "stack.sync.stashed").Return("true")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("old-branch")
		// Clean up orphaned state (user confirmed)
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-a").Return(nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)
		// GetPRForBranch is called for branches not in the cache (to detect merged PRs)
		mockGH.On("GetPRForBranch", "feature-a").Return(nil, nil).Maybe()
		mockGH.On("GetPRForBranch", "main").Return(nil, nil).Maybe()

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
		mockGit.On("FetchBranch", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})

	t.Run("sync aborted when user declines stale state cleanup", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Inject "n" input for the prompt (user declines)
		stdinReader = strings.NewReader("n\n")
		defer func() { stdinReader = os.Stdin }()

		// Orphaned state exists but --resume not passed
		mockGit.On("GetConfig", "stack.sync.stashed").Return("true")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("old-branch")

		// User declined, so sync should abort without calling any other methods

		err := runSync(mockGit, mockGH)

		// Should return nil (not an error) since user chose to abort
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncAutoConfiguresMissingStackparent(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("auto-configures parent branch missing stackparent", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: feature-b has stackparent=feature-a, but feature-a has NO stackparent
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-b").Return(nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-b.stackparent").Return("feature-a")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		// Key difference: feature-a is NOT in stackParents (no stackparent configured)
		stackParents := map[string]string{
			"feature-b": "feature-a", // feature-a is missing!
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		// The fix should auto-configure feature-a with parent=main
		mockGit.On("BranchExists", "feature-a").Return(true)
		mockGit.On("SetConfig", "branch.feature-a.stackparent", "main").Return(nil)

		// Parallel operations
		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetAllPRs").Return(make(map[string]*github.PRInfo), nil)

		// Worktree checks
		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
			"feature-b": true,
		})

		// Process feature-a first (auto-configured)
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranch", "main").Return(nil)
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil)

		// Process feature-b second
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		mockGit.On("GetCommitHash", "feature-b").Return("def456", nil)
		mockGit.On("GetCommitHash", "origin/feature-b").Return("def456", nil)
		mockGit.On("GetUniqueCommitsByPatch", "feature-a", "feature-b").Return([]string{"def456"}, nil)
		mockGit.On("GetMergeBase", "feature-b", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("Rebase", "feature-a").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-b", "def456").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncAbort(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("abort fails when no saved state", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// No saved state
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")

		// No rebase or cherry-pick in progress
		mockGit.On("IsCherryPickInProgress").Return(false)
		mockGit.On("IsRebaseInProgress").Return(false)

		// Set abort flag
		syncAbort = true
		defer func() { syncAbort = false }()

		err := runSync(mockGit, mockGH)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no interrupted sync to abort")
	})

	t.Run("abort succeeds with stashed changes", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Saved state exists with stash
		mockGit.On("GetConfig", "stack.sync.stashed").Return("true")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("feature-a")

		// Rebase in progress (to trigger AbortRebase)
		mockGit.On("IsCherryPickInProgress").Return(false)
		mockGit.On("IsRebaseInProgress").Return(true)

		// Set abort flag
		syncAbort = true
		defer func() { syncAbort = false }()

		// Abort rebase
		mockGit.On("AbortRebase").Return(nil)
		// Restore stashed changes
		mockGit.On("StashPop").Return(nil)
		// Return to original branch
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})

	t.Run("abort succeeds without stashed changes", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Saved state exists without stash (clean working tree)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("feature-a")

		// Rebase in progress (to trigger AbortRebase)
		mockGit.On("IsCherryPickInProgress").Return(false)
		mockGit.On("IsRebaseInProgress").Return(true)

		// Set abort flag
		syncAbort = true
		defer func() { syncAbort = false }()

		// Abort rebase
		mockGit.On("AbortRebase").Return(nil)
		// No stash to restore
		// Return to original branch
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})

	t.Run("abort handles rebase abort failure gracefully", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Saved state exists
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("feature-a")

		// Rebase in progress (to trigger AbortRebase)
		mockGit.On("IsCherryPickInProgress").Return(false)
		mockGit.On("IsRebaseInProgress").Return(true)

		// Set abort flag
		syncAbort = true
		defer func() { syncAbort = false }()

		// Abort rebase fails (simulated failure)
		mockGit.On("AbortRebase").Return(fmt.Errorf("no rebase in progress"))
		// Return to original branch
		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)

		err := runSync(mockGit, mockGH)

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}
