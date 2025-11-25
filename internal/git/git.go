package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Verbose controls whether to print executed commands
var Verbose = false

// DryRun controls whether to actually execute mutation commands
var DryRun = false

// runCmd executes a git command and returns stdout
func runCmd(args ...string) (string, error) {
	if Verbose {
		fmt.Printf("  [git] %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// runCmdMayFail runs a command that might fail (returns empty string on error)
func runCmdMayFail(args ...string) string {
	if Verbose {
		fmt.Printf("  [git] %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	_ = cmd.Run()
	return strings.TrimSpace(stdout.String())
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot() (string, error) {
	return runCmd("rev-parse", "--show-toplevel")
}

// GetCurrentBranch returns the name of the currently checked out branch
func GetCurrentBranch() (string, error) {
	return runCmd("branch", "--show-current")
}

// ListBranches returns a list of all local branches
func ListBranches() ([]string, error) {
	output, err := runCmd("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	branches := strings.Split(output, "\n")
	return branches, nil
}

// GetConfig reads a git config value
func GetConfig(key string) string {
	return runCmdMayFail("config", "--get", key)
}

// SetConfig writes a git config value
func SetConfig(key, value string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git config %s %s\n", key, value)
		return nil
	}
	_, err := runCmd("config", key, value)
	return err
}

// CreateBranch creates a new branch from the specified base and checks it out
func CreateBranch(name, from string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git checkout -b %s %s\n", name, from)
		return nil
	}
	_, err := runCmd("checkout", "-b", name, from)
	return err
}

// CheckoutBranch switches to the specified branch
func CheckoutBranch(name string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git checkout %s\n", name)
		return nil
	}
	_, err := runCmd("checkout", name)
	return err
}

// Rebase rebases the current branch onto the specified base
func Rebase(onto string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git rebase %s\n", onto)
		return nil
	}
	_, err := runCmd("rebase", onto)
	return err
}

// Push pushes a branch to origin
func Push(branch string, force bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	args = append(args, "origin", branch)

	if DryRun {
		fmt.Printf("  [DRY RUN] git %s\n", strings.Join(args, " "))
		return nil
	}

	_, err := runCmd(args...)
	return err
}

// IsWorkingTreeClean returns true if there are no uncommitted changes
func IsWorkingTreeClean() (bool, error) {
	output, err := runCmd("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return output == "", nil
}

// Fetch fetches from origin
func Fetch() error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git fetch origin\n")
		return nil
	}
	_, err := runCmd("fetch", "origin")
	return err
}

// BranchExists checks if a branch exists locally
func BranchExists(name string) bool {
	output := runCmdMayFail("rev-parse", "--verify", "refs/heads/"+name)
	return output != ""
}


