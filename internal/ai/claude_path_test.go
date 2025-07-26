package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindClaudeCLI(t *testing.T) {
	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	tests := []struct {
		name           string
		pathSetup      func() (cleanup func())
		expectedError  bool
		expectedToFind bool
	}{
		{
			name: "Claude in PATH",
			pathSetup: func() (cleanup func()) {
				// Create temporary directory with mock claude executable
				tempDir, err := os.MkdirTemp("", "claude_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				claudePath := filepath.Join(tempDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'claude version 1.0.0'\n"), 0755); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				// Add to PATH
				os.Setenv("PATH", tempDir+":"+originalPath)

				return func() { os.RemoveAll(tempDir) }
			},
			expectedError:  false,
			expectedToFind: true,
		},
		{
			name: "Claude not in PATH but in common location",
			pathSetup: func() (cleanup func()) {
				// Remove claude from PATH
				os.Setenv("PATH", "/nonexistent")

				// Create mock claude in common location
				homeDir, _ := os.UserHomeDir()
				claudeDir := filepath.Join(homeDir, ".claude", "local")
				if err := os.MkdirAll(claudeDir, 0755); err != nil {
					t.Fatalf("Failed to create claude dir: %v", err)
				}

				claudePath := filepath.Join(claudeDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'claude version 1.0.0'\n"), 0755); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				return func() { os.RemoveAll(filepath.Join(homeDir, ".claude")) }
			},
			expectedError:  false,
			expectedToFind: true,
		},
		{
			name: "Claude not found anywhere",
			pathSetup: func() (cleanup func()) {
				// Remove claude from PATH
				os.Setenv("PATH", "/nonexistent")
				return func() {}
			},
			expectedError:  true,
			expectedToFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.pathSetup()
			defer cleanup()

			path, err := findClaudeCLI()

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.expectedToFind && path == "" {
					t.Errorf("Expected to find claude path but got empty string")
				}
			}
		})
	}
}

func TestIsValidClaudeCLI(t *testing.T) {
	tests := []struct {
		name            string
		setupExecutable func() (path string, cleanup func())
		expectedValid   bool
	}{
		{
			name: "Valid Claude CLI",
			setupExecutable: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "claude_valid_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				claudePath := filepath.Join(tempDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'Claude Code CLI version 1.0.0'\n"), 0755); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				return claudePath, func() { os.RemoveAll(tempDir) }
			},
			expectedValid: true,
		},
		{
			name: "Valid Anthropic CLI",
			setupExecutable: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "claude_anthropic_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				claudePath := filepath.Join(tempDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'anthropic-ai/claude-code version 1.0.0'\n"), 0755); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				return claudePath, func() { os.RemoveAll(tempDir) }
			},
			expectedValid: true,
		},
		{
			name: "Invalid executable",
			setupExecutable: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "claude_invalid_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				claudePath := filepath.Join(tempDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'some other tool version 1.0.0'\n"), 0755); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				return claudePath, func() { os.RemoveAll(tempDir) }
			},
			expectedValid: false,
		},
		{
			name: "Non-executable file",
			setupExecutable: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "claude_nonexec_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				claudePath := filepath.Join(tempDir, "claude")
				if err := os.WriteFile(claudePath, []byte("not executable"), 0644); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				return claudePath, func() { os.RemoveAll(tempDir) }
			},
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setupExecutable()
			defer cleanup()

			isValid := isValidClaudeCLI(path)

			if isValid != tt.expectedValid {
				t.Errorf("Expected isValid=%v, got %v", tt.expectedValid, isValid)
			}
		})
	}
}

func TestEnsureClaudeAvailable(t *testing.T) {
	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name          string
		setup         func() (claudePath string, cleanup func())
		expectedError bool
	}{
		{
			name: "Claude already in PATH",
			setup: func() (string, func()) {
				// Create temporary directory with mock claude executable
				tempDir, err := os.MkdirTemp("", "claude_path_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				claudePath := filepath.Join(tempDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'claude version 1.0.0'\n"), 0755); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				// Add to PATH
				os.Setenv("PATH", tempDir+":"+originalPath)

				return claudePath, func() { os.RemoveAll(tempDir) }
			},
			expectedError: false,
		},
		{
			name: "Claude not in PATH, needs symlink",
			setup: func() (string, func()) {
				// Remove claude from PATH
				os.Setenv("PATH", "/nonexistent")

				// Create mock claude in specific location
				tempDir, err := os.MkdirTemp("", "claude_symlink_test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				claudePath := filepath.Join(tempDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'claude version 1.0.0'\n"), 0755); err != nil {
					t.Fatalf("Failed to create mock claude: %v", err)
				}

				return claudePath, func() {
					os.RemoveAll(tempDir)
					// Also cleanup potential symlinks
					symlinkPath := filepath.Join(homeDir, ".local/bin", "claude")
					os.Remove(symlinkPath)
				}
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claudePath, cleanup := tt.setup()
			defer cleanup()

			err := ensureClaudeAvailable(claudePath)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCleanupClaudeSymlink(t *testing.T) {
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

	tests := []struct {
		name          string
		setup         func()
		expectedError bool
	}{
		{
			name: "No symlink exists",
			setup: func() {
				// Ensure no symlink exists
				os.Remove(symlinkPath)
			},
			expectedError: false,
		},
		{
			name: "Reviewtask-managed symlink exists",
			setup: func() {
				// Create reviewtask-managed symlink
				os.MkdirAll(localBinDir, 0755)
				mockTarget := filepath.Join(homeDir, ".claude/local/claude")
				os.Symlink(mockTarget, symlinkPath)
			},
			expectedError: false,
		},
		{
			name: "Non-reviewtask symlink exists",
			setup: func() {
				// Create non-reviewtask symlink
				os.MkdirAll(localBinDir, 0755)
				mockTarget := "/usr/local/bin/claude"
				os.Symlink(mockTarget, symlinkPath)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			err := CleanupClaudeSymlink()

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsReviewtaskManagedSymlink(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		{
			name:     "Claude local path",
			target:   "/home/user/.claude/local/claude",
			expected: true,
		},
		{
			name:     "NPM global path",
			target:   "/home/user/.npm-global/bin/claude",
			expected: true,
		},
		{
			name:     "Volta path",
			target:   "/home/user/.volta/bin/claude",
			expected: true,
		},
		{
			name:     "System installation",
			target:   "/usr/local/bin/claude",
			expected: false,
		},
		{
			name:     "Homebrew installation",
			target:   "/opt/homebrew/bin/claude",
			expected: false,
		},
		{
			name:     "Random path",
			target:   "/some/random/path/claude",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReviewtaskManagedSymlink(tt.target)
			if result != tt.expected {
				t.Errorf("Expected %v for target %s, got %v", tt.expected, tt.target, result)
			}
		})
	}
}

func TestResolveClaudeAlias(t *testing.T) {
	// This test will only verify the function doesn't panic
	// Actual alias resolution depends on user's shell configuration
	t.Run("Basic alias resolution", func(t *testing.T) {
		// The function should not panic even if no alias exists
		path, err := resolveClaudeAlias()

		// It's OK if it returns an error (no alias configured)
		// We just want to ensure it doesn't crash
		if err == nil && path != "" {
			t.Logf("Found alias path: %s", path)
		} else {
			t.Logf("No alias found (expected in test environment): %v", err)
		}
	})
}

func TestParseAliasOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple command",
			input:    "claude",
			expected: "claude",
		},
		{
			name:     "Single quoted alias",
			input:    "'claude'",
			expected: "claude",
		},
		{
			name:     "Double quoted alias",
			input:    "\"claude\"",
			expected: "claude",
		},
		{
			name:     "Alias with prefix",
			input:    "alias claude='claude'",
			expected: "claude",
		},
		{
			name:     "Node script alias",
			input:    "node /usr/local/bin/claude.js",
			expected: "node /usr/local/bin/claude.js",
		},
		{
			name:     "Python script alias",
			input:    "python3 /home/user/claude/cli.py",
			expected: "python3 /home/user/claude/cli.py",
		},
		{
			name:     "Complex command with args",
			input:    "npx @anthropic-ai/claude-code",
			expected: "npx",
		},
		{
			name:     "Path with spaces",
			input:    "\"/path with spaces/claude\"",
			expected: "/path with spaces/claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAliasOutput(tt.input)
			if result != tt.expected {
				t.Errorf("parseAliasOutput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSearchAliasInFile(t *testing.T) {
	// Create a temporary file with test content
	tmpDir, err := os.MkdirTemp("", "claude-alias-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testCases := []struct {
		name          string
		content       string
		expectedPath  string
		expectedFound bool
	}{
		{
			name: "Simple alias",
			content: `# Shell config
alias ll='ls -la'
alias claude='claude'
alias gs='git status'`,
			expectedPath:  "claude",
			expectedFound: true,
		},
		{
			name: "Alias with full path",
			content: `# Shell config
alias claude='/usr/local/bin/claude'`,
			expectedPath:  "/usr/local/bin/claude",
			expectedFound: true,
		},
		{
			name: "Alias with node command",
			content: `# Shell config
alias claude='node /home/user/.npm-global/lib/node_modules/@anthropic-ai/claude-code/dist/cli.js'`,
			expectedPath:  "node",
			expectedFound: true,
		},
		{
			name: "No claude alias",
			content: `# Shell config
alias ll='ls -la'
alias gs='git status'`,
			expectedPath:  "",
			expectedFound: false,
		},
		{
			name: "Commented out alias",
			content: `# Shell config
# alias claude='claude'
alias ll='ls -la'`,
			expectedPath:  "",
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, "test_config")
			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}

			path, found := searchAliasInFile(testFile)
			if found != tc.expectedFound {
				t.Errorf("searchAliasInFile() found = %v, want %v", found, tc.expectedFound)
			}
			if path != tc.expectedPath {
				t.Errorf("searchAliasInFile() path = %q, want %q", path, tc.expectedPath)
			}
		})
	}
}

// Integration test that verifies the complete flow
func TestNewRealClaudeClientWithPathDetection(t *testing.T) {
	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	// Setup: Remove claude from PATH
	os.Setenv("PATH", "/nonexistent")

	// Create mock claude in common location
	claudeDir := filepath.Join(homeDir, ".claude", "local")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}
	defer os.RemoveAll(filepath.Join(homeDir, ".claude"))

	claudePath := filepath.Join(claudeDir, "claude")
	if err := os.WriteFile(claudePath, []byte("#!/bin/bash\necho 'Claude Code CLI version 1.0.0'\n"), 0755); err != nil {
		t.Fatalf("Failed to create mock claude: %v", err)
	}

	// Cleanup potential symlinks
	symlinkPath := filepath.Join(homeDir, ".local/bin", "claude")
	defer os.Remove(symlinkPath)

	// Test: Create RealClaudeClient
	client, err := NewRealClaudeClient()
	if err != nil {
		t.Errorf("Unexpected error creating client: %v", err)
	}

	if client == nil {
		t.Errorf("Expected client to be created, got nil")
	}

	// Verify symlink was created
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Errorf("Expected symlink to be created at %s", symlinkPath)
	}
}
