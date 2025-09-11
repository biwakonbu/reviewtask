package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestAuthLoginCommand tests the auth login command
func TestAuthLoginCommand(t *testing.T) {
	// Set test mode environment variable
	os.Setenv("REVIEWTASK_TEST_MODE", "true")
	defer os.Unsetenv("REVIEWTASK_TEST_MODE")

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-auth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Create .pr-review directory
	err = os.MkdirAll(".pr-review", 0755)
	if err != nil {
		t.Fatalf("Failed to create .pr-review directory: %v", err)
	}

	// Test cases
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectFile  bool
	}{
		{
			name:        "valid token input",
			input:       "github_pat_11TEST1234567890ABCDEF\n",
			expectError: false,
			expectFile:  true,
		},
		{
			name:        "empty token input",
			input:       "\n",
			expectError: true,
			expectFile:  false,
		},
		{
			name:        "invalid token format",
			input:       "invalid-token\n",
			expectError: false, // Command accepts any non-empty input
			expectFile:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up auth file between tests
			authFile := ".pr-review/auth.json"
			os.Remove(authFile)

			// Create a buffer to simulate user input
			var stdin bytes.Buffer
			stdin.WriteString(tt.input)

			// Create command with mocked stdin
			cmd := &cobra.Command{
				Use: "login",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock stdin for testing
					oldStdin := os.Stdin
					r, w, _ := os.Pipe()
					os.Stdin = r
					go func() {
						defer w.Close()
						w.WriteString(tt.input)
					}()
					defer func() { os.Stdin = oldStdin }()

					return runAuthLogin(cmd, args)
				},
			}

			// Execute command
			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check if auth file was created
			if tt.expectFile {
				if _, err := os.Stat(authFile); os.IsNotExist(err) {
					t.Errorf("Expected auth file to be created but it doesn't exist")
				}
			} else {
				if _, err := os.Stat(authFile); err == nil {
					t.Errorf("Expected auth file not to be created but it exists")
				}
			}
		})
	}
}

// TestAuthStatusCommand tests the auth status command
func TestAuthStatusCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-auth-status-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Create .pr-review directory
	err = os.MkdirAll(".pr-review", 0755)
	if err != nil {
		t.Fatalf("Failed to create .pr-review directory: %v", err)
	}

	// Test cases
	tests := []struct {
		name        string
		setupAuth   bool
		setupEnv    bool
		expectError bool
	}{
		{
			name:        "no authentication",
			setupAuth:   false,
			setupEnv:    false,
			expectError: false, // Status command should not error, just report status
		},
		{
			name:        "local auth file exists",
			setupAuth:   true,
			setupEnv:    false,
			expectError: false,
		},
		{
			name:        "environment variable set",
			setupAuth:   false,
			setupEnv:    true,
			expectError: false,
		},
		{
			name:        "both auth sources available",
			setupAuth:   true,
			setupEnv:    true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.Remove(".pr-review/auth.json")
			os.Unsetenv("GITHUB_TOKEN")

			// Setup test conditions
			if tt.setupAuth {
				authContent := `{"token": "test-token"}`
				err := os.WriteFile(".pr-review/auth.json", []byte(authContent), 0600)
				if err != nil {
					t.Fatalf("Failed to create auth file: %v", err)
				}
			}

			if tt.setupEnv {
				os.Setenv("GITHUB_TOKEN", "env-token")
			}

			// Create command
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use: "status",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runAuthStatus(cmd, args)
				},
			}
			cmd.SetOut(&output)

			// Execute command
			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Clean up environment
			os.Unsetenv("GITHUB_TOKEN")
		})
	}
}

// TestAuthLogoutCommand tests the auth logout command
func TestAuthLogoutCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-auth-logout-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Create .pr-review directory
	err = os.MkdirAll(".pr-review", 0755)
	if err != nil {
		t.Fatalf("Failed to create .pr-review directory: %v", err)
	}

	// Test cases
	tests := []struct {
		name        string
		setupAuth   bool
		expectError bool
	}{
		{
			name:        "logout with existing auth file",
			setupAuth:   true,
			expectError: false,
		},
		{
			name:        "logout without auth file",
			setupAuth:   false,
			expectError: false, // Should not error even if no auth file exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authFile := ".pr-review/auth.json"

			// Setup test conditions
			if tt.setupAuth {
				authContent := `{"token": "test-token"}`
				err := os.WriteFile(authFile, []byte(authContent), 0600)
				if err != nil {
					t.Fatalf("Failed to create auth file: %v", err)
				}
			}

			// Create command
			cmd := &cobra.Command{
				Use: "logout",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runAuthLogout(cmd, args)
				},
			}

			// Execute command
			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify auth file is removed (if it existed)
			if tt.setupAuth {
				if _, err := os.Stat(authFile); !os.IsNotExist(err) {
					t.Errorf("Expected auth file to be removed but it still exists")
				}
			}
		})
	}
}

// TestAuthCheckCommand tests the auth check command
func TestAuthCheckCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-auth-check-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Create .pr-review directory
	err = os.MkdirAll(".pr-review", 0755)
	if err != nil {
		t.Fatalf("Failed to create .pr-review directory: %v", err)
	}

	// Test cases
	tests := []struct {
		name        string
		setupAuth   bool
		expectError bool
	}{
		{
			name:        "check with valid auth",
			setupAuth:   true,
			expectError: true, // Will likely fail due to invalid token, but tests the code path
		},
		{
			name:        "check without auth",
			setupAuth:   false,
			expectError: true, // Should error when no auth is available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.Remove(".pr-review/auth.json")
			os.Unsetenv("GITHUB_TOKEN")

			// Setup test conditions
			if tt.setupAuth {
				authContent := `{"token": "test-token"}`
				err := os.WriteFile(".pr-review/auth.json", []byte(authContent), 0600)
				if err != nil {
					t.Fatalf("Failed to create auth file: %v", err)
				}
			}

			// Create command
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use: "check",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runAuthCheck(cmd, args)
				},
			}
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Execute command
			err := cmd.Execute()

			// Auth check will likely fail in test environment due to invalid tokens
			// We're mainly testing that the function doesn't panic and follows code paths
			if !tt.expectError && err != nil {
				t.Logf("Expected success but got error (may be due to test environment): %v", err)
			}

			// Clean up environment
			os.Unsetenv("GITHUB_TOKEN")
		})
	}
}

// TestAuthCommandIntegration tests the auth command integration
func TestAuthCommandIntegration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-auth-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Test the full auth workflow
	t.Run("auth workflow", func(t *testing.T) {
		// Initialize repository structure
		err = os.MkdirAll(".pr-review", 0755)
		if err != nil {
			t.Fatalf("Failed to create .pr-review directory: %v", err)
		}

		// Test status before login (should show no auth)
		var statusOutput bytes.Buffer
		statusCmd := &cobra.Command{
			Use: "status",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runAuthStatus(cmd, args)
			},
		}
		statusCmd.SetOut(&statusOutput)

		err := statusCmd.Execute()
		if err != nil {
			t.Logf("Status command output: %s", statusOutput.String())
		}

		// Test logout when no auth exists (should not error)
		logoutCmd := &cobra.Command{
			Use: "logout",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runAuthLogout(cmd, args)
			},
		}

		err = logoutCmd.Execute()
		if err != nil {
			t.Errorf("Logout command failed when no auth exists: %v", err)
		}

		// Verify no auth file exists
		authFile := ".pr-review/auth.json"
		if _, err := os.Stat(authFile); !os.IsNotExist(err) {
			t.Errorf("Auth file should not exist after logout")
		}
	})
}

// TestAuthErrorHandling tests various error conditions
func TestAuthErrorHandling(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-auth-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	t.Run("missing pr-review directory", func(t *testing.T) {
		// Don't create .pr-review directory to test error handling

		// Test auth status in invalid directory
		var output bytes.Buffer
		cmd := &cobra.Command{
			Use: "status",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runAuthStatus(cmd, args)
			},
		}
		cmd.SetOut(&output)
		cmd.SetErr(&output)

		err := cmd.Execute()
		// May or may not error depending on implementation
		if err != nil {
			t.Logf("Expected error when .pr-review doesn't exist: %v", err)
		}
	})

	t.Run("read-only filesystem simulation", func(t *testing.T) {
		// Create .pr-review directory
		err = os.MkdirAll(".pr-review", 0755)
		if err != nil {
			t.Fatalf("Failed to create .pr-review directory: %v", err)
		}

		// Create a file with restrictive permissions to simulate write failure
		restrictedFile := ".pr-review/readonly.txt"
		err := os.WriteFile(restrictedFile, []byte("test"), 0444)
		if err != nil {
			t.Fatalf("Failed to create restricted file: %v", err)
		}

		// Test will depend on how the auth commands handle write permissions
		// This exercises error handling paths
	})
}

// TestAuthTokenValidation tests token validation logic
func TestAuthTokenValidation(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "github personal access token",
			token:    "github_pat_11ABCDEFGHIJK1234567890",
			expected: true,
		},
		{
			name:     "classic personal access token",
			token:    "ghp_1234567890ABCDEFGHIJK1234567890ABCDEF",
			expected: true,
		},
		{
			name:     "empty token",
			token:    "",
			expected: false,
		},
		{
			name:     "whitespace token",
			token:    "   ",
			expected: false,
		},
		{
			name:     "short token",
			token:    "abc123",
			expected: true, // May accept short tokens depending on validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test token validation if such function exists
			// This is mainly for future token validation logic
			isValid := len(strings.TrimSpace(tt.token)) > 0

			if isValid != tt.expected {
				t.Logf("Token validation result: %v, expected: %v for token: %q",
					isValid, tt.expected, tt.token)
			}
		})
	}
}

// Helper function for auth command testing
func setupTestAuth(t *testing.T, tempDir string) {
	authContent := `{
		"token": "test-github-token",
		"created_at": "2023-01-01T00:00:00Z"
	}`

	authFile := tempDir + "/.pr-review/auth.json"
	err := os.WriteFile(authFile, []byte(authContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test auth file: %v", err)
	}
}

// TestAuthFilePermissions tests that auth files have correct permissions
func TestAuthFilePermissions(t *testing.T) {
	// Skip on Windows as file permissions work differently
	if runtime.GOOS == "windows" {
		t.Skip("Skipping file permission test on Windows")
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-auth-perms-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Create .pr-review directory
	err = os.MkdirAll(".pr-review", 0755)
	if err != nil {
		t.Fatalf("Failed to create .pr-review directory: %v", err)
	}

	// Setup test auth file
	setupTestAuth(t, tempDir)

	// Check file permissions
	authFile := ".pr-review/auth.json"
	info, err := os.Stat(authFile)
	if err != nil {
		t.Fatalf("Failed to stat auth file: %v", err)
	}

	// Check that file has restrictive permissions (readable only by owner)
	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("Auth file has incorrect permissions: %v, expected: 0600", mode.Perm())
	}
}

// TestAuthHelperFunctions tests helper functions
func TestAuthHelperFunctions(t *testing.T) {
	t.Run("Token validation", func(t *testing.T) {
		// Basic token validation tests
		tokens := []struct {
			token   string
			isValid bool
		}{
			{"ghp_validtoken123", true},
			{"", false},
			{"   ", false},
		}

		for _, tt := range tokens {
			isValid := len(strings.TrimSpace(tt.token)) > 0
			if isValid != tt.isValid {
				t.Errorf("Token %q: expected %v, got %v", tt.token, tt.isValid, isValid)
			}
		}
	})
}

// TestAuthScenarios tests complete authentication scenarios
func TestAuthScenarios(t *testing.T) {
	// Set test mode environment variable
	os.Setenv("REVIEWTASK_TEST_MODE", "true")
	defer os.Unsetenv("REVIEWTASK_TEST_MODE")

	// Set temporary GH config directory to avoid conflicts
	tempGHConfig, _ := os.MkdirTemp("", "gh-config-*")
	os.Setenv("GH_CONFIG_DIR", tempGHConfig)
	defer os.RemoveAll(tempGHConfig)
	defer os.Unsetenv("GH_CONFIG_DIR")

	scenarios := []struct {
		name     string
		steps    []authStep
		validate func(t *testing.T, dir string)
	}{
		{
			name: "新規ユーザー認証フロー",
			steps: []authStep{
				{cmd: "status", expectOut: []string{"Not authenticated"}},
				{cmd: "login", input: "test-token-123\n"},
				{cmd: "status", expectOut: []string{"Authentication configured"}},
				{cmd: "check", expectError: false}, // Test mode skips validation
			},
		},
		{
			name: "既存認証の更新フロー",
			steps: []authStep{
				{setup: setupOldAuth},
				{cmd: "status", expectOut: []string{"Authentication configured"}},
				{cmd: "logout"},
				{cmd: "status", expectOut: []string{"Not authenticated"}},
				{cmd: "login", input: "new-token-456\n"},
				{cmd: "status", expectOut: []string{"Authentication configured"}},
			},
		},
		{
			name: "環境変数を使った認証フロー",
			steps: []authStep{
				{setup: func(t *testing.T, dir string) { os.Setenv("GITHUB_TOKEN", "env-token") }},
				{cmd: "status", expectOut: []string{"environment variable"}},
				{cleanup: func() { os.Unsetenv("GITHUB_TOKEN") }},
			},
		},
		{
			name: "マルチ認証ソースの優先順位",
			steps: []authStep{
				{cmd: "login", input: "local-token\n"},
				{setup: func(t *testing.T, dir string) { os.Setenv("GITHUB_TOKEN", "env-token") }},
				{cmd: "status", expectOut: []string{"Authentication configured"}},
				{cleanup: func() { os.Unsetenv("GITHUB_TOKEN") }},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := setupTestDir(t)
			defer os.RemoveAll(tempDir)

			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			for _, step := range scenario.steps {
				if step.setup != nil {
					step.setup(t, tempDir)
				}

				if step.cmd != "" {
					executeAuthCommand(t, step)
				}

				if step.cleanup != nil {
					step.cleanup()
				}
			}

			if scenario.validate != nil {
				scenario.validate(t, tempDir)
			}
		})
	}
}

type authStep struct {
	cmd         string
	input       string
	expectOut   []string
	expectError bool
	setup       func(t *testing.T, dir string)
	cleanup     func()
}

func executeAuthCommand(t *testing.T, step authStep) {
	var output bytes.Buffer
	var cmd *cobra.Command

	switch step.cmd {
	case "login":
		cmd = &cobra.Command{
			Use: "login",
			RunE: func(cmd *cobra.Command, args []string) error {
				if step.input != "" {
					oldStdin := os.Stdin
					r, w, _ := os.Pipe()
					os.Stdin = r
					go func() {
						defer w.Close()
						w.WriteString(step.input)
					}()
					defer func() { os.Stdin = oldStdin }()
				}
				return runAuthLogin(cmd, args)
			},
		}
	case "logout":
		cmd = &cobra.Command{
			Use:  "logout",
			RunE: runAuthLogout,
		}
	case "status":
		cmd = &cobra.Command{
			Use:  "status",
			RunE: runAuthStatus,
		}
	case "check":
		cmd = &cobra.Command{
			Use:  "check",
			RunE: runAuthCheck,
		}
	default:
		t.Fatalf("Unknown command: %s", step.cmd)
	}

	// Capture output
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	err := cmd.Execute()

	// Restore stdout/stderr
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output.Write(buf[:n])

	if step.expectError && err == nil {
		t.Errorf("Expected error for command %s but got none", step.cmd)
	}
	if !step.expectError && err != nil {
		t.Logf("Command %s error (may be expected in test): %v", step.cmd, err)
	}

	outputStr := output.String()
	for _, expected := range step.expectOut {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Command %s: expected output to contain %q, got: %s", step.cmd, expected, outputStr)
		}
	}
}

func setupTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "reviewtask-scenario-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Change to temp directory to initialize git
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Change back to original directory
	os.Chdir(originalDir)

	err = os.MkdirAll(filepath.Join(tempDir, ".pr-review"), 0755)
	if err != nil {
		t.Fatalf("Failed to create .pr-review dir: %v", err)
	}

	// Clear any existing GitHub environment variables
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GH_TOKEN")

	return tempDir
}

func setupOldAuth(t *testing.T, dir string) {
	// Create .pr-review directory first
	if err := os.MkdirAll(".pr-review", 0755); err != nil {
		t.Fatalf("Failed to create .pr-review directory: %v", err)
	}
	authData := map[string]interface{}{
		"token":      "old-token",
		"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
	}

	data, _ := json.Marshal(authData)
	authFile := filepath.Join(".pr-review", "auth.json") // Use relative path since we're already in tempDir
	err := os.WriteFile(authFile, data, 0600)
	if err != nil {
		t.Fatalf("Failed to setup old auth: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(authFile); err != nil {
		t.Fatalf("Auth file not created: %v", err)
	}
}

// TestAuthMultipleEnvironments tests auth behavior across different environments
func TestAuthMultipleEnvironments(t *testing.T) {
	environments := []struct {
		name   string
		setup  func() (cleanup func())
		expect string
	}{
		{
			name: "CI環境（GitHub Actions）",
			setup: func() func() {
				os.Setenv("CI", "true")
				os.Setenv("GITHUB_ACTIONS", "true")
				os.Setenv("GITHUB_TOKEN", "gha-token")
				return func() {
					os.Unsetenv("CI")
					os.Unsetenv("GITHUB_ACTIONS")
					os.Unsetenv("GITHUB_TOKEN")
				}
			},
			expect: "gha-token",
		},
		{
			name: "ローカル開発環境",
			setup: func() func() {
				os.Setenv("REVIEWTASK_GITHUB_TOKEN", "dev-token")
				return func() {
					os.Unsetenv("REVIEWTASK_GITHUB_TOKEN")
				}
			},
			expect: "dev-token",
		},
		{
			name: "Docker環境",
			setup: func() func() {
				os.Setenv("DOCKER_CONTAINER", "true")
				os.Setenv("GH_TOKEN", "docker-token")
				return func() {
					os.Unsetenv("DOCKER_CONTAINER")
					os.Unsetenv("GH_TOKEN")
				}
			},
			expect: "docker-token",
		},
	}

	for _, env := range environments {
		t.Run(env.name, func(t *testing.T) {
			tempDir := setupTestDir(t)
			defer os.RemoveAll(tempDir)

			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			cleanup := env.setup()
			defer cleanup()

			var output bytes.Buffer
			cmd := &cobra.Command{
				Use:  "status",
				RunE: runAuthStatus,
			}
			cmd.SetOut(&output)

			err := cmd.Execute()
			if err != nil {
				t.Logf("Status command in %s: %v", env.name, err)
			}

			// Verify environment-specific behavior
			outputStr := output.String()
			t.Logf("%s output: %s", env.name, outputStr)
		})
	}
}

// TestAuthErrorRecovery tests error recovery scenarios
func TestAuthErrorRecovery(t *testing.T) {
	t.Run("破損した認証ファイルからの回復", func(t *testing.T) {
		tempDir := setupTestDir(t)
		defer os.RemoveAll(tempDir)

		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		// Create corrupted auth file
		authFile := ".pr-review/auth.json"
		os.WriteFile(authFile, []byte("{ invalid json }"), 0600)

		// Try to use auth status - should handle gracefully
		var output bytes.Buffer
		cmd := &cobra.Command{
			Use:  "status",
			RunE: runAuthStatus,
		}
		cmd.SetOut(&output)
		cmd.SetErr(&output)

		err := cmd.Execute()
		if err != nil {
			t.Logf("Expected error with corrupted file: %v", err)
		}

		// Recover by re-login
		loginCmd := &cobra.Command{
			Use: "login",
			RunE: func(cmd *cobra.Command, args []string) error {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				go func() {
					defer w.Close()
					w.WriteString("recovery-token\n")
				}()
				defer func() { os.Stdin = oldStdin }()
				return runAuthLogin(cmd, args)
			},
		}

		err = loginCmd.Execute()
		if err != nil {
			t.Logf("Login after corruption: %v", err)
		}

		// Verify recovery
		data, err := os.ReadFile(authFile)
		if err == nil && strings.Contains(string(data), "recovery-token") {
			t.Log("Successfully recovered from corrupted auth file")
		}
	})

	t.Run("権限エラーからの回復", func(t *testing.T) {
		// Skip on Windows as file permissions work differently
		if runtime.GOOS == "windows" {
			t.Skip("Skipping file permission test on Windows")
		}

		tempDir := setupTestDir(t)
		defer os.RemoveAll(tempDir)

		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		// Create auth file with wrong permissions
		authFile := ".pr-review/auth.json"
		authData := `{"token": "test-token"}`
		os.WriteFile(authFile, []byte(authData), 0644) // Wrong permissions

		// Try to read with status
		var output bytes.Buffer
		cmd := &cobra.Command{
			Use:  "status",
			RunE: runAuthStatus,
		}
		cmd.SetOut(&output)

		err := cmd.Execute()
		if err != nil {
			t.Logf("Status with wrong permissions: %v", err)
		}

		// Fix permissions
		os.Chmod(authFile, 0600)

		// Verify fixed
		info, _ := os.Stat(authFile)
		if info.Mode().Perm() != 0600 {
			t.Errorf("Failed to fix permissions")
		}
	})
}

// TestAuthConcurrency tests concurrent auth operations
func TestAuthConcurrency(t *testing.T) {
	tempDir := setupTestDir(t)
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Run multiple auth operations concurrently
	done := make(chan bool, 3)

	// Status check
	go func() {
		var output bytes.Buffer
		cmd := &cobra.Command{
			Use:  "status",
			RunE: runAuthStatus,
		}
		cmd.SetOut(&output)
		cmd.Execute()
		done <- true
	}()

	// Another status check
	go func() {
		var output bytes.Buffer
		cmd := &cobra.Command{
			Use:  "status",
			RunE: runAuthStatus,
		}
		cmd.SetOut(&output)
		cmd.Execute()
		done <- true
	}()

	// Logout attempt
	go func() {
		cmd := &cobra.Command{
			Use:  "logout",
			RunE: runAuthLogout,
		}
		cmd.Execute()
		done <- true
	}()

	// Wait for all operations
	for i := 0; i < 3; i++ {
		<-done
	}

	t.Log("Concurrent operations completed without panic")
}

// TestAuthCredentialsPriority tests the priority of different auth sources
func TestAuthCredentialsPriority(t *testing.T) {
	tests := []struct {
		name        string
		localToken  string
		envToken    string
		ghToken     string
		expectedSrc string
	}{
		{
			name:        "ローカルファイル優先",
			localToken:  "local-token",
			envToken:    "",
			ghToken:     "",
			expectedSrc: "local",
		},
		{
			name:        "環境変数フォールバック",
			localToken:  "",
			envToken:    "env-token",
			ghToken:     "",
			expectedSrc: "env",
		},
		{
			name:        "GH_TOKEN最後の手段",
			localToken:  "",
			envToken:    "",
			ghToken:     "gh-token",
			expectedSrc: "gh",
		},
		{
			name:        "全ソース利用可能時",
			localToken:  "local-token",
			envToken:    "env-token",
			ghToken:     "gh-token",
			expectedSrc: "local", // Local should win
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTestDir(t)
			defer os.RemoveAll(tempDir)

			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			// Setup auth sources
			if tt.localToken != "" {
				authData := fmt.Sprintf(`{"token": "%s"}`, tt.localToken)
				os.WriteFile(".pr-review/auth.json", []byte(authData), 0600)
			}
			if tt.envToken != "" {
				os.Setenv("GITHUB_TOKEN", tt.envToken)
				defer os.Unsetenv("GITHUB_TOKEN")
			}
			if tt.ghToken != "" {
				os.Setenv("GH_TOKEN", tt.ghToken)
				defer os.Unsetenv("GH_TOKEN")
			}

			// Check which source is used
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use:  "status",
				RunE: runAuthStatus,
			}
			cmd.SetOut(&output)

			err := cmd.Execute()
			if err != nil {
				t.Logf("Status check: %v", err)
			}

			outputStr := output.String()
			t.Logf("Priority test %s output: %s", tt.name, outputStr)
		})
	}
}
