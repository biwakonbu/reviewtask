package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
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
  reviewtask update task-3 cancel`,
	Args: cobra.ExactArgs(2),
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Display AI provider info
	_, err := DisplayAIProviderIfNeeded()
	if err != nil {
		// Continue without config - update can work without it
	}

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

	// Check if auto-resolve is enabled
	cfg, err := config.Load()
	autoResolve := err == nil && cfg.AISettings.AutoResolveThreads

	// Create callback for thread resolution if needed
	var callback func(*storage.Task) error
	if autoResolve && newStatus == "done" {
		callback = func(task *storage.Task) error {
			// Only resolve threads for tasks with source comment IDs
			// (embedded comments from Codex won't have thread IDs)
			if task.SourceCommentID == 0 {
				return nil
			}

			// Create GitHub client
			githubClient, err := github.NewClient()
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			// Resolve the thread
			ctx := context.Background()
			if err := githubClient.ResolveCommentThread(ctx, task.PRNumber, task.SourceCommentID); err != nil {
				return fmt.Errorf("failed to resolve thread: %w", err)
			}

			fmt.Printf("✓ Resolved review thread for comment #%d\n", task.SourceCommentID)
			return nil
		}
	}

	// Update task status with callback
	err = storageManager.UpdateTaskStatusWithCallback(taskID, newStatus, callback)
	if err != nil {
		if err == storage.ErrTaskNotFound {
			return fmt.Errorf("task '%s' not found", taskID)
		}
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("✓ Updated task '%s' status to '%s'\n", taskID, newStatus)

	return nil
}
