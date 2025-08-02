package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/notification"
	"reviewtask/internal/storage"
)

var (
	updateReason string
)

var updateCmd = &cobra.Command{
	Use:   "update <task-id> <new-status>",
	Short: "Update task status",
	Long: `Update the status of a specific task.

Valid statuses:
  todo     - Ready to start
  doing    - Currently in progress  
  done     - Completed
  pending  - Needs evaluation (whether to address or not)
  cancel   - Decided not to address

Examples:
  reviewtask update task-1 doing
  reviewtask update task-2 done
  reviewtask update task-3 cancel --reason "Already implemented in PR #123"
  reviewtask update task-4 pending --reason "Waiting for API design decision"`,
	Args: cobra.ExactArgs(2),
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().StringVar(&updateReason, "reason", "", "Reason for status change (recommended for cancel/pending)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	newStatus := args[1]

	// Validate status
	validStatuses := map[string]bool{
		"todo":    true,
		"doing":   true,
		"done":    true,
		"pending": true,
		"cancel":  true,
	}

	if !validStatuses[newStatus] {
		return fmt.Errorf("invalid status '%s'. Valid statuses: todo, doing, done, pending, cancel", newStatus)
	}

	storageManager := storage.NewManager()

	// Get the task to check if we need to notify
	task, err := storageManager.GetTask(taskID)
	if err != nil {
		if err == storage.ErrTaskNotFound {
			return fmt.Errorf("task '%s' not found", taskID)
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update task status
	err = storageManager.UpdateTaskStatus(taskID, newStatus)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("✓ Updated task '%s' status to '%s'\n", taskID, newStatus)

	// Handle notifications if enabled
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.CommentSettings.Enabled {
		// Create GitHub client
		githubClient, err := github.NewClient()
		if err != nil {
			// Log error but don't fail the update
			fmt.Printf("⚠️  Warning: Could not create GitHub client for notifications: %v\n", err)
			return nil
		}

		// Create notifier
		notifier := notification.New(githubClient, cfg)
		ctx := context.Background()

		// Send appropriate notification based on status
		var notifyErr error
		switch newStatus {
		case "done":
			notifyErr = notifier.NotifyTaskCompletion(ctx, task)
		case "cancel":
			notifyErr = notifier.NotifyTaskCancellation(ctx, task, updateReason)
		case "pending":
			notifyErr = notifier.NotifyTaskPending(ctx, task, updateReason)
		}

		if notifyErr != nil {
			fmt.Printf("⚠️  Warning: Could not post notification: %v\n", notifyErr)
		} else if newStatus == "done" || newStatus == "cancel" || newStatus == "pending" {
			fmt.Printf("📝 Notification posted to GitHub\n")
		}
	}

	return nil
}
