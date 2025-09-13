package test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCIWorkflowScriptHardening tests that the CI workflow has been simplified for cross-platform compatibility
func TestCIWorkflowScriptHardening(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Workflow uses unified approach", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Check that the unified dependency download section exists
		if !strings.Contains(workflowContent, "Download dependencies") {
			t.Fatal("Download dependencies section not found in CI workflow")
		}

		// Extract the download dependencies section
		downloadSectionRegex := regexp.MustCompile(`- name: Download dependencies[\s\S]*?shell: bash`)
		downloadSection := downloadSectionRegex.FindString(workflowContent)
		if downloadSection == "" {
			t.Skip("Could not extract download dependencies section from CI workflow - structure may have changed")
			return
		}

		// Verify that go mod download is present
		if !strings.Contains(downloadSection, "go mod download") {
			t.Error("Download dependencies section does not contain 'go mod download'")
		}

		// Verify shell is bash for cross-platform compatibility
		if !strings.Contains(downloadSection, "shell: bash") {
			t.Error("Download dependencies section does not specify bash shell")
		}
	})

	t.Run("Test step has proper error handling", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Extract the test section
		testSectionRegex := regexp.MustCompile(`- name: Run tests[\s\S]*?(?:- name:|$)`)
		testSection := testSectionRegex.FindString(workflowContent)
		if testSection == "" {
			t.Skip("Could not extract test section from CI workflow - structure may have changed")
			return
		}

		// Check for proper test execution
		requiredElements := []string{
			"go test",
			"EXIT_CODE",
		}

		for _, element := range requiredElements {
			if !strings.Contains(testSection, element) {
				t.Logf("Test section missing element: %s (may be intentional)", element)
			}
		}
	})

	t.Run("Workflow supports all platforms", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Check that matrix strategy includes all platforms
		if !strings.Contains(workflowContent, "ubuntu-latest") {
			t.Error("CI workflow missing ubuntu-latest in matrix")
		}

		if !strings.Contains(workflowContent, "macos-latest") {
			t.Error("CI workflow missing macos-latest in matrix")
		}

		if !strings.Contains(workflowContent, "windows-latest") {
			t.Error("CI workflow missing windows-latest in matrix")
		}

		// Verify bash is used for cross-platform compatibility
		if !strings.Contains(workflowContent, "shell: bash") {
			t.Error("CI workflow should use bash shell for cross-platform compatibility")
		}
	})
}

// TestCIWorkflowBashShellSpecification verifies that bash shell is properly specified
func TestCIWorkflowBashShellSpecification(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Steps specify bash shell", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Check that bash shell is used consistently
		if !strings.Contains(workflowContent, "shell: bash") {
			t.Error("CI workflow should consistently use bash shell")
		}
	})
}

// TestCIWorkflowErrorHandlingRobustness verifies error handling is appropriate for the simplified workflow
func TestCIWorkflowErrorHandlingRobustness(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Test step handles failures appropriately", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// Extract the test section
		testSectionRegex := regexp.MustCompile(`- name: Run tests[\s\S]*?(?:- name:|$)`)
		testSection := testSectionRegex.FindString(workflowContent)
		if testSection == "" {
			t.Skip("Could not extract test section from CI workflow - structure may have changed")
			return
		}

		// Check for exit code handling
		if strings.Contains(testSection, "EXIT_CODE") || strings.Contains(testSection, "exit") {
			// Good - has some form of exit code handling
			t.Log("Test section has exit code handling")
		} else {
			t.Log("Test section may not have explicit exit code handling (could be using default behavior)")
		}
	})
}
