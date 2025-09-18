package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

// GitHubWorkflow represents a GitHub Actions workflow file structure
type GitHubWorkflow struct {
	Name string `yaml:"name"`
	On   struct {
		PullRequest struct {
			Branches    []string `yaml:"branches"`
			PathsIgnore []string `yaml:"paths-ignore"`
		} `yaml:"pull_request"`
	} `yaml:"on"`
	Jobs map[string]struct {
		RunsOn string `yaml:"runs-on"`
		Steps  []struct {
			Name string `yaml:"name"`
			Uses string `yaml:"uses,omitempty"`
			With struct {
				GoVersion string `yaml:"go-version,omitempty"`
			} `yaml:"with,omitempty"`
			Run string `yaml:"run,omitempty"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// TestPRReleaseTestWorkflowStructure tests that the GitHub Actions workflow has the correct structure
func TestPRReleaseTestWorkflowStructure(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	workflowPath := filepath.Join(projectRoot, ".github", "workflows", "pr-release-test.yml")

	// Check that workflow file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Fatalf("GitHub Actions workflow file not found: %s", workflowPath)
	}

	// Read and parse workflow file
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	var workflow GitHubWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Failed to parse workflow YAML: %v", err)
	}

	// Test workflow name
	expectedName := "PR Release Build Test"
	if workflow.Name != expectedName {
		t.Errorf("Expected workflow name %q, got %q", expectedName, workflow.Name)
	}

	// Test trigger conditions
	if len(workflow.On.PullRequest.Branches) == 0 || workflow.On.PullRequest.Branches[0] != "main" {
		t.Errorf("Expected workflow to trigger on main branch, got %v", workflow.On.PullRequest.Branches)
	}

	// Test paths-ignore
	expectedPathsIgnore := []string{"**.md", "docs/**"}
	if len(workflow.On.PullRequest.PathsIgnore) != len(expectedPathsIgnore) {
		t.Errorf("Expected %d paths-ignore patterns, got %d", len(expectedPathsIgnore), len(workflow.On.PullRequest.PathsIgnore))
	}

	// Test job exists
	job, exists := workflow.Jobs["test-release-build"]
	if !exists {
		t.Fatalf("Expected job 'test-release-build' not found")
	}

	// Test runner
	if job.RunsOn != "ubuntu-latest" {
		t.Errorf("Expected runner 'ubuntu-latest', got %q", job.RunsOn)
	}

	// Test steps
	expectedSteps := []string{
		"Checkout code",
		"Setup Go",
		"Verify go.mod",
		"Test cross-platform builds",
		"Test version embedding",
		"Simulate release preparation",
	}

	if len(job.Steps) != len(expectedSteps) {
		t.Errorf("Expected %d steps, got %d", len(expectedSteps), len(job.Steps))
	}

	for i, expectedStep := range expectedSteps {
		if i >= len(job.Steps) {
			t.Errorf("Missing step: %s", expectedStep)
			continue
		}
		if job.Steps[i].Name != expectedStep {
			t.Errorf("Expected step %d name %q, got %q", i, expectedStep, job.Steps[i].Name)
		}
	}
}

// TestWorkflowCommandsValidity tests that the commands in the workflow are valid
func TestWorkflowCommandsValidity(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	workflowPath := filepath.Join(projectRoot, ".github", "workflows", "pr-release-test.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	workflowContent := string(data)

	// Test that required scripts are referenced correctly
	expectedScripts := []string{
		"./scripts/build.sh test",
		"./scripts/version.sh current",
		"./scripts/release.sh prepare patch --dry-run",
	}

	for _, script := range expectedScripts {
		if !strings.Contains(workflowContent, script) {
			t.Errorf("Expected workflow to contain command %q", script)
		}
	}

	// Test that version embedding includes all required variables
	expectedLdflags := []string{
		"-X main.version=",
		"-X main.commitHash=",
		"-X main.buildDate=",
	}

	for _, ldflag := range expectedLdflags {
		if !strings.Contains(workflowContent, ldflag) {
			t.Errorf("Expected workflow to contain ldflags %q", ldflag)
		}
	}
}

// TestRequiredScriptsExist tests that all scripts referenced in the workflow exist
func TestRequiredScriptsExist(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	requiredScripts := []string{
		"scripts/build.sh",
		"scripts/version.sh",
		"scripts/release.sh",
	}

	for _, script := range requiredScripts {
		scriptPath := filepath.Join(projectRoot, script)
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			t.Errorf("Required script not found: %s", scriptPath)
		}
	}
}

// TestWorkflowYamlSyntax tests that the workflow YAML is syntactically correct
func TestWorkflowYamlSyntax(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	workflowPath := filepath.Join(projectRoot, ".github", "workflows", "pr-release-test.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}

	// Parse as generic YAML to check syntax
	var yamlData interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		t.Fatalf("Workflow YAML syntax error: %v", err)
	}
}

// TestWorkflowIntegrationWithExistingScripts tests that the workflow integrates properly with existing scripts
func TestWorkflowIntegrationWithExistingScripts(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	// Test that release.sh supports --dry-run flag for prepare command
	releaseScriptPath := filepath.Join(projectRoot, "scripts", "release.sh")
	data, err := os.ReadFile(releaseScriptPath)
	if err != nil {
		t.Fatalf("Failed to read release script: %v", err)
	}

	scriptContent := string(data)

	// Check that the script has dry-run support
	if !strings.Contains(scriptContent, "--dry-run") {
		t.Error("release.sh script should support --dry-run flag")
	}

	// Check that prepare_release function accepts dry_run parameter
	if !strings.Contains(scriptContent, "local dry_run=${2:-false}") {
		t.Error("prepare_release function should accept dry_run parameter")
	}

	// Test that build.sh has test command
	buildScriptPath := filepath.Join(projectRoot, "scripts", "build.sh")
	data, err = os.ReadFile(buildScriptPath)
	if err != nil {
		t.Fatalf("Failed to read build script: %v", err)
	}

	buildContent := string(data)
	if !strings.Contains(buildContent, "test_cross_compile") {
		t.Error("build.sh script should have test_cross_compile function")
	}
}
