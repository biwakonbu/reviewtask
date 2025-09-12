//go:build !testmode

package cmd

// isTestModeImpl returns false in production builds.
// The test mode bypass is completely disabled in production builds
// for security reasons.
func isTestModeImpl() bool {
	return false
}