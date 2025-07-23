package test

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CIWorkflow represents the structure of a GitHub Actions workflow
type CIWorkflow struct {
	Name string                 `yaml:"name"`
	On   map[string]interface{} `yaml:"on"`
	Jobs map[string]CIJob       `yaml:"jobs"`
}

// CIJob represents a job in the workflow
type CIJob struct {
	Name     string                 `yaml:"name"`
	RunsOn   interface{}            `yaml:"runs-on"`
	Strategy map[string]interface{} `yaml:"strategy,omitempty"`
	Steps    []CIStep               `yaml:"steps"`
}

// CIStep represents a step in a job
type CIStep struct {
	Name  string            `yaml:"name,omitempty"`
	Uses  string            `yaml:"uses,omitempty"`
	With  map[string]string `yaml:"with,omitempty"`
	Run   string            `yaml:"run,omitempty"`
	If    string            `yaml:"if,omitempty"`
	Shell string            `yaml:"shell,omitempty"`
}

// TestCIWorkflowIntegration tests the CI workflow as a complete system
func TestCIWorkflowIntegration(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Workflow file is valid YAML", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		var workflow CIWorkflow
		err = yaml.Unmarshal(content, &workflow)
		if err != nil {
			t.Fatalf("CI workflow file is not valid YAML: %v", err)
		}

		// Basic validation
		if workflow.Name == "" {
			t.Error("Workflow name is empty")
		}

		if len(workflow.Jobs) == 0 {
			t.Error("Workflow has no jobs")
		}
	})

	t.Run("Unix dependency step has proper error handling integration", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		var workflow CIWorkflow
		err = yaml.Unmarshal(content, &workflow)
		if err != nil {
			t.Fatalf("Failed to parse CI workflow: %v", err)
		}

		// Find the test job
		testJob, exists := workflow.Jobs["test"]
		if !exists {
			t.Fatal("Test job not found in CI workflow")
		}

		// Find the Unix dependency download step
		var unixStep *CIStep
		for i, step := range testJob.Steps {
			if strings.Contains(step.Name, "Download dependencies (Unix)") {
				unixStep = &testJob.Steps[i]
				break
			}
		}

		if unixStep == nil {
			t.Fatal("Unix dependency download step not found")
		}

		// Verify the step configuration
		if unixStep.If != "runner.os != 'Windows'" {
			t.Errorf("Unix step condition incorrect: %s", unixStep.If)
		}

		if unixStep.Shell != "bash" {
			t.Errorf("Unix step shell incorrect: %s", unixStep.Shell)
		}

		// Verify the script content includes hardening
		if !strings.Contains(unixStep.Run, "set -euo pipefail") {
			t.Error("Unix step script does not contain error handling hardening")
		}

		// Verify the script maintains existing functionality
		requiredElements := []string{
			"timeout_duration=300",
			"max_retries=3",
			"retry_count=0",
			"go mod download",
			"exit 1",
		}

		for _, element := range requiredElements {
			if !strings.Contains(unixStep.Run, element) {
				t.Errorf("Unix step script missing required element: %s", element)
			}
		}
	})

	t.Run("Windows dependency step remains unchanged", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		var workflow CIWorkflow
		err = yaml.Unmarshal(content, &workflow)
		if err != nil {
			t.Fatalf("Failed to parse CI workflow: %v", err)
		}

		// Find the test job
		testJob, exists := workflow.Jobs["test"]
		if !exists {
			t.Fatal("Test job not found in CI workflow")
		}

		// Find the Windows dependency download step
		var windowsStep *CIStep
		for i, step := range testJob.Steps {
			if strings.Contains(step.Name, "Download dependencies (Windows)") {
				windowsStep = &testJob.Steps[i]
				break
			}
		}

		if windowsStep == nil {
			t.Fatal("Windows dependency download step not found")
		}

		// Verify the step configuration
		if windowsStep.If != "runner.os == 'Windows'" {
			t.Errorf("Windows step condition incorrect: %s", windowsStep.If)
		}

		if windowsStep.Shell != "powershell" {
			t.Errorf("Windows step shell incorrect: %s", windowsStep.Shell)
		}

		// Verify the Windows script does not contain bash-specific hardening
		if strings.Contains(windowsStep.Run, "set -euo pipefail") {
			t.Error("Windows step incorrectly contains bash-specific hardening")
		}

		// Verify Windows script maintains PowerShell functionality
		requiredElements := []string{
			"$maxRetries = 3",
			"$retryCount = 0",
			"\"mod\", \"download\"",
			"exit 1",
		}

		for _, element := range requiredElements {
			if !strings.Contains(windowsStep.Run, element) {
				t.Errorf("Windows step script missing required element: %s", element)
			}
		}
	})

	t.Run("Matrix strategy includes multiple OS platforms", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		var workflow CIWorkflow
		err = yaml.Unmarshal(content, &workflow)
		if err != nil {
			t.Fatalf("Failed to parse CI workflow: %v", err)
		}

		// Find the test job
		testJob, exists := workflow.Jobs["test"]
		if !exists {
			t.Fatal("Test job not found in CI workflow")
		}

		// Verify matrix strategy exists
		if testJob.Strategy == nil {
			t.Fatal("Test job has no matrix strategy")
		}

		matrix, exists := testJob.Strategy["matrix"]
		if !exists {
			t.Fatal("Matrix strategy not found in test job")
		}

		// Check for OS coverage
		matrixMap, ok := matrix.(map[string]interface{})
		if !ok {
			t.Fatal("Matrix is not a map")
		}

		osArray, exists := matrixMap["os"]
		if !exists {
			t.Fatal("OS array not found in matrix strategy")
		}

		osSlice, ok := osArray.([]interface{})
		if !ok {
			t.Fatal("OS is not an array")
		}

		// Verify both Unix and Windows platforms are covered
		hasUnix := false
		hasWindows := false
		for _, os := range osSlice {
			osStr, ok := os.(string)
			if !ok {
				continue
			}
			if strings.Contains(osStr, "ubuntu") || strings.Contains(osStr, "macos") {
				hasUnix = true
			}
			if strings.Contains(osStr, "windows") {
				hasWindows = true
			}
		}

		if !hasUnix {
			t.Error("Matrix strategy does not include Unix platforms")
		}
		if !hasWindows {
			t.Error("Matrix strategy does not include Windows platforms")
		}
	})
}

// TestCIWorkflowErrorHandlingScenarios tests various error scenarios
func TestCIWorkflowErrorHandlingScenarios(t *testing.T) {
	projectPath, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	ciWorkflowPath := filepath.Join(projectPath, ".github", "workflows", "ci.yml")

	t.Run("Unix script handles undefined variables", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// The 'set -u' flag should cause the script to exit if undefined variables are used
		if !strings.Contains(workflowContent, "set -euo pipefail") {
			t.Error("Unix script does not have undefined variable protection")
		}

		// Check that all variables used in the script are properly defined
		unixSectionStart := strings.Index(workflowContent, "Download dependencies (Unix)")
		unixSectionEnd := strings.Index(workflowContent[unixSectionStart:], "Download dependencies (Windows)")
		if unixSectionEnd == -1 {
			unixSectionEnd = len(workflowContent) - unixSectionStart
		}
		unixSection := workflowContent[unixSectionStart : unixSectionStart+unixSectionEnd]

		// Verify that variables are initialized before use
		requiredVarInits := []string{
			"timeout_duration=300",
			"max_retries=3",
			"retry_count=0",
		}

		for _, varInit := range requiredVarInits {
			if !strings.Contains(unixSection, varInit) {
				t.Errorf("Unix script missing variable initialization: %s", varInit)
			}
		}
	})

	t.Run("Unix script handles pipeline failures", func(t *testing.T) {
		content, err := os.ReadFile(ciWorkflowPath)
		if err != nil {
			t.Fatalf("Failed to read CI workflow file: %v", err)
		}

		workflowContent := string(content)

		// The 'set -o pipefail' flag should cause pipelines to fail if any command fails
		if !strings.Contains(workflowContent, "pipefail") {
			t.Error("Unix script does not have pipeline failure protection")
		}

		// Verify that the script has proper error checking around critical operations
		unixSectionStart := strings.Index(workflowContent, "Download dependencies (Unix)")
		unixSectionEnd := strings.Index(workflowContent[unixSectionStart:], "Download dependencies (Windows)")
		if unixSectionEnd == -1 {
			unixSectionEnd = len(workflowContent) - unixSectionStart
		}
		unixSection := workflowContent[unixSectionStart : unixSectionStart+unixSectionEnd]

		// Check for explicit error handling
		if !strings.Contains(unixSection, "exit 1") {
			t.Error("Unix script does not have explicit error exit")
		}
	})
}
