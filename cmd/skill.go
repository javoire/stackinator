package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/javoire/stackinator/internal/ui"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Claude Code skills",
	Long:  `Manage Claude Code skills for the stack CLI.`,
	Annotations: map[string]string{
		"skipGitValidation": "true",
	},
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the stack skill for Claude Code",
	Long: `Install the stack skill so Claude Code knows how to use the stack CLI.

This requires the Claude Code CLI (claude) to be installed.
See https://claude.ai/code for installation instructions.`,
	Annotations: map[string]string{
		"skipGitValidation": "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSkillInstall()
	},
}

func init() {
	skillCmd.AddCommand(skillInstallCmd)
	rootCmd.AddCommand(skillCmd)
}

func runSkillInstall() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Install it from https://claude.ai/code")
	}

	fmt.Println("Adding stackinator marketplace...")
	addCmd := exec.Command("claude", "plugin", "marketplace", "add", "javoire/stackinator")
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add marketplace: %w", err)
	}

	fmt.Println("Installing stack skill...")
	installCmd := exec.Command("claude", "plugin", "install", "stack@stackinator")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install skill: %w", err)
	}

	fmt.Println(ui.Success("Stack skill installed! Claude Code now knows how to use the stack CLI."))
	return nil
}
