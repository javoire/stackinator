package github

// GitHubClient defines the interface for all GitHub operations
type GitHubClient interface {
	GetPRForBranch(branch string) (*PRInfo, error)
	GetPRsForBranches(branches []string) map[string]*PRInfo
	UpdatePRBase(prNumber int, newBase string) error
	IsPRMerged(prNumber int) (bool, error)
}

