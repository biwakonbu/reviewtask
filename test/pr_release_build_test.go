package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestReleaseScriptDryRunMode tests the enhanced dry-run functionality in release.sh
func TestReleaseScriptDryRunMode(t *testing.T) {
	// Get project root directory
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	scriptPath := filepath.Join(projectRoot, "scripts", "release.sh")

	tests := []struct {
		name     string
		args     []string
		wantExit int
		wantOut  []string
	}{
		{
			name:     "prepare with dry-run flag",
			args:     []string{"prepare", "patch", "--dry-run"},
			wantExit: 0,
			wantOut:  []string{"DRY RUN: Simulating release preparation"},
		},
		{
			name:     "prepare without dry-run uses normal flow",
			args:     []string{"prepare", "patch"},
			wantExit: 0,
			wantOut:  []string{"Preparing release", "Testing build process"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command with args
			cmd := exec.Command("bash", append([]string{scriptPath}, tt.args...)...)
			cmd.Dir = projectRoot

			// Set environment to skip interactive prompts
			cmd.Env = append(os.Environ(), "AUTO_CONFIRM=true")

			// Run command and capture output
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Check exit code
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				}
			}

			if exitCode != tt.wantExit {
				t.Errorf("Expected exit code %d, got %d. Output: %s", tt.wantExit, exitCode, outputStr)
			}

			// Check that expected strings are in output
			for _, want := range tt.wantOut {
				if !strings.Contains(outputStr, want) {
					t.Errorf("Expected output to contain %q, got: %s", want, outputStr)
				}
			}
		})
	}
}

// TestBuildScriptCrossPlatformTest tests the build.sh test command functionality
func TestBuildScriptCrossPlatformTest(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	scriptPath := filepath.Join(projectRoot, "scripts", "build.sh")

	// Test the cross-platform build test command
	cmd := exec.Command("bash", scriptPath, "test")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Fatalf("Build test failed: %v\nOutput: %s", err, outputStr)
	}

	// In CI environment, only minimal tests are run
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		// Check for CI-specific success messages
		if !strings.Contains(outputStr, "CI environment detected") {
			t.Errorf("Expected CI environment detection message not found in output: %s", outputStr)
		}
		if !strings.Contains(outputStr, "Native compilation test passed") {
			t.Errorf("Expected native compilation success message not found in output: %s", outputStr)
		}
		if !strings.Contains(outputStr, "Cross-compilation capability verified") {
			t.Errorf("Expected cross-compilation verification message not found in output: %s", outputStr)
		}
		if !strings.Contains(outputStr, "CI cross-compilation tests passed") {
			t.Errorf("Expected CI success message not found in output: %s", outputStr)
		}
	} else {
		// Check that all expected platforms are tested in non-CI environment
		expectedPlatforms := []string{
			"linux/amd64",
			"linux/arm64",
			"darwin/amd64",
			"darwin/arm64",
			"windows/amd64",
			"windows/arm64",
		}

		for _, platform := range expectedPlatforms {
			if !strings.Contains(outputStr, platform) {
				t.Errorf("Expected output to contain platform %q, got: %s", platform, outputStr)
			}
		}

		// Check for success message
		if !strings.Contains(outputStr, "All cross-compilation tests passed") {
			t.Errorf("Expected success message not found in output: %s", outputStr)
		}
	}
}

// TestVersionScriptCurrentCommand tests the version.sh current command
func TestVersionScriptCurrentCommand(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	scriptPath := filepath.Join(projectRoot, "scripts", "version.sh")

	// Test version script current command
	cmd := exec.Command("bash", scriptPath, "current")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version script failed: %v\nOutput: %s", err, string(output))
	}

	versionOutput := strings.TrimSpace(string(output))

	// Check that we get a valid semantic version format
	if !isValidSemanticVersion(versionOutput) {
		t.Errorf("Expected valid semantic version, got: %q", versionOutput)
	}
}

// TestVersionEmbedding tests that version embedding works correctly
func TestVersionEmbedding(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Get current version
	versionCmd := exec.Command("bash", filepath.Join(projectRoot, "scripts", "version.sh"), "current")
	versionCmd.Dir = projectRoot
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}
	version := strings.TrimSpace(string(versionOutput))

	// Build with version embedding
	binName := "test-binary"
	if runtime.GOOS == "windows" {
		binName = "test-binary.exe"
	}

	buildCmd := exec.Command("go", "build", "-ldflags", "-X main.version="+version, "-o", binName, ".")
	buildCmd.Dir = projectRoot
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build with version embedding: %v", err)
	}
	defer os.Remove(filepath.Join(projectRoot, binName))

	// Test version command
	binPath := "./" + binName
	testCmd := exec.Command(binPath, "version")
	testCmd.Dir = projectRoot
	output, err := testCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run version command: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, version) {
		t.Errorf("Expected version output to contain %q, got: %s", version, outputStr)
	}
}

// Helper function to check if a string is a valid semantic version
func isValidSemanticVersion(version string) bool {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	// Check if all parts are numeric
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}

	return true
}
