package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Verbose controls whether to print executed commands
var Verbose = false

// DryRun controls whether to actually execute mutation commands
var DryRun = false

// PRInfo contains information about a Pull Request
type PRInfo struct {
	Number           int
	State            string
	Base             string
	Title            string
	URL              string
	MergeStateStatus string // "BEHIND", "BLOCKED", "CLEAN", "DIRTY", "UNKNOWN", "UNSTABLE"
}

// githubClient implements the GitHubClient interface using exec.Command
type githubClient struct {
	repo string // OWNER/REPO format, used with --repo flag
}

// NewGitHubClient creates a new GitHubClient implementation
// repo should be in OWNER/REPO format (e.g., "javoire/stackinator")
func NewGitHubClient(repo string) GitHubClient {
	return &githubClient{repo: repo}
}

// ParseRepoFromURL extracts HOST/OWNER/REPO or OWNER/REPO from a git remote URL
// For github.com, returns OWNER/REPO (gh CLI default)
// For other hosts (GHE), returns HOST/OWNER/REPO so gh CLI knows which host to use
// Supports formats:
//   - git@github.com:owner/repo.git -> owner/repo
//   - https://github.com/owner/repo.git -> owner/repo
//   - git@ghe.spotify.net:owner/repo.git -> ghe.spotify.net/owner/repo
//   - https://ghe.spotify.net/owner/repo -> ghe.spotify.net/owner/repo
func ParseRepoFromURL(remoteURL string) string {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return ""
	}

	// Remove .git suffix
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	var host, path string

	// Handle SSH format: git@host:owner/repo
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.SplitN(remoteURL, ":", 2)
		if len(parts) == 2 {
			host = strings.TrimPrefix(parts[0], "git@")
			path = parts[1]
		}
	}

	// Handle HTTPS format: https://host/owner/repo
	if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		afterScheme := strings.TrimPrefix(remoteURL, "https://")
		afterScheme = strings.TrimPrefix(afterScheme, "http://")
		slashIdx := strings.Index(afterScheme, "/")
		if slashIdx != -1 {
			host = afterScheme[:slashIdx]
			path = afterScheme[slashIdx+1:]
		}
	}

	if path == "" {
		return ""
	}

	// For github.com, just return OWNER/REPO (it's the default)
	if host == "github.com" {
		return path
	}

	// For other hosts (GHE), return HOST/OWNER/REPO
	return host + "/" + path
}

// runGH executes a gh CLI command and returns stdout
func (c *githubClient) runGH(args ...string) (string, error) {
	// Add --repo flag if repo is set (ensures correct repo with multiple remotes)
	if c.repo != "" {
		args = append([]string{"--repo", c.repo}, args...)
	}
	if Verbose {
		fmt.Printf("  [gh] %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("gh %s failed: %s", strings.Join(args, " "), stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetPRForBranch returns PR info for the specified branch
func (c *githubClient) GetPRForBranch(branch string) (*PRInfo, error) {
	output, err := c.runGH("pr", "view", branch, "--json", "number,state,baseRefName,title,url,mergeStateStatus")
	if err != nil {
		// No PR exists for this branch
		return nil, nil
	}

	var data struct {
		Number           int    `json:"number"`
		State            string `json:"state"`
		BaseRefName      string `json:"baseRefName"`
		Title            string `json:"title"`
		URL              string `json:"url"`
		MergeStateStatus string `json:"mergeStateStatus"`
	}

	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %w", err)
	}

	return &PRInfo{
		Number:           data.Number,
		State:            data.State,
		Base:             data.BaseRefName,
		Title:            data.Title,
		URL:              data.URL,
		MergeStateStatus: data.MergeStateStatus,
	}, nil
}

// GetPRsForBranches fetches PR info for specific branches in parallel.
// This is much faster than bulk-fetching all PRs on large repos (500+ PRs),
// where `gh pr list --limit 500` can time out with 502 Bad Gateway.
func (c *githubClient) GetPRsForBranches(branches []string) map[string]*PRInfo {
	result := make(map[string]*PRInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, branch := range branches {
		wg.Add(1)
		go func(b string) {
			defer wg.Done()
			if pr, err := c.GetPRForBranch(b); err == nil && pr != nil {
				mu.Lock()
				result[b] = pr
				mu.Unlock()
			}
		}(branch)
	}

	wg.Wait()
	return result
}

// UpdatePRBase updates the base branch of a PR
func (c *githubClient) UpdatePRBase(prNumber int, newBase string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] gh pr edit %d --base %s\n", prNumber, newBase)
		return nil
	}

	_, err := c.runGH("pr", "edit", strconv.Itoa(prNumber), "--base", newBase)
	return err
}

// IsPRMerged checks if a PR has been merged
func (c *githubClient) IsPRMerged(prNumber int) (bool, error) {
	output, err := c.runGH("pr", "view", strconv.Itoa(prNumber), "--json", "state")
	if err != nil {
		return false, err
	}

	var data struct {
		State string `json:"state"`
	}

	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return false, fmt.Errorf("failed to parse PR state: %w", err)
	}

	return data.State == "MERGED", nil
}
