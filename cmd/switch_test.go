package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/javoire/stackinator/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRunSwitch(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	t.Run("resolves worktree branch to path", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)

		mockGit.On("GetConfig", "stack.baseBranch").Return("")
		mockGit.On("GetDefaultBranch").Return("main")
		mockGit.On("GetWorktreeBranches").Return(map[string]string{
			"feature-a": "/home/user/.stack/worktrees/repo/feature-a",
			"feature-b": "/home/user/.stack/worktrees/repo/feature-b",
		}, nil)

		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runSwitch(mockGit, "feature-a")

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.NoError(t, err)
		assert.Equal(t, "cd '/home/user/.stack/worktrees/repo/feature-a'\n", buf.String())
		mockGit.AssertExpectations(t)
	})

	t.Run("resolves base branch to repo root", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)

		mockGit.On("GetConfig", "stack.baseBranch").Return("")
		mockGit.On("GetDefaultBranch").Return("main")
		mockGit.On("GetRepoRoot").Return("/home/user/code/repo", nil)

		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runSwitch(mockGit, "main")

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.NoError(t, err)
		assert.Equal(t, "cd '/home/user/code/repo'\n", buf.String())
		mockGit.AssertExpectations(t)
	})

	t.Run("errors for unknown branch", func(t *testing.T) {
		mockGit := new(testutil.MockGitClient)

		mockGit.On("GetConfig", "stack.baseBranch").Return("")
		mockGit.On("GetDefaultBranch").Return("main")
		mockGit.On("GetWorktreeBranches").Return(map[string]string{}, nil)

		err := runSwitch(mockGit, "nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no worktree found")
		mockGit.AssertExpectations(t)
	})
}

func TestRunSwitchInit(t *testing.T) {
	testutil.SetupTest()
	defer testutil.TeardownTest()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runSwitchInit()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	assert.Contains(t, output, "ss()")
	assert.Contains(t, output, "command stack switch")
	assert.Contains(t, output, "eval")
}
