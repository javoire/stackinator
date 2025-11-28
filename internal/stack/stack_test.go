package stack

import (
	"testing"

	"github.com/javoire/stackinator/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetStackBranches(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name             string
		stackParents     map[string]string
		expectedBranches []string
		expectError      bool
	}{
		{
			name: "simple stack",
			stackParents: map[string]string{
				"feature-a": "main",
				"feature-b": "feature-a",
			},
			expectedBranches: []string{"feature-a", "feature-b"},
			expectError:      false,
		},
		{
			name:             "no stack branches",
			stackParents:     map[string]string{},
			expectedBranches: []string{},
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(testutil.MockGitClient)
			mockGit.On("GetAllStackParents").Return(tt.stackParents, nil)

			branches, err := GetStackBranches(mockGit)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, branches, len(tt.expectedBranches))

				branchNames := make(map[string]bool)
				for _, b := range branches {
					branchNames[b.Name] = true
				}

				for _, expectedName := range tt.expectedBranches {
					assert.True(t, branchNames[expectedName], "Expected branch %s not found", expectedName)
				}
			}

			mockGit.AssertExpectations(t)
		})
	}
}

func TestGetStackChain(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name          string
		branch        string
		stackParents  map[string]string
		expectedChain []string
		expectError   bool
	}{
		{
			name:   "simple chain",
			branch: "feature-c",
			stackParents: map[string]string{
				"feature-a": "main",
				"feature-b": "feature-a",
				"feature-c": "feature-b",
			},
			expectedChain: []string{"main", "feature-a", "feature-b", "feature-c"}, // Includes base
			expectError:   false,
		},
		{
			name:   "single branch",
			branch: "feature-a",
			stackParents: map[string]string{
				"feature-a": "main",
			},
			expectedChain: []string{"main", "feature-a"}, // Includes base
			expectError:   false,
		},
		{
			name:   "circular dependency",
			branch: "feature-b",
			stackParents: map[string]string{
				"feature-a": "feature-b",
				"feature-b": "feature-a",
			},
			expectedChain: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(testutil.MockGitClient)
			mockGit.On("GetAllStackParents").Return(tt.stackParents, nil)

			chain, err := GetStackChain(mockGit, tt.branch)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedChain, chain)
			}

			mockGit.AssertExpectations(t)
		})
	}
}

func TestGetBaseBranch(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name              string
		configuredBase    string
		defaultBranch     string
		expectedBase      string
	}{
		{
			name:           "configured base branch",
			configuredBase: "develop",
			defaultBranch:  "main",
			expectedBase:   "develop",
		},
		{
			name:           "no configured base, use default",
			configuredBase: "",
			defaultBranch:  "main",
			expectedBase:   "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(testutil.MockGitClient)
			mockGit.On("GetConfig", "stack.baseBranch").Return(tt.configuredBase)
			if tt.configuredBase == "" {
				mockGit.On("GetDefaultBranch").Return(tt.defaultBranch)
			}

			base := GetBaseBranch(mockGit)

			assert.Equal(t, tt.expectedBase, base)
			mockGit.AssertExpectations(t)
		})
	}
}

func TestBuildStackTree(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name             string
		stackParents     map[string]string
		baseBranch       string
		expectedRootName string
		expectError      bool
	}{
		{
			name: "simple tree",
			stackParents: map[string]string{
				"feature-a": "main",
				"feature-b": "main",
				"feature-c": "feature-a",
			},
			baseBranch:       "main",
			expectedRootName: "main",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGit := new(testutil.MockGitClient)
			mockGit.On("GetAllStackParents").Return(tt.stackParents, nil)
			mockGit.On("GetConfig", "stack.baseBranch").Return("")
			mockGit.On("GetDefaultBranch").Return(tt.baseBranch)

			tree, err := BuildStackTree(mockGit)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tree)
				assert.Equal(t, tt.expectedRootName, tree.Name)
			}

			mockGit.AssertExpectations(t)
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	tests := []struct {
		name          string
		branches      []StackBranch
		expectedOrder []string // Names of branches in expected order
		expectError   bool
	}{
		{
			name: "simple linear stack",
			branches: []StackBranch{
				{Name: "feature-c", Parent: "feature-b"},
				{Name: "feature-a", Parent: "main"},
				{Name: "feature-b", Parent: "feature-a"},
			},
			expectedOrder: []string{"feature-a", "feature-b", "feature-c"},
			expectError:   false,
		},
		{
			name: "branches with shared parent",
			branches: []StackBranch{
				{Name: "feature-a", Parent: "main"},
				{Name: "feature-b", Parent: "main"},
			},
			expectedOrder: []string{"feature-a", "feature-b"}, // Alphabetical within same level
			expectError:   false,
		},
		{
			name: "circular dependency",
			branches: []StackBranch{
				{Name: "feature-a", Parent: "feature-b"},
				{Name: "feature-b", Parent: "feature-a"},
			},
			expectedOrder: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted, err := TopologicalSort(tt.branches)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, sorted, len(tt.expectedOrder))

				// Check order
				for i, expectedName := range tt.expectedOrder {
					assert.Equal(t, expectedName, sorted[i].Name, "Branch at position %d should be %s", i, expectedName)
				}
			}
		})
	}
}

func TestBuildStackTreeForBranch(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	mockGit := new(testutil.MockGitClient)
	stackParents := map[string]string{
		"feature-a": "main",
		"feature-b": "feature-a",
		"feature-c": "feature-b",
		"other-branch": "main",
	}

	// Mock all the calls
	mockGit.On("GetAllStackParents").Return(stackParents, nil).Times(2) // Called twice in the function
	mockGit.On("GetConfig", "stack.baseBranch").Return("")
	mockGit.On("GetDefaultBranch").Return("main")

	// Build tree for feature-c - should only include its chain
	tree, err := BuildStackTreeForBranch(mockGit, "feature-c")

	assert.NoError(t, err)
	assert.NotNil(t, tree)
	assert.Equal(t, "main", tree.Name)

	// Verify the tree structure: main -> feature-a -> feature-b -> feature-c
	// but NOT other-branch
	assert.Len(t, tree.Children, 1)
	assert.Equal(t, "feature-a", tree.Children[0].Name)
	assert.Len(t, tree.Children[0].Children, 1)
	assert.Equal(t, "feature-b", tree.Children[0].Children[0].Name)
	assert.Len(t, tree.Children[0].Children[0].Children, 1)
	assert.Equal(t, "feature-c", tree.Children[0].Children[0].Children[0].Name)

	mockGit.AssertExpectations(t)
}

func TestGetChildrenOf(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	mockGit := new(testutil.MockGitClient)
	stackParents := map[string]string{
		"feature-a": "main",
		"feature-b": "main",
		"feature-c": "feature-a",
	}

	mockGit.On("GetAllStackParents").Return(stackParents, nil)

	children, err := GetChildrenOf(mockGit, "main")

	assert.NoError(t, err)
	assert.Len(t, children, 2)

	// Should be sorted alphabetically
	names := []string{children[0].Name, children[1].Name}
	assert.Contains(t, names, "feature-a")
	assert.Contains(t, names, "feature-b")

	mockGit.AssertExpectations(t)
}

