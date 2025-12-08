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

// gitClient implements the GitClient interface using exec.Command
type gitClient struct{}

// NewGitClient creates a new GitClient implementation
func NewGitClient() GitClient {
	return &gitClient{}
}

// runCmd executes a git command and returns stdout
func (c *gitClient) runCmd(args ...string) (string, error) {
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
func (c *gitClient) runCmdMayFail(args ...string) string {
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
func (c *gitClient) GetRepoRoot() (string, error) {
	return c.runCmd("rev-parse", "--show-toplevel")
}

// GetCurrentBranch returns the name of the currently checked out branch
func (c *gitClient) GetCurrentBranch() (string, error) {
	return c.runCmd("branch", "--show-current")
}

// ListBranches returns a list of all local branches
func (c *gitClient) ListBranches() ([]string, error) {
	output, err := c.runCmd("branch", "--format=%(refname:short)")
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
func (c *gitClient) GetConfig(key string) string {
	return c.runCmdMayFail("config", "--get", key)
}

// GetAllStackParents fetches all stack parent configs in one call (more efficient)
func (c *gitClient) GetAllStackParents() (map[string]string, error) {
	output, err := c.runCmd("config", "--get-regexp", "^branch\\..*\\.stackparent$")
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
func (c *gitClient) SetConfig(key, value string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git config %s %s\n", key, value)
		return nil
	}
	_, err := c.runCmd("config", key, value)
	return err
}

// UnsetConfig removes a git config value
func (c *gitClient) UnsetConfig(key string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git config --unset %s\n", key)
		return nil
	}
	_, err := c.runCmd("config", "--unset", key)
	return err
}

// CreateBranch creates a new branch from the specified base and checks it out
func (c *gitClient) CreateBranch(name, from string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git checkout -b %s %s\n", name, from)
		return nil
	}
	_, err := c.runCmd("checkout", "-b", name, from)
	return err
}

// CheckoutBranch switches to the specified branch
func (c *gitClient) CheckoutBranch(name string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git checkout %s\n", name)
		return nil
	}
	_, err := c.runCmd("checkout", name)
	return err
}

// RenameBranch renames a branch (must be on that branch)
func (c *gitClient) RenameBranch(oldName, newName string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git branch -m %s %s\n", oldName, newName)
		return nil
	}
	_, err := c.runCmd("branch", "-m", oldName, newName)
	return err
}

// Rebase rebases the current branch onto the specified base
func (c *gitClient) Rebase(onto string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git rebase --autostash %s\n", onto)
		return nil
	}
	_, err := c.runCmd("rebase", "--autostash", onto)
	return err
}

// RebaseOnto rebases the current branch onto newBase, excluding commits up to and including oldBase
// This is useful for handling squash merges where oldBase was squashed into newBase
// Equivalent to: git rebase --onto newBase oldBase currentBranch
func (c *gitClient) RebaseOnto(newBase, oldBase, currentBranch string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git rebase --autostash --onto %s %s %s\n", newBase, oldBase, currentBranch)
		return nil
	}
	_, err := c.runCmd("rebase", "--autostash", "--onto", newBase, oldBase, currentBranch)
	return err
}

// FetchBranch fetches a specific branch from origin to update tracking info
func (c *gitClient) FetchBranch(branch string) error {
	// Use refspec to ensure the tracking ref is created/updated
	// git fetch origin <branch> alone only updates FETCH_HEAD, not refs/remotes/origin/<branch>
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
	if DryRun {
		fmt.Printf("  [DRY RUN] git fetch origin %s\n", refspec)
		return nil
	}
	_, err := c.runCmd("fetch", "origin", refspec)
	return err
}

// Push pushes a branch to origin
func (c *gitClient) Push(branch string, forceWithLease bool) error {
	args := []string{"push"}
	if forceWithLease {
		args = append(args, "--force-with-lease")
	}
	args = append(args, "origin", branch)

	if DryRun {
		fmt.Printf("  [DRY RUN] git %s\n", strings.Join(args, " "))
		return nil
	}

	_, err := c.runCmd(args...)
	return err
}

// PushWithExpectedRemote pushes a branch using --force-with-lease with an explicit expected SHA.
// This avoids "stale info" errors that can occur with plain --force-with-lease.
func (c *gitClient) PushWithExpectedRemote(branch string, expectedRemoteSha string) error {
	leaseArg := fmt.Sprintf("--force-with-lease=refs/heads/%s:%s", branch, expectedRemoteSha)
	args := []string{"push", leaseArg, "origin", branch}

	if DryRun {
		fmt.Printf("  [DRY RUN] git %s\n", strings.Join(args, " "))
		return nil
	}

	_, err := c.runCmd(args...)
	return err
}

// ForcePush force pushes a branch to origin (bypasses --force-with-lease safety)
func (c *gitClient) ForcePush(branch string) error {
	args := []string{"push", "--force", "origin", branch}

	if DryRun {
		fmt.Printf("  [DRY RUN] git %s\n", strings.Join(args, " "))
		return nil
	}

	_, err := c.runCmd(args...)
	return err
}

// IsWorkingTreeClean returns true if there are no uncommitted changes
func (c *gitClient) IsWorkingTreeClean() (bool, error) {
	output, err := c.runCmd("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return output == "", nil
}

// Fetch fetches from origin
func (c *gitClient) Fetch() error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git fetch origin\n")
		return nil
	}
	_, err := c.runCmd("fetch", "origin")
	return err
}

// BranchExists checks if a branch exists locally
func (c *gitClient) BranchExists(name string) bool {
	output := c.runCmdMayFail("rev-parse", "--verify", "refs/heads/"+name)
	return output != ""
}

// RemoteBranchExists checks if a branch exists on origin
func (c *gitClient) RemoteBranchExists(name string) bool {
	output := c.runCmdMayFail("rev-parse", "--verify", "refs/remotes/origin/"+name)
	return output != ""
}

// GetRemoteBranchesSet fetches all remote branches from origin in one call
// and returns a set (map[string]bool) for efficient lookups.
// This is more efficient than calling RemoteBranchExists multiple times.
func (c *gitClient) GetRemoteBranchesSet() map[string]bool {
	output := c.runCmdMayFail("for-each-ref", "--format=%(refname:short)", "refs/remotes/origin/")
	if output == "" {
		return make(map[string]bool)
	}

	branches := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove "origin/" prefix to get just the branch name
		if strings.HasPrefix(line, "origin/") {
			branchName := strings.TrimPrefix(line, "origin/")
			branches[branchName] = true
		}
	}

	return branches
}

// AbortRebase aborts an in-progress rebase
func (c *gitClient) AbortRebase() error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git rebase --abort\n")
		return nil
	}
	_, err := c.runCmd("rebase", "--abort")
	return err
}

// ResetToRemote resets the current branch to match the remote branch exactly
func (c *gitClient) ResetToRemote(branch string) error {
	remoteBranch := "origin/" + branch
	if DryRun {
		fmt.Printf("  [DRY RUN] git reset --hard %s\n", remoteBranch)
		return nil
	}
	_, err := c.runCmd("reset", "--hard", remoteBranch)
	return err
}

// GetMergeBase returns the common ancestor of two branches
func (c *gitClient) GetMergeBase(branch1, branch2 string) (string, error) {
	return c.runCmd("merge-base", branch1, branch2)
}

// GetCommitHash returns the commit hash of a ref
func (c *gitClient) GetCommitHash(ref string) (string, error) {
	return c.runCmd("rev-parse", ref)
}

// Stash stashes the current changes
func (c *gitClient) Stash(message string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git stash push -m \"%s\"\n", message)
		return nil
	}
	_, err := c.runCmd("stash", "push", "-m", message)
	return err
}

// StashPop pops the most recent stash
func (c *gitClient) StashPop() error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git stash pop\n")
		return nil
	}
	_, err := c.runCmd("stash", "pop")
	return err
}

// GetDefaultBranch attempts to detect the repository's default branch
// by checking the remote HEAD or falling back to common defaults
func (c *gitClient) GetDefaultBranch() string {
	// Try to get the remote's default branch
	output := c.runCmdMayFail("symbolic-ref", "refs/remotes/origin/HEAD")
	if output != "" {
		// Output format: refs/remotes/origin/master
		parts := strings.Split(output, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Fall back to checking which common branch exists
	for _, branch := range []string{"master", "main"} {
		if c.BranchExists(branch) {
			return branch
		}
	}

	// Final fallback
	return "main"
}

// GetWorktreeBranches returns a map of branch names to their worktree paths (resolved to canonical paths)
func (c *gitClient) GetWorktreeBranches() (map[string]string, error) {
	output := c.runCmdMayFail("worktree", "list", "--porcelain")
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
func (c *gitClient) GetCurrentWorktreePath() (string, error) {
	// Use git rev-parse to get the absolute path to the top-level of the current worktree
	path, err := c.runCmd("rev-parse", "--path-format=absolute", "--show-toplevel")
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
func (c *gitClient) IsCommitsBehind(branch, base string) (bool, error) {
	// NOTE: Caller should fetch first to ensure latest remote refs
	// We don't fetch here to avoid multiple fetches in loops

	// Always use origin/ prefix for the base since we're comparing against what's on the remote
	// (which is what the PR is based on)
	baseBranch := "origin/" + base

	// Get commit count: ahead...behind
	// Format: "ahead<tab>behind"
	output, err := c.runCmd("rev-list", "--left-right", "--count", branch+"..."+baseBranch)
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
func (c *gitClient) DeleteBranch(name string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git branch -d %s\n", name)
		return nil
	}
	_, err := c.runCmd("branch", "-d", name)
	return err
}

// DeleteBranchForce force deletes a branch (equivalent to git branch -D)
// This will delete the branch even if it has unmerged commits
func (c *gitClient) DeleteBranchForce(name string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git branch -D %s\n", name)
		return nil
	}
	_, err := c.runCmd("branch", "-D", name)
	return err
}

// AddWorktree creates a worktree at the specified path for an existing local branch
func (c *gitClient) AddWorktree(path, branch string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git worktree add %s %s\n", path, branch)
		return nil
	}
	_, err := c.runCmd("worktree", "add", path, branch)
	return err
}

// AddWorktreeNewBranch creates a worktree with a new branch at the specified path
// The new branch is created from the given base branch
func (c *gitClient) AddWorktreeNewBranch(path, newBranch, baseBranch string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git worktree add -b %s %s %s\n", newBranch, path, baseBranch)
		return nil
	}
	_, err := c.runCmd("worktree", "add", "-b", newBranch, path, baseBranch)
	return err
}

// AddWorktreeFromRemote creates a worktree tracking a remote branch
// This creates a local branch that tracks the remote branch
func (c *gitClient) AddWorktreeFromRemote(path, branch string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git worktree add --track -b %s %s origin/%s\n", branch, path, branch)
		return nil
	}
	_, err := c.runCmd("worktree", "add", "--track", "-b", branch, path, "origin/"+branch)
	return err
}

// RemoveWorktree removes a worktree at the specified path
func (c *gitClient) RemoveWorktree(path string) error {
	if DryRun {
		fmt.Printf("  [DRY RUN] git worktree remove %s\n", path)
		return nil
	}
	_, err := c.runCmd("worktree", "remove", path)
	return err
}

// ListWorktrees returns a list of all worktree paths
func (c *gitClient) ListWorktrees() ([]string, error) {
	output := c.runCmdMayFail("worktree", "list", "--porcelain")
	if output == "" {
		return []string{}, nil
	}

	var paths []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			paths = append(paths, path)
		}
	}

	return paths, nil
}
