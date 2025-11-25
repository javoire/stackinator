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
	Number int
	State  string
	Base   string
	Title  string
}

// runGH executes a gh CLI command and returns stdout
func runGH(args ...string) (string, error) {
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
func GetPRForBranch(branch string) (*PRInfo, error) {
	output, err := runGH("pr", "view", branch, "--json", "number,state,baseRefName,title")
	if err != nil {
		// No PR exists for this branch
		return nil, nil
	}

	var data struct {
		Number      int    `json:"number"`
		State       string `json:"state"`
		BaseRefName string `json:"baseRefName"`
		Title       string `json:"title"`
	}

	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %w", err)
	}

	return &PRInfo{
		Number: data.Number,
		State:  data.State,
		Base:   data.BaseRefName,
		Title:  data.Title,
	}, nil
}

// UpdatePRBase updates the base branch of a PR
func UpdatePRBase(prNumber int, newBase string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] gh pr edit %d --base %s\n", prNumber, newBase)
		return nil
	}

	_, err := runGH("pr", "edit", strconv.Itoa(prNumber), "--base", newBase)
	return err
}

// IsPRMerged checks if a PR has been merged
func IsPRMerged(prNumber int) (bool, error) {
	output, err := runGH("pr", "view", strconv.Itoa(prNumber), "--json", "state")
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


