package threads

import (
	"context"
	"fmt"
	"strings"

	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

// ResolveMode defines how threads should be resolved
type ResolveMode string

const (
	ResolveModeImmediate ResolveMode = "immediate" // Resolve thread after each task completion
	ResolveModeComplete  ResolveMode = "complete"  // Resolve thread only when all tasks from comment are complete
	ResolveModeDisabled  ResolveMode = "disabled"  // Never auto-resolve threads
)

// ResolutionResult contains the result of thread resolution attempt
type ResolutionResult struct {
	ThreadResolved bool
	CommentID      int64
	TotalTasks     int
	CompletedTasks int
	RemainingTasks int
	ResolveMode    ResolveMode
	Message        string
	ShouldNotify   bool // Whether to show this result to user
}

// ThreadResolver handles review thread resolution logic
type ThreadResolver struct {
	config       *config.Config
	storage      StorageInterface
	githubClient GitHubInterface
}

// StorageInterface defines storage operations needed by resolver
type StorageInterface interface {
	GetAllTasks() ([]storage.Task, error)
	GetTasksByCommentID(commentID int64) ([]storage.Task, error)
}

// GitHubInterface defines GitHub operations needed by resolver
type GitHubInterface interface {
	GetReviewThreadID(ctx context.Context, owner, repo string, prNumber int, commentID int64) (string, error)
	ResolveReviewThread(ctx context.Context, threadID string) error
}

// NewThreadResolver creates a new thread resolver instance
func NewThreadResolver(cfg *config.Config, storageManager StorageInterface, githubClient GitHubInterface) *ThreadResolver {
	return &ThreadResolver{
		config:       cfg,
		storage:      storageManager,
		githubClient: githubClient,
	}
}

// ShouldResolveThread determines if a thread should be resolved based on task completion
func (r *ThreadResolver) ShouldResolveThread(ctx context.Context, task *storage.Task) (*ResolutionResult, error) {
	// Check configuration
	// Normalize to lowercase to prevent case-sensitivity issues
	mode := ResolveMode(strings.ToLower(r.config.DoneWorkflow.EnableAutoResolve))
	if mode == ResolveModeDisabled {
		return &ResolutionResult{
			ThreadResolved: false,
			ResolveMode:    mode,
			Message:        "auto-resolve disabled",
			ShouldNotify:   false,
		}, nil
	}

	// Get all tasks for this comment
	tasks, err := r.storage.GetTasksByCommentID(task.SourceCommentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for comment: %w", err)
	}

	// Count completed and total tasks
	totalTasks := len(tasks)
	completedTasks := 0
	for _, t := range tasks {
		if t.Status == "done" {
			completedTasks++
		}
	}

	result := &ResolutionResult{
		CommentID:      task.SourceCommentID,
		TotalTasks:     totalTasks,
		CompletedTasks: completedTasks,
		RemainingTasks: totalTasks - completedTasks,
		ResolveMode:    mode,
		ShouldNotify:   true,
	}

	// Immediate mode: always try to resolve after each task
	if mode == ResolveModeImmediate {
		result.ThreadResolved = true
		result.Message = "resolving thread (immediate mode)"
		return result, nil
	}

	// Complete mode: only resolve when all tasks are done
	if mode == ResolveModeComplete {
		if completedTasks == totalTasks {
			result.ThreadResolved = true
			result.Message = "all tasks complete, resolving thread"
			return result, nil
		}
		result.ThreadResolved = false
		result.Message = fmt.Sprintf("%d of %d tasks complete", completedTasks, totalTasks)
		return result, nil
	}

	return result, nil
}

// ResolveThreadForTask attempts to resolve the review thread for a task
func (r *ThreadResolver) ResolveThreadForTask(ctx context.Context, task *storage.Task, owner, repo string) error {
	// First check if we should resolve
	result, err := r.ShouldResolveThread(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to check if thread should be resolved: %w", err)
	}

	if !result.ThreadResolved {
		// Thread should not be resolved yet
		return nil
	}

	// Get thread ID from GitHub
	threadID, err := r.githubClient.GetReviewThreadID(ctx, owner, repo, task.PRNumber, task.SourceCommentID)
	if err != nil {
		return fmt.Errorf("failed to get thread ID: %w", err)
	}

	// Resolve the thread
	if err := r.githubClient.ResolveReviewThread(ctx, threadID); err != nil {
		return fmt.Errorf("failed to resolve thread: %w", err)
	}

	return nil
}

// GetResolutionStatus returns the current resolution status for a task's comment thread
func (r *ThreadResolver) GetResolutionStatus(ctx context.Context, task *storage.Task) (*ResolutionResult, error) {
	return r.ShouldResolveThread(ctx, task)
}
