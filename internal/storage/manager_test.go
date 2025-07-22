package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reviewtask/internal/github"
)

// TestManager_GetCurrentBranch tests the current branch detection
func TestManager_GetCurrentBranch(t *testing.T) {
	// Skip this test as it requires actual git repository
	// In practice, this method would be mocked or tested in integration tests
	t.Skip("GetCurrentBranch requires actual git repository - should be tested with mocks or in integration tests")
}

// TestManager_GetPRsForBranch tests branch-based PR filtering
func TestManager_GetPRsForBranch(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	// Create test PR data
	testPRs := []struct {
		prNumber int
		branch   string
	}{
		{1, "feature/test-1"},
		{2, "feature/test-2"},
		{3, "feature/test-1"}, // Same branch as PR 1
		{4, "main"},
	}

	// Create PR directories and info files
	for _, pr := range testPRs {
		prDir := filepath.Join(tempDir, fmt.Sprintf("PR-%d", pr.prNumber))
		if err := os.MkdirAll(prDir, 0755); err != nil {
			t.Fatalf("Failed to create PR directory: %v", err)
		}

		prInfo := github.PRInfo{
			Number: pr.prNumber,
			Branch: pr.branch,
			Title:  "Test PR",
			Author: "testuser",
		}

		data, _ := json.MarshalIndent(prInfo, "", "  ")
		infoPath := filepath.Join(prDir, "info.json")
		if err := os.WriteFile(infoPath, data, 0644); err != nil {
			t.Fatalf("Failed to write info.json: %v", err)
		}
	}

	tests := []struct {
		name        string
		branchName  string
		expectedPRs []int
		expectError bool
	}{
		{
			name:        "Single PR for branch",
			branchName:  "feature/test-2",
			expectedPRs: []int{2},
			expectError: false,
		},
		{
			name:        "Multiple PRs for same branch",
			branchName:  "feature/test-1",
			expectedPRs: []int{1, 3},
			expectError: false,
		},
		{
			name:        "No PRs for branch",
			branchName:  "feature/nonexistent",
			expectedPRs: []int{},
			expectError: false,
		},
		{
			name:        "Main branch",
			branchName:  "main",
			expectedPRs: []int{4},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prNumbers, err := manager.GetPRsForBranch(tt.branchName)

			if tt.expectError && err == nil {
				t.Errorf("Expected error, got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if len(prNumbers) != len(tt.expectedPRs) {
				t.Errorf("Expected %d PRs, got %d", len(tt.expectedPRs), len(prNumbers))
			}

			// Check if all expected PRs are present
			for _, expectedPR := range tt.expectedPRs {
				found := false
				for _, actualPR := range prNumbers {
					if actualPR == expectedPR {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected PR %d not found in results", expectedPR)
				}
			}
		})
	}
}

// TestManager_GetAllPRNumbers tests getting all PR numbers
func TestManager_GetAllPRNumbers(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	// Create test PR directories
	prNumbers := []int{1, 2, 5, 10}
	for _, prNum := range prNumbers {
		prDir := filepath.Join(tempDir, fmt.Sprintf("PR-%d", prNum))
		if err := os.MkdirAll(prDir, 0755); err != nil {
			t.Fatalf("Failed to create PR directory: %v", err)
		}
	}

	// Create a non-PR directory that should be ignored
	if err := os.MkdirAll(filepath.Join(tempDir, "not-a-pr"), 0755); err != nil {
		t.Fatalf("Failed to create non-PR directory: %v", err)
	}

	result, err := manager.GetAllPRNumbers()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedCount := 4 // PR-1, PR-2, PR-5, PR-10
	if len(result) != expectedCount {
		t.Errorf("Expected %d PR numbers, got %d", expectedCount, len(result))
	}
}

// TestManager_MergeTasks tests task merging functionality
func TestManager_MergeTasks(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	prNumber := 123

	// Create initial tasks
	existingTasks := []Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655440011",
			Description:     "Existing task 1",
			SourceCommentID: 1,
			TaskIndex:       1,
			Status:          "done",
			OriginText:      "Original comment",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440012",
			Description:     "Existing task 2",
			SourceCommentID: 1,
			TaskIndex:       2,
			Status:          "todo",
			OriginText:      "Original comment",
		},
	}

	// Save existing tasks
	if err := manager.SaveTasks(prNumber, existingTasks); err != nil {
		t.Fatalf("Failed to save existing tasks: %v", err)
	}

	// Create new tasks with same comment ID but different content
	newTasks := []Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655440021",
			Description:     "Updated task 1",
			SourceCommentID: 1,
			TaskIndex:       1,
			Status:          "todo",
			OriginText:      "Original comment", // Same origin text
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440022",
			Description:     "Updated task 2",
			SourceCommentID: 1,
			TaskIndex:       2,
			Status:          "todo",
			OriginText:      "Original comment",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440023",
			Description:     "New task 3",
			SourceCommentID: 1,
			TaskIndex:       3,
			Status:          "todo",
			OriginText:      "Original comment",
		},
	}

	// Merge tasks
	if err := manager.MergeTasks(prNumber, newTasks); err != nil {
		t.Fatalf("Failed to merge tasks: %v", err)
	}

	// Load merged tasks
	mergedTasks, err := manager.GetTasksByPR(prNumber)
	if err != nil {
		t.Fatalf("Failed to load merged tasks: %v", err)
	}

	// Verify results
	if len(mergedTasks) != 3 {
		t.Errorf("Expected 3 merged tasks, got %d", len(mergedTasks))
	}

	// Find task with SourceCommentID 1 and TaskIndex 1 and verify its status was preserved
	var task1 *Task
	for i := range mergedTasks {
		if mergedTasks[i].SourceCommentID == 1 && mergedTasks[i].TaskIndex == 1 {
			task1 = &mergedTasks[i]
			break
		}
	}

	if task1 == nil {
		t.Fatalf("Task with SourceCommentID 1 and TaskIndex 1 not found in merged tasks")
	}

	// Status should be preserved (was "done")
	if task1.Status != "done" {
		t.Errorf("Expected task 1 status to be preserved as 'done', got: %s", task1.Status)
	}
}

// TestManager_UpdateTaskStatusByCommentAndIndex tests UUID-based task lookup
func TestManager_UpdateTaskStatusByCommentAndIndex(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	prNumber := 456

	// Create test tasks with UUID IDs
	testTasks := []Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655440001",
			Description:     "Test task 1",
			SourceCommentID: 100,
			TaskIndex:       0,
			Status:          "todo",
			OriginText:      "Original comment for task 1",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440002",
			Description:     "Test task 2",
			SourceCommentID: 100,
			TaskIndex:       1,
			Status:          "todo",
			OriginText:      "Original comment for task 2",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440003",
			Description:     "Test task 3",
			SourceCommentID: 200,
			TaskIndex:       0,
			Status:          "todo",
			OriginText:      "Different comment",
		},
	}

	// Save test tasks
	if err := manager.SaveTasks(prNumber, testTasks); err != nil {
		t.Fatalf("Failed to save test tasks: %v", err)
	}

	tests := []struct {
		name        string
		commentID   int64
		taskIndex   int
		newStatus   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Update existing task status",
			commentID:   100,
			taskIndex:   0,
			newStatus:   "done",
			expectError: false,
		},
		{
			name:        "Update second task from same comment",
			commentID:   100,
			taskIndex:   1,
			newStatus:   "doing",
			expectError: false,
		},
		{
			name:        "Update task from different comment",
			commentID:   200,
			taskIndex:   0,
			newStatus:   "done",
			expectError: false,
		},
		{
			name:        "Non-existent comment ID",
			commentID:   999,
			taskIndex:   0,
			newStatus:   "done",
			expectError: true,
			errorMsg:    "task not found",
		},
		{
			name:        "Non-existent task index",
			commentID:   100,
			taskIndex:   5,
			newStatus:   "done",
			expectError: true,
			errorMsg:    "task not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateTaskStatusByCommentAndIndex(prNumber, tt.commentID, tt.taskIndex, tt.newStatus)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				} else {
					// Verify the status was updated correctly
					tasks, loadErr := manager.GetTasksByPR(prNumber)
					if loadErr != nil {
						t.Fatalf("Failed to load tasks after update: %v", loadErr)
					}

					// Find the updated task
					var updatedTask *Task
					for i := range tasks {
						if tasks[i].SourceCommentID == tt.commentID && tasks[i].TaskIndex == tt.taskIndex {
							updatedTask = &tasks[i]
							break
						}
					}

					if updatedTask == nil {
						t.Errorf("Could not find updated task")
					} else if updatedTask.Status != tt.newStatus {
						t.Errorf("Expected status '%s', got '%s'", tt.newStatus, updatedTask.Status)
					}
				}
			}
		})
	}
}

// Helper function to initialize a test git repository
func initTestGitRepo(dir string) error {
	// This would normally use git commands, but for testing we'll mock it
	// In a real implementation, you might use go-git or exec commands
	return nil
}


// TestMergeTasksForCommentCancelStatus tests that mergeTasksForComment uses "cancel" not "cancelled"
func TestMergeTasksForCommentCancelStatus(t *testing.T) {
	// Create manager instance
	m := &Manager{}

	t.Run("empty_new_tasks_cancels_existing", func(t *testing.T) {
		existing := []Task{
			{
				ID:              "task-1",
				Description:     "Existing task 1",
				Status:          "todo",
				Priority:        "high",
				SourceCommentID: 12345,
			},
			{
				ID:              "task-2",
				Description:     "Existing task 2",
				Status:          "doing",
				Priority:        "medium",
				SourceCommentID: 12345,
			},
			{
				ID:              "task-3",
				Description:     "Existing task 3",
				Status:          "done",
				Priority:        "low",
				SourceCommentID: 12345,
			},
		}

		// Call the method with empty new tasks
		result := m.mergeTasksForComment(12345, existing, []Task{})

		// Verify results
		if len(result) != 3 {
			t.Errorf("Expected 3 tasks, got %d", len(result))
		}

		for _, task := range result {
			if task.ID == "task-3" {
				// Done tasks should remain done
				if task.Status != "done" {
					t.Errorf("Done task should remain done, got %s", task.Status)
				}
			} else {
				// Other tasks should be cancelled with "cancel" not "cancelled"
				if task.Status != "cancel" {
					t.Errorf("Non-done task should be marked as 'cancel', got %s", task.Status)
				}
			}
		}
	})
}
