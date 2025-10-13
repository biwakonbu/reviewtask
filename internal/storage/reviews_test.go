package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"reviewtask/internal/github"
)

// TestMarkCommentThreadAsResolved tests that MarkCommentThreadAsResolved
// correctly updates the GitHubThreadResolved field in reviews.json.
// This addresses Issue #233: Update reviews.json when resolving threads.
func TestMarkCommentThreadAsResolved(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	manager := &Manager{baseDir: tmpDir}

	// Create initial reviews with unresolved comments
	prNumber := 123
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:                   100,
					Body:                 "This needs to be fixed",
					GitHubThreadResolved: false,
					LastCheckedAt:        "2023-01-01T10:00:00Z",
				},
				{
					ID:                   200,
					Body:                 "Another issue here",
					GitHubThreadResolved: false,
					LastCheckedAt:        "2023-01-01T10:00:00Z",
				},
			},
		},
	}

	// Save initial reviews
	err := manager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save initial reviews: %v", err)
	}

	// Mark comment 100 as resolved
	err = manager.MarkCommentThreadAsResolved(prNumber, 100)
	if err != nil {
		t.Fatalf("MarkCommentThreadAsResolved failed: %v", err)
	}

	// Load reviews and verify the change
	extendedFile, err := manager.LoadExtendedReviews(prNumber)
	if err != nil {
		t.Fatalf("Failed to load reviews: %v", err)
	}

	// Find comment 100 and verify it's marked as resolved
	found := false
	for _, review := range extendedFile.Reviews {
		for _, comment := range review.Comments {
			if comment.ID == 100 {
				found = true
				if !comment.GitHubThreadResolved {
					t.Errorf("Comment 100 GitHubThreadResolved should be true, got false")
				}
				// Verify LastCheckedAt was updated
				if comment.LastCheckedAt == "2023-01-01T10:00:00Z" {
					t.Errorf("Comment 100 LastCheckedAt should be updated, still has old value")
				}
				// Verify LastCheckedAt is valid timestamp
				_, err := time.Parse("2006-01-02T15:04:05Z", comment.LastCheckedAt)
				if err != nil {
					t.Errorf("Comment 100 LastCheckedAt has invalid format: %v", err)
				}
			}
			if comment.ID == 200 {
				// Comment 200 should remain unchanged
				if comment.GitHubThreadResolved {
					t.Errorf("Comment 200 GitHubThreadResolved should still be false, got true")
				}
			}
		}
	}

	if !found {
		t.Errorf("Comment 100 not found in reviews")
	}
}

// TestMarkCommentThreadAsResolved_NonExistentComment tests error handling
// when trying to mark a non-existent comment as resolved.
func TestMarkCommentThreadAsResolved_NonExistentComment(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{baseDir: tmpDir}

	prNumber := 123
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:                   100,
					Body:                 "Test comment",
					GitHubThreadResolved: false,
				},
			},
		},
	}

	err := manager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save initial reviews: %v", err)
	}

	// Try to mark non-existent comment 999 as resolved
	err = manager.MarkCommentThreadAsResolved(prNumber, 999)
	if err == nil {
		t.Errorf("Expected error for non-existent comment, got nil")
	}

	// Verify error message
	expectedErrMsg := "comment 999 not found"
	if err != nil && err.Error() != "comment 999 not found in PR 123" {
		t.Errorf("Expected error containing %q, got %q", expectedErrMsg, err.Error())
	}
}

// TestMarkCommentThreadAsResolved_NonExistentPR tests error handling
// when trying to mark a comment in a non-existent PR.
func TestMarkCommentThreadAsResolved_NonExistentPR(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{baseDir: tmpDir}

	// Try to mark comment in non-existent PR
	err := manager.MarkCommentThreadAsResolved(999, 100)
	if err == nil {
		t.Errorf("Expected error for non-existent PR, got nil")
	}
}

// TestMarkCommentThreadAsResolved_MultipleComments tests marking multiple
// comments as resolved independently.
func TestMarkCommentThreadAsResolved_MultipleComments(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{baseDir: tmpDir}

	prNumber := 123
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "reviewer1",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:                   100,
					Body:                 "Issue 1",
					GitHubThreadResolved: false,
				},
				{
					ID:                   200,
					Body:                 "Issue 2",
					GitHubThreadResolved: false,
				},
				{
					ID:                   300,
					Body:                 "Issue 3",
					GitHubThreadResolved: false,
				},
			},
		},
	}

	err := manager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save initial reviews: %v", err)
	}

	// Mark comments as resolved one by one
	for _, commentID := range []int64{100, 200, 300} {
		err = manager.MarkCommentThreadAsResolved(prNumber, commentID)
		if err != nil {
			t.Fatalf("Failed to mark comment %d as resolved: %v", commentID, err)
		}

		// Verify the specific comment is marked as resolved
		extendedFile, err := manager.LoadExtendedReviews(prNumber)
		if err != nil {
			t.Fatalf("Failed to load reviews: %v", err)
		}

		for _, review := range extendedFile.Reviews {
			for _, comment := range review.Comments {
				if comment.ID == commentID {
					if !comment.GitHubThreadResolved {
						t.Errorf("Comment %d should be resolved after marking", commentID)
					}
				}
			}
		}
	}

	// Verify all comments are now resolved
	extendedFile, err := manager.LoadExtendedReviews(prNumber)
	if err != nil {
		t.Fatalf("Failed to load reviews: %v", err)
	}

	for _, review := range extendedFile.Reviews {
		for _, comment := range review.Comments {
			if !comment.GitHubThreadResolved {
				t.Errorf("Comment %d should be resolved, got false", comment.ID)
			}
		}
	}
}

// TestMarkCommentThreadAsResolved_Idempotent tests that marking an already
// resolved comment as resolved again is idempotent (no error, state unchanged).
func TestMarkCommentThreadAsResolved_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{baseDir: tmpDir}

	prNumber := 123
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:                   100,
					Body:                 "Test comment",
					GitHubThreadResolved: false,
				},
			},
		},
	}

	err := manager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save initial reviews: %v", err)
	}

	// Mark as resolved first time
	err = manager.MarkCommentThreadAsResolved(prNumber, 100)
	if err != nil {
		t.Fatalf("First MarkCommentThreadAsResolved failed: %v", err)
	}

	// Get the timestamp after first marking
	extendedFile, err := manager.LoadExtendedReviews(prNumber)
	if err != nil {
		t.Fatalf("Failed to load reviews: %v", err)
	}
	firstTimestamp := extendedFile.Reviews[0].Comments[0].LastCheckedAt

	// Wait a bit to ensure timestamp would change if updated
	time.Sleep(10 * time.Millisecond)

	// Mark as resolved second time (should be idempotent)
	err = manager.MarkCommentThreadAsResolved(prNumber, 100)
	if err != nil {
		t.Fatalf("Second MarkCommentThreadAsResolved failed: %v", err)
	}

	// Verify comment is still resolved
	extendedFile, err = manager.LoadExtendedReviews(prNumber)
	if err != nil {
		t.Fatalf("Failed to load reviews: %v", err)
	}

	comment := extendedFile.Reviews[0].Comments[0]
	if !comment.GitHubThreadResolved {
		t.Errorf("Comment should still be resolved after second marking")
	}

	// Timestamp should be updated (operation is not a no-op)
	if comment.LastCheckedAt == firstTimestamp {
		t.Logf("Note: Timestamp not updated on second marking (may be expected behavior)")
	}
}

// TestMarkCommentThreadAsResolved_PreservesOtherFields tests that marking
// a comment as resolved doesn't modify other comment fields.
func TestMarkCommentThreadAsResolved_PreservesOtherFields(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &Manager{baseDir: tmpDir}

	prNumber := 123
	originalComment := github.Comment{
		ID:                   100,
		File:                 "test.go",
		Line:                 42,
		Body:                 "This is a test comment with important content",
		Author:               "test-author",
		CreatedAt:            "2023-01-01T10:00:00Z",
		URL:                  "https://github.com/test/repo/pull/123#discussion_r100",
		GitHubThreadResolved: false,
		LastCheckedAt:        "2023-01-01T10:00:00Z",
		TasksGenerated:       true,
		AllTasksCompleted:    false,
		Replies: []github.Reply{
			{
				ID:        101,
				Body:      "Test reply",
				Author:    "reply-author",
				CreatedAt: "2023-01-01T11:00:00Z",
			},
		},
	}

	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{originalComment},
		},
	}

	err := manager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save initial reviews: %v", err)
	}

	// Mark comment as resolved
	err = manager.MarkCommentThreadAsResolved(prNumber, 100)
	if err != nil {
		t.Fatalf("MarkCommentThreadAsResolved failed: %v", err)
	}

	// Load and verify all fields are preserved except GitHubThreadResolved and LastCheckedAt
	extendedFile, err := manager.LoadExtendedReviews(prNumber)
	if err != nil {
		t.Fatalf("Failed to load reviews: %v", err)
	}

	comment := extendedFile.Reviews[0].Comments[0]

	// Verify modified fields
	if !comment.GitHubThreadResolved {
		t.Errorf("GitHubThreadResolved should be true")
	}
	if comment.LastCheckedAt == originalComment.LastCheckedAt {
		t.Errorf("LastCheckedAt should be updated")
	}

	// Verify preserved fields
	if comment.ID != originalComment.ID {
		t.Errorf("ID changed: expected %d, got %d", originalComment.ID, comment.ID)
	}
	if comment.File != originalComment.File {
		t.Errorf("File changed: expected %s, got %s", originalComment.File, comment.File)
	}
	if comment.Line != originalComment.Line {
		t.Errorf("Line changed: expected %d, got %d", originalComment.Line, comment.Line)
	}
	if comment.Body != originalComment.Body {
		t.Errorf("Body changed")
	}
	if comment.Author != originalComment.Author {
		t.Errorf("Author changed: expected %s, got %s", originalComment.Author, comment.Author)
	}
	if comment.CreatedAt != originalComment.CreatedAt {
		t.Errorf("CreatedAt changed")
	}
	if comment.URL != originalComment.URL {
		t.Errorf("URL changed")
	}
	if comment.TasksGenerated != originalComment.TasksGenerated {
		t.Errorf("TasksGenerated changed")
	}
	if comment.AllTasksCompleted != originalComment.AllTasksCompleted {
		t.Errorf("AllTasksCompleted changed")
	}
	if len(comment.Replies) != len(originalComment.Replies) {
		t.Errorf("Replies count changed: expected %d, got %d", len(originalComment.Replies), len(comment.Replies))
	}
}

// TestMarkCommentThreadAsResolved_FileSystemPersistence tests that changes
// are actually persisted to disk and can be read back after process restart.
func TestMarkCommentThreadAsResolved_FileSystemPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first manager instance and save reviews
	manager1 := &Manager{baseDir: tmpDir}
	prNumber := 123
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:                   100,
					Body:                 "Test comment",
					GitHubThreadResolved: false,
				},
			},
		},
	}

	err := manager1.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save initial reviews: %v", err)
	}

	// Mark comment as resolved
	err = manager1.MarkCommentThreadAsResolved(prNumber, 100)
	if err != nil {
		t.Fatalf("MarkCommentThreadAsResolved failed: %v", err)
	}

	// Create second manager instance (simulating process restart)
	manager2 := &Manager{baseDir: tmpDir}

	// Load reviews with second instance
	extendedFile, err := manager2.LoadExtendedReviews(prNumber)
	if err != nil {
		t.Fatalf("Failed to load reviews with second manager: %v", err)
	}

	// Verify the change persisted across manager instances
	if len(extendedFile.Reviews) == 0 || len(extendedFile.Reviews[0].Comments) == 0 {
		t.Fatalf("No reviews or comments found after reload")
	}

	comment := extendedFile.Reviews[0].Comments[0]
	if !comment.GitHubThreadResolved {
		t.Errorf("GitHubThreadResolved should be true after persistence")
	}

	// Verify the actual file exists and is readable
	// The path should be: <tmpDir>/.pr-review/pr-<number>/reviews.json
	reviewsPath := filepath.Join(manager2.getPRDir(prNumber), "reviews.json")
	if _, err := os.Stat(reviewsPath); os.IsNotExist(err) {
		t.Errorf("reviews.json file does not exist at expected path: %s", reviewsPath)
	}
}
