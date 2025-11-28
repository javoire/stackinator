package testutil

import "github.com/javoire/stackinator/internal/github"

// BuildStackParents creates a map of branch names to their stack parents for testing
func BuildStackParents(config map[string]string) map[string]string {
	return config
}

// CreatePRMap creates a map of branch names to PR info for testing
func CreatePRMap(prs map[string]*github.PRInfo) map[string]*github.PRInfo {
	return prs
}

// NewPRInfo creates a PR info struct for testing
func NewPRInfo(number int, state, base, title, url string) *github.PRInfo {
	return &github.PRInfo{
		Number:           number,
		State:            state,
		Base:             base,
		Title:            title,
		URL:              url,
		MergeStateStatus: "CLEAN",
	}
}

// BuildGitConfig simulates git config output
func BuildGitConfig(configs map[string]string) map[string]string {
	return configs
}

// BuildRemoteBranchesSet creates a set of remote branches for testing
func BuildRemoteBranchesSet(branches []string) map[string]bool {
	set := make(map[string]bool)
	for _, branch := range branches {
		set[branch] = true
	}
	return set
}

