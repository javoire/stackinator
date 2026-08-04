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
	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "codex.log")
	t.Setenv("CODEX_TEST_LOG", logPath)
	t.Setenv("PATH", binDir)

	codexPath := filepath.Join(binDir, "codex")
	require.NoError(t, os.WriteFile(codexPath, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$CODEX_TEST_LOG\"\n"), 0755))

	err := installCodex()
	require.NoError(t, err)

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "plugin marketplace add javoire/stackinator\nplugin add stack@stackinator\n", string(content))
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
	assert.Contains(t, string(content), "## Inspect and navigate")
	// Should not contain SKILL.md frontmatter
	assert.NotContains(t, string(content), "name: stack")
}

func TestSkillBody(t *testing.T) {
	body := skillBody()
	assert.False(t, strings.HasPrefix(body, "---"))
	assert.Contains(t, body, "## Inspect and navigate")
}

func TestSkillDescription(t *testing.T) {
	description := skillDescription()
	assert.Contains(t, description, "Manage stacked Git branches")
	assert.NotContains(t, description, "description:")
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
