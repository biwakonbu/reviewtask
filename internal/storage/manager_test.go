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

// TestManager_ReviewCache tests the review caching functionality
func TestManager_ReviewCache(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	prNumber := 789

	// Test loading non-existent cache returns empty cache
	cache, err := manager.LoadReviewCache(prNumber)
	if err != nil {
		t.Fatalf("Expected no error loading non-existent cache, got: %v", err)
	}
	if cache.PRNumber != prNumber {
		t.Errorf("Expected PR number %d, got %d", prNumber, cache.PRNumber)
	}
	if len(cache.CommentCaches) != 0 {
		t.Errorf("Expected empty comment caches, got %d entries", len(cache.CommentCaches))
	}

	// Add some comment caches
	cache.CommentCaches = []CommentCache{
		{
			CommentID:      12345,
			ContentHash:    "hash123",
			ThreadDepth:    2,
			LastProcessed:  "2024-01-01T10:00:00Z",
			TasksGenerated: []string{"task-1", "task-2"},
		},
		{
			CommentID:      67890,
			ContentHash:    "hash456",
			ThreadDepth:    0,
			LastProcessed:  "2024-01-01T10:05:00Z",
			TasksGenerated: []string{"task-3"},
		},
	}

	// Save cache
	if err := manager.SaveReviewCache(cache); err != nil {
		t.Fatalf("Failed to save review cache: %v", err)
	}

	// Load cache again and verify
	loadedCache, err := manager.LoadReviewCache(prNumber)
	if err != nil {
		t.Fatalf("Failed to load saved cache: %v", err)
	}

	if len(loadedCache.CommentCaches) != 2 {
		t.Errorf("Expected 2 comment caches, got %d", len(loadedCache.CommentCaches))
	}

	// Verify first comment cache
	comment1 := loadedCache.CommentCaches[0]
	if comment1.CommentID != 12345 {
		t.Errorf("Expected comment ID 12345, got %d", comment1.CommentID)
	}
	if comment1.ContentHash != "hash123" {
		t.Errorf("Expected hash 'hash123', got '%s'", comment1.ContentHash)
	}
	if comment1.ThreadDepth != 2 {
		t.Errorf("Expected thread depth 2, got %d", comment1.ThreadDepth)
	}
	if len(comment1.TasksGenerated) != 2 {
		t.Errorf("Expected 2 generated tasks, got %d", len(comment1.TasksGenerated))
	}
}

// TestManager_GenerateContentHash tests content hashing functionality
func TestManager_GenerateContentHash(t *testing.T) {
	manager := &Manager{}

	// Test comment with no replies
	comment1 := github.Comment{
		ID:      123,
		Body:    "This is a test comment",
		Author:  "testuser",
		File:    "test.go",
		Line:    42,
		Replies: []github.Reply{},
	}

	hash1 := manager.GenerateContentHash(comment1)
	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	// Test that identical comments produce same hash
	comment2 := github.Comment{
		ID:      123, // Same content
		Body:    "This is a test comment",
		Author:  "testuser",
		File:    "test.go",
		Line:    42,
		Replies: []github.Reply{},
	}

	hash2 := manager.GenerateContentHash(comment2)
	if hash1 != hash2 {
		t.Errorf("Expected identical comments to have same hash, got %s != %s", hash1, hash2)
	}

	// Test that different content produces different hash
	comment3 := github.Comment{
		ID:      123,
		Body:    "This is a different comment", // Different body
		Author:  "testuser",
		File:    "test.go",
		Line:    42,
		Replies: []github.Reply{},
	}

	hash3 := manager.GenerateContentHash(comment3)
	if hash1 == hash3 {
		t.Error("Expected different comments to have different hashes")
	}

	// Test comment with replies
	comment4 := github.Comment{
		ID:     123,
		Body:   "This is a test comment",
		Author: "testuser",
		File:   "test.go",
		Line:   42,
		Replies: []github.Reply{
			{
				ID:        456,
				Body:      "This is a reply",
				Author:    "reviewer",
				CreatedAt: "2024-01-01T10:00:00Z",
			},
		},
	}

	hash4 := manager.GenerateContentHash(comment4)
	if hash1 == hash4 {
		t.Error("Expected comment with replies to have different hash than comment without replies")
	}

	// Test that comments with same content but different IDs produce different hashes
	comment5 := github.Comment{
		ID:      456,                      // Different ID
		Body:    "This is a test comment", // Same content as comment1
		Author:  "testuser",
		File:    "test.go",
		Line:    42,
		Replies: []github.Reply{},
	}

	hash5 := manager.GenerateContentHash(comment5)
	if hash1 == hash5 {
		t.Error("Expected comments with same content but different IDs to have different hashes")
	}
}

// TestManager_DetectCommentChanges tests comment change detection
func TestManager_DetectCommentChanges(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	prNumber := 101

	// Setup: Create a cache with existing comments
	cache := &ReviewCache{
		PRNumber:    prNumber,
		LastUpdated: "2024-01-01T10:00:00Z",
		CommentCaches: []CommentCache{
			{
				CommentID:      100,
				ContentHash:    "existing_hash_100",
				ThreadDepth:    0,
				LastProcessed:  "2024-01-01T10:00:00Z",
				TasksGenerated: []string{"task-100"},
			},
			{
				CommentID:      200,
				ContentHash:    "existing_hash_200",
				ThreadDepth:    1,
				LastProcessed:  "2024-01-01T10:00:00Z",
				TasksGenerated: []string{"task-200"},
			},
		},
	}

	if err := manager.SaveReviewCache(cache); err != nil {
		t.Fatalf("Failed to save initial cache: %v", err)
	}

	// Create current comments
	currentComments := []github.Comment{
		// Existing comment - unchanged
		{
			ID:      100,
			Body:    "Original comment body",
			Author:  "user1",
			File:    "test.go",
			Line:    10,
			Replies: []github.Reply{},
		},
		// Existing comment - modified (different body)
		{
			ID:      200,
			Body:    "Modified comment body", // Different from cached version
			Author:  "user2",
			File:    "test.go",
			Line:    20,
			Replies: []github.Reply{},
		},
		// New comment
		{
			ID:      300,
			Body:    "New comment body",
			Author:  "user3",
			File:    "test.go",
			Line:    30,
			Replies: []github.Reply{},
		},
	}

	// We need to set up the hash for the existing unchanged comment to match
	existingHash := manager.GenerateContentHash(currentComments[0])
	cache.CommentCaches[0].ContentHash = existingHash
	if err := manager.SaveReviewCache(cache); err != nil {
		t.Fatalf("Failed to update cache with correct hash: %v", err)
	}

	// Test change detection
	newComments, modifiedComments, err := manager.DetectCommentChanges(prNumber, currentComments)
	if err != nil {
		t.Fatalf("Failed to detect comment changes: %v", err)
	}

	// Verify results
	if len(newComments) != 1 {
		t.Errorf("Expected 1 new comment, got %d", len(newComments))
	} else if newComments[0].ID != 300 {
		t.Errorf("Expected new comment ID 300, got %d", newComments[0].ID)
	}

	if len(modifiedComments) != 1 {
		t.Errorf("Expected 1 modified comment, got %d", len(modifiedComments))
	} else if modifiedComments[0].ID != 200 {
		t.Errorf("Expected modified comment ID 200, got %d", modifiedComments[0].ID)
	}
}

// TestManager_GetCachedComments tests retrieval of cached comments
func TestManager_GetCachedComments(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	prNumber := 102

	// Create test comments
	comments := []github.Comment{
		{
			ID:      100,
			Body:    "Comment 1",
			Author:  "user1",
			File:    "test.go",
			Line:    10,
			Replies: []github.Reply{},
		},
		{
			ID:      200,
			Body:    "Comment 2",
			Author:  "user2",
			File:    "test.go",
			Line:    20,
			Replies: []github.Reply{},
		},
	}

	// Setup cache for first comment only
	cache := &ReviewCache{
		PRNumber:    prNumber,
		LastUpdated: "2024-01-01T10:00:00Z",
		CommentCaches: []CommentCache{
			{
				CommentID:      100,
				ContentHash:    manager.GenerateContentHash(comments[0]),
				ThreadDepth:    0,
				LastProcessed:  "2024-01-01T10:00:00Z",
				TasksGenerated: []string{"task-100"},
			},
		},
	}

	if err := manager.SaveReviewCache(cache); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Get cached comments
	cachedComments, err := manager.GetCachedComments(prNumber, comments)
	if err != nil {
		t.Fatalf("Failed to get cached comments: %v", err)
	}

	// Should return only the first comment (cached)
	if len(cachedComments) != 1 {
		t.Errorf("Expected 1 cached comment, got %d", len(cachedComments))
	} else if cachedComments[0].ID != 100 {
		t.Errorf("Expected cached comment ID 100, got %d", cachedComments[0].ID)
	}
}

// TestManager_ClearCache tests cache clearing functionality
func TestManager_ClearCache(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	prNumber := 103

	// Create and save a cache
	cache := &ReviewCache{
		PRNumber:    prNumber,
		LastUpdated: "2024-01-01T10:00:00Z",
		CommentCaches: []CommentCache{
			{
				CommentID:      100,
				ContentHash:    "hash123",
				ThreadDepth:    0,
				LastProcessed:  "2024-01-01T10:00:00Z",
				TasksGenerated: []string{"task-100"},
			},
		},
	}

	if err := manager.SaveReviewCache(cache); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify cache exists
	loadedCache, err := manager.LoadReviewCache(prNumber)
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}
	if len(loadedCache.CommentCaches) != 1 {
		t.Errorf("Expected cache to exist with 1 entry, got %d", len(loadedCache.CommentCaches))
	}

	// Clear cache
	if err := manager.ClearCache(prNumber); err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify cache is cleared (should return empty cache)
	clearedCache, err := manager.LoadReviewCache(prNumber)
	if err != nil {
		t.Fatalf("Failed to load cache after clearing: %v", err)
	}
	if len(clearedCache.CommentCaches) != 0 {
		t.Errorf("Expected empty cache after clearing, got %d entries", len(clearedCache.CommentCaches))
	}

	// Test clearing non-existent cache (should not error)
	if err := manager.ClearCache(999); err != nil {
		t.Errorf("Expected no error when clearing non-existent cache, got: %v", err)
	}
}

// TestManager_UpdateCommentCache tests updating cache with grouped task IDs
func TestManager_UpdateCommentCache(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{baseDir: tempDir}

	prNumber := 104

	// Create test comments
	comments := []github.Comment{
		{
			ID:      100,
			Body:    "Comment 1",
			Author:  "user1",
			File:    "test.go",
			Line:    10,
			Replies: []github.Reply{},
		},
		{
			ID:      200,
			Body:    "Comment 2",
			Author:  "user2",
			File:    "test.go",
			Line:    20,
			Replies: []github.Reply{},
		},
	}

	// Test with grouped task IDs (comment 1 has 2 tasks, comment 2 has 1 task)
	taskIDGroups := [][]string{
		{"task-100-1", "task-100-2"}, // Comment 100 has 2 tasks
		{"task-200-1"},               // Comment 200 has 1 task
	}

	// Update cache with grouped task IDs
	if err := manager.UpdateCommentCache(prNumber, comments, taskIDGroups); err != nil {
		t.Fatalf("Failed to update cache with groups: %v", err)
	}

	// Load cache and verify
	cache, err := manager.LoadReviewCache(prNumber)
	if err != nil {
		t.Fatalf("Failed to load updated cache: %v", err)
	}

	if len(cache.CommentCaches) != 2 {
		t.Errorf("Expected 2 comment caches, got %d", len(cache.CommentCaches))
	}

	// Find cache for comment 100
	var comment100Cache *CommentCache
	for _, cc := range cache.CommentCaches {
		if cc.CommentID == 100 {
			comment100Cache = &cc
			break
		}
	}

	if comment100Cache == nil {
		t.Fatalf("Cache for comment 100 not found")
	}

	// Verify comment 100 has 2 task IDs
	if len(comment100Cache.TasksGenerated) != 2 {
		t.Errorf("Expected comment 100 to have 2 task IDs, got %d", len(comment100Cache.TasksGenerated))
	}

	expectedTaskIDs := []string{"task-100-1", "task-100-2"}
	for i, expectedID := range expectedTaskIDs {
		if i < len(comment100Cache.TasksGenerated) && comment100Cache.TasksGenerated[i] != expectedID {
			t.Errorf("Expected task ID %s at index %d, got %s", expectedID, i, comment100Cache.TasksGenerated[i])
		}
	}

	// Find cache for comment 200
	var comment200Cache *CommentCache
	for _, cc := range cache.CommentCaches {
		if cc.CommentID == 200 {
			comment200Cache = &cc
			break
		}
	}

	if comment200Cache == nil {
		t.Fatalf("Cache for comment 200 not found")
	}

	// Verify comment 200 has 1 task ID
	if len(comment200Cache.TasksGenerated) != 1 {
		t.Errorf("Expected comment 200 to have 1 task ID, got %d", len(comment200Cache.TasksGenerated))
	}

	if comment200Cache.TasksGenerated[0] != "task-200-1" {
		t.Errorf("Expected task ID 'task-200-1', got '%s'", comment200Cache.TasksGenerated[0])
	}
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
