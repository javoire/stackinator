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

func TestFilterMergedBranches(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name            string
		tree            *stack.TreeNode
		prCache         map[string]*github.PRInfo
		currentBranch   string
		expectFiltered  bool
		expectedBranches []string
	}{
		{
			name: "keep merged branch with children",
			tree: &stack.TreeNode{
				Name: "main",
				Children: []*stack.TreeNode{
					{
						Name: "feature-a",
						Children: []*stack.TreeNode{
							{Name: "feature-b", Children: nil},
						},
					},
				},
			},
			prCache: map[string]*github.PRInfo{
				"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
				"feature-b": testutil.NewPRInfo(2, "OPEN", "feature-a", "Feature B", "url"),
			},
			currentBranch: "feature-b",
			expectedBranches: []string{"main", "feature-a", "feature-b"}, // Keep feature-a because it has children
		},
		{
			name: "filter merged leaf branch",
			tree: &stack.TreeNode{
				Name: "main",
				Children: []*stack.TreeNode{
					{
						Name:     "feature-a",
						Children: nil,
					},
				},
			},
			prCache: map[string]*github.PRInfo{
				"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
			},
			currentBranch: "main",
			expectedBranches: []string{"main"}, // Filter out feature-a because it's a merged leaf
		},
		{
			name: "keep current branch even if merged",
			tree: &stack.TreeNode{
				Name: "main",
				Children: []*stack.TreeNode{
					{
						Name:     "feature-a",
						Children: nil,
					},
				},
			},
			prCache: map[string]*github.PRInfo{
				"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
			},
			currentBranch: "feature-a",
			expectedBranches: []string{"main", "feature-a"}, // Keep feature-a because it's current branch
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterMergedBranches(tt.tree, tt.prCache, tt.currentBranch)

			// Collect all branch names from filtered tree
			var branches []string
			var collectBranches func(*stack.TreeNode)
			collectBranches = func(node *stack.TreeNode) {
				if node == nil {
					return
				}
				branches = append(branches, node.Name)
				for _, child := range node.Children {
					collectBranches(child)
				}
			}
			collectBranches(filtered)

			assert.Equal(t, tt.expectedBranches, branches)
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
		name                string
		stackBranches       []stack.StackBranch
		prCache             map[string]*github.PRInfo
		setupMocks          func(*testutil.MockGitClient)
		expectedIssues      int
		expectedMerged      int
	}{
		{
			name: "branch behind parent",
			stackBranches: []stack.StackBranch{
				{Name: "feature-a", Parent: "main"},
			},
			prCache: make(map[string]*github.PRInfo),
			setupMocks: func(mockGit *testutil.MockGitClient) {
				mockGit.On("IsCommitsBehind", "feature-a", "main").Return(true, nil)
			},
			expectedIssues: 1,
			expectedMerged: 0,
		},
		{
			name: "branch with merged PR",
			stackBranches: []stack.StackBranch{
				{Name: "feature-a", Parent: "main"},
			},
			prCache: map[string]*github.PRInfo{
				"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
			},
			setupMocks: func(mockGit *testutil.MockGitClient) {
				// No calls expected for merged branches
			},
			expectedIssues: 0,
			expectedMerged: 1,
		},
		{
			name: "parent PR merged",
			stackBranches: []stack.StackBranch{
				{Name: "feature-b", Parent: "feature-a"},
			},
			prCache: map[string]*github.PRInfo{
				"feature-a": testutil.NewPRInfo(1, "MERGED", "main", "Feature A", "url"),
			},
			setupMocks: func(mockGit *testutil.MockGitClient) {
				mockGit.On("GetDefaultBranch").Return("main")
				mockGit.On("IsCommitsBehind", "feature-b", "feature-a").Return(false, nil)
			},
			expectedIssues: 1, // Issue because parent is merged
			expectedMerged: 0,
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
			assert.Len(t, result.mergedBranches, tt.expectedMerged, "Expected %d merged branches, got %d", tt.expectedMerged, len(result.mergedBranches))

			mockGit.AssertExpectations(t)
		})
	}
}

