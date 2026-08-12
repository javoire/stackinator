package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPRInfoParsing(t *testing.T) {
	// Test that PRInfo struct is properly defined
	pr := &PRInfo{
		Number:           123,
		State:            "OPEN",
		Base:             "main",
		Title:            "Test PR",
		URL:              "https://github.com/test/repo/pull/123",
		MergeStateStatus: "CLEAN",
	}

	assert.Equal(t, 123, pr.Number)
	assert.Equal(t, "OPEN", pr.State)
	assert.Equal(t, "main", pr.Base)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "https://github.com/test/repo/pull/123", pr.URL)
	assert.Equal(t, "CLEAN", pr.MergeStateStatus)
}

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient("owner/repo")
	assert.NotNil(t, client)
}

func TestParseRepoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH format",
			url:      "git@github.com:javoire/stackinator.git",
			expected: "javoire/stackinator",
		},
		{
			name:     "HTTPS format",
			url:      "https://github.com/javoire/stackinator.git",
			expected: "javoire/stackinator",
		},
		{
			name:     "HTTPS without .git",
			url:      "https://github.com/javoire/stackinator",
			expected: "javoire/stackinator",
		},
		{
			name:     "GHE SSH format",
			url:      "git@ghe.spotify.net:some-org/some-repo.git",
			expected: "ghe.spotify.net/some-org/some-repo",
		},
		{
			name:     "GHE HTTPS format",
			url:      "https://ghe.spotify.net/some-org/some-repo",
			expected: "ghe.spotify.net/some-org/some-repo",
		},
		{
			name:     "empty string",
			url:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRepoFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPRsForBranchesTimesOut(t *testing.T) {
	tempDir := t.TempDir()
	ghPath := filepath.Join(tempDir, "gh")
	assert.NoError(t, os.WriteFile(ghPath, []byte("#!/bin/sh\nexec sleep 10\n"), 0o755))
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	previousTimeout := commandTimeout
	commandTimeout = 20 * time.Millisecond
	t.Cleanup(func() { commandTimeout = previousTimeout })

	prs, err := NewGitHubClient("ghe.spotify.net/org/repo").GetPRsForBranches([]string{"feature"})
	assert.Empty(t, prs)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to load PR for feature"), err.Error())
	assert.True(t, strings.Contains(err.Error(), "timed out after 20ms"), err.Error())
}

func TestGetPRForBranchTreatsMissingPRAsAbsent(t *testing.T) {
	tempDir := t.TempDir()
	ghPath := filepath.Join(tempDir, "gh")
	script := "#!/bin/sh\necho 'no pull requests found for branch feature' >&2\nexit 1\n"
	assert.NoError(t, os.WriteFile(ghPath, []byte(script), 0o755))
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	pr, err := NewGitHubClient("owner/repo").GetPRForBranch("feature")
	assert.NoError(t, err)
	assert.Nil(t, pr)
}

// Note: More comprehensive tests would require mocking exec.Command or running actual gh CLI commands
// For unit tests focused on critical path, we rely on integration tests or testutil mocks
