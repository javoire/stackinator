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

// GetAllStackParents fetches all stack parent configs in one call (more efficient)
func GetAllStackParents() (map[string]string, error) {
	output, err := runCmd("config", "--get-regexp", "^branch\\..*\\.stackparent$")
	if err != nil {
		// No stack parents configured
		return make(map[string]string), nil
	}

	parents := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		// Extract branch name from "branch.<name>.stackparent"
		configKey := parts[0]
		parent := parts[1]

		// Remove "branch." prefix and ".stackparent" suffix
		if strings.HasPrefix(configKey, "branch.") && strings.HasSuffix(configKey, ".stackparent") {
			branchName := configKey[7 : len(configKey)-12]
			parents[branchName] = parent
		}
	}

	return parents, nil
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

// UnsetConfig removes a git config value
func UnsetConfig(key string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git config --unset %s\n", key)
		return nil
	}
	_, err := runCmd("config", "--unset", key)
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

// RenameBranch renames a branch (must be on that branch)
func RenameBranch(oldName, newName string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git branch -m %s %s\n", oldName, newName)
		return nil
	}
	_, err := runCmd("branch", "-m", oldName, newName)
	return err
}

// Rebase rebases the current branch onto the specified base
func Rebase(onto string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git rebase --autostash %s\n", onto)
		return nil
	}
	_, err := runCmd("rebase", "--autostash", onto)
	return err
}

// RebaseOnto rebases the current branch onto newBase, excluding commits up to and including oldBase
// This is useful for handling squash merges where oldBase was squashed into newBase
// Equivalent to: git rebase --onto newBase oldBase currentBranch
func RebaseOnto(newBase, oldBase, currentBranch string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git rebase --autostash --onto %s %s %s\n", newBase, oldBase, currentBranch)
		return nil
	}
	_, err := runCmd("rebase", "--autostash", "--onto", newBase, oldBase, currentBranch)
	return err
}

// FetchBranch fetches a specific branch from origin to update tracking info
func FetchBranch(branch string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git fetch origin %s\n", branch)
		return nil
	}
	_, err := runCmd("fetch", "origin", branch)
	return err
}

// Push pushes a branch to origin
func Push(branch string, forceWithLease bool) error {
	args := []string{"push"}
	if forceWithLease {
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

// ForcePush force pushes a branch to origin (bypasses --force-with-lease safety)
func ForcePush(branch string) error {
	args := []string{"push", "--force", "origin", branch}

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

// RemoteBranchExists checks if a branch exists on origin
func RemoteBranchExists(name string) bool {
	output := runCmdMayFail("rev-parse", "--verify", "refs/remotes/origin/"+name)
	return output != ""
}

// AbortRebase aborts an in-progress rebase
func AbortRebase() error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git rebase --abort\n")
		return nil
	}
	_, err := runCmd("rebase", "--abort")
	return err
}

// ResetToRemote resets the current branch to match the remote branch exactly
func ResetToRemote(branch string) error {
	remoteBranch := "origin/" + branch
	if DryRun {
		fmt.Printf("  [DRY RUN] git reset --hard %s\n", remoteBranch)
		return nil
	}
	_, err := runCmd("reset", "--hard", remoteBranch)
	return err
}

// GetMergeBase returns the common ancestor of two branches
func GetMergeBase(branch1, branch2 string) (string, error) {
	return runCmd("merge-base", branch1, branch2)
}

// GetCommitHash returns the commit hash of a ref
func GetCommitHash(ref string) (string, error) {
	return runCmd("rev-parse", ref)
}

// Stash stashes the current changes
func Stash(message string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git stash push -m \"%s\"\n", message)
		return nil
	}
	_, err := runCmd("stash", "push", "-m", message)
	return err
}

// StashPop pops the most recent stash
func StashPop() error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git stash pop\n")
		return nil
	}
	_, err := runCmd("stash", "pop")
	return err
}

// GetDefaultBranch attempts to detect the repository's default branch
// by checking the remote HEAD or falling back to common defaults
func GetDefaultBranch() string {
	// Try to get the remote's default branch
	output := runCmdMayFail("symbolic-ref", "refs/remotes/origin/HEAD")
	if output != "" {
		// Output format: refs/remotes/origin/master
		parts := strings.Split(output, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Fall back to checking which common branch exists
	for _, branch := range []string{"master", "main"} {
		if BranchExists(branch) {
			return branch
		}
	}

	// Final fallback
	return "main"
}

// GetWorktreeBranches returns a map of branch names to their worktree paths (resolved to canonical paths)
func GetWorktreeBranches() (map[string]string, error) {
	output := runCmdMayFail("worktree", "list", "--porcelain")
	if output == "" {
		return make(map[string]string), nil
	}

	worktrees := make(map[string]string)
	var currentPath string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") && currentPath != "" {
			branch := strings.TrimPrefix(line, "branch refs/heads/")
			// Resolve symlinks to get canonical path for accurate comparison
			canonicalPath, err := resolveSymlinks(currentPath)
			if err != nil {
				// If we can't resolve, use the original path
				canonicalPath = currentPath
			}
			worktrees[branch] = canonicalPath
			currentPath = "" // Reset for next worktree
		}
	}

	return worktrees, nil
}

// GetCurrentWorktreePath returns the absolute path of the current worktree
func GetCurrentWorktreePath() (string, error) {
	// Use git rev-parse to get the absolute path to the top-level of the current worktree
	path, err := runCmd("rev-parse", "--path-format=absolute", "--show-toplevel")
	if err != nil {
		return "", err
	}
	// Resolve symlinks to get the canonical path for accurate comparison
	return resolveSymlinks(path)
}

// resolveSymlinks resolves any symlinks in a path to get the canonical path
func resolveSymlinks(path string) (string, error) {
	// Use readlink -f to resolve all symlinks (this is the canonical path)
	// Note: This uses the system readlink command, not a git command
	cmd := exec.Command("readlink", "-f", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// If readlink fails (e.g., on macOS where -f isn't available),
		// try with realpath instead
		cmd = exec.Command("realpath", path)
		stdout.Reset()
		stderr.Reset()
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			// If both fail, return the original path
			return path, nil
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsCommitsBehind checks if the 'branch' is behind 'base' (i.e., base has commits that branch doesn't)
func IsCommitsBehind(branch, base string) (bool, error) {
	// NOTE: Caller should fetch first to ensure latest remote refs
	// We don't fetch here to avoid multiple fetches in loops

	// Always use origin/ prefix for the base since we're comparing against what's on the remote
	// (which is what the PR is based on)
	baseBranch := "origin/" + base

	// Get commit count: ahead...behind
	// Format: "ahead<tab>behind"
	output, err := runCmd("rev-list", "--left-right", "--count", branch+"..."+baseBranch)
	if err != nil {
		return false, err
	}

	parts := strings.Fields(output)
	if len(parts) != 2 {
		return false, fmt.Errorf("unexpected output from git rev-list: %s", output)
	}

	// parts[0] = ahead count, parts[1] = behind count
	// We only care if behind count > 0
	return parts[1] != "0", nil
}

// DeleteBranch deletes a branch safely (equivalent to git branch -d)
// This will fail if the branch has unmerged commits
func DeleteBranch(name string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git branch -d %s\n", name)
		return nil
	}
	_, err := runCmd("branch", "-d", name)
	return err
}

// DeleteBranchForce force deletes a branch (equivalent to git branch -D)
// This will delete the branch even if it has unmerged commits
func DeleteBranchForce(name string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git branch -D %s\n", name)
		return nil
	}
	_, err := runCmd("branch", "-D", name)
	return err
}
