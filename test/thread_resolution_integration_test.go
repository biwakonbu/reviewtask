package test

import (
	"context"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
	"reviewtask/internal/threads"
)

// TestThreadResolutionWorkflow tests the complete workflow of thread resolution
// including checking GitHubThreadResolved field and updating reviews.json.
// This addresses Issue #233: isCommentResolved and reviews.json update integration.
func TestThreadResolutionWorkflow(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tmpDir)

	// Setup: Create test PR with reviews
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
					File:                 "test.go",
					Line:                 10,
					Author:               "reviewer1",
				},
				{
					ID:                   200,
					Body:                 "Another issue",
					GitHubThreadResolved: false,
					File:                 "test.go",
					Line:                 20,
					Author:               "reviewer1",
				},
				{
					ID:                   300,
					Body:                 "Already resolved issue",
					GitHubThreadResolved: true,
					File:                 "test.go",
					Line:                 30,
					Author:               "reviewer1",
				},
			},
		},
	}

	// Save initial reviews
	err := storageManager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save initial reviews: %v", err)
	}

	// Create tasks for comments 100 and 200 (300 is resolved, should not generate tasks)
	tasks := []storage.Task{
		{
			ID:              "task-100-1",
			Description:     "Fix issue in comment 100",
			SourceCommentID: 100,
			PRNumber:        prNumber,
			Status:          "todo",
			Priority:        "high",
		},
		{
			ID:              "task-200-1",
			Description:     "Fix issue in comment 200",
			SourceCommentID: 200,
			PRNumber:        prNumber,
			Status:          "todo",
			Priority:        "medium",
		},
	}

	// Save all tasks at once
	err = storageManager.SaveTasks(prNumber, tasks)
	if err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Test 1: Verify initial state - comment 300 should not have tasks due to GitHubThreadResolved=true
	t.Run("ResolvedCommentHasNoTasks", func(t *testing.T) {
		allTasks, err := storageManager.GetAllTasks()
		if err != nil {
			t.Fatalf("Failed to get all tasks: %v", err)
		}

		// Should only have 2 tasks (for comments 100 and 200), not comment 300
		if len(allTasks) != 2 {
			t.Errorf("Expected 2 tasks (excluding resolved comment 300), got %d", len(allTasks))
		}

		// Verify no task exists for comment 300
		for _, task := range allTasks {
			if task.SourceCommentID == 300 {
				t.Errorf("Found task for resolved comment 300, should not exist")
			}
		}
	})

	// Test 2: Complete task for comment 100 and mark thread as resolved
	t.Run("CompleteTaskAndMarkThreadResolved", func(t *testing.T) {
		// Update task status to done
		err := storageManager.UpdateTaskStatus("task-100-1", "done")
		if err != nil {
			t.Fatalf("Failed to update task status: %v", err)
		}

		// Mark comment thread as resolved (simulating done workflow)
		err = storageManager.MarkCommentThreadAsResolved(prNumber, 100)
		if err != nil {
			t.Fatalf("Failed to mark comment thread as resolved: %v", err)
		}

		// Verify comment 100 is now marked as resolved in reviews.json
		extendedFile, err := storageManager.LoadExtendedReviews(prNumber)
		if err != nil {
			t.Fatalf("Failed to load reviews: %v", err)
		}

		found := false
		for _, review := range extendedFile.Reviews {
			for _, comment := range review.Comments {
				if comment.ID == 100 {
					found = true
					if !comment.GitHubThreadResolved {
						t.Errorf("Comment 100 should be marked as resolved after done workflow")
					}
					// Verify LastCheckedAt was updated
					if comment.LastCheckedAt == "" {
						t.Errorf("Comment 100 LastCheckedAt should be updated")
					}
				}
			}
		}

		if !found {
			t.Errorf("Comment 100 not found in reviews")
		}
	})

	// Test 3: Verify that subsequent task generation would skip comment 100
	t.Run("SubsequentGenerationSkipsResolvedComment", func(t *testing.T) {
		// Load reviews again
		extendedFile, err := storageManager.LoadExtendedReviews(prNumber)
		if err != nil {
			t.Fatalf("Failed to load reviews: %v", err)
		}

		// Count comments that should generate tasks
		unresolvedCount := 0
		for _, review := range extendedFile.Reviews {
			for _, comment := range review.Comments {
				// Comment should be skipped if GitHubThreadResolved is true
				if !comment.GitHubThreadResolved {
					unresolvedCount++
				}
			}
		}

		// Only comment 200 should be unresolved
		// (100 was marked resolved, 300 was already resolved)
		if unresolvedCount != 1 {
			t.Errorf("Expected 1 unresolved comment, got %d", unresolvedCount)
		}
	})

	// Test 4: Thread resolution mode - complete all tasks for a comment
	t.Run("ThreadResolutionModeComplete", func(t *testing.T) {
		// Mark task-200-1 as done first
		err := storageManager.UpdateTaskStatus("task-200-1", "done")
		if err != nil {
			t.Fatalf("Failed to update task status: %v", err)
		}

		cfg := &config.Config{
			DoneWorkflow: config.DoneWorkflow{
				EnableAutoResolve: "complete",
			},
		}

		resolver := threads.NewThreadResolver(cfg, storageManager, nil)

		// Get task for comment 200
		task := &storage.Task{
			ID:              "task-200-1",
			Description:     "Fix issue in comment 200",
			SourceCommentID: 200,
			PRNumber:        prNumber,
			Status:          "done",
		}

		// Check resolution status
		status, err := resolver.GetResolutionStatus(context.Background(), task)
		if err != nil {
			t.Fatalf("Failed to get resolution status: %v", err)
		}

		// In complete mode, thread should be resolved when all tasks are done
		if status.TotalTasks != 1 || status.CompletedTasks != 1 {
			t.Errorf("Expected 1/1 tasks complete, got %d/%d", status.CompletedTasks, status.TotalTasks)
		}

		if !status.ThreadResolved {
			t.Errorf("Thread should be resolved when all tasks complete in 'complete' mode")
		}
	})
}

// TestThreadResolutionWithMultipleTasks tests thread resolution when a comment has multiple tasks.
func TestThreadResolutionWithMultipleTasks(t *testing.T) {
	tmpDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tmpDir)

	prNumber := 456
	commentID := int64(1000)

	// Setup: Create review with one comment
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:                   commentID,
					Body:                 "Multiple issues here",
					GitHubThreadResolved: false,
					File:                 "test.go",
					Line:                 50,
				},
			},
		},
	}

	err := storageManager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save reviews: %v", err)
	}

	// Create multiple tasks for the same comment
	tasks := []storage.Task{
		{
			ID:              "task-1000-1",
			Description:     "Fix issue 1",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "todo",
			TaskIndex:       0,
		},
		{
			ID:              "task-1000-2",
			Description:     "Fix issue 2",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "todo",
			TaskIndex:       1,
		},
		{
			ID:              "task-1000-3",
			Description:     "Fix issue 3",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "todo",
			TaskIndex:       2,
		},
	}

	// Save all tasks at once
	err = storageManager.SaveTasks(prNumber, tasks)
	if err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	cfg := &config.Config{
		DoneWorkflow: config.DoneWorkflow{
			EnableAutoResolve: "complete",
		},
	}

	resolver := threads.NewThreadResolver(cfg, storageManager, nil)

	// Test 1: Complete first task - thread should NOT be resolved
	t.Run("PartialCompletion_ThreadNotResolved", func(t *testing.T) {
		err := storageManager.UpdateTaskStatus("task-1000-1", "done")
		if err != nil {
			t.Fatalf("Failed to update task status: %v", err)
		}

		task := &storage.Task{
			ID:              "task-1000-1",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "done",
		}

		status, err := resolver.GetResolutionStatus(context.Background(), task)
		if err != nil {
			t.Fatalf("Failed to get resolution status: %v", err)
		}

		if status.ThreadResolved {
			t.Errorf("Thread should NOT be resolved when only 1/3 tasks complete")
		}

		if status.CompletedTasks != 1 || status.TotalTasks != 3 {
			t.Errorf("Expected 1/3 tasks complete, got %d/%d", status.CompletedTasks, status.TotalTasks)
		}
	})

	// Test 2: Complete second task - thread should still NOT be resolved
	t.Run("TwoOfThree_ThreadNotResolved", func(t *testing.T) {
		err := storageManager.UpdateTaskStatus("task-1000-2", "done")
		if err != nil {
			t.Fatalf("Failed to update task status: %v", err)
		}

		task := &storage.Task{
			ID:              "task-1000-2",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "done",
		}

		status, err := resolver.GetResolutionStatus(context.Background(), task)
		if err != nil {
			t.Fatalf("Failed to get resolution status: %v", err)
		}

		if status.ThreadResolved {
			t.Errorf("Thread should NOT be resolved when only 2/3 tasks complete")
		}

		if status.CompletedTasks != 2 || status.TotalTasks != 3 {
			t.Errorf("Expected 2/3 tasks complete, got %d/%d", status.CompletedTasks, status.TotalTasks)
		}
	})

	// Test 3: Complete third task - thread SHOULD be resolved
	t.Run("AllComplete_ThreadResolved", func(t *testing.T) {
		err := storageManager.UpdateTaskStatus("task-1000-3", "done")
		if err != nil {
			t.Fatalf("Failed to update task status: %v", err)
		}

		task := &storage.Task{
			ID:              "task-1000-3",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "done",
		}

		status, err := resolver.GetResolutionStatus(context.Background(), task)
		if err != nil {
			t.Fatalf("Failed to get resolution status: %v", err)
		}

		if !status.ThreadResolved {
			t.Errorf("Thread SHOULD be resolved when all 3/3 tasks complete")
		}

		if status.CompletedTasks != 3 || status.TotalTasks != 3 {
			t.Errorf("Expected 3/3 tasks complete, got %d/%d", status.CompletedTasks, status.TotalTasks)
		}

		// Verify we can mark the thread as resolved without error
		err = storageManager.MarkCommentThreadAsResolved(prNumber, commentID)
		if err != nil {
			t.Errorf("Failed to mark thread as resolved: %v", err)
		}

		// Verify the field was updated
		extendedFile, err := storageManager.LoadExtendedReviews(prNumber)
		if err != nil {
			t.Fatalf("Failed to load reviews: %v", err)
		}

		found := false
		for _, review := range extendedFile.Reviews {
			for _, comment := range review.Comments {
				if comment.ID == commentID {
					found = true
					if !comment.GitHubThreadResolved {
						t.Errorf("Comment should be marked as resolved after MarkCommentThreadAsResolved")
					}
				}
			}
		}

		if !found {
			t.Errorf("Comment not found in reviews after resolution")
		}
	})
}

// TestThreadResolutionModes tests different thread resolution modes (immediate, complete, disabled).
func TestThreadResolutionModes(t *testing.T) {
	tmpDir := t.TempDir()
	storageManager := storage.NewManagerWithBase(tmpDir)

	prNumber := 789
	commentID := int64(2000)

	// Setup: Create review with one comment and two tasks
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "COMMENTED",
			Comments: []github.Comment{
				{
					ID:                   commentID,
					Body:                 "Test comment",
					GitHubThreadResolved: false,
				},
			},
		},
	}

	err := storageManager.SaveExtendedReviews(prNumber, reviews)
	if err != nil {
		t.Fatalf("Failed to save reviews: %v", err)
	}

	tasks := []storage.Task{
		{
			ID:              "task-2000-1",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "done",
		},
		{
			ID:              "task-2000-2",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "todo",
		},
	}

	// Save all tasks at once
	err = storageManager.SaveTasks(prNumber, tasks)
	if err != nil {
		t.Fatalf("Failed to save tasks: %v", err)
	}

	// Test 1: Immediate mode - should always resolve
	t.Run("ImmediateMode_AlwaysResolves", func(t *testing.T) {
		cfg := &config.Config{
			DoneWorkflow: config.DoneWorkflow{
				EnableAutoResolve: "immediate",
			},
		}

		resolver := threads.NewThreadResolver(cfg, storageManager, nil)

		task := &storage.Task{
			ID:              "task-2000-1",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "done",
		}

		status, err := resolver.GetResolutionStatus(context.Background(), task)
		if err != nil {
			t.Fatalf("Failed to get resolution status: %v", err)
		}

		// Immediate mode should resolve even when not all tasks are complete
		if !status.ThreadResolved {
			t.Errorf("Immediate mode should resolve thread even when only 1/2 tasks complete")
		}
	})

	// Test 2: Complete mode - should NOT resolve until all tasks done
	t.Run("CompleteMode_WaitsForAllTasks", func(t *testing.T) {
		cfg := &config.Config{
			DoneWorkflow: config.DoneWorkflow{
				EnableAutoResolve: "complete",
			},
		}

		resolver := threads.NewThreadResolver(cfg, storageManager, nil)

		task := &storage.Task{
			ID:              "task-2000-1",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "done",
		}

		status, err := resolver.GetResolutionStatus(context.Background(), task)
		if err != nil {
			t.Fatalf("Failed to get resolution status: %v", err)
		}

		// Complete mode should NOT resolve when only 1/2 tasks complete
		if status.ThreadResolved {
			t.Errorf("Complete mode should NOT resolve thread when only 1/2 tasks complete")
		}

		if status.CompletedTasks != 1 || status.TotalTasks != 2 {
			t.Errorf("Expected 1/2 tasks complete, got %d/%d", status.CompletedTasks, status.TotalTasks)
		}
	})

	// Test 3: Disabled mode - should never resolve
	t.Run("DisabledMode_NeverResolves", func(t *testing.T) {
		cfg := &config.Config{
			DoneWorkflow: config.DoneWorkflow{
				EnableAutoResolve: "disabled",
			},
		}

		resolver := threads.NewThreadResolver(cfg, storageManager, nil)

		task := &storage.Task{
			ID:              "task-2000-1",
			SourceCommentID: commentID,
			PRNumber:        prNumber,
			Status:          "done",
		}

		status, err := resolver.GetResolutionStatus(context.Background(), task)
		if err != nil {
			t.Fatalf("Failed to get resolution status: %v", err)
		}

		// Disabled mode should never resolve
		if status.ThreadResolved {
			t.Errorf("Disabled mode should never resolve thread")
		}

		if status.ResolveMode != threads.ResolveModeDisabled {
			t.Errorf("Expected disabled mode, got: %s", status.ResolveMode)
		}
	})
}
