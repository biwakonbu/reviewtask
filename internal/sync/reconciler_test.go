package sync

import (
	"context"
	"fmt"
	"testing"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// MockGitHubClient implements GitHubInterface for testing
type MockGitHubClient struct {
	threadStates          map[int64]bool
	resolvedThreads       []int64
	postedReplies         map[int64]string
	getAllThreadStatesErr error
	resolveThreadErr      error
	postReplyErr          error
}

func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		threadStates:    make(map[int64]bool),
		resolvedThreads: []int64{},
		postedReplies:   make(map[int64]string),
	}
}

func (m *MockGitHubClient) GetAllThreadStates(ctx context.Context, prNumber int) (map[int64]bool, error) {
	if m.getAllThreadStatesErr != nil {
		return nil, m.getAllThreadStatesErr
	}
	return m.threadStates, nil
}

func (m *MockGitHubClient) ResolveCommentThread(ctx context.Context, prNumber int, commentID int64) error {
	if m.resolveThreadErr != nil {
		return m.resolveThreadErr
	}
	m.resolvedThreads = append(m.resolvedThreads, commentID)
	// Update thread state to resolved
	m.threadStates[commentID] = true
	return nil
}

func (m *MockGitHubClient) PostReviewCommentReply(ctx context.Context, prNumber int, commentID int64, body string) error {
	if m.postReplyErr != nil {
		return m.postReplyErr
	}
	m.postedReplies[commentID] = body
	return nil
}

// MockStorageManager implements StorageInterface for testing
type MockStorageManager struct {
	tasks       []storage.Task
	getAllError error
}

func NewMockStorageManager() *MockStorageManager {
	return &MockStorageManager{
		tasks: []storage.Task{},
	}
}

func (m *MockStorageManager) GetAllTasks() ([]storage.Task, error) {
	if m.getAllError != nil {
		return nil, m.getAllError
	}
	return m.tasks, nil
}

// TestReconcileWithGitHub_AllTasksCompleteButUnresolved tests that threads are resolved
// when all local tasks are complete but the thread is unresolved on GitHub
func TestReconcileWithGitHub_AllTasksCompleteButUnresolved(t *testing.T) {
	ctx := context.Background()

	// Setup mock GitHub client with unresolved thread
	mockGitHub := NewMockGitHubClient()
	mockGitHub.threadStates[101] = false // Thread 101 is unresolved

	// Setup mock storage with completed tasks
	mockStorage := NewMockStorageManager()
	mockStorage.tasks = []storage.Task{
		{ID: "task1", SourceCommentID: 101, Status: "done", PRNumber: 1},
		{ID: "task2", SourceCommentID: 101, Status: "done", PRNumber: 1},
	}

	// Create reviews
	reviews := []github.Review{
		{
			Comments: []github.Comment{
				{ID: 101, Body: "Please fix this"},
			},
		},
	}

	// Create reconciler and run
	reconciler := NewReconciler(mockGitHub, mockStorage)
	result, err := reconciler.ReconcileWithGitHub(ctx, 1, reviews)

	// Verify
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.LocalTasksNeedingResolve != 1 {
		t.Errorf("Expected 1 task needing resolve, got %d", result.LocalTasksNeedingResolve)
	}

	if len(result.ResolvedThreads) != 1 {
		t.Errorf("Expected 1 resolved thread, got %d", len(result.ResolvedThreads))
	}

	if len(mockGitHub.resolvedThreads) != 1 || mockGitHub.resolvedThreads[0] != 101 {
		t.Errorf("Expected thread 101 to be resolved, got %v", mockGitHub.resolvedThreads)
	}
}

// TestReconcileWithGitHub_CancelTaskWithoutReply tests detection of cancel tasks without replies
func TestReconcileWithGitHub_CancelTaskWithoutReply(t *testing.T) {
	ctx := context.Background()

	// Setup mock GitHub client with unresolved thread
	mockGitHub := NewMockGitHubClient()
	mockGitHub.threadStates[102] = false // Thread 102 is unresolved

	// Setup mock storage with cancel task without reply
	mockStorage := NewMockStorageManager()
	mockStorage.tasks = []storage.Task{
		{
			ID:                  "task1",
			SourceCommentID:     102,
			Status:              "cancel",
			CancelCommentPosted: false, // No reply posted
			PRNumber:            1,
		},
	}

	// Create reviews
	reviews := []github.Review{
		{
			Comments: []github.Comment{
				{ID: 102, Body: "Consider refactoring"},
			},
		},
	}

	// Create reconciler and run
	reconciler := NewReconciler(mockGitHub, mockStorage)
	result, err := reconciler.ReconcileWithGitHub(ctx, 1, reviews)

	// Verify
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.CancelTasksWithoutReply != 1 {
		t.Errorf("Expected 1 cancel task without reply, got %d", result.CancelTasksWithoutReply)
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warnings about cancel task without reply")
	}

	// Thread should NOT be resolved if cancel task has no reply
	if len(mockGitHub.resolvedThreads) != 0 {
		t.Errorf("Thread should not be resolved when cancel task has no reply")
	}
}

// TestReconcileWithGitHub_MixedTaskStates tests handling of mixed task states
func TestReconcileWithGitHub_MixedTaskStates(t *testing.T) {
	ctx := context.Background()

	// Setup mock GitHub client
	mockGitHub := NewMockGitHubClient()
	mockGitHub.threadStates[103] = false // Thread 103 is unresolved

	// Setup mock storage with mixed task states
	mockStorage := NewMockStorageManager()
	mockStorage.tasks = []storage.Task{
		{ID: "task1", SourceCommentID: 103, Status: "done", PRNumber: 1},
		{ID: "task2", SourceCommentID: 103, Status: "pending", PRNumber: 1}, // Still pending
	}

	// Create reviews
	reviews := []github.Review{
		{
			Comments: []github.Comment{
				{ID: 103, Body: "Multiple issues here"},
			},
		},
	}

	// Create reconciler and run
	reconciler := NewReconciler(mockGitHub, mockStorage)
	result, err := reconciler.ReconcileWithGitHub(ctx, 1, reviews)

	// Verify
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Thread should NOT be resolved because not all tasks are complete
	if len(mockGitHub.resolvedThreads) != 0 {
		t.Errorf("Thread should not be resolved when tasks are incomplete")
	}

	if result.LocalTasksNeedingResolve != 0 {
		t.Errorf("Expected 0 tasks needing resolve, got %d", result.LocalTasksNeedingResolve)
	}
}

// TestReconcileWithGitHub_AlreadyResolvedOnGitHub tests that already resolved threads are not re-resolved
func TestReconcileWithGitHub_AlreadyResolvedOnGitHub(t *testing.T) {
	ctx := context.Background()

	// Setup mock GitHub client with already resolved thread
	mockGitHub := NewMockGitHubClient()
	mockGitHub.threadStates[104] = true // Thread 104 is already resolved

	// Setup mock storage with completed tasks
	mockStorage := NewMockStorageManager()
	mockStorage.tasks = []storage.Task{
		{ID: "task1", SourceCommentID: 104, Status: "done", PRNumber: 1},
	}

	// Create reviews
	reviews := []github.Review{
		{
			Comments: []github.Comment{
				{ID: 104, Body: "Already fixed"},
			},
		},
	}

	// Create reconciler and run
	reconciler := NewReconciler(mockGitHub, mockStorage)
	result, err := reconciler.ReconcileWithGitHub(ctx, 1, reviews)

	// Verify
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Thread should NOT be re-resolved
	if len(mockGitHub.resolvedThreads) != 0 {
		t.Errorf("Thread should not be re-resolved")
	}

	if result.LocalTasksNeedingResolve != 0 {
		t.Errorf("Expected 0 tasks needing resolve, got %d", result.LocalTasksNeedingResolve)
	}
}

// TestReconcileWithGitHub_ErrorHandling tests error handling
func TestReconcileWithGitHub_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("GetAllThreadStates error", func(t *testing.T) {
		mockGitHub := NewMockGitHubClient()
		mockGitHub.getAllThreadStatesErr = fmt.Errorf("API error")
		mockStorage := NewMockStorageManager()

		reconciler := NewReconciler(mockGitHub, mockStorage)
		_, err := reconciler.ReconcileWithGitHub(ctx, 1, []github.Review{})

		if err == nil {
			t.Error("Expected error when GetAllThreadStates fails")
		}
	})

	t.Run("GetAllTasks error", func(t *testing.T) {
		mockGitHub := NewMockGitHubClient()
		mockStorage := NewMockStorageManager()
		mockStorage.getAllError = fmt.Errorf("storage error")

		reconciler := NewReconciler(mockGitHub, mockStorage)
		_, err := reconciler.ReconcileWithGitHub(ctx, 1, []github.Review{})

		if err == nil {
			t.Error("Expected error when GetAllTasks fails")
		}
	})

	t.Run("ResolveCommentThread error", func(t *testing.T) {
		mockGitHub := NewMockGitHubClient()
		mockGitHub.threadStates[105] = false
		mockGitHub.resolveThreadErr = fmt.Errorf("resolve error")

		mockStorage := NewMockStorageManager()
		mockStorage.tasks = []storage.Task{
			{ID: "task1", SourceCommentID: 105, Status: "done", PRNumber: 1},
		}

		reviews := []github.Review{
			{Comments: []github.Comment{{ID: 105, Body: "Test"}}},
		}

		reconciler := NewReconciler(mockGitHub, mockStorage)
		result, err := reconciler.ReconcileWithGitHub(ctx, 1, reviews)

		if err != nil {
			t.Fatalf("Should not return error, but got: %v", err)
		}

		// Should have a warning about the failure
		if len(result.Warnings) == 0 {
			t.Error("Expected warning about resolve failure")
		}
	})
}

// TestUpdateCommentResolutionStates tests batch update of comment resolution states
func TestUpdateCommentResolutionStates(t *testing.T) {
	ctx := context.Background()

	mockGitHub := NewMockGitHubClient()
	mockGitHub.threadStates[201] = true
	mockGitHub.threadStates[202] = false

	mockStorage := NewMockStorageManager()

	reviews := []github.Review{
		{
			Comments: []github.Comment{
				{ID: 201, Body: "Comment 1"},
				{ID: 202, Body: "Comment 2"},
			},
		},
	}

	reconciler := NewReconciler(mockGitHub, mockStorage)
	updatedReviews, err := reconciler.UpdateCommentResolutionStates(ctx, 1, reviews)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(updatedReviews) != 1 {
		t.Fatalf("Expected 1 review, got %d", len(updatedReviews))
	}

	if len(updatedReviews[0].Comments) != 2 {
		t.Fatalf("Expected 2 comments, got %d", len(updatedReviews[0].Comments))
	}

	// Verify resolution states
	if !updatedReviews[0].Comments[0].GitHubThreadResolved {
		t.Error("Comment 201 should be marked as resolved")
	}

	if updatedReviews[0].Comments[1].GitHubThreadResolved {
		t.Error("Comment 202 should be marked as unresolved")
	}

	// Verify LastCheckedAt is set
	if updatedReviews[0].Comments[0].LastCheckedAt == "" {
		t.Error("LastCheckedAt should be set for comment 201")
	}
}
