package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/javoire/stackinator/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetWorktreesBaseDir(t *testing.T) {
	t.Setenv("HOME", "/home/test")

	tests := []struct {
		name     string
		config   string
		repoRoot string
		repoErr  error
		expected string
	}{
		{
			name:     "default when no config",
			config:   "",
			repoRoot: "/repo",
			expected: "/home/test/.stack/worktrees",
		},
		{
			name:     "tilde expands",
			config:   "~/worktrees",
			repoRoot: "/repo",
			expected: "/home/test/worktrees",
		},
		{
			name:     "env expands",
			config:   "$HOME/custom",
			repoRoot: "/repo",
			expected: "/home/test/custom",
		},
		{
			name:     "relative uses repo root",
			config:   ".worktrees",
			repoRoot: "/repo",
			expected: "/repo/.worktrees",
		},
		{
			name:     "relative falls back to home when repo root missing",
			config:   ".worktrees",
			repoErr:  errors.New("no repo"),
			expected: "/home/test/.worktrees",
		},
		{
			name:     "absolute kept as is",
			config:   "/abs/worktrees",
			repoRoot: "/repo",
			expected: "/abs/worktrees",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &testutil.MockGitClient{}
			mockGit.On("GetRepoRoot").Return(tt.repoRoot, tt.repoErr)
			mockGit.On("GetConfig", "stack.worktreesDir").Return(tt.config)

			dir, err := getWorktreesBaseDir(mockGit)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, dir)
			mockGit.AssertExpectations(t)
		})
	}
}

func TestRunConfigSet(t *testing.T) {
	t.Setenv("HOME", "/home/test")

	tests := []struct {
		name        string
		input       string
		expectValue string
		expectErr   bool
	}{
		{
			name:        "default selection",
			input:       "\n",
			expectValue: "~/.stack/worktrees",
		},
		{
			name:        "select repo local",
			input:       "2\n",
			expectValue: ".worktrees",
		},
		{
			name:      "invalid selection",
			input:     "3\n",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockGit := &testutil.MockGitClient{}
			mockGit.On("GetRepoRoot").Return("/repo", nil)
			if !tt.expectErr {
				mockGit.On("SetConfig", "stack.worktreesDir", tt.expectValue).Return(nil)
			}

			previousStdin := configStdin
			configStdin = strings.NewReader(tt.input)
			defer func() { configStdin = previousStdin }()

			err := runConfigSet(mockGit)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockGit.AssertExpectations(t)
		})
	}
}
