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
type githubClient struct{}

// NewGitHubClient creates a new GitHubClient implementation
func NewGitHubClient() GitHubClient {
	return &githubClient{}
}

// runGH executes a gh CLI command and returns stdout
func (c *githubClient) runGH(args ...string) (string, error) {
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
	prMap := make(map[string]*PRInfo)
	for _, pr := range prs {
		prMap[pr.HeadRefName] = &PRInfo{
			Number:           pr.Number,
			State:            pr.State,
			Base:             pr.BaseRefName,
			Title:            pr.Title,
			URL:              pr.URL,
			MergeStateStatus: pr.MergeStateStatus,
		}
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
