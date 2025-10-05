package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
)

// TestUpdateCommandRejectsCancelStatus tests that update command rejects cancel status
func TestUpdateCommandRejectsCancelStatus(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test task
	tasks := []storage.Task{
		{
			ID:          "test-task-1",
			Description: "Test task",
			Status:      "todo",
			PRNumber:    123,
		},
	}

	// Setup storage with .pr-review subdirectory
	prReviewDir := filepath.Join(tempDir, ".pr-review")
	storageManager := storage.NewManagerWithBase(prReviewDir)
	prDir := filepath.Join(prReviewDir, "PR-123")
	if err := os.MkdirAll(prDir, 0755); err != nil {
		t.Fatalf("Failed to create PR directory: %v", err)
	}

	if err := storageManager.SaveTasks(123, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Change to temp directory (not prReviewDir)
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create update command
	cmd := &cobra.Command{}
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	// Try to update task to cancel status
	err := runUpdate(cmd, []string{"test-task-1", "cancel"})

	// Should return error
	if err == nil {
		t.Fatal("Expected error when updating to cancel status, got nil")
	}

	// Check error message
	errMsg := err.Error()
	expectedPhrases := []string{
		"cannot set status to 'cancel' using update command",
		"reviewtask cancel test-task-1 --reason",
		"thread auto-resolution",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(errMsg, phrase) {
			t.Errorf("Error message should contain %q, got:\n%s", phrase, errMsg)
		}
	}

	// Verify task status was NOT changed
	updatedTasks, err := storageManager.GetTasksByPR(123)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	if len(updatedTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(updatedTasks))
	}

	if updatedTasks[0].Status != "todo" {
		t.Errorf("Task status should remain 'todo', got '%s'", updatedTasks[0].Status)
	}
}

// TestUpdateCommandAcceptsValidStatuses tests that update command accepts valid statuses
func TestUpdateCommandAcceptsValidStatuses(t *testing.T) {
	validStatuses := []string{"todo", "doing", "done", "pending"}

	for _, status := range validStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()

			// Create test task
			tasks := []storage.Task{
				{
					ID:          "test-task-1",
					Description: "Test task",
					Status:      "todo",
					PRNumber:    123,
				},
			}

			// Setup storage with .pr-review subdirectory
			prReviewDir := filepath.Join(tempDir, ".pr-review")
			storageManager := storage.NewManagerWithBase(prReviewDir)
			prDir := filepath.Join(prReviewDir, "PR-123")
			if err := os.MkdirAll(prDir, 0755); err != nil {
				t.Fatalf("Failed to create PR directory: %v", err)
			}

			if err := storageManager.SaveTasks(123, tasks); err != nil {
				t.Fatalf("Failed to save tasks: %v", err)
			}

			// Change to temp directory (not prReviewDir)
			origDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(origDir)

			// Create update command
			cmd := &cobra.Command{}
			var outBuf, errBuf bytes.Buffer
			cmd.SetOut(&outBuf)
			cmd.SetErr(&errBuf)

			// Update task to valid status
			err := runUpdate(cmd, []string{"test-task-1", status})

			// Should not return error
			if err != nil {
				t.Fatalf("Expected no error for valid status '%s', got: %v", status, err)
			}

			// Verify task status was changed
			updatedTasks, err := storageManager.GetTasksByPR(123)
			if err != nil {
				t.Fatalf("Failed to get tasks: %v", err)
			}

			if len(updatedTasks) != 1 {
				t.Fatalf("Expected 1 task, got %d", len(updatedTasks))
			}

			if updatedTasks[0].Status != status {
				t.Errorf("Task status should be '%s', got '%s'", status, updatedTasks[0].Status)
			}
		})
	}
}

// TestUpdateCommandInvalidStatus tests that update command rejects invalid statuses
func TestUpdateCommandInvalidStatus(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create test task
	tasks := []storage.Task{
		{
			ID:          "test-task-1",
			Description: "Test task",
			Status:      "todo",
			PRNumber:    123,
		},
	}

	// Setup storage with .pr-review subdirectory
	prReviewDir := filepath.Join(tempDir, ".pr-review")
	storageManager := storage.NewManagerWithBase(prReviewDir)
	prDir := filepath.Join(prReviewDir, "PR-123")
	if err := os.MkdirAll(prDir, 0755); err != nil {
		t.Fatalf("Failed to create PR directory: %v", err)
	}

	if err := storageManager.SaveTasks(123, tasks); err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Change to temp directory (not prReviewDir)
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Create update command
	cmd := &cobra.Command{}
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	// Try to update task to invalid status
	invalidStatus := "invalid_status"
	err := runUpdate(cmd, []string{"test-task-1", invalidStatus})

	// Should return error
	if err == nil {
		t.Fatal("Expected error for invalid status, got nil")
	}

	// Check error message
	errMsg := err.Error()
	if !strings.Contains(errMsg, "invalid status") {
		t.Errorf("Error message should contain 'invalid status', got: %s", errMsg)
	}

	// Verify task status was NOT changed
	updatedTasks, err := storageManager.GetTasksByPR(123)
	if err != nil {
		t.Fatalf("Failed to get tasks: %v", err)
	}

	if updatedTasks[0].Status != "todo" {
		t.Errorf("Task status should remain 'todo', got '%s'", updatedTasks[0].Status)
	}
}

// TestUpdateCommandHelpText tests that help text is correct
func TestUpdateCommandHelpText(t *testing.T) {
	help := updateCmd.Long

	// Should not mention cancel as a valid status
	if strings.Contains(help, "cancel   - Decided not to address") {
		t.Error("Help text should not list 'cancel' as a valid status")
	}

	// Should mention the cancel command instead
	if !strings.Contains(help, "reviewtask cancel") {
		t.Error("Help text should mention 'reviewtask cancel' command")
	}

	// Should list valid statuses
	validStatuses := []string{"todo", "doing", "done", "pending"}
	for _, status := range validStatuses {
		if !strings.Contains(help, status) {
			t.Errorf("Help text should mention valid status '%s'", status)
		}
	}
}

// TestUpdateCommandExamples tests that examples are correct
func TestUpdateCommandExamples(t *testing.T) {
	help := updateCmd.Long

	// Should have example for pending, not cancel
	if strings.Contains(help, "reviewtask update task-3 cancel") {
		t.Error("Examples should not show 'cancel' status")
	}

	if !strings.Contains(help, "reviewtask update task-3 pending") {
		t.Error("Examples should show 'pending' status instead of 'cancel'")
	}
}
