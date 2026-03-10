package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/stack"
	"github.com/spf13/cobra"
)

var switchInit bool
var switchInstall bool

var switchCmd = &cobra.Command{
	Use:   "switch [branch]",
	Short: "Print cd command to switch to a branch's worktree",
	Long: `Print a cd command to switch to the worktree for a given branch.

With no arguments, shows an interactive picker of all worktrees.
Use --init to output a shell function that wraps this command with actual cd.
Use --install to add the shell function to your shell config.`,
	Example: `  # Switch to a branch's worktree
  eval "$(stack switch my-feature)"

  # Interactive picker
  eval "$(stack switch)"

  # Output shell function
  stack switch --init

  # Install shell function to ~/.zshrc
  stack switch --install`,
	Annotations: map[string]string{},
	Args:        cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if switchInit {
			runSwitchInit()
			return
		}

		if switchInstall {
			if err := runSwitchInstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		gitClient := git.NewGitClient()

		var err error
		if len(args) == 1 {
			err = runSwitch(gitClient, args[0])
		} else {
			err = runSwitchInteractive(gitClient)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	switchCmd.Flags().BoolVar(&switchInit, "init", false, "Output shell function for wrapping switch with cd")
	switchCmd.Flags().BoolVar(&switchInstall, "install", false, "Add shell function to shell config")

	// Skip git validation for --init and --install
	switchCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Run root's PersistentPreRun only if we need git
		if !switchInit && !switchInstall {
			rootCmd.PersistentPreRun(cmd, args)
		}
	}
}

func runSwitchInit() {
	fmt.Print(`ss() {
  local dir
  dir=$(command stack switch "$@" 2>/dev/null)
  if [ $? -eq 0 ] && [ -n "$dir" ]; then
    eval "$dir"
  else
    command stack switch "$@"
  fi
}
`)
}

func runSwitch(gitClient git.GitClient, branchName string) error {
	path, err := resolveWorktreePath(gitClient, branchName)
	if err != nil {
		return err
	}
	fmt.Printf("cd %s\n", path)
	return nil
}

func resolveWorktreePath(gitClient git.GitClient, branchName string) (string, error) {
	// Check if it's the base branch
	baseBranch := stack.GetBaseBranch(gitClient)
	if branchName == baseBranch {
		repoRoot, err := gitClient.GetRepoRoot()
		if err != nil {
			return "", fmt.Errorf("failed to get repo root: %w", err)
		}
		return repoRoot, nil
	}

	// Look up in worktree branches
	worktreeBranches, err := gitClient.GetWorktreeBranches()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree branches: %w", err)
	}

	if path, ok := worktreeBranches[branchName]; ok {
		return path, nil
	}

	return "", fmt.Errorf("no worktree found for branch %s", branchName)
}

func runSwitchInteractive(gitClient git.GitClient) error {
	worktreesBaseDir, err := getWorktreesBaseDir(gitClient)
	if err != nil {
		return err
	}

	repoName, err := gitClient.GetRepoName()
	if err != nil {
		return fmt.Errorf("failed to get repo name: %w", err)
	}

	worktreesDir := filepath.Join(worktreesBaseDir, repoName)

	// Get worktree branches
	worktreeBranches, err := gitClient.GetWorktreeBranches()
	if err != nil {
		return fmt.Errorf("failed to get worktree branches: %w", err)
	}

	// Get current worktree path for marking
	currentPath, _ := gitClient.GetCurrentWorktreePath()

	// Collect options: worktrees in the worktrees dir + main repo
	type option struct {
		branch string
		path   string
	}
	var options []option

	// Add main repo
	baseBranch := stack.GetBaseBranch(gitClient)
	repoRoot, err := gitClient.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repo root: %w", err)
	}
	options = append(options, option{branch: baseBranch, path: repoRoot})

	// Add worktrees filtered to this repo's worktrees dir
	var sortedBranches []string
	for branch, path := range worktreeBranches {
		if pathWithinDir(path, worktreesDir) {
			sortedBranches = append(sortedBranches, branch)
		}
	}
	sort.Strings(sortedBranches)
	for _, branch := range sortedBranches {
		options = append(options, option{branch: branch, path: worktreeBranches[branch]})
	}

	if len(options) <= 1 {
		return fmt.Errorf("no worktrees found")
	}

	// Display options
	fmt.Fprintf(os.Stderr, "Select a worktree:\n\n")
	for i, opt := range options {
		marker := " "
		if opt.path == currentPath {
			marker = "*"
		}
		fmt.Fprintf(os.Stderr, "  %s %d) %s\n", marker, i+1, opt.branch)
		fmt.Fprintf(os.Stderr, "       %s\n", opt.path)
	}
	fmt.Fprintf(os.Stderr, "\n> ")

	// Read selection
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(options) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	selected := options[idx-1]
	fmt.Printf("cd %s\n", selected.path)
	return nil
}

func runSwitchInstall() error {
	// Detect shell
	shell := os.Getenv("SHELL")
	var rcFile string
	switch {
	case strings.HasSuffix(shell, "/bash"):
		homeDir, err := getHomeDir()
		if err != nil {
			return err
		}
		rcFile = filepath.Join(homeDir, ".bashrc")
	default:
		// Default to zsh
		homeDir, err := getHomeDir()
		if err != nil {
			return err
		}
		rcFile = filepath.Join(homeDir, ".zshrc")
	}

	// Read existing file
	content, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", rcFile, err)
	}

	// Check if already installed
	initLine := `eval "$(stack switch --init)"`
	if strings.Contains(string(content), "stack switch --init") {
		fmt.Fprintf(os.Stderr, "Already installed in %s\n", rcFile)
		return nil
	}

	// Append
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", rcFile, err)
	}
	defer f.Close()

	entry := fmt.Sprintf("\n# stackinator: ss() function for quick worktree switching\n%s\n", initLine)
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to %s: %w", rcFile, err)
	}

	fmt.Fprintf(os.Stderr, "Added ss() function to %s\n", rcFile)
	fmt.Fprintf(os.Stderr, "Run 'source %s' or start a new shell to use it.\n", rcFile)
	return nil
}
