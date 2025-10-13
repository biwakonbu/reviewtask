package sync

import (
	"context"
	"fmt"
	"time"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// ReconciliationResult contains the results of reconciling local tasks with GitHub state
type ReconciliationResult struct {
	TotalComments            int
	ResolvedOnGitHub         int
	LocalTasksNeedingResolve int
	CancelTasksWithoutReply  int
	ResolvedThreads          []int64
	Warnings                 []string
}

// Reconciler handles synchronization between local task state and GitHub thread state
type Reconciler struct {
	githubClient GitHubInterface
	storage      StorageInterface
}

// GitHubInterface defines GitHub operations needed by reconciler
type GitHubInterface interface {
	GetAllThreadStates(ctx context.Context, prNumber int) (map[int64]bool, error)
	ResolveCommentThread(ctx context.Context, prNumber int, commentID int64) error
	PostReviewCommentReply(ctx context.Context, prNumber int, commentID int64, body string) error
}

// StorageInterface defines storage operations needed by reconciler
type StorageInterface interface {
	GetAllTasks() ([]storage.Task, error)
}

// NewReconciler creates a new reconciler instance
func NewReconciler(githubClient GitHubInterface, storageManager StorageInterface) *Reconciler {
	return &Reconciler{
		githubClient: githubClient,
		storage:      storageManager,
	}
}

// ReconcileWithGitHub reconciles local task state with GitHub thread resolution state
// This ensures that:
// 1. Tasks marked as done/cancel locally but unresolved on GitHub are resolved
// 2. Cancel tasks have a reply comment posted explaining the cancellation
func (r *Reconciler) ReconcileWithGitHub(ctx context.Context, prNumber int, reviews []github.Review) (*ReconciliationResult, error) {
	result := &ReconciliationResult{
		ResolvedThreads: []int64{},
		Warnings:        []string{},
	}

	// Step 1: Get all thread states from GitHub in batch
	threadStates, err := r.githubClient.GetAllThreadStates(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread states from GitHub: %w", err)
	}

	// Count total comments and resolved threads
	for _, review := range reviews {
		for _, comment := range review.Comments {
			if comment.ID == 0 {
				continue // Skip embedded comments without ID
			}
			result.TotalComments++
			if isResolved, exists := threadStates[comment.ID]; exists && isResolved {
				result.ResolvedOnGitHub++
			}
		}
	}

	// Step 2: Load all local tasks
	tasks, err := r.storage.GetAllTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load local tasks: %w", err)
	}

	// Step 3: Build a map of comment ID -> tasks for that comment
	commentTasksMap := make(map[int64][]storage.Task)
	for _, task := range tasks {
		if task.SourceCommentID != 0 {
			commentTasksMap[task.SourceCommentID] = append(commentTasksMap[task.SourceCommentID], task)
		}
	}

	// Step 4: Reconcile each comment's tasks with GitHub state
	for commentID, tasksForComment := range commentTasksMap {
		// Check if thread is resolved on GitHub
		isResolvedOnGitHub, exists := threadStates[commentID]
		if !exists {
			// Comment doesn't have a thread on GitHub, skip
			continue
		}

		// Check if all tasks for this comment are completed locally
		allTasksComplete := true
		hasCancelTaskWithoutReply := false

		for _, task := range tasksForComment {
			// Check task completion
			// For cancel tasks, they're only considered complete if reply was posted
			if task.Status == "done" {
				// Done tasks are always complete
				continue
			} else if task.Status == "cancel" && task.CancelCommentPosted {
				// Cancel tasks with posted reply are complete
				continue
			} else if task.Status == "cancel" && !task.CancelCommentPosted {
				// Cancel tasks without reply are NOT complete
				allTasksComplete = false
				hasCancelTaskWithoutReply = true
				result.CancelTasksWithoutReply++
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Task %s (comment %d) is cancelled but no reply comment posted", task.ID, commentID))
			} else {
				// Pending, doing, or any other status
				allTasksComplete = false
			}
		}

		// If all tasks are complete locally but thread is not resolved on GitHub, resolve it
		// Note: This will only happen if all cancel tasks have posted replies
		if allTasksComplete && !isResolvedOnGitHub && !hasCancelTaskWithoutReply {
			result.LocalTasksNeedingResolve++

			// Attempt to resolve the thread on GitHub
			if err := r.githubClient.ResolveCommentThread(ctx, prNumber, commentID); err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Failed to resolve thread for comment %d: %v", commentID, err))
			} else {
				result.ResolvedThreads = append(result.ResolvedThreads, commentID)
			}
		}

		// If there are cancel tasks without reply, warn user (additional warning)
		if hasCancelTaskWithoutReply && !isResolvedOnGitHub {
			// Thread is not resolved, user should post reply before resolving
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("⚠️  Comment %d has cancelled tasks without explanation - please post reply before resolving", commentID))
		}
	}

	return result, nil
}

// UpdateCommentResolutionStates updates the GitHubThreadResolved field for all comments
// based on the current GitHub state
func (r *Reconciler) UpdateCommentResolutionStates(ctx context.Context, prNumber int, reviews []github.Review) ([]github.Review, error) {
	// Get all thread states from GitHub in batch
	threadStates, err := r.githubClient.GetAllThreadStates(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread states: %w", err)
	}

	// Update resolution state for each comment
	now := time.Now().Format("2006-01-02T15:04:05Z")
	for i := range reviews {
		for j := range reviews[i].Comments {
			comment := &reviews[i].Comments[j]
			if comment.ID == 0 {
				continue // Skip embedded comments
			}

			// Update resolution state from GitHub
			if isResolved, exists := threadStates[comment.ID]; exists {
				comment.GitHubThreadResolved = isResolved
				comment.LastCheckedAt = now
			}
		}
	}

	return reviews, nil
}
