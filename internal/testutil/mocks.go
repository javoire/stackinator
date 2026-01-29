package testutil

import (
	"github.com/javoire/stackinator/internal/github"
	"github.com/stretchr/testify/mock"
)

// MockGitClient is a mock implementation of git.GitClient for testing
type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) GetRepoRoot() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) GetRepoName() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) GetCurrentBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) ListBranches() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGitClient) GetConfig(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockGitClient) GetAllStackParents() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockGitClient) SetConfig(key, value string) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockGitClient) UnsetConfig(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockGitClient) CreateBranch(name, from string) error {
	args := m.Called(name, from)
	return args.Error(0)
}

func (m *MockGitClient) CreateBranchAndCheckout(name, from string) error {
	args := m.Called(name, from)
	return args.Error(0)
}

func (m *MockGitClient) CheckoutBranch(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockGitClient) RenameBranch(oldName, newName string) error {
	args := m.Called(oldName, newName)
	return args.Error(0)
}

func (m *MockGitClient) Rebase(onto string) error {
	args := m.Called(onto)
	return args.Error(0)
}

func (m *MockGitClient) RebaseOnto(newBase, oldBase, currentBranch string) error {
	args := m.Called(newBase, oldBase, currentBranch)
	return args.Error(0)
}

func (m *MockGitClient) FetchBranch(branch string) error {
	args := m.Called(branch)
	return args.Error(0)
}

func (m *MockGitClient) Push(branch string, forceWithLease bool) error {
	args := m.Called(branch, forceWithLease)
	return args.Error(0)
}

func (m *MockGitClient) PushWithExpectedRemote(branch string, expectedRemoteSha string) error {
	args := m.Called(branch, expectedRemoteSha)
	return args.Error(0)
}

func (m *MockGitClient) ForcePush(branch string) error {
	args := m.Called(branch)
	return args.Error(0)
}

func (m *MockGitClient) IsWorkingTreeClean() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockGitClient) Fetch() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) BranchExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockGitClient) RemoteBranchExists(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockGitClient) GetRemoteBranchesSet() map[string]bool {
	args := m.Called()
	return args.Get(0).(map[string]bool)
}

func (m *MockGitClient) IsRebaseInProgress() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockGitClient) IsCherryPickInProgress() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockGitClient) AbortRebase() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) AbortCherryPick() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) ResetToRemote(branch string) error {
	args := m.Called(branch)
	return args.Error(0)
}

func (m *MockGitClient) GetMergeBase(branch1, branch2 string) (string, error) {
	args := m.Called(branch1, branch2)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) GetCommitHash(ref string) (string, error) {
	args := m.Called(ref)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) GetUniqueCommits(base, branch string) ([]string, error) {
	args := m.Called(base, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGitClient) GetUniqueCommitsByPatch(base, branch string) ([]string, error) {
	args := m.Called(base, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGitClient) CherryPick(commit string) error {
	args := m.Called(commit)
	return args.Error(0)
}

func (m *MockGitClient) ResetHard(ref string) error {
	args := m.Called(ref)
	return args.Error(0)
}

func (m *MockGitClient) Stash(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockGitClient) StashPop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClient) GetDefaultBranch() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockGitClient) GetWorktreeBranches() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockGitClient) GetCurrentWorktreePath() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) IsCommitsBehind(branch, base string) (bool, error) {
	args := m.Called(branch, base)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitClient) DeleteBranch(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockGitClient) DeleteBranchForce(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockGitClient) AddWorktree(path, branch string) error {
	args := m.Called(path, branch)
	return args.Error(0)
}

func (m *MockGitClient) AddWorktreeNewBranch(path, newBranch, baseBranch string) error {
	args := m.Called(path, newBranch, baseBranch)
	return args.Error(0)
}

func (m *MockGitClient) AddWorktreeFromRemote(path, branch string) error {
	args := m.Called(path, branch)
	return args.Error(0)
}

func (m *MockGitClient) RemoveWorktree(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockGitClient) ListWorktrees() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGitClient) GetRemoteURL(remoteName string) string {
	args := m.Called(remoteName)
	return args.String(0)
}

// MockGitHubClient is a mock implementation of github.GitHubClient for testing
type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) GetPRForBranch(branch string) (*github.PRInfo, error) {
	args := m.Called(branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.PRInfo), args.Error(1)
}

func (m *MockGitHubClient) GetAllPRs() (map[string]*github.PRInfo, error) {
	args := m.Called()
	return args.Get(0).(map[string]*github.PRInfo), args.Error(1)
}

func (m *MockGitHubClient) UpdatePRBase(prNumber int, newBase string) error {
	args := m.Called(prNumber, newBase)
	return args.Error(0)
}

func (m *MockGitHubClient) IsPRMerged(prNumber int) (bool, error) {
	args := m.Called(prNumber)
	return args.Bool(0), args.Error(1)
}
