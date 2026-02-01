package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/javoire/stackinator/internal/git"
	"github.com/javoire/stackinator/internal/ui"
	"github.com/spf13/cobra"
)

var configStdin io.Reader = os.Stdin

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show stackinator configuration for this repository",
	Long: `Show stackinator configuration for this repository.

Use 'stack config set' to update settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()
		if err := runConfigShow(gitClient); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show stackinator configuration for this repository",
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()
		if err := runConfigShow(gitClient); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Interactively configure stackinator settings",
	Long: `Interactively configure stackinator settings.

Currently supports the worktrees directory location.`,
	Run: func(cmd *cobra.Command, args []string) {
		gitClient := git.NewGitClient()
		if err := runConfigSet(gitClient); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(gitClient git.GitClient) error {
	configured := strings.TrimSpace(gitClient.GetConfig("stack.worktreesDir"))
	if configured == "" {
		configured = "(default)"
	}

	effective, err := getWorktreesBaseDir(gitClient)
	if err != nil {
		return err
	}

	fmt.Println("Worktrees directory:")
	fmt.Printf("  Configured: %s\n", configured)
	fmt.Printf("  Effective:  %s\n", effective)

	return nil
}

func runConfigSet(gitClient git.GitClient) error {
	defaultDir, err := getDefaultWorktreesBaseDir()
	if err != nil {
		return err
	}

	repoRoot, err := gitClient.GetRepoRoot()
	if err != nil {
		return err
	}

	projectDir := filepath.Join(repoRoot, ".worktrees")

	fmt.Println("Choose worktrees directory:")
	fmt.Printf("  1) %s (default)\n", defaultDir)
	fmt.Printf("  2) %s (this repo)\n", projectDir)
	fmt.Printf("Select [1/2] (default 1): ")

	reader := bufio.NewReader(configStdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(input)
	if choice == "" {
		choice = "1"
	}

	var configValue string
	switch choice {
	case "1":
		configValue = "~/.stack/worktrees"
	case "2":
		configValue = ".worktrees"
	default:
		return fmt.Errorf("invalid selection: %s", choice)
	}

	if err := gitClient.SetConfig("stack.worktreesDir", configValue); err != nil {
		return fmt.Errorf("failed to set worktrees directory: %w", err)
	}

	if !dryRun {
		fmt.Println(ui.Success(fmt.Sprintf("Worktrees directory set to %s", configValue)))
	}

	return nil
}
