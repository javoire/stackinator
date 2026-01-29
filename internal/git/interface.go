package git

// GitClient defines the interface for all git operations
type GitClient interface {
	GetRepoRoot() (string, error)
	GetRepoName() (string, error)
	GetCurrentBranch() (string, error)
	ListBranches() ([]string, error)
	GetConfig(key string) string
	GetAllStackParents() (map[string]string, error)
	SetConfig(key, value string) error
	UnsetConfig(key string) error
	CreateBranch(name, from string) error
	CreateBranchAndCheckout(name, from string) error
	CheckoutBranch(name string) error
	RenameBranch(oldName, newName string) error
	Rebase(onto string) error
	RebaseOnto(newBase, oldBase, currentBranch string) error
	FetchBranch(branch string) error
	Push(branch string, forceWithLease bool) error
	PushWithExpectedRemote(branch string, expectedRemoteSha string) error
	ForcePush(branch string) error
	IsWorkingTreeClean() (bool, error)
	Fetch() error
	BranchExists(name string) bool
	RemoteBranchExists(name string) bool
	GetRemoteBranchesSet() map[string]bool
	IsRebaseInProgress() bool
	IsCherryPickInProgress() bool
	AbortRebase() error
	AbortCherryPick() error
	ResetToRemote(branch string) error
	GetMergeBase(branch1, branch2 string) (string, error)
	GetCommitHash(ref string) (string, error)
	GetUniqueCommits(base, branch string) ([]string, error)
	GetUniqueCommitsByPatch(base, branch string) ([]string, error)
	CherryPick(commit string) error
	ResetHard(ref string) error
	Stash(message string) error
	StashPop() error
	GetDefaultBranch() string
	GetWorktreeBranches() (map[string]string, error)
	GetCurrentWorktreePath() (string, error)
	IsCommitsBehind(branch, base string) (bool, error)
	DeleteBranch(name string) error
	DeleteBranchForce(name string) error
	AddWorktree(path, branch string) error
	AddWorktreeNewBranch(path, newBranch, baseBranch string) error
	AddWorktreeFromRemote(path, branch string) error
	RemoveWorktree(path string) error
	ListWorktrees() ([]string, error)
	GetRemoteURL(remoteName string) string
}
