package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSkillInstall_NoToolsFound(t *testing.T) {
	t.Setenv("PATH", "")
	t.Setenv("HOME", t.TempDir())

	err := runSkillInstall()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no supported AI coding tools found")
}

func TestInstallCodex(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := installCodex()
	require.NoError(t, err)

	dest := filepath.Join(home, ".agents", "skills", "stack", "SKILL.md")
	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Contains(t, string(content), "name: stack")
	assert.Contains(t, string(content), "## Common Commands")
}

func TestInstallCursor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := installCursor()
	require.NoError(t, err)

	dest := filepath.Join(home, ".cursor", "rules", "stack.mdc")
	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Contains(t, string(content), "alwaysApply: true")
	assert.Contains(t, string(content), "description:")
	assert.Contains(t, string(content), "## Common Commands")
	// Should not contain SKILL.md frontmatter
	assert.NotContains(t, string(content), "name: stack")
}

func TestSkillBody(t *testing.T) {
	body := skillBody()
	assert.False(t, strings.HasPrefix(body, "---"))
	assert.Contains(t, body, "## Common Commands")
}

func TestDetectCursor_WithDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "")

	require.NoError(t, os.MkdirAll(filepath.Join(home, ".cursor"), 0755))
	assert.True(t, detectCursor())
}

func TestDetectCursor_NothingFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "")

	assert.False(t, detectCursor())
}
