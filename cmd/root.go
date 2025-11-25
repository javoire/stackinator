package cmd

import (
	"fmt"
	"os"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/github"
	"github.com/javoire/stackinator/internal/spinner"
	"github.com/spf13/cobra"
)

var (
	dryRun  bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage stacked branches and sync them to GitHub PRs",
	Long: `A CLI tool for managing stacks of branches and syncing them to GitHub Pull Requests.

Stack branches are tracked using git config, where each branch stores its parent.
The tool helps you create, navigate, and sync stacked branches with minimal overhead.`,
	Example: `  # Create a new feature branch
  stack new feature-auth

  # View your stack structure
  stack status

  # Sync all branches and update PRs
  stack sync

  # Preview sync without executing
  stack sync --dry-run

  # See detailed output
  stack sync --verbose`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set global flags
		git.DryRun = dryRun
		git.Verbose = verbose
		github.DryRun = dryRun
		github.Verbose = verbose

		// Disable spinners in verbose mode to avoid visual conflicts
		spinner.Enabled = !verbose

		// Validate we're in a git repository
		if _, err := git.GetRepoRoot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: not in a git repository\n")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without executing")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")

	// Add subcommands
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(parentCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(reparentCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
