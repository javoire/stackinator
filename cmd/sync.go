package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/spinner"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

// errAlreadyPrinted is a sentinel error indicating the error message was already displayed
var errAlreadyPrinted = errors.New("")

var (
	syncForce  bool
	syncResume bool
	syncAbort  bool
)

// Git config keys for sync state persistence
const (
	configSyncStashed        = "stack.sync.stashed"
	configSyncOriginalBranch = "stack.sync.originalBranch"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync all stack branches with their parents and update PRs",
	Long: `Perform a full sync of the stack:
  1. Fetch latest changes from origin
  2. Rebase each stack branch onto its parent (in bottom-to-top order)
  3. Force push each branch to origin
  4. Update PR base branches to match the stack (if PRs exist)

This ensures your stack is up-to-date and all PRs have the correct base branches.

If a parent PR has been merged, the child branches will be rebased to point to
the merged parent's parent.

Uncommitted changes are automatically stashed and reapplied (using --autostash).`,
	Example: `  # Sync all branches and update PRs
  stack sync

  # Preview what would happen
  stack sync --dry-run

  # Show detailed git/gh commands
  stack sync --verbose

  # Force push even if branches have diverged
  stack sync --force

  # Resume after resolving rebase conflicts
  stack sync --resume

  # Abort an interrupted sync
  stack sync --abort

  # Common workflow after updating main
  git checkout main && git pull
  stack sync`,
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()
		githubClient := github.NewGitHubClient()

		if err := runSync(gitClient, githubClient); err != nil {
			// Don't print if error was already displayed with detailed message
			if !errors.Is(err, errAlreadyPrinted) {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			os.Exit(1)
		}
	},
}

func init() {
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Use --force instead of --force-with-lease for push (bypasses safety checks)")
	syncCmd.Flags().BoolVarP(&syncResume, "resume", "r", false, "Resume a sync after resolving rebase conflicts")
	syncCmd.Flags().BoolVarP(&syncAbort, "abort", "a", false, "Abort an interrupted sync and clean up state")
}

func runSync(gitClient git.GitClient, githubClient github.GitHubClient) error {
	// Track state for stash handling
	var originalBranch string
	stashed := false
	rebaseConflict := false

	// Check for existing sync state (from previous interrupted sync)
	savedStashed := gitClient.GetConfig(configSyncStashed)
	savedOriginalBranch := gitClient.GetConfig(configSyncOriginalBranch)
	hasSavedState := savedStashed == "true" || savedOriginalBranch != ""

	if syncAbort {
		// Check if there's actually anything to abort
		hasCherryPick := gitClient.IsCherryPickInProgress()
		hasRebase := gitClient.IsRebaseInProgress()

		if !hasSavedState && !hasCherryPick && !hasRebase {
			return fmt.Errorf("no interrupted sync to abort\n\nUse 'stack sync' to start a new sync")
		}

		fmt.Println("Aborting sync and cleaning up...")
		fmt.Println()

		// Abort cherry-pick if one is in progress
		if hasCherryPick {
			if err := gitClient.AbortCherryPick(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to abort cherry-pick: %v\n", err)
			} else {
				fmt.Println("✓ Aborted cherry-pick")
			}
		} else if git.Verbose {
			fmt.Fprintf(os.Stderr, "Note: no cherry-pick in progress\n")
		}

		// Abort rebase if one is in progress
		if hasRebase {
			if err := gitClient.AbortRebase(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to abort rebase: %v\n", err)
			} else {
				fmt.Println("✓ Aborted rebase")
			}
		} else if git.Verbose {
			fmt.Fprintf(os.Stderr, "Note: no rebase in progress\n")
		}

		// Restore stashed changes if any
		if savedStashed == "true" {
			fmt.Println("Restoring stashed changes...")
			if err := gitClient.StashPop(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restore stashed changes: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'git stash pop' manually to restore your changes\n")
			} else {
				fmt.Println("✓ Restored stashed changes")
			}
		}

		// Return to original branch if we have one saved
		if savedOriginalBranch != "" {
			currentBranch, err := gitClient.GetCurrentBranch()
			if err == nil && currentBranch != savedOriginalBranch {
				fmt.Printf("Returning to %s...\n", savedOriginalBranch)
				if err := gitClient.CheckoutBranch(savedOriginalBranch); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to return to original branch: %v\n", err)
				} else {
					fmt.Printf("✓ Returned to %s\n", savedOriginalBranch)
				}
			}
		}

		// Clean up sync state
		_ = gitClient.UnsetConfig(configSyncStashed)
		_ = gitClient.UnsetConfig(configSyncOriginalBranch)

		fmt.Println()
		fmt.Println("✓ Sync aborted and state cleaned up")
		return nil
	}

	if syncResume {
		// Resuming after conflict resolution
		if !hasSavedState {
			return fmt.Errorf("no interrupted sync to resume\n\nUse 'stack sync' to start a new sync")
		}
		stashed = true
		originalBranch = savedOriginalBranch
		fmt.Println("Resuming sync...")
		fmt.Println()
	} else {
		// Starting a fresh sync
		if hasSavedState {
			fmt.Fprintf(os.Stderr, "Warning: found state from a previous interrupted sync\n")
			fmt.Fprintf(os.Stderr, "If you resolved rebase conflicts, run 'stack sync --resume'\n")
			fmt.Fprintf(os.Stderr, "Otherwise, cleaning up stale state and starting fresh...\n\n")
			// Clean up stale state
			_ = gitClient.UnsetConfig(configSyncStashed)
			_ = gitClient.UnsetConfig(configSyncOriginalBranch)
		}

		// Get current branch so we can return to it
		var err error
		originalBranch, err = gitClient.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// Save original branch state for potential --abort
		if err := gitClient.SetConfig(configSyncOriginalBranch, originalBranch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save sync state: %v\n", err)
		}

		// Check if working tree is clean and stash if needed
		clean, err := gitClient.IsWorkingTreeClean()
		if err != nil {
			return fmt.Errorf("failed to check working tree status: %w", err)
		}

		if !clean {
			fmt.Println("Stashing uncommitted changes...")
			if err := gitClient.Stash("stack-sync-autostash"); err != nil {
				return fmt.Errorf("failed to stash changes: %w", err)
			}
			stashed = true

			// Mark that we stashed changes
			if err := gitClient.SetConfig(configSyncStashed, "true"); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save sync state: %v\n", err)
			}

			fmt.Println()
		}
	}

	// Track if we complete successfully
	success := false

	// Ensure stash is popped on error (if we don't complete successfully)
	// But NOT if we hit a rebase conflict - user needs to resolve and --resume
	defer func() {
		if stashed && !success && !rebaseConflict {
			fmt.Println("\nRestoring stashed changes...")
			if err := gitClient.StashPop(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restore stashed changes: %v\n", err)
				fmt.Fprintf(os.Stderr, "Run 'git stash pop' manually to restore your changes\n")
			}
			// Clean up sync state since we're restoring the stash
			_ = gitClient.UnsetConfig(configSyncStashed)
			_ = gitClient.UnsetConfig(configSyncOriginalBranch)
		}
	}()

	// Check if current branch is in a stack BEFORE doing any network operations
	// This allows us to prompt the user immediately if needed
	baseBranch := stack.GetBaseBranch(gitClient)
	parent := gitClient.GetConfig(fmt.Sprintf("branch.%s.stackparent", originalBranch))

	if parent == "" && originalBranch != baseBranch {
		fmt.Printf("Branch '%s' is not in a stack.\n", originalBranch)
		fmt.Printf("Add it with parent '%s'? [Y/n] ", baseBranch)

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input != "" && input != "y" && input != "yes" {
			fmt.Println("Aborted.")
			return nil
		}

		// Set the parent
		configKey := fmt.Sprintf("branch.%s.stackparent", originalBranch)
		if err := gitClient.SetConfig(configKey, baseBranch); err != nil {
			return fmt.Errorf("failed to set parent: %w", err)
		}
		fmt.Printf("✓ Added '%s' to stack with parent '%s'\n", originalBranch, baseBranch)
	}

	// Start parallel fetch operations (git fetch and GitHub PR fetch)
	// These are the slowest operations and have no dependencies between them
	var wg sync.WaitGroup
	var fetchErr error
	var prCache map[string]*github.PRInfo
	var prErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		fetchErr = gitClient.Fetch()
	}()
	go func() {
		defer wg.Done()
		prCache, prErr = githubClient.GetAllPRs()
	}()

	// While network operations run in background, do local work
	// Get only branches in the current branch's stack
	chain, err := stack.GetStackChain(gitClient, originalBranch)
	if err != nil {
		return fmt.Errorf("failed to get stack chain: %w", err)
	}

	if len(chain) == 0 {
		// Wait for parallel operations before returning
		wg.Wait()
		fmt.Println("No stack branches found.")
		return nil
	}

	// Build set of branches in current stack
	chainSet := make(map[string]bool)
	for _, b := range chain {
		chainSet[b] = true
	}

	// Get all stack branches and filter to current stack only
	allStackBranches, err := stack.GetStackBranches(gitClient)
	if err != nil {
		return fmt.Errorf("failed to get stack branches: %w", err)
	}

	var stackBranches []stack.StackBranch
	for _, b := range allStackBranches {
		if chainSet[b.Name] {
			stackBranches = append(stackBranches, b)
		}
	}

	// Sort branches in topological order (bottom to top)
	sorted, err := stack.TopologicalSort(stackBranches)
	if err != nil {
		return fmt.Errorf("failed to sort branches: %w", err)
	}

	// Check if any branches in the current stack are in worktrees
	worktrees, err := gitClient.GetWorktreeBranches()
	if err != nil {
		// Non-fatal, continue without worktree detection
		worktrees = make(map[string]string)
	}

	// Get current worktree path to check if we're already in the right place
	currentWorktreePath, err := gitClient.GetCurrentWorktreePath()
	if err != nil {
		// Non-fatal, continue without worktree path detection
		currentWorktreePath = ""
	}

	for _, branch := range sorted {
		if worktreePath, inWorktree := worktrees[branch.Name]; inWorktree {
			// Only error if we're NOT already in this worktree
			if currentWorktreePath != worktreePath {
				return fmt.Errorf(
					"cannot sync: branch '%s' is checked out in worktree at %s\n\n"+
						"To sync this stack:\n"+
						"  1. cd %s\n"+
						"  2. stack sync\n\n"+
						"Or remove the worktree: git worktree remove %s",
					branch.Name,
					worktreePath,
					worktreePath,
					worktreePath,
				)
			}
		}
	}

	// Wait for parallel network operations to complete
	if err := spinner.WrapWithSuccess("Fetching from origin and loading PRs...", "Fetched from origin and loaded PRs", func() error {
		wg.Wait()
		return nil
	}); err != nil {
		return err
	}

	// Check for fetch errors
	if fetchErr != nil {
		return fmt.Errorf("failed to fetch: %w", fetchErr)
	}

	// Handle PR fetch errors gracefully
	if prErr != nil {
		prCache = make(map[string]*github.PRInfo)
	}

	// Get all remote branches in one call (more efficient than checking each branch individually)
	remoteBranches := gitClient.GetRemoteBranchesSet()

	fmt.Printf("Processing %d branch(es)...\n\n", len(sorted))

	// Build a set of stack branch names for quick lookup
	stackBranchSet := make(map[string]bool)
	for _, sb := range stackBranches {
		stackBranchSet[sb.Name] = true
	}

	// Process each branch
	for i, branch := range sorted {
		progress := fmt.Sprintf("(%d/%d)", i+1, len(sorted))

		// Check if this branch has a merged PR - if so, remove from stack tracking
		if pr, exists := prCache[branch.Name]; exists && pr.State == "MERGED" {
			fmt.Printf("%s Skipping %s (PR #%d is merged)...\n", progress, branch.Name, pr.Number)
			fmt.Printf("  Removing from stack tracking...\n")
			configKey := fmt.Sprintf("branch.%s.stackparent", branch.Name)
			if err := gitClient.UnsetConfig(configKey); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to remove stack config: %v\n", err)
			} else {
				fmt.Printf("  ✓ Removed. You can delete this branch with: git branch -d %s\n", branch.Name)
			}
			fmt.Println()
			continue
		}

		fmt.Printf("%s Processing %s...\n", progress, branch.Name)

		// Check if parent PR is merged
		oldParent := "" // Track old parent for --onto rebase
		parentPR := prCache[branch.Parent]
		if parentPR != nil && parentPR.State == "MERGED" {
			fmt.Printf("  Parent PR #%d has been merged\n", parentPR.Number)

			// Save old parent for --onto rebase
			oldParent = branch.Parent

			// Update parent to grandparent
			grandparent := gitClient.GetConfig(fmt.Sprintf("branch.%s.stackparent", branch.Parent))
			if grandparent == "" {
				grandparent = stack.GetBaseBranch(gitClient)
			}

			fmt.Printf("  Updating parent from %s to %s\n", branch.Parent, grandparent)
			configKey := fmt.Sprintf("branch.%s.stackparent", branch.Name)
			if err := gitClient.SetConfig(configKey, grandparent); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to update parent config: %v\n", err)
			} else {
				branch.Parent = grandparent
			}
		}

		// Checkout the branch
		if err := gitClient.CheckoutBranch(branch.Name); err != nil {
			return fmt.Errorf("failed to checkout %s: %w", branch.Name, err)
		}

		// Sync with remote branch if it exists (unless --force is set)
		remoteBranch := "origin/" + branch.Name
		// Check if we have a local tracking ref for the remote branch
		hasLocalRef := remoteBranches[branch.Name]
		// Branch exists on remote if we have local tracking ref OR if there's a PR for it
		// (PR existence proves branch is on origin, even if local ref is missing)
		branchExistsOnRemote := hasLocalRef || prCache[branch.Name] != nil

		// If branch is on remote but we don't have the local tracking ref, fetch it
		if branchExistsOnRemote && !hasLocalRef {
			if git.Verbose {
				fmt.Printf("  Fetching remote branch (local tracking ref missing)...\n")
			}
			if err := gitClient.FetchBranch(branch.Name); err != nil {
				// If fetch fails, the branch might have been deleted on remote
				// Fall back to treating it as a new branch
				if git.Verbose {
					fmt.Printf("  Could not fetch remote branch, treating as new branch\n")
				}
				branchExistsOnRemote = false
			}
		}

		if branchExistsOnRemote && !syncForce {
			// Check if local and remote have diverged
			localHash, err := gitClient.GetCommitHash(branch.Name)
			if err != nil {
				return fmt.Errorf("failed to get local commit hash: %w", err)
			}

			remoteHash, err := gitClient.GetCommitHash(remoteBranch)
			if err != nil {
				return fmt.Errorf("failed to get remote commit hash: %w", err)
			}

			if localHash != remoteHash {
				// Check merge base to determine relationship
				mergeBase, err := gitClient.GetMergeBase(branch.Name, remoteBranch)
				if err != nil {
					return fmt.Errorf("failed to get merge base: %w", err)
				}

				if mergeBase == remoteHash {
					// Local is ahead of remote (we have new commits)
					if git.Verbose {
						fmt.Printf("  Local branch is ahead of origin (has new commits)\n")
					}
				} else if mergeBase == localHash {
					// Local is behind remote (safe to fast-forward)
					fmt.Printf("  Fast-forwarding to origin/%s...\n", branch.Name)
					if err := gitClient.ResetToRemote(branch.Name); err != nil {
						return fmt.Errorf("failed to fast-forward: %w", err)
					}
				} else {
					// Branches have diverged - this is normal after rebasing onto an updated parent
					// --force-with-lease will safely handle this during push
					if git.Verbose {
						fmt.Printf("  Local and remote have diverged (normal after rebase)\n")
					}
				}
			} else if git.Verbose {
				fmt.Printf("  Local branch is up-to-date with origin/%s\n", branch.Name)
			}
		} else if syncForce && branchExistsOnRemote {
			if git.Verbose {
				fmt.Printf("  Skipping divergence check (--force enabled)\n")
			}
		} else {
			if git.Verbose {
				fmt.Printf("  Remote branch origin/%s doesn't exist yet (new branch)\n", branch.Name)
			}
		}

		// Determine rebase target: origin/<parent> for base branches, local for stack branches
		rebaseTarget := branch.Parent
		if !stackBranchSet[branch.Parent] {
			// Parent is not a stack branch, so it's a base branch - use origin/<parent>
			rebaseTarget = "origin/" + branch.Parent
		}

		// Rebase onto parent
		// If parent was just merged (oldParent set), use --onto to exclude old parent's commits
		if err := spinner.WrapWithSuccess(
			fmt.Sprintf("  Rebasing onto %s...", rebaseTarget),
			fmt.Sprintf("  Rebased onto %s", rebaseTarget),
			func() error {
				if oldParent != "" {
					// Parent was merged - use --onto to handle squash merge
					// This excludes commits from oldParent that are now in rebaseTarget
					fmt.Printf("  Using --onto to handle squash merge (excluding commits from %s)\n", oldParent)
					return gitClient.RebaseOnto(rebaseTarget, oldParent, branch.Name)
				}

				// Get unique commits in this branch by comparing patch content (not just SHAs)
				// This detects duplicate changes even if commits were rebased with different SHAs
				uniqueCommits, err := gitClient.GetUniqueCommitsByPatch(rebaseTarget, branch.Name)
				if err != nil {
					// If we can't get unique commits, fall back to regular rebase
					if git.Verbose {
						fmt.Printf("  Could not get unique commits by patch, using regular rebase: %v\n", err)
					}
					return gitClient.Rebase(rebaseTarget)
				}

				// If no unique commits, branch is up-to-date
				if len(uniqueCommits) == 0 {
					if git.Verbose {
						fmt.Printf("  Branch is up-to-date with %s (no unique patches)\n", rebaseTarget)
					}
					return nil
				}

				if git.Verbose {
					fmt.Printf("  Found %d unique commit(s) by patch comparison\n", len(uniqueCommits))
				}

				// Get merge-base to understand the history
				mergeBase, err := gitClient.GetMergeBase(branch.Name, rebaseTarget)
				if err != nil {
					// If we can't find merge-base, fall back to regular rebase
					if git.Verbose {
						fmt.Printf("  Could not find merge-base, using regular rebase: %v\n", err)
					}
					return gitClient.Rebase(rebaseTarget)
				}

				rebaseTargetHash, err := gitClient.GetCommitHash(rebaseTarget)
				if err == nil && mergeBase == rebaseTargetHash {
					// Parent hasn't changed since we branched, regular rebase is fine
					return gitClient.Rebase(rebaseTarget)
				}

				// Count commits from merge-base to current branch (total commits in branch history)
				allCommits, err := gitClient.GetUniqueCommits(mergeBase, branch.Name)
				if err == nil && len(allCommits) > len(uniqueCommits)*2 {
					// Branch has polluted history: many more commits than unique patches
					// This usually means branch diverged from parent's history (e.g., based on old backup)
					rebaseConflict = true
					fmt.Fprintf(os.Stderr, "\n")
					fmt.Fprintf(os.Stderr, "⚠ Detected polluted branch history:\n")
					fmt.Fprintf(os.Stderr, "  - %d commits in branch history\n", len(allCommits))
					fmt.Fprintf(os.Stderr, "  - Only %d unique patch(es)\n", len(uniqueCommits))
					fmt.Fprintf(os.Stderr, "\n")
					fmt.Fprintf(os.Stderr, "This usually means your branch diverged from the parent's history.\n")
					fmt.Fprintf(os.Stderr, "Rebasing may result in many conflicts.\n")
					fmt.Fprintf(os.Stderr, "\n")
					fmt.Fprintf(os.Stderr, "Recommended: Rebuild branch manually with cherry-pick:\n")
					fmt.Fprintf(os.Stderr, "  1. git checkout %s\n", branch.Parent)
					fmt.Fprintf(os.Stderr, "  2. git checkout -b %s-clean\n", branch.Name)
					for i, commit := range uniqueCommits {
						if i < 5 { // Show first 5 commits
							fmt.Fprintf(os.Stderr, "  3. git cherry-pick %s\n", commit[:8])
						}
					}
					if len(uniqueCommits) > 5 {
						fmt.Fprintf(os.Stderr, "     ... (%d more commits)\n", len(uniqueCommits)-5)
					}
					fmt.Fprintf(os.Stderr, "  4. git branch -D %s\n", branch.Name)
					fmt.Fprintf(os.Stderr, "  5. git branch -m %s\n", branch.Name)
					fmt.Fprintf(os.Stderr, "  6. git push --force-with-lease\n")
					fmt.Fprintf(os.Stderr, "\n")
					return fmt.Errorf("branch history is polluted, manual cleanup recommended")
				}

				// Use --onto to only replay commits unique to this branch
				// This prevents conflicts from duplicate commits when parent was rebased
				if git.Verbose {
					fmt.Printf("  Using --onto with merge-base %s to handle rebased parent\n", mergeBase[:8])
				}
				return gitClient.RebaseOnto(rebaseTarget, mergeBase, branch.Name)
			},
		); err != nil {
			rebaseConflict = true
			fmt.Fprintf(os.Stderr, "\n  Rebase conflict detected. To continue:\n")
			fmt.Fprintf(os.Stderr, "    1. Resolve the conflicts\n")
			fmt.Fprintf(os.Stderr, "    2. Run 'git add <resolved files>'\n")
			fmt.Fprintf(os.Stderr, "    3. Run 'git rebase --continue'\n")
			fmt.Fprintf(os.Stderr, "    4. Run 'stack sync --resume'\n")
			fmt.Fprintf(os.Stderr, "\n  Or to abort the sync:\n")
			fmt.Fprintf(os.Stderr, "    Run 'stack sync --abort'\n")
			if stashed {
				fmt.Fprintf(os.Stderr, "\n  Note: Your uncommitted changes have been stashed and will be restored when you run --resume or --abort\n")
			}
			return fmt.Errorf("failed to rebase: %w", errAlreadyPrinted)
		}

		// Push to origin - only if the branch already exists remotely
		if branchExistsOnRemote {
			pushErr := spinner.WrapWithSuccess(
				"  Pushing to origin...",
				"  Pushed to origin",
				func() error {
					if syncForce {
						// Use regular --force (bypasses --force-with-lease safety checks)
						if git.Verbose {
							fmt.Printf("  Using --force (bypassing safety checks)\n")
						}
						return gitClient.ForcePush(branch.Name)
					}

					// Fetch one more time right before push to get the current remote SHA
					if git.Verbose {
						fmt.Printf("  Refreshing remote tracking ref before push...\n")
					}
					if err := gitClient.FetchBranch(branch.Name); err != nil {
						// Non-fatal, continue with push using plain --force-with-lease
						if git.Verbose {
							fmt.Fprintf(os.Stderr, "  Note: could not refresh tracking ref: %v\n", err)
						}
						return gitClient.Push(branch.Name, true)
					}

					// Get the remote SHA to use with explicit --force-with-lease
					// This avoids "stale info" errors that can occur with plain --force-with-lease
					remoteSha, err := gitClient.GetCommitHash("origin/" + branch.Name)
					if err != nil {
						// Fall back to plain --force-with-lease
						if git.Verbose {
							fmt.Fprintf(os.Stderr, "  Note: could not get remote SHA, using plain force-with-lease: %v\n", err)
						}
						return gitClient.Push(branch.Name, true)
					}

					return gitClient.PushWithExpectedRemote(branch.Name, remoteSha)
				},
			)

			if pushErr != nil {
				if !syncForce {
					fmt.Fprintf(os.Stderr, "\nPossible cause:\n")
					fmt.Fprintf(os.Stderr, "  Remote branch was updated after fetch - try running 'stack sync' again\n")
				}
				return fmt.Errorf("push failed for %s", branch.Name)
			}
		} else {
			fmt.Printf("  Skipping push (branch not yet on origin)\n")
		}

		// Check if PR exists and update base if needed
		pr := prCache[branch.Name]
		if pr != nil {
			if pr.Base != branch.Parent {
				fmt.Printf("  Updating PR #%d base from %s to %s...\n", pr.Number, pr.Base, branch.Parent)
				if err := githubClient.UpdatePRBase(pr.Number, branch.Parent); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: failed to update PR base: %v\n", err)
				} else {
					fmt.Printf("✓   PR #%d updated\n", pr.Number)
				}
			} else {
				fmt.Printf("✓   PR #%d base is already correct (%s)\n", pr.Number, pr.Base)
			}
		} else {
			fmt.Printf("  No PR found (create one with 'gh pr create')\n")
		}

		fmt.Println()
	}

	// Return to original branch
	fmt.Printf("Returning to %s...\n", originalBranch)
	if err := gitClient.CheckoutBranch(originalBranch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to return to original branch: %v\n", err)
	}

	fmt.Println()

	// Display the updated stack status (reuse prCache to avoid redundant API call)
	if err := displayStatusAfterSync(gitClient, githubClient, prCache); err != nil {
		// Don't fail if we can't display status, just warn
		fmt.Fprintf(os.Stderr, "Warning: failed to display stack status: %v\n", err)
	}

	// Mark as successful so defer doesn't restore stash
	success = true

	// Restore stashed changes before success message
	if stashed {
		fmt.Println()
		fmt.Println("Restoring stashed changes...")
		if err := gitClient.StashPop(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore stashed changes: %v\n", err)
			fmt.Fprintf(os.Stderr, "Run 'git stash pop' manually to restore your changes\n")
		}
	}

	// Clean up sync state (both stash flag and original branch)
	_ = gitClient.UnsetConfig(configSyncStashed)
	_ = gitClient.UnsetConfig(configSyncOriginalBranch)

	fmt.Println()
	fmt.Println("✓ Sync complete!")

	return nil
}

// displayStatusAfterSync shows the stack tree after a successful sync
// It reuses the prCache from earlier to avoid a redundant API call
func displayStatusAfterSync(gitClient git.GitClient, githubClient github.GitHubClient, prCache map[string]*github.PRInfo) error {
	currentBranch, err := gitClient.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	tree, err := stack.BuildStackTreeForBranch(gitClient, currentBranch)
	if err != nil {
		return fmt.Errorf("failed to build stack tree: %w", err)
	}

	// Filter out branches with merged PRs (leaf nodes only)
	tree = filterMergedBranchesForSync(tree, prCache)

	// Print the tree
	printTreeForSync(gitClient, tree, currentBranch, prCache)

	return nil
}

// filterMergedBranchesForSync removes branches with merged PRs from the tree,
// but only if they don't have children (to keep the stack structure visible)
func filterMergedBranchesForSync(node *stack.TreeNode, prCache map[string]*github.PRInfo) *stack.TreeNode {
	if node == nil {
		return nil
	}

	// Filter children recursively first
	var filteredChildren []*stack.TreeNode
	for _, child := range node.Children {
		// Recurse first to process all descendants
		filtered := filterMergedBranchesForSync(child, prCache)

		// Only filter out merged branches if they have no children
		if pr, exists := prCache[child.Name]; exists && pr.State == "MERGED" {
			// If this merged branch still has children after filtering, keep it
			if filtered != nil && len(filtered.Children) > 0 {
				filteredChildren = append(filteredChildren, filtered)
			}
			// Otherwise skip this merged leaf branch
		} else {
			// Not merged, keep it
			if filtered != nil {
				filteredChildren = append(filteredChildren, filtered)
			}
		}
	}

	node.Children = filteredChildren
	return node
}

// printTreeForSync prints the stack tree after sync
func printTreeForSync(gitClient git.GitClient, node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo) {
	if node == nil {
		return
	}
	printTreeVerticalForSync(gitClient, node, currentBranch, prCache, false)
}

func printTreeVerticalForSync(gitClient git.GitClient, node *stack.TreeNode, currentBranch string, prCache map[string]*github.PRInfo, isPipe bool) {
	if node == nil {
		return
	}

	// Determine the current branch marker
	marker := ""
	if node.Name == currentBranch {
		marker = " *"
	}

	// Get PR info from cache
	prInfo := ""
	if node.Name != stack.GetBaseBranch(gitClient) {
		if pr, exists := prCache[node.Name]; exists {
			prInfo = fmt.Sprintf(" [%s :%s]", pr.URL, strings.ToLower(pr.State))
		}
	}

	// Print pipe if needed
	if isPipe {
		fmt.Println("  |")
	}

	// Print current node
	fmt.Printf(" %s%s%s\n", node.Name, prInfo, marker)

	// Print children vertically
	for _, child := range node.Children {
		printTreeVerticalForSync(gitClient, child, currentBranch, prCache, true)
	}
}
