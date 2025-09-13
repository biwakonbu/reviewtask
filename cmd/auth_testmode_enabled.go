//go:build testmode

package cmd

import "os"

// isTestModeImpl returns true only in test builds when the environment variable is set.
// This provides compile-time safety - test mode is only available in builds
// compiled with the 'testmode' build tag.
func isTestModeImpl() bool {
	return os.Getenv("REVIEWTASK_TEST_MODE") == "true"
}
