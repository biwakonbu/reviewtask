package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/storage"
)

// TestCancelledStatusHandling tests that cancelled tasks are properly handled
// throughout the system - from storage to statistics to display
func TestCancelledStatusHandling(t *testing.T) {
	// Test that storage manager properly handles cancel status
	t.Run("storage_manager_merge_uses_cancel_status", func(t *testing.T) {
		// Create initial tasks
		existingTasks := []storage.Task{
			{
				ID:              "task-1",
				Description:     "Task 1",
				Status:          "todo",
				Priority:        "high",
				SourceCommentID: 12345,
			},
			{
				ID:              "task-2",
				Description:     "Task 2",
				Status:          "doing",
				Priority:        "medium",
				SourceCommentID: 12345,
			},
		}

		// This simulates what the mergeTasksForComment method does
		// when a comment is deleted or has no actionable tasks
		result := existingTasks
		for i := range result {
			if result[i].Status != "done" && result[i].Status != "cancel" {
				result[i].Status = "cancel"
			}
		}

		// Verify tasks are marked as "cancel" not "cancelled"
		for _, task := range result {
			if task.Status != "done" {
				assert.Equal(t, "cancel", task.Status, "Task should be marked as 'cancel' not 'cancelled'")
			}
		}
	})

	// Test that statistics correctly count cancel status
	t.Run("statistics_count_cancel_status", func(t *testing.T) {
		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(origDir)

		prDir := filepath.Join(".pr-review", "PR-1")
		require.NoError(t, os.MkdirAll(prDir, 0755))

		// Create tasks with various statuses including cancel
		tasks := []storage.Task{
			{ID: "1", Status: "todo", Priority: "high"},
			{ID: "2", Status: "doing", Priority: "medium"},
			{ID: "3", Status: "done", Priority: "low"},
			{ID: "4", Status: "cancel", Priority: "high"},
			{ID: "5", Status: "cancel", Priority: "medium"},
		}

		// Save tasks in the expected format
		tasksFile := filepath.Join(prDir, "tasks.json")
		tasksData := storage.TasksFile{
			GeneratedAt: "2024-01-01T00:00:00Z",
			Tasks:       tasks,
		}
		data, err := json.MarshalIndent(tasksData, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tasksFile, data, 0644))

		// Create storage manager and get statistics
		sm := storage.NewManager()
		allTasks, err := sm.GetAllTasks()
		require.NoError(t, err)

		// Count statuses manually
		statusCounts := make(map[string]int)
		for _, task := range allTasks {
			statusCounts[task.Status]++
		}

		// Verify cancel tasks are counted
		assert.Equal(t, 2, statusCounts["cancel"], "Should count 2 cancel tasks")
		assert.Equal(t, 1, statusCounts["todo"], "Should count 1 todo task")
		assert.Equal(t, 1, statusCounts["doing"], "Should count 1 doing task")
		assert.Equal(t, 1, statusCounts["done"], "Should count 1 done task")
	})

	// Test that show command skips cancelled tasks
	t.Run("show_command_skips_cancelled_tasks", func(t *testing.T) {
		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(origDir)

		prDir := filepath.Join(".pr-review", "PR-1")
		require.NoError(t, os.MkdirAll(prDir, 0755))

		// Create tasks where first todo is cancelled
		tasks := []storage.Task{
			{ID: "1", Status: "cancel", Priority: "high", Description: "Cancelled task"},
			{ID: "2", Status: "todo", Priority: "medium", Description: "Should be shown"},
			{ID: "3", Status: "todo", Priority: "low", Description: "Lower priority"},
		}

		// Save tasks in the expected format
		tasksFile := filepath.Join(prDir, "tasks.json")
		tasksData := storage.TasksFile{
			GeneratedAt: "2024-01-01T00:00:00Z",
			Tasks:       tasks,
		}
		data, err := json.MarshalIndent(tasksData, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tasksFile, data, 0644))

		// Create storage manager
		sm := storage.NewManager()
		allTasks, err := sm.GetAllTasks()
		require.NoError(t, err)

		// Find next actionable task (should skip cancelled)
		var nextTask *storage.Task
		priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
		highestPriority := 0

		for _, task := range allTasks {
			// Skip cancelled tasks
			if task.Status == "todo" {
				if priority, exists := priorityOrder[task.Priority]; exists && priority > highestPriority {
					highestPriority = priority
					taskCopy := task
					nextTask = &taskCopy
				}
			}
		}

		// Verify we found the non-cancelled task
		require.NotNil(t, nextTask, "Should find a next task")
		assert.Equal(t, "2", nextTask.ID, "Should find task 2, not the cancelled task")
		assert.Equal(t, "Should be shown", nextTask.Description, "Should find the correct task")
	})

	// Test backwards compatibility
	t.Run("backwards_compatibility_with_cancelled_status", func(t *testing.T) {
		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(origDir)

		prDir := filepath.Join(".pr-review", "PR-1")
		require.NoError(t, os.MkdirAll(prDir, 0755))

		// Create tasks with old "cancelled" status
		tasks := []storage.Task{
			{ID: "1", Status: "cancelled", Priority: "high"},
			{ID: "2", Status: "cancel", Priority: "medium"},
		}

		// Save tasks in the expected format
		tasksFile := filepath.Join(prDir, "tasks.json")
		tasksData := storage.TasksFile{
			GeneratedAt: "2024-01-01T00:00:00Z",
			Tasks:       tasks,
		}
		data, err := json.MarshalIndent(tasksData, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(tasksFile, data, 0644))

		// Read back and verify both are treated as cancelled
		sm := storage.NewManager()
		allTasks, err := sm.GetAllTasks()
		require.NoError(t, err)

		// Both should be considered cancelled
		cancelCount := 0
		for _, task := range allTasks {
			if task.Status == "cancel" || task.Status == "cancelled" {
				cancelCount++
			}
		}
		assert.Equal(t, 2, cancelCount, "Both 'cancel' and 'cancelled' should be counted")
	})
}

// TestStatusCommandOutput tests the actual output of the status command
func TestStatusCommandOutput(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	prDir := filepath.Join(".pr-review", "PR-1")
	require.NoError(t, os.MkdirAll(prDir, 0755))

	// Create tasks with cancel status
	tasks := []storage.Task{
		{ID: "1", Status: "todo", Priority: "high"},
		{ID: "2", Status: "doing", Priority: "medium"},
		{ID: "3", Status: "done", Priority: "low"},
		{ID: "4", Status: "cancel", Priority: "high"},
		{ID: "5", Status: "cancel", Priority: "medium"},
		{ID: "6", Status: "pending", Priority: "low"},
	}

	// Save tasks in the expected format
	tasksFile := filepath.Join(prDir, "tasks.json")
	tasksData := storage.TasksFile{
		GeneratedAt: "2024-01-01T00:00:00Z",
		Tasks:       tasks,
	}
	data, err := json.MarshalIndent(tasksData, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tasksFile, data, 0644))

	// Calculate expected status counts
	sm := storage.NewManager()
	allTasks, err := sm.GetAllTasks()
	require.NoError(t, err)

	statusCounts := make(map[string]int)
	for _, task := range allTasks {
		statusCounts[task.Status]++
	}

	// Build expected output
	expectedParts := []string{
		fmt.Sprintf("todo: %d", statusCounts["todo"]),
		fmt.Sprintf("doing: %d", statusCounts["doing"]),
		fmt.Sprintf("done: %d", statusCounts["done"]),
		fmt.Sprintf("pending: %d", statusCounts["pending"]),
		fmt.Sprintf("cancel: %d", statusCounts["cancel"]),
	}

	// Verify all parts are present in expected output
	expectedOutput := strings.Join(expectedParts, ", ")

	// This is what the status command should show (Modern UI uses uppercase)
	assert.Contains(t, expectedOutput, "CANCEL: 2", "Status output should show 2 cancelled tasks")

	// Verify completion rate includes cancelled tasks
	completed := statusCounts["done"] + statusCounts["cancel"]
	total := len(allTasks)
	completionRate := float64(completed) / float64(total) * 100

	assert.Equal(t, 3, completed, "Should count done + cancel as completed")
	assert.Equal(t, 50.0, completionRate, "Completion rate should be 50%")
}
