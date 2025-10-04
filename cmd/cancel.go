package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

var (
	cancelReason     string
	cancelAllPending bool
)

var cancelCmd = &cobra.Command{
	Use:   "cancel <task-id>",
	Short: "Cancel a task and post the reason to PR review thread",
	Long: `Cancel a task and post the cancellation reason as a comment on the PR review thread.

This command:
- Updates the task status to 'cancel'
- Posts the cancellation reason to the PR review thread
- Sets CancelCommentPosted flag to true after successful posting

The cancellation reason is required and helps reviewers understand why the feedback was not addressed.

Examples:
  # Cancel a single task with a reason
  reviewtask cancel task-1 --reason "This was addressed in PR #123"

  # Cancel all pending tasks with the same reason
  reviewtask cancel --all-pending --reason "Deferring to follow-up PR #124"`,
	RunE: runCancel,
}

func init() {
	cancelCmd.Flags().StringVar(&cancelReason, "reason", "", "Reason for cancelling the task (required)")
	cancelCmd.Flags().BoolVar(&cancelAllPending, "all-pending", false, "Cancel all pending tasks")
	cancelCmd.MarkFlagRequired("reason")
}

func runCancel(cmd *cobra.Command, args []string) error {
	// Display AI provider info
	_, err := DisplayAIProviderIfNeeded()
	if err != nil {
		// Continue without config - cancel can work without it
	}

	// Validate input
	if !cancelAllPending && len(args) != 1 {
		return fmt.Errorf("task ID is required (or use --all-pending flag)")
	}

	if cancelAllPending && len(args) > 0 {
		return fmt.Errorf("cannot specify task ID when using --all-pending flag")
	}

	if strings.TrimSpace(cancelReason) == "" {
		return fmt.Errorf("cancellation reason cannot be empty")
	}

	storageManager := storage.NewManager()

	// Get tasks to cancel
	var tasksToCancel []storage.Task
	if cancelAllPending {
		// Get all tasks
		allTasks, err := storageManager.GetAllTasks()
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		// Filter pending tasks
		for _, task := range allTasks {
			if task.Status == "pending" {
				tasksToCancel = append(tasksToCancel, task)
			}
		}

		if len(tasksToCancel) == 0 {
			fmt.Println("No pending tasks found to cancel")
			return nil
		}

		fmt.Printf("Found %d pending task(s) to cancel\n", len(tasksToCancel))
	} else {
		// Get single task
		taskID := args[0]
		allTasks, err := storageManager.GetAllTasks()
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		var found bool
		for _, task := range allTasks {
			if task.ID == taskID {
				tasksToCancel = append(tasksToCancel, task)
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("task '%s' not found", taskID)
		}
	}

	// Create GitHub client
	githubClient, err := github.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Cancel each task
	successCount := 0
	failureCount := 0
	var firstErr error

	for _, task := range tasksToCancel {
		if err := cancelTask(cmd, storageManager, githubClient, &task, cancelReason); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "✗ Failed to cancel task '%s': %v\n", task.ID, err)
			failureCount++
			// For single-task cancellations, return error immediately
			if !cancelAllPending {
				return err
			}
			// For batch cancellations, capture first error
			if firstErr == nil {
				firstErr = err
			}
		} else {
			successCount++
		}
	}

	// Print summary
	if cancelAllPending {
		fmt.Printf("\n✓ Successfully cancelled %d task(s)\n", successCount)
		if failureCount > 0 {
			fmt.Printf("✗ Failed to cancel %d task(s)\n", failureCount)
			return fmt.Errorf("failed to cancel %d task(s)", failureCount)
		}
	}

	return nil
}

// cancelTask cancels a single task and posts the reason to GitHub
func cancelTask(cmd *cobra.Command, storageManager *storage.Manager, githubClient *github.Client, task *storage.Task, reason string) error {
	// Check if task has a source comment ID (embedded comments from Codex won't have thread IDs)
	if task.SourceCommentID == 0 {
		fmt.Printf("⚠ Task '%s' has no associated review comment, skipping GitHub comment posting\n", task.ID)
		// Still update the task status locally
		return updateTaskCancelStatus(storageManager, task.ID, reason, false)
	}

	// Post cancel reason as a reply to the review comment
	ctx := context.Background()
	commentBody := formatCancelComment(task, reason)

	if err := githubClient.PostReviewCommentReply(ctx, task.PRNumber, task.SourceCommentID, commentBody); err != nil {
		// If comment posting fails, still update task but mark comment as not posted
		updateErr := updateTaskCancelStatus(storageManager, task.ID, reason, false)
		if updateErr != nil {
			return fmt.Errorf("failed to post comment: %w (and failed to update task: %v)", err, updateErr)
		}
		return fmt.Errorf("failed to post comment to GitHub: %w", err)
	}

	// Update task status with successful comment posting
	if err := updateTaskCancelStatus(storageManager, task.ID, reason, true); err != nil {
		return fmt.Errorf("comment posted successfully but failed to update task: %w", err)
	}

	fmt.Printf("✓ Cancelled task '%s' and posted reason to PR #%d\n", task.ID, task.PRNumber)
	return nil
}

// updateTaskCancelStatus updates the task with cancel status and reason
func updateTaskCancelStatus(storageManager *storage.Manager, taskID, reason string, commentPosted bool) error {
	return storageManager.UpdateTaskCancelStatus(taskID, reason, commentPosted)
}

// formatCancelComment formats the cancellation comment for posting to GitHub
func formatCancelComment(task *storage.Task, reason string) string {
	var comment strings.Builder

	comment.WriteString("**Task Cancelled**\n\n")
	comment.WriteString("This feedback item has been cancelled for the following reason:\n\n")
	comment.WriteString(fmt.Sprintf("> %s\n\n", reason))

	// Add task information
	if task.Description != "" {
		comment.WriteString(fmt.Sprintf("**Original task**: %s\n", task.Description))
	}

	if task.URL != "" {
		comment.WriteString(fmt.Sprintf("**Comment**: %s\n", task.URL))
	}

	return comment.String()
}
