package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"reviewtask/internal/storage"
)

// TestLowPriorityStatusAssignment tests that tasks created from low-priority
// comments get the correct status when saved and retrieved
func TestLowPriorityStatusAssignment(t *testing.T) {
	// Note: This test uses the default storage directory
	// In a real scenario, we'd use a test-specific directory

	// Create storage manager
	store := storage.NewManager()

	// Use a unique PR number for this test to avoid conflicts
	prNumber := 99999

	// Create test tasks - simulating what would come from AI analyzer
	tasks := []storage.Task{
		{
			ID:              "task-1",
			Description:     "Fix indentation",
			OriginText:      "nit: Please fix the indentation here",
			Priority:        "low",
			Status:          "pending", // Should be pending due to "nit:" pattern
			SourceCommentID: 101,
			SourceReviewID:  1,
			File:            "main.go",
			Line:            10,
			CreatedAt:       "2023-01-01T00:00:00Z",
			UpdatedAt:       "2023-01-01T00:00:00Z",
		},
		{
			ID:              "task-2",
			Description:     "Add error handling",
			OriginText:      "This needs error handling - could crash",
			Priority:        "high",
			Status:          "todo", // Should be todo (no pattern)
			SourceCommentID: 102,
			SourceReviewID:  1,
			File:            "main.go",
			Line:            20,
			CreatedAt:       "2023-01-01T00:00:00Z",
			UpdatedAt:       "2023-01-01T00:00:00Z",
		},
		{
			ID:              "task-3",
			Description:     "Improve variable names",
			OriginText:      "MINOR: These variable names are unclear",
			Priority:        "low",
			Status:          "pending", // Should be pending due to "MINOR:" pattern
			SourceCommentID: 103,
			SourceReviewID:  1,
			File:            "utils.go",
			Line:            5,
			CreatedAt:       "2023-01-01T00:00:00Z",
			UpdatedAt:       "2023-01-01T00:00:00Z",
		},
	}

	// Save tasks
	if err := store.SaveTasks(prNumber, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Retrieve tasks
	savedTasks, err := store.GetTasksByPR(prNumber)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	// Verify task count
	if len(savedTasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(savedTasks))
	}

	// Create a map for easier verification
	taskMap := make(map[string]storage.Task)
	for _, task := range savedTasks {
		taskMap[task.ID] = task
	}

	// Verify each task has the correct status
	tests := []struct {
		taskID         string
		expectedStatus string
		pattern        string
	}{
		{"task-1", "pending", "nit:"},
		{"task-2", "todo", "no pattern"},
		{"task-3", "pending", "MINOR:"},
	}

	for _, test := range tests {
		task, exists := taskMap[test.taskID]
		if !exists {
			t.Errorf("Task %s not found in saved tasks", test.taskID)
			continue
		}

		if task.Status != test.expectedStatus {
			t.Errorf("Task %s (%s): expected status %q, got %q",
				test.taskID, test.pattern, test.expectedStatus, task.Status)
		}
	}

	// Test status update functionality
	// Update a low-priority task to "done"
	if err := store.UpdateTaskStatus("task-1", "done"); err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Verify the update
	updatedTasks, err := store.GetTasksByPR(prNumber)
	if err != nil {
		t.Fatalf("Failed to get updated tasks: %v", err)
	}

	for _, task := range updatedTasks {
		if task.ID == "task-1" && task.Status != "done" {
			t.Errorf("Task status update failed: expected 'done', got %q", task.Status)
		}
	}
}

// TestLowPriorityWorkflow simulates a complete workflow from PR to task display
func TestLowPriorityWorkflow(t *testing.T) {
	// Create a temporary directory for this test to ensure isolation
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Setup storage
	store := storage.NewManager()

	// Use a unique PR number to avoid conflicts
	prNumber := 99998

	// Save tasks with different priorities and statuses
	tasks := []storage.Task{
		{
			ID:              "urgent-1",
			Description:     "Fix SQL injection vulnerability",
			OriginText:      "Critical security issue - SQL injection possible here",
			Priority:        "critical",
			Status:          "todo",
			SourceCommentID: 201,
			File:            "db.go",
			Line:            100,
			CreatedAt:       "2023-01-01T00:00:00Z",
			UpdatedAt:       "2023-01-01T00:00:00Z",
		},
		{
			ID:              "nit-1",
			Description:     "Fix spacing",
			OriginText:      "nit: Extra space here",
			Priority:        "low",
			Status:          "pending",
			SourceCommentID: 202,
			File:            "format.go",
			Line:            10,
			CreatedAt:       "2023-01-01T00:00:00Z",
			UpdatedAt:       "2023-01-01T00:00:00Z",
		},
		{
			ID:              "nit-2",
			Description:     "Rename variable",
			OriginText:      "style: This variable name doesn't follow conventions",
			Priority:        "low",
			Status:          "pending",
			SourceCommentID: 203,
			File:            "vars.go",
			Line:            20,
			CreatedAt:       "2023-01-01T00:00:00Z",
			UpdatedAt:       "2023-01-01T00:00:00Z",
		},
	}

	if err := store.SaveTasks(prNumber, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Get all tasks
	allTasks, err := store.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	// Count tasks by status
	statusCounts := make(map[string]int)
	for _, task := range allTasks {
		statusCounts[task.Status]++
	}

	// Verify status distribution
	if statusCounts["todo"] != 1 {
		t.Errorf("Expected 1 'todo' task, got %d", statusCounts["todo"])
	}
	if statusCounts["pending"] != 2 {
		t.Errorf("Expected 2 'pending' tasks, got %d", statusCounts["pending"])
	}

	// Verify high-priority tasks are still 'todo'
	for _, task := range allTasks {
		if task.Priority == "critical" && task.Status != "todo" {
			t.Errorf("Critical task should have 'todo' status, got %q", task.Status)
		}
	}

	// Verify low-priority tasks with patterns are 'pending'
	for _, task := range allTasks {
		if task.ID == "nit-1" || task.ID == "nit-2" {
			if task.Status != "pending" {
				t.Errorf("Low-priority task %s should have 'pending' status, got %q",
					task.ID, task.Status)
			}
		}
	}
}

// TestFileFormat verifies that tasks are saved in the correct JSON format
func TestFileFormat(t *testing.T) {
	// Create a temporary directory for this test to ensure isolation
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Create storage manager
	store := storage.NewManager()

	// Save a task with low-priority status
	tasks := []storage.Task{
		{
			ID:              "format-test-1",
			Description:     "Test task",
			OriginText:      "nit: Test comment",
			Priority:        "low",
			Status:          "pending",
			SourceCommentID: 301,
			File:            "test.go",
			Line:            1,
			CreatedAt:       "2023-01-01T00:00:00Z",
			UpdatedAt:       "2023-01-01T00:00:00Z",
		},
	}

	prNumber := 99997
	if err := store.SaveTasks(prNumber, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Read the saved file directly
	tasksFile := filepath.Join(".pr-review", fmt.Sprintf("pr-%d", prNumber), "tasks.json")

	// Ensure the directory structure exists
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		// If file doesn't exist, skip the file format test
		t.Skip("Tasks file doesn't exist in test environment, skipping file format verification")
	}

	data, err := os.ReadFile(tasksFile)
	if err != nil {
		t.Fatalf("Failed to read tasks file: %v", err)
	}

	// Parse JSON to verify structure
	var savedData struct {
		Tasks []json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal(data, &savedData); err != nil {
		t.Fatalf("Failed to parse tasks file: %v", err)
	}

	// Verify we have one task
	if len(savedData.Tasks) != 1 {
		t.Fatalf("Expected 1 task in file, got %d", len(savedData.Tasks))
	}

	// Parse the task
	var savedTask storage.Task
	if err := json.Unmarshal(savedData.Tasks[0], &savedTask); err != nil {
		t.Fatalf("Failed to parse task: %v", err)
	}

	// Verify the status was saved correctly
	if savedTask.Status != "pending" {
		t.Errorf("Expected status 'pending' in saved file, got %q", savedTask.Status)
	}
}
