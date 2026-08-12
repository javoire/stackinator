package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGitClient(t *testing.T) {
	client := NewGitClient()
	assert.NotNil(t, client)
}

func TestGitClientInterface(t *testing.T) {
	// Verify that gitClient implements GitClient interface
	var _ GitClient = &gitClient{}
}

func TestFetchRemoteTimesOut(t *testing.T) {
	tempDir := t.TempDir()
	gitPath := filepath.Join(tempDir, "git")
	assert.NoError(t, os.WriteFile(gitPath, []byte("#!/bin/sh\nexec sleep 10\n"), 0o755))
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	previousTimeout := fetchTimeout
	fetchTimeout = 20 * time.Millisecond
	t.Cleanup(func() { fetchTimeout = previousTimeout })

	err := NewGitClient().FetchRemote("origin")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "git fetch origin timed out"), err.Error())
}

// Note: More comprehensive tests would require mocking exec.Command or running actual git commands
// For unit tests focused on critical path, we rely on integration tests or testutil mocks
// The real value is in testing the stack package and command packages with mocked clients
