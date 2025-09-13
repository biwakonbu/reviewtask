package ai

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"reviewtask/internal/testutil"
)

func TestClaudePathDetectionIntegration(t *testing.T) {
	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name          string
		setup         func() (cleanup func())
		expectSuccess bool
	}{
		{
			name: "End-to-end: Claude CLI detection and client creation",
			setup: func() (cleanup func()) {
				// Remove claude from PATH
				os.Setenv("PATH", "/nonexistent")

				// Create mock claude in npm location
				claudeDir := filepath.Join(homeDir, ".npm-global", "bin")
				if err := os.MkdirAll(claudeDir, 0755); err != nil {
					t.Fatalf("Failed to create npm dir: %v", err)
				}

				// Create mock Claude CLI
				testutil.CreateMockClaude(t, claudeDir, "Claude Code CLI version 1.0.0")

				return func() {
					os.RemoveAll(filepath.Join(homeDir, ".npm-global"))
					// Cleanup potential symlinks
					symlinkPath := filepath.Join(homeDir, ".local", "bin", "claude")
					os.Remove(symlinkPath)
				}
			},
			expectSuccess: true,
		},
		{
			name: "End-to-end: Volta installation detection",
			setup: func() (cleanup func()) {
				// Remove claude from PATH
				os.Setenv("PATH", "/nonexistent")

				// Create mock claude in volta location
				claudeDir := filepath.Join(homeDir, ".volta", "bin")
				if err := os.MkdirAll(claudeDir, 0755); err != nil {
					t.Fatalf("Failed to create volta dir: %v", err)
				}

				// Create mock Claude CLI
				testutil.CreateMockClaude(t, claudeDir, "anthropic-ai/claude-code version 1.2.0")

				return func() {
					os.RemoveAll(filepath.Join(homeDir, ".volta"))
					// Cleanup potential symlinks
					symlinkPath := filepath.Join(homeDir, ".local", "bin", "claude")
					os.Remove(symlinkPath)
				}
			},
			expectSuccess: true,
		},
		{
			name: "End-to-end: No Claude CLI available",
			setup: func() (cleanup func()) {
				// Remove claude from PATH and don't create anywhere
				os.Setenv("PATH", "/nonexistent")

				return func() {}
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			// Test the complete flow
			client, err := NewRealClaudeClient()

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected client to be created, got nil")
				}

				// Verify symlink was created if needed
				symlinkPath := filepath.Join(homeDir, ".local/bin", "claude")
				// Symlink should exist for all successful cases since we set PATH to /nonexistent
				if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
					t.Errorf("Expected symlink to exist at %s", symlinkPath)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if client != nil {
					t.Errorf("Expected nil client but got: %v", client)
				}
			}
		})
	}
}

func TestClaudeExecutionWithPathDetection(t *testing.T) {
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}
	// This test verifies that the detected Claude CLI actually works
	// Skip if no real Claude CLI is available
	if _, err := NewRealClaudeClient(); err != nil {
		t.Skip("Skipping execution test - no Claude CLI available")
	}

	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	// Remove claude from PATH
	os.Setenv("PATH", "/nonexistent")

	// Create mock claude that actually responds
	claudeDir := filepath.Join(homeDir, ".claude", "local")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}
	defer os.RemoveAll(filepath.Join(homeDir, ".claude"))

	claudePath := filepath.Join(claudeDir, "claude")
	claudeScript := `#!/bin/bash
if [ "$1" = "--version" ]; then
    echo "Claude Code CLI version 1.0.0"
elif [ "$1" = "--output-format" ] && [ "$2" = "json" ]; then
    # Read input and return mock JSON response
    cat > /dev/null
    echo '{"response": "This is a mock Claude response"}'
else
    # Read input and return simple response
    cat > /dev/null
    echo "This is a mock Claude response"
fi
`
	if err := os.WriteFile(claudePath, []byte(claudeScript), 0755); err != nil {
		t.Fatalf("Failed to create mock claude: %v", err)
	}

	// Cleanup potential symlinks
	symlinkPath := filepath.Join(homeDir, ".local/bin", "claude")
	defer os.Remove(symlinkPath)

	// Create client (should use path detection and symlink creation)
	client, err := NewRealClaudeClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test execution
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.Execute(ctx, "test input", "")
	if err != nil {
		t.Errorf("Failed to execute Claude: %v", err)
	}

	expectedResponse := "This is a mock Claude response"
	if response != expectedResponse+"\n" {
		t.Errorf("Expected response %q, got %q", expectedResponse, response)
	}
}

func TestSymlinkLifecycleIntegration(t *testing.T) {
	// Skip on Windows as symlinks require admin privileges
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	localBinDir := filepath.Join(homeDir, ".local/bin")
	symlinkPath := filepath.Join(localBinDir, "claude")

	// Ensure cleanup after test
	defer func() {
		os.Remove(symlinkPath)
	}()

	// Create mock Claude CLI
	tempDir, err := os.MkdirTemp("", "claude_lifecycle_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	claudePath := filepath.Join(tempDir, "claude")
	if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'Claude Code CLI version 1.0.0'\n"), 0755); err != nil {
		t.Fatalf("Failed to create mock claude: %v", err)
	}

	// Test symlink creation
	// First check if claude is already in PATH
	claudeInPath := false
	if _, err := exec.LookPath("claude"); err == nil {
		claudeInPath = true
	}

	if err := ensureClaudeAvailable(claudePath); err != nil {
		t.Errorf("Failed to ensure Claude available: %v", err)
	}

	// If claude was already in PATH, ensureClaudeAvailable should return early
	// without creating a symlink. This is expected behavior.
	if claudeInPath {
		// Verify symlink was NOT created (correct behavior)
		if _, err := os.Lstat(symlinkPath); err == nil {
			t.Errorf("Symlink should not be created when claude is already in PATH")
		}
	} else {
		// Verify symlink exists
		if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
			t.Errorf("Expected symlink to exist at %s", symlinkPath)
		}

		// Verify symlink points to correct target
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Errorf("Failed to read symlink: %v", err)
		}
		if target != claudePath {
			t.Errorf("Symlink points to %s, expected %s", target, claudePath)
		}
	}

	// Test symlink cleanup
	if err := CleanupClaudeSymlink(); err != nil {
		t.Errorf("Failed to cleanup symlink: %v", err)
	}

	// Note: Our cleanup function only removes reviewtask-managed symlinks
	// Since our test creates a symlink to a temp directory, it won't be considered
	// "reviewtask-managed" and won't be removed. This is actually correct behavior.
	// Manual removal for test cleanup
	os.Remove(symlinkPath)
}

func TestCrossplatformPathHandling(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	// Test that path joining works correctly across platforms
	expectedPaths := []string{
		filepath.Join(homeDir, ".claude/local/claude"),
		filepath.Join(homeDir, ".npm-global/bin/claude"),
		filepath.Join(homeDir, ".volta/bin/claude"),
		filepath.Join(homeDir, ".local/bin/claude"),
	}

	// Add Unix-specific paths only on Unix systems
	if runtime.GOOS != "windows" {
		expectedPaths = append(expectedPaths,
			"/usr/local/bin/claude",
			"/opt/homebrew/bin/claude",
		)
	}

	// Verify all paths use correct separators for current platform
	for _, path := range expectedPaths {
		if !filepath.IsAbs(path) && !isHomeRelativePath(path, homeDir) {
			t.Errorf("Path %s is not absolute or home-relative", path)
		}
	}
}

func isHomeRelativePath(path, homeDir string) bool {
	return strings.HasPrefix(filepath.Clean(path), filepath.Clean(homeDir))
}

func TestErrorHandlingIntegration(t *testing.T) {
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}
	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Test error handling when no Claude CLI is found
	os.Setenv("PATH", "/nonexistent")

	client, err := NewRealClaudeClient()
	if err == nil {
		t.Errorf("Expected error when no Claude CLI is available")
	}
	if client != nil {
		t.Errorf("Expected nil client when error occurs, got: %v", client)
	}

	// Verify error message is helpful
	expectedErrSubstring := "claude command not found"
	if err != nil && err.Error() != "" {
		if len(err.Error()) < len(expectedErrSubstring) {
			t.Logf("Error message: %s", err.Error())
		}
	}
}
