package cmd

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomWorktreeName(t *testing.T) {
	first, err := generateRandomWorktreeName()
	require.NoError(t, err)
	second, err := generateRandomWorktreeName()
	require.NoError(t, err)

	assert.Regexp(t, regexp.MustCompile(`^worktree-[0-9a-f]{16}$`), first)
	assert.Regexp(t, regexp.MustCompile(`^worktree-[0-9a-f]{16}$`), second)
	assert.NotEqual(t, first, second)
}

func TestWorktreeArgs(t *testing.T) {
	originalList := worktreeList
	originalPrune := worktreePrune
	t.Cleanup(func() {
		worktreeList = originalList
		worktreePrune = originalPrune
	})

	worktreeList = false
	worktreePrune = false

	assert.NoError(t, worktreeCmd.Args(worktreeCmd, nil))
	assert.NoError(t, worktreeCmd.Args(worktreeCmd, []string{"feature"}))
	assert.NoError(t, worktreeCmd.Args(worktreeCmd, []string{"feature", "main"}))
	assert.Error(t, worktreeCmd.Args(worktreeCmd, []string{"feature", "main", "extra"}))
}
