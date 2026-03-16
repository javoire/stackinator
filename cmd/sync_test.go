package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/javoire/stackinator/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()
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
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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
		mockGit.On("FetchRemote", "origin").Return(nil)

		// Parent PR is merged
		prCache := map[string]*github.PRInfo{
			"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
		}
		mockGH.On("GetPRsForBranches", mock.Anything).Return(prCache)
		// Git-based merge detection fallback (for branches without PRs)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

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
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("RebaseOnto", "origin/main", "feature-a", "feature-b").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-b", "def456").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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
		mockGit.On("FetchRemote", "origin").Return(nil)

		// PRs with mismatched base
		prCache := map[string]*github.PRInfo{
			"feature-a": testutil.NewPRInfo(1, "OPEN", "main", "Feature A", "url"),
			"feature-b": testutil.NewPRInfo(2, "OPEN", "main", "Feature B", "url"), // Wrong base!
		}
		mockGH.On("GetPRsForBranches", mock.Anything).Return(prCache)

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
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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

		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

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
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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

		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})

		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		// Rebase fails
		mockGit.On("Rebase", "origin/main").Return(fmt.Errorf("rebase conflict"))
		// Note: StashPop is NOT called because rebaseConflict=true

		err := runSync(mockGit, mockGH, "origin")

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

		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})

		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		// Rebase fails - stash should NOT be popped (preserved for --resume)
		mockGit.On("Rebase", "origin/main").Return(fmt.Errorf("rebase conflict"))
		// Note: StashPop is NOT called because rebaseConflict=true

		err := runSync(mockGit, mockGH, "origin")

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

	// When there are no stack branches, code returns early after parallel ops
	mockGit.On("FetchRemote", "origin").Return(nil).Maybe()
	mockGit.On("FastForwardToRemote", "main").Return(nil).Maybe()

	// These calls don't happen when there are no stack branches (early return)

	err := runSync(mockGit, mockGH, "origin")

	assert.NoError(t, err)
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

		err := runSync(mockGit, mockGH, "origin")

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

		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

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
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

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

		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

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
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil) // Fetch base branch before rebase
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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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

		err := runSync(mockGit, mockGH, "origin")

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
		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

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
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil)
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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncNoUniqueCommits(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("rebases even when no unique commits by patch", func(t *testing.T) {
		// This tests that sync still rebases when GetUniqueCommitsByPatch returns 0 commits.
		// A branch may have no unique patches but still be behind origin/master
		// (e.g., branch with only merge commits, or branch whose changes are already in master).
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		// Setup: Get current branch
		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-a").Return(nil)
		// Check working tree
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		// Get base branch
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()
		// Get stack chain
		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()
		// Parallel operations
		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()
		// Check if any branches in the current stack are in worktrees
		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		// Get remote branches
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})
		// Process feature-a
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil)
		// GetUniqueCommitsByPatch returns empty slice - no unique commits
		// But Rebase should STILL be called to incorporate target updates
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{}, nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil)
		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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

		err := runSync(mockGit, mockGH, "origin")

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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunSyncWithUpstreamRemote(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("sync uses upstream remote for base branch fetch and rebase", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: Check for existing sync state (none)
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		// Setup: Get current branch
		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		// Save original branch state
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-a").Return(nil)
		// Check working tree
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		// Get base branch
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()
		// Get stack chain
		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()
		// Parallel operations: fetch from upstream (not origin)
		mockGit.On("FetchRemote", "upstream").Return(nil)
		// Also fetch origin since syncRemote != "origin"
		mockGit.On("Fetch").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()
		// Worktree checks
		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		// Get remote branches (from origin)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})
		// Process feature-a: base branch uses upstream remote
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranchFromRemote", "upstream", "main").Return(nil) // Fetch from upstream!
		mockGit.On("GetUniqueCommitsByPatch", "upstream/main", "feature-a").Return([]string{"abc123"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "upstream/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "upstream/main").Return("main123", nil)
		mockGit.On("Rebase", "upstream/main").Return(nil) // Rebase onto upstream/main
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "abc123").Return(nil) // Push to origin
		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "upstream")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestDetermineSyncRemote(t *testing.T) {
	t.Run("uses CLI arg when provided", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		result := determineSyncRemote(mockGit, []string{"upstream"})
		assert.Equal(t, "upstream", result)
	})

	t.Run("uses git config when set", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGit.On("GetConfig", "stack.fetchRemote").Return("upstream")
		result := determineSyncRemote(mockGit, []string{})
		assert.Equal(t, "upstream", result)
	})

	t.Run("auto-detects upstream remote", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGit.On("GetConfig", "stack.fetchRemote").Return("")
		mockGit.On("RemoteExists", "upstream").Return(true)
		result := determineSyncRemote(mockGit, []string{})
		assert.Equal(t, "upstream", result)
	})

	t.Run("falls back to origin", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGit.On("GetConfig", "stack.fetchRemote").Return("")
		mockGit.On("RemoteExists", "upstream").Return(false)
		result := determineSyncRemote(mockGit, []string{})
		assert.Equal(t, "origin", result)
	})
}

func TestRunSyncSkipsWorktreeBranches(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("skips branch checked out in another worktree", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: no existing sync state
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		mockGit.On("GetCurrentBranch").Return("feature-c", nil)
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-c").Return(nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-c.stackparent").Return("feature-b")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
			"feature-b": "feature-a",
			"feature-c": "feature-b",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		// Parallel operations
		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

		// feature-b is in another worktree
		mockGit.On("GetWorktreeBranches").Return(map[string]string{
			"feature-b": "/other/worktree",
		}, nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)

		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
			"feature-b": true,
			"feature-c": true,
		})

		// Process feature-a (not skipped)
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("aaa111", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("aaa111", nil)
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil)
		mockGit.On("GetUniqueCommitsByPatch", "origin/main", "feature-a").Return([]string{"aaa111"}, nil)
		mockGit.On("GetMergeBase", "feature-a", "origin/main").Return("main123", nil)
		mockGit.On("GetCommitHash", "origin/main").Return("main123", nil)
		mockGit.On("Rebase", "origin/main").Return(nil)
		mockGit.On("FetchBranch", "feature-a").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-a", "aaa111").Return(nil)

		// feature-b is SKIPPED (in another worktree) - no checkout/rebase/push calls

		// Process feature-c (not skipped)
		mockGit.On("CheckoutBranch", "feature-c").Return(nil)
		mockGit.On("GetCommitHash", "feature-c").Return("ccc333", nil)
		mockGit.On("GetCommitHash", "origin/feature-c").Return("ccc333", nil)
		mockGit.On("GetUniqueCommitsByPatch", "feature-b", "feature-c").Return([]string{"ccc333"}, nil)
		mockGit.On("GetMergeBase", "feature-c", "feature-b").Return("bbb222", nil)
		mockGit.On("GetCommitHash", "feature-b").Return("bbb222", nil)
		mockGit.On("Rebase", "feature-b").Return(nil)
		mockGit.On("FetchBranch", "feature-c").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-c", "ccc333").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-c").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

		assert.NoError(t, err)
		// Verify feature-b was never checked out
		mockGit.AssertNotCalled(t, "CheckoutBranch", "feature-b")
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}

func TestRunPostSyncInstall(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("disabled via config", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false")

		// Should return early without calling GetRepoRoot
		runPostSyncInstall(mockGit)

		mockGit.AssertNotCalled(t, "GetRepoRoot")
		mockGit.AssertExpectations(t)
	})

	t.Run("auto-detect with no lockfile", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("")
		mockGit.On("GetRepoRoot").Return(t.TempDir(), nil)

		// Empty temp dir has no lockfiles, should be a no-op
		runPostSyncInstall(mockGit)

		mockGit.AssertExpectations(t)
	})

	t.Run("auto-detect picks correct package manager", func(t *testing.T) {
		for _, tc := range []struct {
			lockfile string
			expected string
		}{
			{"pnpm-lock.yaml", "pnpm"},
			{"yarn.lock", "yarn"},
			{"bun.lockb", "bun"},
			{"bun.lock", "bun"},
			{"package-lock.json", "npm"},
		} {
			t.Run(tc.lockfile, func(t *testing.T) {
				dir := t.TempDir()
				err := os.WriteFile(filepath.Join(dir, tc.lockfile), []byte{}, 0644)
				assert.NoError(t, err)

				detected := detectPackageManager(dir)
				assert.NotNil(t, detected, "expected detection for %s", tc.lockfile)
				assert.Equal(t, tc.expected, detected.command)
			})
		}
	})

	t.Run("pnpm-lock.yaml wins over yarn.lock", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte{}, 0644)
		_ = os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte{}, 0644)

		detected := detectPackageManager(dir)
		assert.NotNil(t, detected)
		assert.Equal(t, "pnpm", detected.command)
	})
}

func TestRunSyncGitBasedMergeDetection(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("removes branch from stack when merged via git history (no PR)", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: no existing sync state
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		mockGit.On("GetCurrentBranch").Return("feature-b", nil)
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-b").Return(nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-b.stackparent").Return("feature-a")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
			"feature-b": "feature-a",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		// No PRs found at all
		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
			"feature-b": true,
		})

		// feature-a: no PR, but IsAncestor returns true → merged via git history
		mockGit.On("IsAncestor", "feature-a", "origin/main").Return(true, nil)
		// Reverse check: origin/main is NOT ancestor of feature-a (branch has diverged, truly merged)
		mockGit.On("IsAncestor", "origin/main", "feature-a").Return(false, nil)
		// Remove feature-a from stack
		mockGit.On("UnsetConfig", "branch.feature-a.stackparent").Return(nil)

		// feature-b: no PR, not merged via git
		mockGit.On("IsAncestor", "feature-b", "origin/main").Return(false, nil)

		// feature-b's parent (feature-a) has no PR, parent merged via git
		// IsAncestor("feature-a", "origin/main") already mocked above → true
		// IsAncestor("origin/main", "feature-a") already mocked above → false (truly merged)

		// Reparent feature-b from feature-a to main
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("SetConfig", "branch.feature-b.stackparent", "main").Return(nil)
		// Also remove parent from stack tracking (called again for parent cleanup)
		mockGit.On("UnsetConfig", "branch.feature-a.stackparent").Return(nil)

		// Process feature-b: checkout, rebase onto origin/main (new parent)
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		mockGit.On("GetCommitHash", "feature-b").Return("def456", nil)
		mockGit.On("GetCommitHash", "origin/feature-b").Return("def456", nil)
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil)
		// --onto rebase since parent was merged
		mockGit.On("RebaseOnto", "origin/main", "feature-a", "feature-b").Return(nil)
		mockGit.On("FetchBranch", "feature-b").Return(nil)
		mockGit.On("PushWithExpectedRemote", "feature-b", "def456").Return(nil)

		// Return to original branch
		mockGit.On("CheckoutBranch", "feature-b").Return(nil)
		// Clean up sync state
		mockGit.On("UnsetConfig", "stack.sync.stashed").Return(nil)
		mockGit.On("UnsetConfig", "stack.sync.originalBranch").Return(nil)
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})

	t.Run("branch not merged via git is processed normally", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Setup: no existing sync state
		mockGit.On("GetConfig", "stack.sync.stashed").Return("")
		mockGit.On("GetConfig", "stack.sync.originalBranch").Return("")
		mockGit.On("GetCurrentBranch").Return("feature-a", nil)
		mockGit.On("SetConfig", "stack.sync.originalBranch", "feature-a").Return(nil)
		mockGit.On("IsWorkingTreeClean").Return(true, nil)
		mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
		mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
		mockGit.On("GetDefaultBranch").Return("main").Maybe()

		stackParents := map[string]string{
			"feature-a": "main",
		}
		mockGit.On("GetAllStackParents").Return(stackParents, nil).Maybe()

		// No PRs found
		mockGit.On("FetchRemote", "origin").Return(nil)
		mockGH.On("GetPRsForBranches", mock.Anything).Return(make(map[string]*github.PRInfo))
		// Git-based merge detection (no branches are merged)
		mockGit.On("IsAncestor", mock.Anything, mock.Anything).Return(false, nil).Maybe()

		mockGit.On("GetWorktreeBranches").Return(make(map[string]string), nil)
		mockGit.On("GetCurrentWorktreePath").Return("/Users/test/repo", nil)
		mockGit.On("GetRemoteBranchesSet").Return(map[string]bool{
			"main":      true,
			"feature-a": true,
		})

		// feature-a: no PR, NOT merged via git either
		mockGit.On("IsAncestor", "feature-a", "origin/main").Return(false, nil)

		// Should process normally (no parent reparenting since parent is baseBranch)
		mockGit.On("CheckoutBranch", "feature-a").Return(nil)
		mockGit.On("GetCommitHash", "feature-a").Return("abc123", nil)
		mockGit.On("GetCommitHash", "origin/feature-a").Return("abc123", nil)
		mockGit.On("FetchBranchFromRemote", "origin", "main").Return(nil)
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
		mockGit.On("GetConfig", "stack.postSyncInstall").Return("false").Maybe()

		err := runSync(mockGit, mockGH, "origin")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
		mockGH.AssertExpectations(t)
	})
}
