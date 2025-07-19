package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gh-review-task/internal/github"
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
		prDir := filepath.Join(tempDir, "PR-"+string(rune(pr.prNumber+'0')))
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
		prDir := filepath.Join(tempDir, "PR-"+string(rune(prNum+'0')))
		if prNum >= 10 {
			prDir = filepath.Join(tempDir, "PR-10")
		}
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
			ID:              "comment-1-task-1",
			Description:     "Existing task 1",
			SourceCommentID: 1,
			TaskIndex:       1,
			Status:          "done",
			OriginText:      "Original comment",
		},
		{
			ID:              "comment-1-task-2",
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
			ID:              "comment-1-task-1",
			Description:     "Updated task 1",
			SourceCommentID: 1,
			TaskIndex:       1,
			Status:          "todo",
			OriginText:      "Original comment", // Same origin text
		},
		{
			ID:              "comment-1-task-2",
			Description:     "Updated task 2",
			SourceCommentID: 1,
			TaskIndex:       2,
			Status:          "todo",
			OriginText:      "Original comment",
		},
		{
			ID:              "comment-1-task-3",
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

	// Find task 1 and verify its status was preserved
	var task1 *Task
	for i := range mergedTasks {
		if mergedTasks[i].ID == "comment-1-task-1" {
			task1 = &mergedTasks[i]
			break
		}
	}

	if task1 == nil {
		t.Fatalf("Task 1 not found in merged tasks")
	}

	// Status should be preserved (was "done")
	if task1.Status != "done" {
		t.Errorf("Expected task 1 status to be preserved as 'done', got: %s", task1.Status)
	}
}

// Helper function to initialize a test git repository
func initTestGitRepo(dir string) error {
	// This would normally use git commands, but for testing we'll mock it
	// In a real implementation, you might use go-git or exec commands
	return nil
}
