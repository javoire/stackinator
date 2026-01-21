package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

// GetAllPRs fetches all PRs for the repository in a single call
func (c *githubClient) GetAllPRs() (map[string]*PRInfo, error) {
	// Fetch all PRs (open, closed, and merged) in one call
	output, err := c.runGH("pr", "list", "--state", "all", "--json", "number,state,headRefName,baseRefName,title,url,mergeStateStatus", "--limit", "1000")
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	var prs []struct {
		Number           int    `json:"number"`
		State            string `json:"state"`
		HeadRefName      string `json:"headRefName"`
		BaseRefName      string `json:"baseRefName"`
		Title            string `json:"title"`
		URL              string `json:"url"`
		MergeStateStatus string `json:"mergeStateStatus"`
	}

	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list: %w", err)
	}

	// Create a map of branch name -> PR info
	// When multiple PRs exist for the same branch, prefer OPEN over closed/merged
	prMap := make(map[string]*PRInfo)
	for _, pr := range prs {
		existing, exists := prMap[pr.HeadRefName]
		prInfo := &PRInfo{
			Number:           pr.Number,
			State:            pr.State,
			Base:             pr.BaseRefName,
			Title:            pr.Title,
			URL:              pr.URL,
			MergeStateStatus: pr.MergeStateStatus,
		}

		if !exists {
			// No PR for this branch yet, add it
			prMap[pr.HeadRefName] = prInfo
		} else if pr.State == "OPEN" && existing.State != "OPEN" {
			// New PR is open and existing is not - prefer the open one
			prMap[pr.HeadRefName] = prInfo
		}
		// Otherwise keep the existing PR (first open PR wins, or first closed if no open)
	}

	return prMap, nil
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
