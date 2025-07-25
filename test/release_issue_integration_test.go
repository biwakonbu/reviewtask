package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestReleaseIssueIntegration tests integration with release scripts
func TestReleaseIssueIntegration(t *testing.T) {
	// Skip integration tests in CI unless specifically enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration tests (set RUN_INTEGRATION_TESTS=1 to enable)")
	}

	releaseScript := filepath.Join("..", "scripts", "release.sh")
	issueScript := filepath.Join("..", "scripts", "create-release-issue.sh")

	// Verify both scripts exist
	for _, script := range []string{releaseScript, issueScript} {
		if _, err := os.Stat(script); os.IsNotExist(err) {
			t.Fatalf("Required script not found: %s", script)
		}
	}

	t.Run("ReleaseScriptContainsIssueCreation", func(t *testing.T) {
		// Read release script content to verify integration
		content, err := os.ReadFile(releaseScript)
		if err != nil {
			t.Fatalf("Failed to read release script: %v", err)
		}

		scriptContent := string(content)

		// Check that release script references the issue creation script
		if !strings.Contains(scriptContent, "create-release-issue.sh") {
			t.Error("Release script should reference create-release-issue.sh")
		}

		if !strings.Contains(scriptContent, "Creating GitHub Issue for release documentation") {
			t.Error("Release script should contain issue creation step")
		}

		if !strings.Contains(scriptContent, "RELEASE_ISSUE_SCRIPT") {
			t.Error("Release script should define RELEASE_ISSUE_SCRIPT variable")
		}
	})

	t.Run("GitHubWorkflowContainsIssueCreation", func(t *testing.T) {
		workflowPath := filepath.Join("..", ".github", "workflows", "release.yml")
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		workflowContent := string(content)

		// Check that GitHub workflow includes issue creation step
		if !strings.Contains(workflowContent, "Create Release Issue") {
			t.Error("GitHub workflow should contain 'Create Release Issue' step")
		}

		if !strings.Contains(workflowContent, "create-release-issue.sh") {
			t.Error("GitHub workflow should reference create-release-issue.sh")
		}
	})
}

// TestReleaseIssueErrorHandling tests error handling scenarios
func TestReleaseIssueErrorHandling(t *testing.T) {
	scriptPath := filepath.Join("..", "scripts", "create-release-issue.sh")

	// Check if script exists first
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("Script not found: %s (skipping test)", scriptPath)
	}

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectedOutput string
	}{
		{
			name:           "Missing version",
			args:           []string{},
			expectError:    true,
			expectedOutput: "Version is required", // Partial match - full message is "Version is required. Use --version or provide as first argument."
		},
		{
			name:           "Invalid version format - no dots",
			args:           []string{"--version", "v123"},
			expectError:    true,
			expectedOutput: "Invalid version format",
		},
		{
			name:           "Invalid version format - letters in version",
			args:           []string{"--version", "va.b.c"},
			expectError:    true,
			expectedOutput: "Invalid version format",
		},
		{
			name:           "Invalid version format - too many parts",
			args:           []string{"--version", "v1.2.3.4"},
			expectError:    true,
			expectedOutput: "Invalid version format",
		},
		{
			name:           "Valid format but likely GitHub API failure",
			args:           []string{"--version", "v999.999.999"},
			expectError:    true, // Will fail on GitHub API call or git operations
			expectedOutput: "",   // Don't check specific output as it may vary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command with timeout handling
			// Convert relative script path to absolute path
			absoluteScriptPath, absErr := filepath.Abs(scriptPath)
			if absErr != nil {
				t.Fatalf("Failed to get absolute path for script: %v", absErr)
			}

			cmd := exec.Command("bash", absoluteScriptPath)
			cmd.Args = append(cmd.Args, tt.args...)

			// Set working directory to project root to ensure consistent execution
			projectRoot := filepath.Join("..", "..")
			absoluteProjectRoot, absErr2 := filepath.Abs(projectRoot)
			if absErr2 != nil {
				t.Fatalf("Failed to get absolute path for project root: %v", absErr2)
			}
			cmd.Dir = absoluteProjectRoot

			// Disable git operations to prevent test pollution and ensure consistent error handling
			cmd.Env = append(os.Environ(), "TESTING=true")

			// Set timeout using context
			done := make(chan bool, 1)
			var output []byte
			var err error

			go func() {
				output, err = cmd.CombinedOutput()
				done <- true
			}()

			select {
			case <-done:
				// Command completed
			case <-time.After(30 * time.Second):
				// Timeout occurred
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				t.Fatalf("Command timed out after 30 seconds")
			}

			outputStr := string(output)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but command succeeded. Output: %s", outputStr)
					// Log debugging info on unexpected success
					t.Logf("Command: bash %s %v", absoluteScriptPath, tt.args)
					t.Logf("Working directory: %s", absoluteProjectRoot)
				}

				if tt.expectedOutput != "" && !strings.Contains(outputStr, tt.expectedOutput) {
					t.Errorf("Expected output to contain '%s', got: %s", tt.expectedOutput, outputStr)
					// Additional debugging information
					t.Logf("Command: bash %s %v", absoluteScriptPath, tt.args)
					t.Logf("Working directory: %s", absoluteProjectRoot)
					t.Logf("Exit code: %v", err)
					t.Logf("Output length: %d bytes", len(output))
					t.Logf("Full output for debugging:\n%s", outputStr)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success but got error: %v. Output: %s", err, outputStr)
					// Log debugging info on unexpected failure
					t.Logf("Command: bash %s %v", absoluteScriptPath, tt.args)
					t.Logf("Working directory: %s", absoluteProjectRoot)
					t.Logf("Exit code: %v", err)
					t.Logf("Full output for debugging:\n%s", outputStr)
				}
			}
		})
	}
}

// TestReleaseScriptDryRunWithIssueCreation tests dry run functionality
func TestReleaseScriptDryRunWithIssueCreation(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration tests (set RUN_INTEGRATION_TESTS=1 to enable)")
	}

	releaseScript := filepath.Join("..", "scripts", "release.sh")

	// Test dry run to ensure it doesn't actually create issues
	cmd := exec.Command("bash", releaseScript, "dry-run", "patch")
	cmd.Dir = filepath.Dir(releaseScript)

	// Set timeout to avoid hanging
	done := make(chan error, 1)
	go func() {
		_, err := cmd.CombinedOutput()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			// Dry run might fail due to various reasons (not on main branch, etc.)
			// but it should not hang or crash
			t.Logf("Dry run completed with error (expected): %v", err)
		} else {
			t.Log("Dry run completed successfully")
		}
	case <-time.After(60 * time.Second):
		cmd.Process.Kill()
		t.Fatal("Dry run timed out - script may be hanging")
	}
}

// TestScriptExecutionPermissions tests that scripts have correct permissions
func TestScriptExecutionPermissions(t *testing.T) {
	scripts := []string{
		"create-release-issue.sh",
		"release.sh",
		"version.sh",
		"build.sh",
	}

	scriptsDir := filepath.Join("..", "scripts")

	for _, script := range scripts {
		t.Run(script, func(t *testing.T) {
			scriptPath := filepath.Join(scriptsDir, script)

			info, err := os.Stat(scriptPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Skipf("Script not found: %s (skipping)", scriptPath)
				} else {
					t.Fatalf("Failed to stat script %s: %v", scriptPath, err)
				}
			}

			mode := info.Mode()
			// On Windows, executable permission checking is different
			// Skip executable check on Windows as Git Bash handles .sh files regardless
			if runtime.GOOS != "windows" && mode&0111 == 0 {
				t.Errorf("Script %s is not executable (mode: %v)", script, mode)
			}

			// Verify it's a valid shell script
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("Failed to read script: %v", err)
			}

			scriptContent := string(content)
			if !strings.HasPrefix(scriptContent, "#!/bin/bash") {
				t.Errorf("Script %s should start with #!/bin/bash shebang", script)
			}
		})
	}
}

// TestScriptCompatibilityWithExistingWorkflow tests compatibility
func TestScriptCompatibilityWithExistingWorkflow(t *testing.T) {
	releaseScript := filepath.Join("..", "scripts", "release.sh")

	// Check if script exists first
	if _, err := os.Stat(releaseScript); os.IsNotExist(err) {
		t.Skipf("Script not found: %s (skipping test)", releaseScript)
	}

	// Test that release script still supports existing functionality
	// Try invalid command to trigger usage output
	cmd := exec.Command("bash", releaseScript, "invalid-command")
	output, _ := cmd.CombinedOutput()

	// Should show usage/help without crashing
	outputStr := string(output)

	if !strings.Contains(outputStr, "Usage:") && !strings.Contains(outputStr, "COMMANDS:") {
		t.Errorf("Release script should show usage information when run with invalid command")
	}

	// Check that all existing commands are still documented
	requiredSections := []string{
		"prepare",
		"release",
		"dry-run",
		"RELEASE TYPES:",
		"major",
		"minor",
		"patch",
	}

	for _, section := range requiredSections {
		if !strings.Contains(outputStr, section) {
			t.Errorf("Release script help missing section: %s", section)
		}
	}
}

// TestIssueScriptLogOutput tests logging and output formatting
func TestIssueScriptLogOutput(t *testing.T) {
	scriptPath := filepath.Join("..", "scripts", "create-release-issue.sh")

	// Check if script exists first
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("Script not found: %s (skipping test)", scriptPath)
	}

	// Test help output formatting
	cmd := exec.Command("bash", scriptPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	helpOutput := string(output)

	// Verify help output structure
	expectedHelpSections := []string{
		"Usage:",
		"Create a GitHub Issue for release notes",
		"OPTIONS:",
		"EXAMPLES:",
		"ENVIRONMENT:",
	}

	for _, section := range expectedHelpSections {
		if !strings.Contains(helpOutput, section) {
			t.Errorf("Help output missing section: %s", section)
		}
	}

	// Test error output format (should contain colored log messages)
	cmd = exec.Command("bash", scriptPath, "--version", "invalid")
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for invalid version format")
	}

	errorOutput := string(output)
	if !strings.Contains(errorOutput, "[ERROR]") && !strings.Contains(errorOutput, "Invalid version format") {
		t.Errorf("Error output should contain proper error message format, got: %s", errorOutput)
	}
}

// BenchmarkVersionValidation benchmarks version validation performance
func BenchmarkVersionValidation(t *testing.B) {
	versions := []string{
		"v1.0.0",
		"v1.2.3",
		"v10.20.30",
		"v1.0.0-alpha",
		"v2.0.0-rc.1",
		"1.2.3",
		"invalid",
		"v1.0",
	}

	// Create validation script
	validationScript := `#!/bin/bash
version=$1
if [[ ! "$version" =~ ^v ]]; then
    version="v$version"
fi
if [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    exit 0
else
    exit 1
fi`

	tmpFile, err := os.CreateTemp("", "benchmark_version_*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(validationScript); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		t.Fatalf("Failed to make script executable: %v", err)
	}

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		version := versions[i%len(versions)]
		cmd := exec.Command("bash", tmpFile.Name(), version)
		cmd.Dir = filepath.Dir(tmpFile.Name())
		cmd.Run() // Don't care about the result for benchmarking
	}
}
