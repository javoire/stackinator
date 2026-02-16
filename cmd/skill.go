package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/javoire/stackinator/internal/ui"
	skillcontent "github.com/javoire/stackinator/plugins/stack/skills/stack"
	"github.com/spf13/cobra"
)

type aiTool struct {
	Name    string
	Detect  func() bool
	Install func() error
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage AI coding tool skills",
	Long:  `Manage AI coding tool skills for the stack CLI.`,
	Annotations: map[string]string{
		"skipGitValidation": "true",
	},
}

var skillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the stack skill for AI coding tools",
	Long: `Install the stack skill so AI coding tools know how to use the stack CLI.

Supported tools: Claude Code, Codex, Cursor.
Automatically detects which tools are installed and installs for all of them.`,
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

func getAITools() []aiTool {
	return []aiTool{
		{Name: "Claude Code", Detect: detectClaude, Install: installClaude},
		{Name: "Codex", Detect: detectCodex, Install: installCodex},
		{Name: "Cursor", Detect: detectCursor, Install: installCursor},
	}
}

func detectClaude() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func detectCodex() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func detectCursor() bool {
	if _, err := exec.LookPath("cursor"); err == nil {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".cursor"))
	return err == nil
}

// skillBody returns the SKILL.md content with YAML frontmatter stripped.
func skillBody() string {
	content := skillcontent.SkillMD
	if strings.HasPrefix(content, "---") {
		if idx := strings.Index(content[3:], "---"); idx != -1 {
			content = strings.TrimLeft(content[3+idx+3:], "\n")
		}
	}
	return content
}

func installClaude() error {
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
	return nil
}

func installCodex() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(home, ".agents", "skills", "stack")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	dest := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(dest, []byte(skillcontent.SkillMD), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dest, err)
	}

	fmt.Printf("Wrote %s\n", dest)
	return nil
}

func installCursor() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(home, ".cursor", "rules")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	body := skillBody()
	mdc := "---\ndescription: Manage stacked branches with the stack CLI. Covers branch creation, navigation, syncing, and PR management.\nalwaysApply: true\n---\n\n" + body

	dest := filepath.Join(dir, "stack.mdc")
	if err := os.WriteFile(dest, []byte(mdc), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", dest, err)
	}

	fmt.Printf("Wrote %s\n", dest)
	return nil
}

func runSkillInstall() error {
	tools := getAITools()

	var detected []aiTool
	for _, t := range tools {
		if t.Detect() {
			detected = append(detected, t)
		}
	}

	if len(detected) == 0 {
		return fmt.Errorf("no supported AI coding tools found. Supported tools: Claude Code (claude), Codex (codex), Cursor (cursor or ~/.cursor/)")
	}

	var installed []string
	for _, t := range detected {
		fmt.Printf("Installing for %s...\n", t.Name)
		if err := t.Install(); err != nil {
			return fmt.Errorf("failed to install for %s: %w", t.Name, err)
		}
		installed = append(installed, t.Name)
	}

	fmt.Println(ui.Success(fmt.Sprintf("Stack skill installed for: %s", strings.Join(installed, ", "))))
	return nil
}
