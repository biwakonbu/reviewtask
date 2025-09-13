package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reviewtask/internal/github"
)

// TestTasksFileOperations tests TasksFile struct operations
func TestTasksFileOperations(t *testing.T) {
	scenarios := []struct {
		name   string
		setup  func() *TasksFile
		verify func(t *testing.T, tf *TasksFile)
	}{
		{
			name: "新規タスクファイル作成",
			setup: func() *TasksFile {
				return &TasksFile{
					Tasks: []Task{
						{
							ID:              "TSK-001",
							Description:     "バグを修正する",
							Status:          "todo",
							Priority:        "high",
							PRNumber:        100,
							SourceCommentID: 123456,
						},
					},
					GeneratedAt: time.Now().Format(time.RFC3339),
				}
			},
			verify: func(t *testing.T, tf *TasksFile) {
				if len(tf.Tasks) != 1 {
					t.Errorf("Expected 1 task, got %d", len(tf.Tasks))
				}
				if tf.Tasks[0].Description != "バグを修正する" {
					t.Error("Japanese content not preserved")
				}
			},
		},
		{
			name: "複数タスクのファイル",
			setup: func() *TasksFile {
				return &TasksFile{
					Tasks: []Task{
						{ID: "1", Status: "todo", Priority: "critical", PRNumber: 200},
						{ID: "2", Status: "doing", Priority: "high", PRNumber: 200},
						{ID: "3", Status: "done", Priority: "medium", PRNumber: 200},
						{ID: "4", Status: "pending", Priority: "low", PRNumber: 200},
					},
					GeneratedAt: time.Now().Format(time.RFC3339),
				}
			},
			verify: func(t *testing.T, tf *TasksFile) {
				if len(tf.Tasks) != 4 {
					t.Errorf("Expected 4 tasks, got %d", len(tf.Tasks))
				}
				// Verify each status exists
				statuses := make(map[string]bool)
				for _, task := range tf.Tasks {
					statuses[task.Status] = true
				}
				expectedStatuses := []string{"todo", "doing", "done", "pending"}
				for _, status := range expectedStatuses {
					if !statuses[status] {
						t.Errorf("Missing status: %s", status)
					}
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tf := scenario.setup()
			scenario.verify(t, tf)
		})
	}
}

// TestTaskJSONSerialization tests JSON serialization of tasks
func TestTaskJSONSerialization(t *testing.T) {
	task := Task{
		ID:              "TSK-100",
		Description:     "メモリリークを修正\n複数行の\n説明文",
		OriginText:      "Original review comment",
		Status:          "doing",
		Priority:        "critical",
		PRNumber:        500,
		SourceCommentID: 789012,
		File:            "src/main.go",
		Line:            100,
		CreatedAt:       time.Now().Format(time.RFC3339),
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize task: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	expectedFields := []string{
		"id",
		"description",
		"status",
		"priority",
		"pr_number",
		"source_comment_id",
		"file",
		"line",
	}
	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON missing field: %s", field)
		}
	}

	// Deserialize back
	var decoded Task
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to deserialize task: %v", err)
	}

	// Verify content preserved
	if decoded.Description != task.Description {
		t.Error("Description not preserved through serialization")
	}
	if decoded.File != "src/main.go" {
		t.Error("File not preserved")
	}
	if decoded.Line != 100 {
		t.Error("Line not preserved")
	}
}

// TestStorageWorkflows tests complete storage workflows
func TestStorageWorkflows(t *testing.T) {
	scenarios := []struct {
		name  string
		steps []func(t *testing.T, tempDir string)
	}{
		{
			name: "PR情報保存と読み込みフロー",
			steps: []func(t *testing.T, tempDir string){
				// Step 1: Create storage directory
				func(t *testing.T, tempDir string) {
					storageDir := filepath.Join(tempDir, ".pr-review")
					err := os.MkdirAll(storageDir, 0755)
					if err != nil {
						t.Fatalf("Failed to create storage dir: %v", err)
					}
				},
				// Step 2: Save PR info
				func(t *testing.T, tempDir string) {
					prDir := filepath.Join(tempDir, ".pr-review", "PR-100")
					err := os.MkdirAll(prDir, 0755)
					if err != nil {
						t.Fatalf("Failed to create PR dir: %v", err)
					}

					prInfo := github.PRInfo{
						Number:    100,
						Title:     "新機能追加",
						State:     "open",
						Author:    "developer",
						CreatedAt: time.Now().Format(time.RFC3339),
						UpdatedAt: time.Now().Format(time.RFC3339),
					}

					data, err := json.MarshalIndent(prInfo, "", "  ")
					if err != nil {
						t.Fatalf("Failed to marshal PR info: %v", err)
					}
					err = os.WriteFile(filepath.Join(prDir, "pr_info.json"), data, 0644)
					if err != nil {
						t.Fatalf("Failed to save PR info: %v", err)
					}
				},
				// Step 3: Verify PR info
				func(t *testing.T, tempDir string) {
					prFile := filepath.Join(tempDir, ".pr-review", "PR-100", "pr_info.json")
					data, err := os.ReadFile(prFile)
					if err != nil {
						t.Fatalf("Failed to read PR info: %v", err)
					}

					var prInfo github.PRInfo
					err = json.Unmarshal(data, &prInfo)
					if err != nil {
						t.Fatalf("Failed to parse PR info: %v", err)
					}

					if prInfo.Title != "新機能追加" {
						t.Error("PR title not preserved")
					}
				},
			},
		},
		{
			name: "タスク更新とマージフロー",
			steps: []func(t *testing.T, tempDir string){
				// Step 1: Create initial tasks
				func(t *testing.T, tempDir string) {
					prDir := filepath.Join(tempDir, ".pr-review", "PR-200")
					if err := os.MkdirAll(prDir, 0755); err != nil {
						t.Fatalf("Failed to create PR directory: %v", err)
					}

					tasksFile := TasksFile{
						Tasks: []Task{
							{ID: "1", Description: "Task 1", Status: "todo", Priority: "high", PRNumber: 200, SourceCommentID: 1001},
							{ID: "2", Description: "Task 2", Status: "todo", Priority: "medium", PRNumber: 200, SourceCommentID: 1001},
						},
						GeneratedAt: time.Now().Format(time.RFC3339),
					}

					data, err := json.MarshalIndent(tasksFile, "", "  ")
					if err != nil {
						t.Fatalf("Failed to marshal tasks file: %v", err)
					}
					if err := os.WriteFile(filepath.Join(prDir, "tasks.json"), data, 0644); err != nil {
						t.Fatalf("Failed to write tasks file: %v", err)
					}
				},
				// Step 2: Update task status
				func(t *testing.T, tempDir string) {
					tasksFile := filepath.Join(tempDir, ".pr-review", "PR-200", "tasks.json")
					data, _ := os.ReadFile(tasksFile)

					var tf TasksFile
					json.Unmarshal(data, &tf)

					// Update first task to doing
					tf.Tasks[0].Status = "doing"
					tf.Tasks[0].UpdatedAt = time.Now().Format(time.RFC3339)

					newData, _ := json.MarshalIndent(tf, "", "  ")
					os.WriteFile(tasksFile, newData, 0644)
				},
				// Step 3: Verify update
				func(t *testing.T, tempDir string) {
					tasksFile := filepath.Join(tempDir, ".pr-review", "PR-200", "tasks.json")
					data, _ := os.ReadFile(tasksFile)

					var tf TasksFile
					json.Unmarshal(data, &tf)

					if tf.Tasks[0].Status != "doing" {
						t.Error("Task status not updated")
					}
					if tf.Tasks[1].Status != "todo" {
						t.Error("Other task status changed unexpectedly")
					}
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "storage_test")
			if err != nil {
				t.Fatalf("Failed to create temporary directory for storage test: %v", err)
			}
			defer os.RemoveAll(tempDir)

			for _, step := range scenario.steps {
				step(t, tempDir)
			}
		})
	}
}

// TestTaskFileFieldHandling tests task file and line field handling
func TestTaskFileFieldHandling(t *testing.T) {
	tests := []struct {
		name   string
		task   Task
		verify func(t *testing.T, task Task)
	}{
		{
			name: "基本的なファイル情報",
			task: Task{
				ID:          "test-1",
				Description: "Fix bug",
				File:        "src/components/Button.tsx",
				Line:        42,
				Status:      "todo",
				Priority:    "high",
			},
			verify: func(t *testing.T, task Task) {
				if task.File != "src/components/Button.tsx" {
					t.Error("File not correct")
				}
				if task.Line != 42 {
					t.Error("Line not correct")
				}
			},
		},
		{
			name: "日本語パスのファイル",
			task: Task{
				ID:          "test-2",
				Description: "Update docs",
				File:        "ドキュメント/設計書.md",
				Line:        1,
				Status:      "todo",
				Priority:    "medium",
			},
			verify: func(t *testing.T, task Task) {
				if task.File != "ドキュメント/設計書.md" {
					t.Error("Japanese path not preserved")
				}
			},
		},
		{
			name: "ファイル情報なし",
			task: Task{
				ID:          "test-3",
				Description: "General task",
				Status:      "todo",
				Priority:    "low",
			},
			verify: func(t *testing.T, task Task) {
				if task.File != "" {
					t.Error("Expected empty file")
				}
				if task.Line != 0 {
					t.Error("Expected zero line")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, tt.task)

			// Test JSON serialization
			data, err := json.Marshal(tt.task)
			if err != nil {
				t.Fatalf("Failed to serialize: %v", err)
			}

			var decoded Task
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to deserialize: %v", err)
			}

			if decoded.File != tt.task.File {
				t.Error("File not preserved through JSON")
			}
			if decoded.Line != tt.task.Line {
				t.Error("Line not preserved through JSON")
			}
		})
	}
}

// TestTaskCommentTracking tests comment ID tracking for tasks
func TestTaskCommentTracking(t *testing.T) {
	scenarios := []struct {
		name   string
		tasks  []Task
		verify func(t *testing.T, grouped map[int64][]Task)
	}{
		{
			name: "単一コメントからのタスク",
			tasks: []Task{
				{ID: "1", SourceCommentID: 1001, Description: "Task 1"},
				{ID: "2", SourceCommentID: 1001, Description: "Task 2"},
				{ID: "3", SourceCommentID: 1001, Description: "Task 3"},
			},
			verify: func(t *testing.T, grouped map[int64][]Task) {
				if len(grouped) != 1 {
					t.Errorf("Expected 1 comment group, got %d", len(grouped))
				}
				if len(grouped[1001]) != 3 {
					t.Errorf("Expected 3 tasks for comment 1001, got %d", len(grouped[1001]))
				}
			},
		},
		{
			name: "複数コメントからのタスク",
			tasks: []Task{
				{ID: "1", SourceCommentID: 2001, Description: "Review 1 Task 1"},
				{ID: "2", SourceCommentID: 2001, Description: "Review 1 Task 2"},
				{ID: "3", SourceCommentID: 2002, Description: "Review 2 Task 1"},
				{ID: "4", SourceCommentID: 2003, Description: "Review 3 Task 1"},
			},
			verify: func(t *testing.T, grouped map[int64][]Task) {
				if len(grouped) != 3 {
					t.Errorf("Expected 3 comment groups, got %d", len(grouped))
				}
				if len(grouped[2001]) != 2 {
					t.Error("Comment 2001 should have 2 tasks")
				}
				if len(grouped[2002]) != 1 {
					t.Error("Comment 2002 should have 1 task")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Group tasks by comment ID
			grouped := make(map[int64][]Task)
			for _, task := range scenario.tasks {
				grouped[task.SourceCommentID] = append(grouped[task.SourceCommentID], task)
			}

			scenario.verify(t, grouped)
		})
	}
}

// TestPRDirectoryNaming tests PR directory naming conventions
func TestPRDirectoryNaming(t *testing.T) {
	tests := []struct {
		prNumber int
		expected string
	}{
		{1, "PR-1"},
		{100, "PR-100"},
		{9999, "PR-9999"},
		{0, "PR-0"},
		{-1, "PR--1"}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getPRDirName(tt.prNumber)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestTaskPriorityValidation tests priority value validation
func TestTaskPriorityValidation(t *testing.T) {
	validPriorities := []string{"critical", "high", "medium", "low"}

	for _, priority := range validPriorities {
		task := Task{
			ID:       "test",
			Priority: priority,
		}

		if !isValidPriority(task.Priority) {
			t.Errorf("Priority %s should be valid", priority)
		}
	}

	// Test invalid priorities
	invalidPriorities := []string{"", "urgent", "normal", "CRITICAL", "High"}
	for _, priority := range invalidPriorities {
		if isValidPriority(priority) {
			t.Errorf("Priority %s should be invalid", priority)
		}
	}
}

// TestTaskStatusValidation tests status value validation
func TestTaskStatusValidation(t *testing.T) {
	validStatuses := []string{"todo", "doing", "done", "pending", "cancel", "cancelled"}

	for _, status := range validStatuses {
		task := Task{
			ID:     "test",
			Status: status,
		}

		// Normalize cancelled to cancel
		normalized := normalizeStatus(task.Status)
		if normalized == "cancelled" {
			normalized = "cancel"
		}

		if !isValidStatus(normalized) {
			t.Errorf("Status %s should be valid", status)
		}
	}
}

// TestConcurrentTaskOperations tests concurrent access to tasks
func TestConcurrentTaskOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	prDir := filepath.Join(tempDir, ".pr-review", "PR-300")
	if err := os.MkdirAll(prDir, 0755); err != nil {
		t.Fatalf("Failed to create PR directory: %v", err)
	}

	// Create initial tasks file
	tasksFile := TasksFile{
		Tasks: []Task{
			{ID: "1", Status: "todo", Priority: "high", PRNumber: 300},
		},
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(tasksFile, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal tasks file: %v", err)
	}
	tasksPath := filepath.Join(prDir, "tasks.json")
	if err := os.WriteFile(tasksPath, data, 0644); err != nil {
		t.Fatalf("Failed to write tasks file: %v", err)
	}

	// Simulate concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			data, err := os.ReadFile(tasksPath)
			if err != nil {
				t.Logf("Failed to read: %v", err)
				return
			}
			var tf TasksFile
			if err := json.Unmarshal(data, &tf); err != nil {
				t.Logf("Failed to unmarshal: %v", err)
				return
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper functions
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func getPRDirName(prNumber int) string {
	return fmt.Sprintf("PR-%d", prNumber)
}

func isValidPriority(priority string) bool {
	switch priority {
	case "critical", "high", "medium", "low":
		return true
	default:
		return false
	}
}

func isValidStatus(status string) bool {
	switch status {
	case "todo", "doing", "done", "pending", "cancel":
		return true
	default:
		return false
	}
}

func normalizeStatus(status string) string {
	if status == "cancelled" {
		return "cancel"
	}
	return status
}
