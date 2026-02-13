package cmd

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunSkillInstall_ClaudeNotFound(t *testing.T) {
	// Set PATH to empty so claude can't be found
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() { os.Setenv("PATH", originalPath) }()

	// Clear the exec.LookPath cache by ensuring fresh lookup
	err := runSkillInstall()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude CLI not found")
}

func TestRunSkillInstall_ClaudeFound(t *testing.T) {
	// Skip if claude is not installed
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not installed, skipping integration test")
	}

	// This would actually run the commands, so we just verify claude is found
	// A full integration test would require mocking exec.Command
	t.Log("claude CLI found in PATH")
}
