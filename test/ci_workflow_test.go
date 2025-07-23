package test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCIWorkflowScriptHardening tests that the CI workflow Unix script has proper error handling
func TestCIWorkflowScriptHardening(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Unix script has set -euo pipefail", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Check that the Unix dependency download section exists
		if !strings.Contains(workflowContent, "Download dependencies (Unix)") {
			t.Fatal("Unix dependency download section not found in CI workflow")
		}

		// Extract the Unix script section
		unixSectionRegex := regexp.MustCompile(`- name: Download dependencies \(Unix\)[\s\S]*?run: \|[\s\S]*?shell: bash`)
		unixSection := unixSectionRegex.FindString(workflowContent)
		if unixSection == "" {
			t.Fatal("Could not extract Unix script section from CI workflow")
		}

		// Verify that 'set -euo pipefail' is present in the Unix script
		if !strings.Contains(unixSection, "set -euo pipefail") {
			t.Error("Unix script section does not contain 'set -euo pipefail'")
		}

		// Verify that it appears near the beginning of the script
		scriptLines := strings.Split(unixSection, "\n")
		var scriptStartFound bool
		var setPipeFailFound bool
		for _, line := range scriptLines {
			if strings.Contains(line, "run: |") {
				scriptStartFound = true
				continue
			}
			if scriptStartFound && strings.TrimSpace(line) != "" {
				// This should be the first non-empty line after 'run: |'
				if strings.Contains(line, "set -euo pipefail") {
					setPipeFailFound = true
					break
				}
				// If we find any other non-comment line first, that's wrong
				if !strings.Contains(line, "#") {
					break
				}
			}
		}
		if !setPipeFailFound {
			t.Error("'set -euo pipefail' is not the first line of the Unix script")
		}
	})

	t.Run("Script contains proper error handling elements", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Extract the Unix script section
		unixSectionRegex := regexp.MustCompile(`- name: Download dependencies \(Unix\)[\s\S]*?run: \|[\s\S]*?shell: bash`)
		unixSection := unixSectionRegex.FindString(workflowContent)
		if unixSection == "" {
			t.Fatal("Could not extract Unix script section from CI workflow")
		}

		// Verify retry logic is still present
		if !strings.Contains(unixSection, "max_retries=3") {
			t.Error("Max retries configuration not found in Unix script")
		}

		if !strings.Contains(unixSection, "while [ $retry_count -lt $max_retries ]") {
			t.Error("Retry loop not found in Unix script")
		}

		// Verify timeout logic is still present
		if !strings.Contains(unixSection, "timeout_duration=300") {
			t.Error("Timeout duration configuration not found in Unix script")
		}

		// Verify error exit is still present
		if !strings.Contains(unixSection, "exit 1") {
			t.Error("Error exit not found in Unix script")
		}
	})

	t.Run("Windows script is unchanged", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Check that Windows section exists and doesn't have the Unix hardening
		if !strings.Contains(workflowContent, "Download dependencies (Windows)") {
			t.Fatal("Windows dependency download section not found in CI workflow")
		}

		// Extract the Windows script section
		windowsSectionRegex := regexp.MustCompile(`- name: Download dependencies \(Windows\)[\s\S]*?shell: powershell`)
		windowsSection := windowsSectionRegex.FindString(workflowContent)
		if windowsSection == "" {
			t.Fatal("Could not extract Windows script section from CI workflow")
		}

		// Windows script should not contain bash-specific hardening
		if strings.Contains(windowsSection, "set -euo pipefail") {
			t.Error("Windows script incorrectly contains bash-specific 'set -euo pipefail'")
		}
	})
}

// TestCIWorkflowBashShellSpecification tests that Unix scripts use bash shell
func TestCIWorkflowBashShellSpecification(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Unix script specifies bash shell", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Extract the Unix script section
		unixSectionRegex := regexp.MustCompile(`- name: Download dependencies \(Unix\)[\s\S]*?shell: bash`)
		unixSection := unixSectionRegex.FindString(workflowContent)
		if unixSection == "" {
			t.Fatal("Unix script section does not specify bash shell")
		}

		// Verify bash shell is explicitly specified
		if !strings.Contains(unixSection, "shell: bash") {
			t.Error("Unix script does not explicitly specify bash shell")
		}
	})
}

// TestCIWorkflowErrorHandlingRobustness tests the robustness of error handling
func TestCIWorkflowErrorHandlingRobustness(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Script hardening flags are correctly ordered", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Extract the Unix script section
		unixSectionRegex := regexp.MustCompile(`- name: Download dependencies \(Unix\)[\s\S]*?run: \|[\s\S]*?shell: bash`)
		unixSection := unixSectionRegex.FindString(workflowContent)
		if unixSection == "" {
			t.Fatal("Could not extract Unix script section from CI workflow")
		}

		// Find the set command
		setCommandRegex := regexp.MustCompile(`set -[a-z]+`)
		setCommand := setCommandRegex.FindString(unixSection)
		if setCommand == "" {
			t.Fatal("Could not find set command in Unix script")
		}

		// Verify it contains the required flags
		expectedFlags := []string{"e", "u", "o"}
		for _, flag := range expectedFlags {
			if !strings.Contains(setCommand, flag) {
				t.Errorf("Set command missing flag '%s': %s", flag, setCommand)
			}
		}

		// Verify pipefail option
		if !strings.Contains(unixSection, "pipefail") {
			t.Error("Script does not contain pipefail option")
		}
	})
}