package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// CreateExecutable creates a mock executable file with proper permissions for the current platform
func CreateExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()

	// Add .exe extension on Windows
	if runtime.GOOS == "windows" {
		if filepath.Ext(name) == "" {
			name += ".exe"
		}
	}

	path := filepath.Join(dir, name)

	// Write the file
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create executable: %v", err)
	}

	// On Unix, ensure executable permissions
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0755); err != nil {
			t.Fatalf("Failed to set executable permissions: %v", err)
		}
	}

	return path
}

// CreateMockClaude creates a mock Claude CLI for testing
func CreateMockClaude(t *testing.T, dir string, response string) string {
	t.Helper()

	var content string
	if runtime.GOOS == "windows" {
		// Create a batch file for Windows
		content = "@echo off\necho " + response
		return CreateExecutable(t, dir, "claude.cmd", content)
	} else {
		// Create a shell script for Unix
		content = "#!/bin/sh\necho '" + response + "'"
		return CreateExecutable(t, dir, "claude", content)
	}
}

// NormalizePath normalizes file paths for comparison across platforms
func NormalizePath(path string) string {
	return filepath.Clean(filepath.ToSlash(path))
}

// CreateTestDir creates a temporary test directory with proper cleanup
func CreateTestDir(t *testing.T, name string) string {
	t.Helper()

	dir, err := os.MkdirTemp("", name)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// CreateTestFile creates a test file with the given content
func CreateTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return path
}

// SetReadOnly sets a file or directory to read-only mode
func SetReadOnly(t *testing.T, path string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		// On Windows, use os.Chmod directly as attrib might not be available in CI
		// Windows permission model is different, but os.Chmod can still make files read-only
		if err := os.Chmod(path, 0444); err != nil {
			// On Windows, sometimes we need to handle directories differently
			// Try with directory-specific permissions
			if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
				// For directories on Windows, try a different permission set
				if err := os.Chmod(path, 0555); err != nil {
					t.Fatalf("Failed to set directory read-only on Windows: %v", err)
				}
			} else {
				t.Fatalf("Failed to set read-only on Windows: %v", err)
			}
		}
	} else {
		// On Unix, use chmod
		if err := os.Chmod(path, 0444); err != nil {
			t.Fatalf("Failed to set read-only: %v", err)
		}
	}
}

// SetWritable sets a file or directory to writable mode
func SetWritable(t *testing.T, path string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		// On Windows, use os.Chmod directly as attrib might not be available in CI
		// Check if it's a directory to apply appropriate permissions
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Failed to stat path: %v", err)
		}

		if info.IsDir() {
			// For directories, need execute permission too
			if err := os.Chmod(path, 0755); err != nil {
				t.Fatalf("Failed to set directory writable on Windows: %v", err)
			}
		} else {
			// For files
			if err := os.Chmod(path, 0644); err != nil {
				t.Fatalf("Failed to set file writable on Windows: %v", err)
			}
		}
	} else {
		// On Unix, use chmod
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatalf("Failed to set writable: %v", err)
		}
	}
}

// GetExecutableName returns the platform-specific executable name
func GetExecutableName(name string) string {
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		return name + ".exe"
	}
	return name
}

// GetScriptExtension returns the platform-specific script extension
func GetScriptExtension() string {
	if runtime.GOOS == "windows" {
		return ".cmd"
	}
	return ""
}

// IsExecutable checks if a file is executable on the current platform
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, check file extension
		ext := filepath.Ext(path)
		return ext == ".exe" || ext == ".cmd" || ext == ".bat"
	}

	// On Unix, check executable bit
	return info.Mode()&0111 != 0
}

// SkipIfWindows skips a test on Windows
func SkipIfWindows(t *testing.T, reason string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows: " + reason)
	}
}

// SkipIfNotWindows skips a test on non-Windows platforms
func SkipIfNotWindows(t *testing.T, reason string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("Skipping on non-Windows: " + reason)
	}
}

// AssertFileExists asserts that a file exists
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("File does not exist: %s", path)
	}
}

// AssertFileNotExists asserts that a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("File should not exist: %s", path)
	}
}
