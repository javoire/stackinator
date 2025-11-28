package git

import (
	"testing"

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

// Note: More comprehensive tests would require mocking exec.Command or running actual git commands
// For unit tests focused on critical path, we rely on integration tests or testutil mocks
// The real value is in testing the stack package and command packages with mocked clients

