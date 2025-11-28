package cmd

import (
	"fmt"
	"testing"

	"github.com/javoire/stackinator/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRunNew(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name           string
		branchName     string
		explicitParent string
		setupMocks     func(*testutil.MockGitClient, *testutil.MockGitHubClient)
		expectError    bool
	}{
		{
			name:           "create branch with explicit parent",
			branchName:     "feature-b",
			explicitParent: "feature-a",
			setupMocks: func(mockGit *testutil.MockGitClient, mockGH *testutil.MockGitHubClient) {
				// Branch doesn't exist
				mockGit.On("BranchExists", "feature-b").Return(false)
				// Parent exists
				mockGit.On("BranchExists", "feature-a").Return(true)
				// Create branch
				mockGit.On("CreateBranch", "feature-b", "feature-a").Return(nil)
				// Set config
				mockGit.On("SetConfig", "branch.feature-b.stackparent", "feature-a").Return(nil)
			},
			expectError: false,
		},
		{
			name:           "create branch from current",
			branchName:     "feature-b",
			explicitParent: "",
			setupMocks: func(mockGit *testutil.MockGitClient, mockGH *testutil.MockGitHubClient) {
				// Branch doesn't exist
				mockGit.On("BranchExists", "feature-b").Return(false)
				// Get current branch
				mockGit.On("GetCurrentBranch").Return("feature-a", nil)
				// Check if current branch has parent
				mockGit.On("GetConfig", "branch.feature-a.stackparent").Return("main")
				// Create branch from current
				mockGit.On("CreateBranch", "feature-b", "feature-a").Return(nil)
				// Set config
				mockGit.On("SetConfig", "branch.feature-b.stackparent", "feature-a").Return(nil)
			},
			expectError: false,
		},
		{
			name:           "error when branch exists",
			branchName:     "feature-a",
			explicitParent: "main",
			setupMocks: func(mockGit *testutil.MockGitClient, mockGH *testutil.MockGitHubClient) {
				// Branch already exists
				mockGit.On("BranchExists", "feature-a").Return(true)
			},
			expectError: true,
		},
		{
			name:           "error when parent doesn't exist",
			branchName:     "feature-b",
			explicitParent: "non-existent",
			setupMocks: func(mockGit *testutil.MockGitClient, mockGH *testutil.MockGitHubClient) {
				// Branch doesn't exist
				mockGit.On("BranchExists", "feature-b").Return(false)
				// Parent doesn't exist
				mockGit.On("BranchExists", "non-existent").Return(false)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(testutil.MockGitClient)
			mockGH := new(testutil.MockGitHubClient)

			tt.setupMocks(mockGit, mockGH)

			// Set dryRun to true to skip the display logic at the end
			dryRun = true

			err := runNew(mockGit, mockGH, tt.branchName, tt.explicitParent)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockGit.AssertExpectations(t)
			mockGH.AssertExpectations(t)

			// Reset dryRun
			dryRun = false
		})
	}
}

func TestRunNewValidation(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("validates branch name", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Branch already exists
		mockGit.On("BranchExists", "existing-branch").Return(true)

		err := runNew(mockGit, mockGH, "existing-branch", "main")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")

		mockGit.AssertExpectations(t)
	})

	t.Run("validates parent exists", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		// Branch doesn't exist
		mockGit.On("BranchExists", "new-branch").Return(false)
		// Parent doesn't exist
		mockGit.On("BranchExists", "non-existent-parent").Return(false)

		err := runNew(mockGit, mockGH, "new-branch", "non-existent-parent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")

		mockGit.AssertExpectations(t)
	})
}

func TestRunNewSetConfig(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	mockGit := new(testutil.MockGitClient)
	mockGH := new(testutil.MockGitHubClient)

	// Branch doesn't exist
	mockGit.On("BranchExists", "new-branch").Return(false)
	// Parent exists
	mockGit.On("BranchExists", "parent-branch").Return(true)
	// Create branch
	mockGit.On("CreateBranch", "new-branch", "parent-branch").Return(nil)
	// Verify SetConfig is called with correct parameters
	mockGit.On("SetConfig", "branch.new-branch.stackparent", "parent-branch").Return(nil)

	dryRun = true
	err := runNew(mockGit, mockGH, "new-branch", "parent-branch")
	dryRun = false

	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
}

func TestRunNewFromCurrentBranch(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	mockGit := new(testutil.MockGitClient)
	mockGH := new(testutil.MockGitHubClient)

	// Branch doesn't exist
	mockGit.On("BranchExists", "new-branch").Return(false)
	// Get current branch
	mockGit.On("GetCurrentBranch").Return("current-branch", nil)
	// Current branch has a parent (it's in a stack)
	mockGit.On("GetConfig", "branch.current-branch.stackparent").Return("main")
	// Create branch from current
	mockGit.On("CreateBranch", "new-branch", "current-branch").Return(nil)
	// Set config
	mockGit.On("SetConfig", "branch.new-branch.stackparent", "current-branch").Return(nil)

	dryRun = true
	err := runNew(mockGit, mockGH, "new-branch", "")
	dryRun = false

	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
}

func TestRunNewErrorHandling(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("error on CreateBranch failure", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		mockGit.On("BranchExists", "new-branch").Return(false)
		mockGit.On("BranchExists", "parent").Return(true)
		mockGit.On("CreateBranch", "new-branch", "parent").Return(fmt.Errorf("git error"))

		err := runNew(mockGit, mockGH, "new-branch", "parent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create branch")

		mockGit.AssertExpectations(t)
	})

	t.Run("error on SetConfig failure", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)
		mockGH := new(testutil.MockGitHubClient)

		mockGit.On("BranchExists", "new-branch").Return(false)
		mockGit.On("BranchExists", "parent").Return(true)
		mockGit.On("CreateBranch", "new-branch", "parent").Return(nil)
		mockGit.On("SetConfig", "branch.new-branch.stackparent", "parent").Return(fmt.Errorf("config error"))

		err := runNew(mockGit, mockGH, "new-branch", "parent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set parent config")

		mockGit.AssertExpectations(t)
	})
}

