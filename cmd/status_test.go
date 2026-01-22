package cmd

import (
	"testing"

	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/javoire/stackinator/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRunStatus(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name        string
		setupMocks  func(*testutil.MockGitClient, *testutil.MockGitHubClient)
		expectError bool
	}{
		{
			name: "display simple stack",
			setupMocks: func(mockGit *testutil.MockGitClient, mockGH *testutil.MockGitHubClient) {
				// Get current branch
				mockGit.On("GetCurrentBranch").Return("feature-a", nil)
				// Get stack branches (called multiple times in BuildStackTreeForBranch)
				stackParents := map[string]string{
					"feature-a": "main",
				}
				mockGit.On("GetAllStackParents").Return(stackParents, nil).Times(3) // Called 3 times
				// Get base branch
				mockGit.On("GetConfig", "stack.baseBranch").Return("")
				mockGit.On("GetDefaultBranch").Return("main")
				// Note: GetAllPRs is NOT called because noPR is true
			},
			expectError: false,
		},
		{
			name: "no stack branches",
			setupMocks: func(mockGit *testutil.MockGitClient, mockGH *testutil.MockGitHubClient) {
				// Get current branch
				mockGit.On("GetCurrentBranch").Return("main", nil)
				// Get stack branches (empty)
				mockGit.On("GetAllStackParents").Return(make(map[string]string), nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(testutil.MockGitClient)
			mockGH := new(testutil.MockGitHubClient)

			tt.setupMocks(mockGit, mockGH)

			// Set noPR to true to skip PR fetching in parallel goroutines
			noPR = true

			err := runStatus(mockGit, mockGH)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockGit.AssertExpectations(t)
			mockGH.AssertExpectations(t)

			// Reset noPR
			noPR = false
		})
	}
}

func TestGetAllBranchNamesFromTree(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tree := &stack.TreeNode{
		Name: "main",
		Children: []*stack.TreeNode{
			{
				Name: "feature-a",
				Children: []*stack.TreeNode{
					{Name: "feature-b", Children: nil},
				},
			},
			{Name: "feature-c", Children: nil},
		},
	}

	branches := getAllBranchNamesFromTree(tree)

	assert.Len(t, branches, 4)
	assert.Contains(t, branches, "main")
	assert.Contains(t, branches, "feature-a")
	assert.Contains(t, branches, "feature-b")
	assert.Contains(t, branches, "feature-c")
}

func TestDetectSyncIssues(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name           string
		stackBranches  []stack.StackBranch
		prCache        map[string]*github.PRInfo
		setupMocks     func(*testutil.MockGitClient)
		expectedIssues int
	}{
		{
			name: "branch behind parent",
			stackBranches: []stack.StackBranch{
				{Name: "feature-a", Parent: "main"},
			},
			prCache: make(map[string]*github.PRInfo),
			setupMocks: func(mockGit *testutil.MockGitClient) {
				mockGit.On("IsCommitsBehind", "feature-a", "main").Return(true, nil)
				mockGit.On("RemoteBranchExists", "feature-a").Return(false)
			},
			expectedIssues: 1,
		},
		{
			name: "branch up to date",
			stackBranches: []stack.StackBranch{
				{Name: "feature-a", Parent: "main"},
			},
			prCache: make(map[string]*github.PRInfo),
			setupMocks: func(mockGit *testutil.MockGitClient) {
				mockGit.On("IsCommitsBehind", "feature-a", "main").Return(false, nil)
				mockGit.On("RemoteBranchExists", "feature-a").Return(false)
			},
			expectedIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(testutil.MockGitClient)
			tt.setupMocks(mockGit)

			// Mock GetBaseBranch calls
			mockGit.On("GetConfig", "stack.baseBranch").Return("").Maybe()
			mockGit.On("GetDefaultBranch").Return("main").Maybe()

			nopProgress := func(msg string) {} // No-op progress function
			result, err := detectSyncIssues(mockGit, tt.stackBranches, tt.prCache, nopProgress, true)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Len(t, result.issues, tt.expectedIssues, "Expected %d issues, got %d", tt.expectedIssues, len(result.issues))

			mockGit.AssertExpectations(t)
		})
	}
}
