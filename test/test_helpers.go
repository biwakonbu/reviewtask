package test

import (
	"path/filepath"
	"runtime"
)

// findProjectRoot returns the project root directory based on the test file location
func findProjectRoot() (string, error) {
	// Get the directory of the current test file
	_, filename, _, _ := runtime.Caller(1)
	testDir := filepath.Dir(filename)
	// Project root is one level up from the test directory
	return filepath.Dir(testDir), nil
}
