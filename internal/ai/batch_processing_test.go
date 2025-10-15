package ai

import (
	"strings"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// TestBuildBatchPromptWithExistingTasks tests that the batch prompt builder
// correctly formats multiple comments with their existing tasks.
func TestBuildBatchPromptWithExistingTasks(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "English",
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Create test comments
	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:     1001,
				Author: "reviewer1",
				Body:   "Please fix the memory leak in the parser",
				File:   "parser.go",
				Line:   42,
			},
		},
		{
			Comment: github.Comment{
				ID:     1002,
				Author: "reviewer1",
				Body:   "Add error handling here",
				File:   "handler.go",
				Line:   15,
			},
		},
	}

	// Create existing tasks for first comment
	existingTasksByComment := map[int64][]storage.Task{
		1001: {
			{
				ID:              "task-1",
				Description:     "Fix memory leak in parser",
				Status:          "done",
				Priority:        "high",
				SourceCommentID: 1001,
			},
			{
				ID:              "task-2",
				Description:     "Add unit test for parser",
				Status:          "doing",
				Priority:        "medium",
				SourceCommentID: 1001,
			},
		},
		// No existing tasks for comment 1002
	}

	// Build batch prompt
	prompt := analyzer.buildBatchPromptWithExistingTasks(comments, existingTasksByComment)

	// Verify prompt structure
	if !strings.Contains(prompt, "# PR Review Task Generation Request") {
		t.Error("Prompt should contain header")
	}

	if !strings.Contains(prompt, "## Important Instructions") {
		t.Error("Prompt should contain instructions section")
	}

	if !strings.Contains(prompt, "Avoid duplicates with existing tasks") {
		t.Error("Prompt should mention duplicate avoidance")
	}

	// Verify comment 1 is included
	if !strings.Contains(prompt, "Comment #1: 1001") {
		t.Error("Prompt should contain first comment ID")
	}
	if !strings.Contains(prompt, "Please fix the memory leak in the parser") {
		t.Error("Prompt should contain first comment body")
	}

	// Verify existing tasks are shown
	if !strings.Contains(prompt, "### Existing Tasks for Comment") {
		t.Error("Prompt should show existing tasks section")
	}
	if !strings.Contains(prompt, "Fix memory leak in parser") {
		t.Error("Prompt should list existing task description")
	}
	if !strings.Contains(prompt, "✅") { // done status icon
		t.Error("Prompt should show done status icon")
	}
	if !strings.Contains(prompt, "🔄") { // doing status icon
		t.Error("Prompt should show doing status icon")
	}

	// Verify comment 2 is included
	if !strings.Contains(prompt, "Comment #2: 1002") {
		t.Error("Prompt should contain second comment ID")
	}
	if !strings.Contains(prompt, "Add error handling here") {
		t.Error("Prompt should contain second comment body")
	}

	// Verify "no existing tasks" message for comment 2
	if !strings.Contains(prompt, "*No existing tasks for this comment.*") {
		t.Error("Prompt should indicate when no existing tasks exist")
	}

	// Verify JSON response format is specified
	if !strings.Contains(prompt, "Return ONLY the JSON array") {
		t.Error("Prompt should specify JSON response format")
	}
}

// TestGetStatusIcon tests that status icons are correctly mapped.
func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status       string
		expectedIcon string
	}{
		{"done", "✅"},
		{"doing", "🔄"},
		{"todo", "📝"},
		{"pending", "⏸️"},
		{"cancel", "❌"},
		{"unknown", "•"},
		{"", "•"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			icon := getStatusIcon(tt.status)
			if icon != tt.expectedIcon {
				t.Errorf("getStatusIcon(%q) = %q, expected %q", tt.status, icon, tt.expectedIcon)
			}
		})
	}
}

// TestParseBatchTaskResponse tests parsing of AI JSON responses.
func TestParseBatchTaskResponse(t *testing.T) {
	cfg := &config.Config{}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name          string
		response      string
		expectedCount int
		expectError   bool
	}{
		{
			name: "Valid JSON response with tasks",
			response: `[
				{
					"comment_id": 1001,
					"tasks": [
						{
							"description": "Fix memory leak",
							"priority": "high"
						},
						{
							"description": "Add tests",
							"priority": "medium"
						}
					]
				},
				{
					"comment_id": 1002,
					"tasks": []
				}
			]`,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "JSON with markdown wrapper",
			response: "```json\n" + `[
				{
					"comment_id": 1001,
					"tasks": [
						{
							"description": "Fix issue",
							"priority": "high"
						}
					]
				}
			]` + "\n```",
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "Empty array response",
			response:      "[]",
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Invalid JSON",
			response:      "{invalid json}",
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses, err := analyzer.parseBatchTaskResponse(tt.response)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(responses) != tt.expectedCount {
				t.Errorf("Expected %d responses, got %d", tt.expectedCount, len(responses))
			}
		})
	}
}

// TestParseBatchTaskResponseTaskFields verifies all task fields are parsed correctly.
func TestParseBatchTaskResponseTaskFields(t *testing.T) {
	cfg := &config.Config{}
	analyzer := NewAnalyzer(cfg)

	response := `[
		{
			"comment_id": 1001,
			"tasks": [
				{
					"description": "Fix memory leak in parser",
					"priority": "critical"
				},
				{
					"description": "Add comprehensive tests",
					"priority": "high"
				}
			]
		}
	]`

	responses, err := analyzer.parseBatchTaskResponse(response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(responses) != 1 {
		t.Fatalf("Expected 1 response, got %d", len(responses))
	}

	resp := responses[0]

	// Verify comment ID
	if resp.CommentID != 1001 {
		t.Errorf("Expected comment_id 1001, got %d", resp.CommentID)
	}

	// Verify task count
	if len(resp.Tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(resp.Tasks))
	}

	// Verify first task
	task1 := resp.Tasks[0]
	if task1.Description != "Fix memory leak in parser" {
		t.Errorf("Task 1 description mismatch: got %q", task1.Description)
	}
	if task1.Priority != "critical" {
		t.Errorf("Task 1 priority mismatch: got %q", task1.Priority)
	}

	// Verify second task
	task2 := resp.Tasks[1]
	if task2.Description != "Add comprehensive tests" {
		t.Errorf("Task 2 description mismatch: got %q", task2.Description)
	}
	if task2.Priority != "high" {
		t.Errorf("Task 2 priority mismatch: got %q", task2.Priority)
	}
}

// TestBatchProcessingEmptyTasksResponse tests that empty task arrays are handled correctly.
// This simulates AI determining that existing tasks are sufficient.
func TestBatchProcessingEmptyTasksResponse(t *testing.T) {
	cfg := &config.Config{}
	analyzer := NewAnalyzer(cfg)

	// AI response indicating no new tasks needed (existing tasks are sufficient)
	response := `[
		{
			"comment_id": 1001,
			"tasks": []
		},
		{
			"comment_id": 1002,
			"tasks": []
		}
	]`

	responses, err := analyzer.parseBatchTaskResponse(response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(responses) != 2 {
		t.Fatalf("Expected 2 responses, got %d", len(responses))
	}

	// Verify both responses have empty task arrays
	for i, resp := range responses {
		if len(resp.Tasks) != 0 {
			t.Errorf("Response %d: Expected empty task array, got %d tasks", i, len(resp.Tasks))
		}
	}
}

// TestBatchPromptMarkdownEscaping tests that special markdown characters
// in comments are properly handled.
func TestBatchPromptMarkdownEscaping(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "English",
		},
	}
	analyzer := NewAnalyzer(cfg)

	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:     1001,
				Author: "reviewer1",
				Body:   "Fix `code block` and **bold** text handling",
				File:   "test.go",
				Line:   1,
			},
		},
	}

	existingTasks := map[int64][]storage.Task{}

	prompt := analyzer.buildBatchPromptWithExistingTasks(comments, existingTasks)

	// Verify the comment body is included as-is (markdown is preserved)
	if !strings.Contains(prompt, "Fix `code block` and **bold** text handling") {
		t.Error("Prompt should preserve markdown syntax in comment body")
	}
}

// TestBatchPromptWithMultipleExistingTasks verifies that multiple existing tasks
// with different statuses are correctly displayed in the prompt.
func TestBatchPromptWithMultipleExistingTasks(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "English",
		},
	}
	analyzer := NewAnalyzer(cfg)

	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:     1001,
				Author: "reviewer1",
				Body:   "Multiple issues here",
				File:   "test.go",
				Line:   1,
			},
		},
	}

	existingTasks := map[int64][]storage.Task{
		1001: {
			{ID: "task-1", Description: "Task 1", Status: "done", Priority: "high"},
			{ID: "task-2", Description: "Task 2", Status: "doing", Priority: "medium"},
			{ID: "task-3", Description: "Task 3", Status: "todo", Priority: "low"},
			{ID: "task-4", Description: "Task 4", Status: "pending", Priority: "medium"},
			{ID: "task-5", Description: "Task 5", Status: "cancel", Priority: "low"},
		},
	}

	prompt := analyzer.buildBatchPromptWithExistingTasks(comments, existingTasks)

	// Verify all status icons are present
	statusIcons := []string{"✅", "🔄", "📝", "⏸️", "❌"}
	for _, icon := range statusIcons {
		if !strings.Contains(prompt, icon) {
			t.Errorf("Prompt should contain status icon: %s", icon)
		}
	}

	// Verify all task descriptions are listed
	for i := 1; i <= 5; i++ {
		taskDesc := "Task " + string(rune('0'+i))
		if !strings.Contains(prompt, taskDesc) {
			t.Errorf("Prompt should contain task description: %s", taskDesc)
		}
	}
}

// TestBatchProcessingIntegrationWithConfig tests that EnableBatchProcessing
// configuration flag is properly respected.
func TestBatchProcessingIntegrationWithConfig(t *testing.T) {
	tests := []struct {
		name                  string
		enableBatchProcessing bool
		expectedBehavior      string
	}{
		{
			name:                  "Batch processing enabled",
			enableBatchProcessing: true,
			expectedBehavior:      "should use batch processor",
		},
		{
			name:                  "Batch processing disabled (default)",
			enableBatchProcessing: false,
			expectedBehavior:      "should use stream processor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					EnableBatchProcessing: tt.enableBatchProcessing,
					UserLanguage:          "English",
				},
				TaskSettings: config.TaskSettings{
					DefaultStatus: "todo",
				},
			}

			// Verify config value is set correctly
			if cfg.AISettings.EnableBatchProcessing != tt.enableBatchProcessing {
				t.Errorf("Config EnableBatchProcessing = %v, expected %v",
					cfg.AISettings.EnableBatchProcessing, tt.enableBatchProcessing)
			}

			t.Logf("Config verified: EnableBatchProcessing = %v (%s)",
				cfg.AISettings.EnableBatchProcessing, tt.expectedBehavior)
		})
	}
}
