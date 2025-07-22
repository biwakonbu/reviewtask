package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReviewTaskWorkflowDocumentationUpdate validates that the review-task-workflow.md
// documentation has been updated to reflect current reviewtask tool specifications
func TestReviewTaskWorkflowDocumentationUpdate(t *testing.T) {
	// Note: Parallel execution disabled due to getwd issues in CI environment

	// Use a relative path from the test directory
	// This works whether tests are run from project root or test directory
	workflowPath := filepath.Join("..", ".claude", "commands", "pr-review", "review-task-workflow.md")

	// If the file doesn't exist at the relative path, try from current directory
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		workflowPath = filepath.Join(".", ".claude", "commands", "pr-review", "review-task-workflow.md")
	}

	// Read the documentation file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow documentation: %v", err)
	}

	docContent := string(content)

	// Test 1: Verify current command syntax is documented
	t.Run("CurrentCommandSyntax", func(t *testing.T) {
		requiredCommands := []string{
			"reviewtask",
			"reviewtask status",
			"reviewtask show",
			"reviewtask show <task-id>",
			"reviewtask update <task-id> <status>",
		}

		for _, cmd := range requiredCommands {
			if !strings.Contains(docContent, cmd) {
				t.Errorf("Documentation missing command syntax: %s", cmd)
			}
		}
	})

	// Test 2: Verify task priority system is documented
	t.Run("TaskPrioritySystem", func(t *testing.T) {
		priorities := []string{"critical", "high", "medium", "low"}

		// Check for priority system section
		if !strings.Contains(docContent, "Task Priority System") {
			t.Error("Documentation missing Task Priority System section")
		}

		for _, priority := range priorities {
			if !strings.Contains(docContent, priority) {
				t.Errorf("Documentation missing priority level: %s", priority)
			}
		}
	})

	// Test 3: Verify status options are documented
	t.Run("StatusOptions", func(t *testing.T) {
		statusOptions := []string{"todo", "doing", "done", "pending", "cancel"}

		for _, status := range statusOptions {
			if !strings.Contains(docContent, status) {
				t.Errorf("Documentation missing status option: %s", status)
			}
		}
	})

	// Test 4: Verify AI processing features are documented
	t.Run("AIProcessingFeatures", func(t *testing.T) {
		aiFeatures := []string{
			"AI Processing and Task Generation",
			"Automatic Task Creation",
			"Task Deduplication",
			"Priority Assignment",
			"Task Validation",
		}

		for _, feature := range aiFeatures {
			if !strings.Contains(docContent, feature) {
				t.Errorf("Documentation missing AI feature: %s", feature)
			}
		}
	})

	// Test 5: Verify current tool features are documented
	t.Run("CurrentToolFeatures", func(t *testing.T) {
		toolFeatures := []string{
			"Multi-source Authentication",
			"Task Management",
			"AI-Enhanced Analysis",
			"Progress Tracking",
		}

		for _, feature := range toolFeatures {
			if !strings.Contains(docContent, feature) {
				t.Errorf("Documentation missing tool feature: %s", feature)
			}
		}
	})

	// Test 6: Verify realistic example outputs are included
	t.Run("ExampleOutputs", func(t *testing.T) {
		if !strings.Contains(docContent, "Example Tool Output") {
			t.Error("Documentation missing example tool output section")
		}

		// Check for example status table
		if !strings.Contains(docContent, "PR Review Tasks Status") {
			t.Error("Documentation missing example status output")
		}

		// Check for example show output
		if !strings.Contains(docContent, "Task ID: task-001") {
			t.Error("Documentation missing example show command output")
		}
	})

	// Test 7: Verify priority-based processing is mentioned
	t.Run("PriorityBasedProcessing", func(t *testing.T) {
		priorityProcessing := []string{
			"prioritized by: critical → high → medium → low",
			"Priority-Based Processing",
		}

		for _, phrase := range priorityProcessing {
			if !strings.Contains(docContent, phrase) {
				t.Errorf("Documentation missing priority processing phrase: %s", phrase)
			}
		}
	})
}

// TestDocumentationStructure validates the overall structure of the documentation
func TestDocumentationStructure(t *testing.T) {
	// Note: Parallel execution disabled due to getwd issues in CI environment

	// Use a relative path from the test directory
	// This works whether tests are run from project root or test directory
	workflowPath := filepath.Join("..", ".claude", "commands", "pr-review", "review-task-workflow.md")

	// If the file doesn't exist at the relative path, try from current directory
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		workflowPath = filepath.Join(".", ".claude", "commands", "pr-review", "review-task-workflow.md")
	}

	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read workflow documentation: %v", err)
	}

	docContent := string(content)

	// Test required sections exist
	requiredSections := []string{
		"Available Commands",
		"Task Priority System",
		"Initial Setup",
		"Workflow Steps",
		"Task Classification Guidelines",
		"AI Processing and Task Generation",
		"Current Tool Features",
		"Important Notes",
		"Example Tool Output",
	}

	for _, section := range requiredSections {
		if !strings.Contains(docContent, section) {
			t.Errorf("Documentation missing required section: %s", section)
		}
	}
}
