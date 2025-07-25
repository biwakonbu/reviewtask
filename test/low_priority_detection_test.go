package test

import (
	"testing"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// TestLowPriorityDetectionE2E tests the complete end-to-end workflow
// for detecting and handling low-priority comments using a mock Claude client.
func TestLowPriorityDetectionE2E(t *testing.T) {

	// Create configuration with low-priority patterns
	cfg := &config.Config{
		PriorityRules: config.PriorityRules{
			Critical: "Security vulnerabilities",
			High:     "Performance issues",
			Medium:   "Functional bugs",
			Low:      "Code style, naming conventions",
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			AutoPrioritize:      true,
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: config.AISettings{
			UserLanguage:         "English",
			OutputFormat:         "json",
			MaxTasksPerComment:   2,
			DeduplicationEnabled: false, // Disable for testing
			SimilarityThreshold:  0.8,
		},
	}

	// Create mock Claude client
	mockClient := &ai.MockClaudeClient{
		Responses: make(map[string]string),
	}

	// Set up mock responses for each comment (use 0 as placeholder for dynamic values)
	mockClient.Responses["nit: Variable naming is inconsistent"] = `[{
		"description": "Fix inconsistent variable naming",
		"origin_text": "nit: Variable naming is inconsistent",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "pending"
	}]`

	mockClient.Responses["This error handling is missing"] = `[{
		"description": "Add missing error handling to prevent crashes in production",
		"origin_text": "This error handling is missing - could cause crashes in production",
		"priority": "high",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "todo"
	}]`

	mockClient.Responses["MINOR: Variable names could be more descriptive"] = `[{
		"description": "Improve variable names for better readability",
		"origin_text": "MINOR: Variable names could be more descriptive",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "pending"
	}]`

	mockClient.Responses["suggestion: You could add unit tests"] = `[{
		"description": "Add unit tests for this function",
		"origin_text": "Good implementation!\nsuggestion: You could add unit tests for this function",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "pending"
	}]`

	// Create analyzer with mock client
	analyzer := ai.NewAnalyzerWithClient(cfg, mockClient)

	// Test case 1: Comments with various low-priority patterns
	reviews := []github.Review{
		{
			ID:          1,
			State:       "COMMENTED",
			Body:        "Overall review comment",
			SubmittedAt: "2023-01-01T00:00:00Z",
			Comments: []github.Comment{
				{
					ID:   101,
					Body: "nit: Consider using const instead of let for this variable",
					File: "main.js",
					Line: 10,
				},
				{
					ID:   102,
					Body: "This error handling is missing - could cause crashes in production",
					File: "main.js",
					Line: 20,
				},
				{
					ID:   103,
					Body: "MINOR: Variable names could be more descriptive",
					File: "utils.js",
					Line: 5,
				},
				{
					ID:   104,
					Body: "Good implementation!\nsuggestion: You could add unit tests for this function",
					File: "utils.js",
					Line: 15,
				},
			},
		},
	}

	// Generate tasks from reviews
	tasks, err := analyzer.GenerateTasks(reviews)
	if err != nil {
		t.Fatalf("Failed to generate tasks: %v", err)
	}

	// NOTE: This test assumes that SourceCommentID is preserved from the
	// GitHub comment ID to the generated task. This assumption is validated
	// in the unit tests at internal/ai/analyzer_test.go
	// Expected outcomes
	expectedStatuses := map[int64]string{
		101: "pending", // nit: pattern
		102: "todo",    // no pattern
		103: "pending", // MINOR: pattern
		104: "pending", // suggestion: pattern after newline
	}

	// Verify tasks
	for _, task := range tasks {
		t.Logf("Task: ID=%s, CommentID=%d, Status=%s, Origin=%q",
			task.ID, task.SourceCommentID, task.Status, task.OriginText)

		expectedStatus, exists := expectedStatuses[task.SourceCommentID]
		if !exists {
			t.Errorf("Unexpected task from comment ID %d", task.SourceCommentID)
			continue
		}

		if task.Status != expectedStatus {
			t.Errorf("Comment %d: Expected status %q, got %q (origin: %q)",
				task.SourceCommentID, expectedStatus, task.Status, task.OriginText)
		}
	}
}

// TestConfigurationBackwardCompatibility ensures the feature works
// when configuration fields are missing (backward compatibility)
func TestConfigurationBackwardCompatibility(t *testing.T) {
	// Create minimal configuration without low-priority fields
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:  "todo",
			AutoPrioritize: true,
			// LowPriorityPatterns and LowPriorityStatus are intentionally not set
		},
		AISettings: config.AISettings{
			UserLanguage: "English",
			OutputFormat: "json",
		},
	}

	// Create mock Claude client
	mockClient := &ai.MockClaudeClient{
		Responses: make(map[string]string),
	}
	
	// Set up mock response for the test
	mockClient.Responses["nit: Fix indentation"] = `[{
		"description": "Fix indentation issues",
		"origin_text": "nit: Fix indentation",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "todo"
	}]`
	
	analyzer := ai.NewAnalyzerWithClient(cfg, mockClient)

	// Create review with "nit:" comment
	reviews := []github.Review{
		{
			ID:    1,
			State: "COMMENTED",
			Comments: []github.Comment{
				{
					ID:   201,
					Body: "nit: Fix indentation",
					File: "test.go",
					Line: 10,
				},
			},
		},
	}

	// Generate tasks
	tasks, err := analyzer.GenerateTasks(reviews)
	if err != nil {
		t.Fatalf("Failed to generate tasks: %v", err)
	}

	// Should use default status when low-priority config is missing
	if len(tasks) > 0 && tasks[0].Status != "todo" {
		t.Errorf("Expected default status 'todo' when low-priority config missing, got %q", tasks[0].Status)
	}
}

// TestComplexCommentPatterns tests various edge cases and complex patterns.
// NOTE: This test also uses real Analyzer with Claude Code CLI dependency.
// See TestLowPriorityDetectionE2E comments for architectural notes.
func TestComplexCommentPatterns(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "style:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: config.AISettings{
			UserLanguage: "English",
			OutputFormat: "json",
		},
	}

	// Create mock Claude client with dynamic response handling
	mockClient := &ai.MockClaudeClient{
		Responses: make(map[string]string),
	}

	// Set up responses for each test case (use 0 as placeholder for dynamic values)
	mockClient.Responses["nit:   Extra spaces should still match"] = `[{
		"description": "Fix extra spaces issue",
		"origin_text": "nit:   Extra spaces should still match",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "pending"
	}]`

	mockClient.Responses["error handling in this function needs improvement"] = `[{
		"description": "Improve error handling in function",
		"origin_text": "The error handling in this function needs improvement. It should return proper error messages instead of generic ones. Here's an example of what NOT to do:\n` + "```" + `\n// nit: this is in a code block\nreturn fmt.Errorf(\"error\")\n` + "```" + `\nPlease update the error handling to include context about what operation failed.",
		"priority": "medium",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "todo"
	}]`

	mockClient.Responses["style: Fix formatting"] = `[{
		"description": "Fix formatting issues",
		"origin_text": "style: Fix formatting\nnit: Also fix indentation",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "pending"
	}, {
		"description": "Fix indentation",
		"origin_text": "style: Fix formatting\nnit: Also fix indentation",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 1,
		"status": "pending"
	}]`

	mockClient.Responses["nit: Fix this 修正してください"] = `[{
		"description": "Fix the issue mentioned (修正してください)",
		"origin_text": "nit: Fix this 修正してください",
		"priority": "low",
		"source_review_id": 0,
		"source_comment_id": 0,
		"file": "",
		"line": 0,
		"task_index": 0,
		"status": "pending"
	}]`

	analyzer := ai.NewAnalyzerWithClient(cfg, mockClient)

	testCases := []struct {
		name           string
		comment        github.Comment
		expectedStatus string
	}{
		{
			name: "Pattern with extra spaces",
			comment: github.Comment{
				ID:   301,
				Body: "nit:   Extra spaces should still match",
				File: "file.go",
				Line: 1,
			},
			expectedStatus: "pending",
		},
		{
			name: "Pattern in code block should not match",
			comment: github.Comment{
				ID:   302,
				Body: "The error handling in this function needs improvement. It should return proper error messages instead of generic ones. Here's an example of what NOT to do:\n```\n// nit: this is in a code block\nreturn fmt.Errorf(\"error\")\n```\nPlease update the error handling to include context about what operation failed.",
				File: "file.go",
				Line: 2,
			},
			expectedStatus: "todo",
		},
		{
			name: "Multiple patterns in same comment",
			comment: github.Comment{
				ID:   303,
				Body: "style: Fix formatting\nnit: Also fix indentation",
				File: "file.go",
				Line: 3,
			},
			expectedStatus: "pending",
		},
		{
			name: "Unicode in comment",
			comment: github.Comment{
				ID:   304,
				Body: "nit: Fix this 修正してください",
				File: "file.go",
				Line: 4,
			},
			expectedStatus: "pending",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reviews := []github.Review{
				{
					ID:       1,
					State:    "COMMENTED",
					Comments: []github.Comment{tc.comment},
				},
			}

			tasks, err := analyzer.GenerateTasks(reviews)
			if err != nil {
				t.Fatalf("Failed to generate tasks: %v", err)
			}

			if len(tasks) == 0 {
				t.Fatal("No tasks generated")
			}

			if tasks[0].Status != tc.expectedStatus {
				t.Errorf("Expected status %q, got %q for comment: %q",
					tc.expectedStatus, tasks[0].Status, tc.comment.Body)
			}
		})
	}
}

// TestStorageIntegration verifies that low-priority tasks are correctly
// stored and retrieved with their assigned status
func TestStorageIntegration(t *testing.T) {
	// Create storage manager
	store := storage.NewManager()

	// Create tasks with different statuses
	now := "2023-01-01T00:00:00Z"
	tasks := []storage.Task{
		{
			ID:              "test-1",
			Description:     "Fix indentation",
			OriginText:      "nit: Fix the indentation",
			Priority:        "low",
			Status:          "pending", // Low-priority status
			SourceCommentID: 1,
			File:            "test.go",
			Line:            10,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "test-2",
			Description:     "Fix security issue",
			OriginText:      "Critical security vulnerability here",
			Priority:        "critical",
			Status:          "todo", // Default status
			SourceCommentID: 2,
			File:            "auth.go",
			Line:            20,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	// Save tasks
	if err := store.SaveTasks(1, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Retrieve and verify
	savedTasks, err := store.GetTasksByPR(1)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	if len(savedTasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(savedTasks))
	}

	// Verify statuses are preserved
	statusMap := make(map[string]string)
	for _, task := range savedTasks {
		statusMap[task.ID] = task.Status
	}

	if statusMap["test-1"] != "pending" {
		t.Errorf("Expected 'pending' status for low-priority task, got %q", statusMap["test-1"])
	}

	if statusMap["test-2"] != "todo" {
		t.Errorf("Expected 'todo' status for normal task, got %q", statusMap["test-2"])
	}
}
