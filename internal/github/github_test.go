package github

import (
	"testing"

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
	client := NewGitHubClient()
	assert.NotNil(t, client)
}

// Note: More comprehensive tests would require mocking exec.Command or running actual gh CLI commands
// For unit tests focused on critical path, we rely on integration tests or testutil mocks

