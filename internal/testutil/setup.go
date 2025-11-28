package testutil

import (
	"github.com/javoire/stackinator/internal/spinner"
)

// SetupTest initializes test environment (disable spinners, etc.)
func SetupTest() {
	spinner.Enabled = false
}

// TeardownTest cleans up after tests
func TeardownTest() {
	// Currently no cleanup needed, but keeping for future use
}

