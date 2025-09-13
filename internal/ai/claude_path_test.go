package ai

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"reviewtask/internal/testutil"
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
				tempDir := testutil.CreateTestDir(t, "claude_test")

				// Create mock Claude CLI
				testutil.CreateMockClaude(t, tempDir, "claude version 1.0.0")

				// Add to PATH
				pathSep := string(os.PathListSeparator)
				os.Setenv("PATH", tempDir+pathSep+originalPath)

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

				// Create mock Claude CLI
				testutil.CreateMockClaude(t, claudeDir, "claude version 1.0.0")

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
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}
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
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}
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
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}
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
			expectedPath:  "node /home/user/.npm-global/lib/node_modules/@anthropic-ai/claude-code/dist/cli.js",
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

// TestClaudeAliasDetectionIntegration tests the full alias detection flow
func TestClaudeAliasDetectionIntegration(t *testing.T) {
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}
	// Skip if running in CI environment where shell configs might not be available
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping alias integration test in CI environment")
	}

	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "claude-alias-integration-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock claude executable
	mockClaudePath := filepath.Join(tmpDir, "mock-claude")
	mockClaudeScript := `#!/bin/bash
echo "Claude Code CLI v1.0.0"
`
	if err := os.WriteFile(mockClaudePath, []byte(mockClaudeScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Create temporary shell config with alias
	tempBashrc := filepath.Join(tmpDir, ".bashrc")
	bashrcContent := fmt.Sprintf(`# Test bashrc
alias claude='%s'
alias other='ls -la'
`, mockClaudePath)
	if err := os.WriteFile(tempBashrc, []byte(bashrcContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test cases
	testCases := []struct {
		name           string
		setupFunc      func()
		cleanupFunc    func()
		expectError    bool
		expectAliasUse bool
	}{
		{
			name: "Detect claude via direct shell config file reading",
			setupFunc: func() {
				// Remove claude from PATH to force alias detection
				os.Setenv("PATH", "/nonexistent")
				// Temporarily backup and replace HOME to use our test directory
				os.Setenv("TEST_ORIGINAL_HOME", os.Getenv("HOME"))
				os.Setenv("HOME", tmpDir)
			},
			cleanupFunc: func() {
				// Restore original HOME
				os.Setenv("HOME", os.Getenv("TEST_ORIGINAL_HOME"))
				os.Unsetenv("TEST_ORIGINAL_HOME")
			},
			expectError:    false,
			expectAliasUse: true,
		},
		{
			name: "Detect claude with node interpreter alias",
			setupFunc: func() {
				// Create a mock node script
				nodeScriptPath := filepath.Join(tmpDir, "claude.js")
				nodeScript := `console.log("Claude Code CLI v1.0.0");`
				os.WriteFile(nodeScriptPath, []byte(nodeScript), 0644)

				// Update bashrc with node-based alias
				bashrcContent := fmt.Sprintf(`# Test bashrc with node
alias claude='node %s'
`, nodeScriptPath)
				os.WriteFile(tempBashrc, []byte(bashrcContent), 0644)

				os.Setenv("PATH", "/nonexistent")
				os.Setenv("TEST_ORIGINAL_HOME", os.Getenv("HOME"))
				os.Setenv("HOME", tmpDir)
			},
			cleanupFunc: func() {
				os.Setenv("HOME", os.Getenv("TEST_ORIGINAL_HOME"))
				os.Unsetenv("TEST_ORIGINAL_HOME")
			},
			expectError:    false,
			expectAliasUse: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			// Cleanup
			if tc.cleanupFunc != nil {
				defer tc.cleanupFunc()
			}

			// Test searchAliasInFile directly with our test bashrc
			aliasPath, found := searchAliasInFile(tempBashrc)
			if !found && tc.expectAliasUse {
				t.Errorf("Expected to find alias in test bashrc, but didn't")
			}
			if found && aliasPath == "" {
				t.Errorf("Found alias but path is empty")
			}

			// Test checkShellConfigFiles if HOME is set to test directory
			if os.Getenv("HOME") == tmpDir {
				path, err := checkShellConfigFiles()
				if tc.expectError && err == nil {
					t.Errorf("Expected error but got none")
				}
				if !tc.expectError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !tc.expectError && tc.expectAliasUse && path == "" {
					t.Errorf("Expected to find claude path via alias, but got empty")
				}

				// Log the detected path for debugging
				if path != "" {
					t.Logf("Detected claude path via alias: %s", path)
				}
			}
		})
	}
}

// TestAliasDetectionEndToEnd demonstrates that alias detection works in a real scenario
func TestAliasDetectionEndToEnd(t *testing.T) {
	// This test verifies that the alias detection logic itself works correctly
	// by testing the individual components that make up the detection flow

	t.Run("searchAliasInFile correctly parses aliases", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "alias-parse-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create mock executable
		mockPath := filepath.Join(tmpDir, "my-claude")
		if err := os.WriteFile(mockPath, []byte("#!/bin/sh\necho claude\n"), 0755); err != nil {
			t.Fatal(err)
		}

		// Create bashrc with various alias formats
		bashrc := filepath.Join(tmpDir, ".bashrc")
		content := fmt.Sprintf(`
# My shell config
alias claude='%s'
alias claude2="%s"
alias claude3=%s
`, mockPath, mockPath, mockPath)

		if err := os.WriteFile(bashrc, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		// Test detection
		path, found := searchAliasInFile(bashrc)
		if !found {
			t.Error("Failed to find alias")
		}
		if path != mockPath {
			t.Errorf("Expected %s, got %s", mockPath, path)
		}
	})

	t.Run("Complex alias formats are parsed correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			alias    string
			expected string
		}{
			{"Simple path", "/usr/bin/claude", "/usr/bin/claude"},
			{"Node script", "node /path/to/claude.js", "node /path/to/claude.js"},
			{"Python script", "python3 /opt/claude/cli.py", "python3 /opt/claude/cli.py"},
			{"NPX command", "npx @anthropic-ai/claude-code", "npx"},
			{"Path with spaces", "\"/Program Files/claude\"", "/Program Files/claude"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := parseAliasOutput(tc.alias)
				if result != tc.expected {
					t.Errorf("parseAliasOutput(%q) = %q, want %q", tc.alias, result, tc.expected)
				}
			})
		}
	})

	t.Run("Real world scenario simulation", func(t *testing.T) {
		// This demonstrates how a user would set up an alias and reviewtask would detect it
		t.Log("Scenario: User has installed claude via npm in a custom location")
		t.Log("and created an alias: alias claude='node ~/.npm-global/lib/node_modules/@anthropic-ai/claude-code/dist/cli.js'")
		t.Log("")
		t.Log("When reviewtask runs:")
		t.Log("1. findClaudeCLI() first checks PATH - not found")
		t.Log("2. Then calls resolveClaudeAlias() which may fail in test env")
		t.Log("3. Falls back to checkShellConfigFiles() which reads ~/.bashrc")
		t.Log("4. Finds the alias and parses it correctly")
		t.Log("5. Returns the full command for execution")

		// The actual implementation is tested above
		// This just documents the expected flow
	})
}

// TestFindClaudeCLIWithAlias tests the complete findClaudeCLI function with alias support
func TestFindClaudeCLIWithAlias(t *testing.T) {
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}

	// Save original PATH and restore after test
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "claude-find-alias-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock claude executable that reports version correctly
	mockClaudePath := filepath.Join(tmpDir, "claude-mock")
	// Write a simple shell script that works on Linux/macOS
	mockClaudeScript := []byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n    echo \"Claude Code CLI v1.0.0\"\n    exit 0\nfi\necho \"Mock claude output\"\n")
	if err := os.WriteFile(mockClaudePath, mockClaudeScript, 0755); err != nil {
		t.Fatal(err)
	}

	// Ensure the script is executable
	if err := os.Chmod(mockClaudePath, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("Find claude via PATH", func(t *testing.T) {
		// Add mock claude to PATH
		os.Setenv("PATH", tmpDir+":"+originalPath)

		// Rename mock to "claude"
		claudeInPath := filepath.Join(tmpDir, "claude")
		os.Rename(mockClaudePath, claudeInPath)
		defer os.Rename(claudeInPath, mockClaudePath) // restore for next test

		path, err := findClaudeCLI()
		if err != nil {
			t.Errorf("Expected to find claude in PATH, got error: %v", err)
		}
		if path != claudeInPath {
			t.Errorf("Expected path %s, got %s", claudeInPath, path)
		}
	})

	t.Run("Find claude via shell config when not in PATH", func(t *testing.T) {
		// Remove claude from PATH
		os.Setenv("PATH", "/nonexistent")

		// Create temporary HOME with bashrc
		tempHome := filepath.Join(tmpDir, "home")
		os.MkdirAll(tempHome, 0755)

		// Create bashrc with alias
		bashrcPath := filepath.Join(tempHome, ".bashrc")
		bashrcContent := fmt.Sprintf(`# Test bashrc
alias claude='%s'
`, mockClaudePath)
		if err := os.WriteFile(bashrcPath, []byte(bashrcContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Temporarily change HOME
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tempHome)
		defer os.Setenv("HOME", originalHome)

		// First test that our mock claude works
		if !isValidClaudeCLI(mockClaudePath) {
			// Try to run the command manually to see output
			cmd := exec.Command(mockClaudePath, "--version")
			output, err := cmd.CombinedOutput()
			t.Errorf("Mock claude at %s is not recognized as valid. Output: %s, Error: %v", mockClaudePath, string(output), err)
		}

		// Test that searchAliasInFile works
		aliasPath, found := searchAliasInFile(bashrcPath)
		if !found {
			t.Errorf("searchAliasInFile failed to find alias in %s", bashrcPath)
		} else {
			t.Logf("searchAliasInFile found: %s", aliasPath)
		}

		// Test checkShellConfigFiles
		configPath, err := checkShellConfigFiles()
		if err != nil {
			t.Logf("checkShellConfigFiles error: %v", err)
		} else {
			t.Logf("checkShellConfigFiles found: %s", configPath)

			// Test if the found path is valid
			if isValidClaudeCLI(configPath) {
				t.Logf("checkShellConfigFiles path is valid Claude CLI")
			} else {
				t.Errorf("checkShellConfigFiles path %s is NOT valid Claude CLI", configPath)
			}
		}

		// Now test the full findClaudeCLI
		// The problem might be that findClaudeCLI calls resolveClaudeAlias()
		// which tries to use the shell, not checkShellConfigFiles

		// Debug: test resolveClaudeAlias directly
		resolvedPath, resolveErr := resolveClaudeAlias()
		if resolveErr != nil {
			t.Logf("resolveClaudeAlias error: %v", resolveErr)
		} else {
			t.Logf("resolveClaudeAlias found: %s", resolvedPath)
			if isValidClaudeCLI(resolvedPath) {
				t.Logf("resolveClaudeAlias path is valid")
			} else {
				t.Logf("resolveClaudeAlias path is NOT valid")
			}
		}

		path, err := findClaudeCLI()
		if err != nil {
			t.Errorf("Expected to find claude via alias, got error: %v", err)
		}
		if path != mockClaudePath {
			t.Errorf("Expected path %s, got %s", mockClaudePath, path)
		}
	})
}

// TestClaudePathDetectionWindows tests basic path detection functionality on Windows
func TestClaudePathDetectionWindows(t *testing.T) {
	// Only run on Windows
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	t.Run("parseAliasOutput handles Windows paths", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{`C:\Program Files\claude\claude.exe`, `C:\Program Files\claude\claude.exe`},
			{`"C:\Program Files\claude\claude.exe"`, `C:\Program Files\claude\claude.exe`},
			{`node.exe C:\Users\test\claude.js`, `node.exe C:\Users\test\claude.js`},
		}

		for _, tc := range tests {
			result := parseAliasOutput(tc.input)
			if result != tc.expected {
				t.Errorf("parseAliasOutput(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		}
	})

	t.Run("findClaudeCLI handles Windows environment", func(t *testing.T) {
		// This test just verifies the function doesn't panic on Windows
		_, _ = findClaudeCLI()
	})
}

// Integration test that verifies the complete flow
func TestNewRealClaudeClientWithPathDetection(t *testing.T) {
	// Skip on Windows as shell scripts won't work
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell script based test on Windows")
	}
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
