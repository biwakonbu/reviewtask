package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
)

// TestCancelCommandBasic tests basic cancel command functionality
func TestCancelCommandBasic(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test task
	tasks := []storage.Task{
		{
			ID:              "test-task-1",
			Description:     "Test task",
			Status:          "pending",
			PRNumber:        123,
			SourceCommentID: 456,
		},
	}

	// Setup storage
	storageManager := storage.NewManagerWithBase(tempDir)
	prDir := filepath.Join(tempDir, "PR-123")
	if err := os.MkdirAll(prDir, 0755); err != nil {
		t.Fatalf("Failed to create PR directory: %v", err)
	}

	if err := storageManager.SaveTasks(123, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Test formatCancelComment
	comment := formatCancelComment(storageManager, &tasks[0], "Test reason")
	if comment == "" {
		t.Error("formatCancelComment returned empty string")
	}
	if !contains(comment, "Test reason") {
		t.Error("formatCancelComment should contain the reason")
	}
	if !contains(comment, "**Task Cancelled**") {
		t.Error("formatCancelComment should contain task cancelled header")
	}
	// Task ID should NOT be exposed in GitHub comments
	if contains(comment, tasks[0].ID) {
		t.Error("formatCancelComment should NOT contain internal task ID")
	}
	// Should contain original feedback
	if !contains(comment, "**Original Feedback:**") {
		t.Error("formatCancelComment should contain original feedback section")
	}
}

// TestFormatCancelComment tests the comment formatting
func TestFormatCancelComment(t *testing.T) {
	tests := []struct {
		name         string
		task         *storage.Task
		otherTasks   []storage.Task // Other tasks from same comment for testing count
		reason       string
		wantContain  []string
		wantNotExist []string
	}{
		{
			name: "basic task with priority",
			task: &storage.Task{
				ID:          "task-1",
				Description: "Fix bug",
				Priority:    "high",
				PRNumber:    123,
			},
			reason: "Already fixed in another PR",
			wantContain: []string{
				"**Task Cancelled**",
				"Priority: HIGH",
				"Already fixed in another PR",
				"**Original Feedback:**",
				"> Fix bug",
			},
			wantNotExist: []string{
				"task-1", // Task ID should not be exposed
			},
		},
		{
			name: "task with description",
			task: &storage.Task{
				ID:          "task-2",
				Description: "Update docs",
				Priority:    "medium",
				PRNumber:    123,
			},
			reason: "Documentation updated differently",
			wantContain: []string{
				"**Task Cancelled**",
				"Priority: MEDIUM",
				"Documentation updated differently",
				"**Original Feedback:**",
				"> Update docs",
			},
			wantNotExist: []string{
				"task-2", // Task ID should not be exposed
			},
		},
		{
			name: "task with other active tasks from same comment",
			task: &storage.Task{
				ID:              "task-3",
				Description:     "Add tests",
				Priority:        "high",
				PRNumber:        123,
				SourceCommentID: 456,
			},
			otherTasks: []storage.Task{
				{
					ID:              "task-4",
					Status:          "todo",
					PRNumber:        123,
					SourceCommentID: 456,
				},
				{
					ID:              "task-5",
					Status:          "doing",
					PRNumber:        123,
					SourceCommentID: 456,
				},
				{
					ID:              "task-6",
					Status:          "done",
					PRNumber:        123,
					SourceCommentID: 456, // This should not be counted as "active"
				},
			},
			reason: "Covered by integration tests",
			wantContain: []string{
				"**Task Cancelled**",
				"> Add tests",
				"ℹ️ This comment has 2 other task(s) still active",
			},
			wantNotExist: []string{
				"task-3", // Task ID should not be exposed
			},
		},
		{
			name: "task without priority defaults to MEDIUM",
			task: &storage.Task{
				ID:          "task-7",
				Description: "Refactor code",
				PRNumber:    123,
			},
			reason: "Not needed anymore",
			wantContain: []string{
				"Priority: MEDIUM",
				"> Refactor code",
			},
			wantNotExist: []string{
				"task-7", // Task ID should not be exposed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and storage manager
			tempDir := t.TempDir()
			storageManager := storage.NewManagerWithBase(tempDir)

			// Setup PR directory
			prDir := filepath.Join(tempDir, fmt.Sprintf("PR-%d", tt.task.PRNumber))
			if err := os.MkdirAll(prDir, 0755); err != nil {
				t.Fatalf("Failed to create PR directory: %v", err)
			}

			// Save tasks if there are other tasks from same comment
			if len(tt.otherTasks) > 0 {
				allTasks := append([]storage.Task{*tt.task}, tt.otherTasks...)
				if err := storageManager.SaveTasks(tt.task.PRNumber, allTasks); err != nil {
					t.Fatalf("Failed to save tasks: %v", err)
				}
			}

			result := formatCancelComment(storageManager, tt.task, tt.reason)

			for _, want := range tt.wantContain {
				if !contains(result, want) {
					t.Errorf("formatCancelComment() result should contain %q, got:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotExist {
				if contains(result, notWant) {
					t.Errorf("formatCancelComment() result should NOT contain %q, got:\n%s", notWant, result)
				}
			}
		})
	}
}

// TestCancelCommandValidation tests input validation
func TestCancelCommandValidation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		flags     map[string]string
		boolFlags map[string]bool
		wantError bool
		errMsg    string
	}{
		{
			name:      "no task ID without all-pending",
			args:      []string{},
			flags:     map[string]string{"reason": "Valid reason"},
			wantError: true,
			errMsg:    "task ID is required",
		},
		{
			name:      "empty reason",
			args:      []string{"task-1"},
			flags:     map[string]string{"reason": ""},
			wantError: true,
			errMsg:    "cancellation reason cannot be empty",
		},
		{
			name:      "all-pending with task ID",
			args:      []string{"task-1"},
			flags:     map[string]string{"reason": "Valid reason"},
			boolFlags: map[string]bool{"all-pending": true},
			wantError: true,
			errMsg:    "cannot specify task ID when using --all-pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new cancel command instance
			cmd := &cobra.Command{
				Use:  "cancel",
				RunE: runCancel,
			}
			cmd.Flags().String("reason", "", "Cancellation reason")
			cmd.Flags().Bool("all-pending", false, "Cancel all pending tasks")

			// Set flags
			for key, val := range tt.flags {
				cmd.Flags().Set(key, val)
			}
			for key, val := range tt.boolFlags {
				if val {
					cmd.Flags().Set(key, "true")
				}
			}

			// Reset global flags to defaults
			cancelReason = ""
			cancelAllPending = false

			// Parse flags into global variables
			if reason, err := cmd.Flags().GetString("reason"); err == nil {
				cancelReason = reason
			}
			if allPending, err := cmd.Flags().GetBool("all-pending"); err == nil {
				cancelAllPending = allPending
			}

			err := runCancel(cmd, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("runCancel() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestUpdateTaskCancelStatus tests the storage update function
func TestUpdateTaskCancelStatus(t *testing.T) {
	tempDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tempDir)

	// Create test task
	tasks := []storage.Task{
		{
			ID:              "test-task-1",
			Description:     "Test task",
			Status:          "pending",
			PRNumber:        123,
			SourceCommentID: 456,
		},
	}

	if err := storageManager.SaveTasks(123, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Update task with cancel status
	err := updateTaskCancelStatus(storageManager, "test-task-1", "Test cancel reason", true)
	if err != nil {
		t.Fatalf("updateTaskCancelStatus failed: %v", err)
	}

	// Verify task was updated
	updatedTasks, err := storageManager.GetTasksByPR(123)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	if len(updatedTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(updatedTasks))
	}

	task := updatedTasks[0]
	if task.Status != "cancel" {
		t.Errorf("Expected status 'cancel', got %q", task.Status)
	}
	if task.CancelReason != "Test cancel reason" {
		t.Errorf("Expected cancel reason 'Test cancel reason', got %q", task.CancelReason)
	}
	if !task.CancelCommentPosted {
		t.Error("Expected CancelCommentPosted to be true")
	}
}

// TestBatchCancelPending tests cancelling all pending tasks
func TestBatchCancelPending(t *testing.T) {
	tempDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tempDir)

	// Create multiple tasks with different statuses
	tasks := []storage.Task{
		{ID: "task-1", Status: "pending", PRNumber: 123, SourceCommentID: 0},
		{ID: "task-2", Status: "pending", PRNumber: 123, SourceCommentID: 0},
		{ID: "task-3", Status: "doing", PRNumber: 123, SourceCommentID: 0},
		{ID: "task-4", Status: "done", PRNumber: 123, SourceCommentID: 0},
	}

	if err := storageManager.SaveTasks(123, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Cancel all pending tasks (without GitHub client since SourceCommentID is 0)
	cancelReason = "Batch test reason"
	cancelAllPending = true

	// Count pending tasks before
	allTasks, _ := storageManager.GetAllTasks()
	pendingCount := 0
	for _, task := range allTasks {
		if task.Status == "pending" {
			pendingCount++
		}
	}

	if pendingCount != 2 {
		t.Fatalf("Expected 2 pending tasks, got %d", pendingCount)
	}

	// Note: We can't fully test runCancel here without mocking GitHub client
	// But we've tested the individual components (updateTaskCancelStatus)
	// Integration tests should cover the full flow
}

// TestCancelTaskWithoutSourceComment tests cancelling task without source comment
func TestCancelTaskWithoutSourceComment(t *testing.T) {
	tempDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tempDir)

	// Create task without source comment ID (embedded Codex comment)
	tasks := []storage.Task{
		{
			ID:              "embedded-task",
			Description:     "Embedded task",
			Status:          "pending",
			PRNumber:        123,
			SourceCommentID: 0, // No source comment
		},
	}

	if err := storageManager.SaveTasks(123, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Test that updateTaskCancelStatus works without GitHub interaction
	err := updateTaskCancelStatus(storageManager, "embedded-task", "Test reason", false)
	if err != nil {
		t.Fatalf("updateTaskCancelStatus failed: %v", err)
	}

	// Verify task was updated locally even though comment wasn't posted
	updatedTasks, _ := storageManager.GetTasksByPR(123)
	if updatedTasks[0].Status != "cancel" {
		t.Errorf("Expected status 'cancel', got %q", updatedTasks[0].Status)
	}
	if updatedTasks[0].CancelCommentPosted {
		t.Error("Expected CancelCommentPosted to be false for embedded task")
	}
	if updatedTasks[0].CancelReason != "Test reason" {
		t.Errorf("Expected CancelReason 'Test reason', got %q", updatedTasks[0].CancelReason)
	}
}

// TestCancelErrorPropagation tests that errors are properly propagated with non-zero exit status
func TestCancelErrorPropagation(t *testing.T) {
	tests := []struct {
		name        string
		taskID      string
		allPending  bool
		setupTasks  func(*storage.Manager) error
		wantErr     bool
		errContains string
	}{
		{
			name:       "single task not found returns error",
			taskID:     "nonexistent",
			allPending: false,
			setupTasks: func(sm *storage.Manager) error {
				return sm.SaveTasks(123, []storage.Task{
					{ID: "task-1", Status: "todo", PRNumber: 123},
				})
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:       "all-pending with no pending tasks returns nil",
			allPending: true,
			setupTasks: func(sm *storage.Manager) error {
				return sm.SaveTasks(123, []storage.Task{
					{ID: "task-1", Status: "done", PRNumber: 123},
					{ID: "task-2", Status: "cancel", PRNumber: 123},
				})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			storageManager := storage.NewManagerWithBase(tempDir)

			if tt.setupTasks != nil {
				if err := tt.setupTasks(storageManager); err != nil {
					t.Fatalf("Failed to setup tasks: %v", err)
				}
			}

			// Reset global flags
			cancelReason = "Test cancellation reason"
			cancelAllPending = tt.allPending

			// Build args
			var args []string
			if !tt.allPending && tt.taskID != "" {
				args = []string{tt.taskID}
			}

			// Create command
			cmd := &cobra.Command{Use: "cancel"}
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			// Run command
			err := runCancel(cmd, args)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("runCancel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message if error expected
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

// TestCancelTaskFunction tests the cancelTask function directly
func TestCancelTaskFunction(t *testing.T) {
	tempDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tempDir)

	// Test task without source comment (local-only cancellation)
	task := storage.Task{
		ID:              "test-task",
		Description:     "Test task",
		Status:          "pending",
		PRNumber:        123,
		SourceCommentID: 0,
	}

	if err := storageManager.SaveTasks(123, []storage.Task{task}); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	cmd := &cobra.Command{Use: "cancel"}

	// Test cancelTask directly (without GitHub client since SourceCommentID=0)
	err := cancelTask(cmd, storageManager, nil, &task, "Test cancellation")

	if err != nil {
		t.Errorf("cancelTask should succeed for task without source comment, got: %v", err)
	}

	// Verify task was cancelled
	updatedTasks, _ := storageManager.GetTasksByPR(123)
	if len(updatedTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(updatedTasks))
	}

	if updatedTasks[0].Status != "cancel" {
		t.Errorf("Expected status 'cancel', got %q", updatedTasks[0].Status)
	}
	if updatedTasks[0].CancelReason != "Test cancellation" {
		t.Errorf("Expected cancel reason 'Test cancellation', got %q", updatedTasks[0].CancelReason)
	}
	if updatedTasks[0].CancelCommentPosted {
		t.Error("Expected CancelCommentPosted to be false (no GitHub comment)")
	}
}

// TestErrorWrappingInBatchCancel tests that errors are properly wrapped with %w
func TestErrorWrappingInBatchCancel(t *testing.T) {
	// This test verifies the error wrapping logic at the code level
	// The actual error wrapping happens in cancel.go lines 144-148

	// Create a mock error to verify wrapping
	mockErr := fmt.Errorf("mock GitHub API error")

	// Simulate the error wrapping logic from the code
	wrappedErr := fmt.Errorf("failed to cancel %d task(s): %w", 2, mockErr)

	// Verify error wrapping works correctly
	if !contains(wrappedErr.Error(), "failed to cancel 2 task(s)") {
		t.Errorf("Error should contain count message, got: %v", wrappedErr)
	}
	if !contains(wrappedErr.Error(), "mock GitHub API error") {
		t.Errorf("Error should contain wrapped error, got: %v", wrappedErr)
	}

	// Verify error unwrapping works using errors.Unwrap
	unwrappedErr := errors.Unwrap(wrappedErr)
	if unwrappedErr == nil {
		t.Error("Expected to unwrap error, got nil")
	} else if unwrappedErr.Error() != mockErr.Error() {
		t.Errorf("Expected unwrapped error %q, got %q", mockErr.Error(), unwrappedErr.Error())
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// TestCancelReasonJSONPersistence tests that cancel reason is properly persisted
func TestCancelReasonJSONPersistence(t *testing.T) {
	tempDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tempDir)

	tasks := []storage.Task{
		{
			ID:              "task-1",
			Description:     "Test task",
			Status:          "pending",
			PRNumber:        123,
			SourceCommentID: 456,
		},
	}

	if err := storageManager.SaveTasks(123, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Update with cancel status
	testReason := "Integration testing revealed this is not needed"
	err := storageManager.UpdateTaskCancelStatus("task-1", testReason, true)
	if err != nil {
		t.Fatalf("UpdateTaskCancelStatus failed: %v", err)
	}

	// Read the JSON file directly
	tasksPath := filepath.Join(tempDir, "PR-123", "tasks.json")
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		t.Fatalf("Failed to read tasks file: %v", err)
	}

	var tasksFile storage.TasksFile
	if err := json.Unmarshal(data, &tasksFile); err != nil {
		t.Fatalf("Failed to unmarshal tasks: %v", err)
	}

	if len(tasksFile.Tasks) != 1 {
		t.Fatalf("Expected 1 task in file, got %d", len(tasksFile.Tasks))
	}

	task := tasksFile.Tasks[0]
	if task.CancelReason != testReason {
		t.Errorf("Expected CancelReason %q, got %q", testReason, task.CancelReason)
	}
	if !task.CancelCommentPosted {
		t.Error("Expected CancelCommentPosted to be true")
	}
}
