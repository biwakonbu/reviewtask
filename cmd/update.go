package cmd

import (
	"context"
	"fmt"
	"strings"

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

Note: To cancel a task, use 'reviewtask cancel <task-id> --reason "..."'
This ensures cancellation reasons are posted to the PR.

Examples:
  reviewtask update task-1 doing
  reviewtask update task-2 done
  reviewtask update task-3 pending`,
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

	// Reject cancel status - must use dedicated cancel command
	if newStatus == "cancel" {
		return fmt.Errorf("cannot set status to 'cancel' using update command.\n\n"+
			"Use 'reviewtask cancel %s --reason \"...\"' instead.\n\n"+
			"This ensures cancellation reasons are posted to PR for reviewer visibility\n"+
			"and enables proper thread auto-resolution.", taskID)
	}

	// Validate status
	validStatuses := map[string]bool{
		"todo":    true,
		"doing":   true,
		"done":    true,
		"pending": true,
	}

	if !validStatuses[newStatus] {
		return fmt.Errorf("invalid status '%s'. Valid statuses: todo, doing, done, pending", newStatus)
	}

	storageManager := storage.NewManager()

	// Check if auto-resolve is enabled
	cfg, err := config.Load()

	// Determine auto-resolve behavior based on configuration
	// Priority: AutoResolveMode > AutoResolveThreads (legacy)
	autoResolveMode := "disabled"
	if err == nil {
		mode := strings.ToLower(strings.TrimSpace(cfg.AISettings.AutoResolveMode))
		switch mode {
		case "immediate", "complete":
			autoResolveMode = mode
		case "", "disabled":
			// keep disabled
		default:
			fmt.Fprintf(cmd.ErrOrStderr(), "unknown auto_resolve_mode %q; defaulting to disabled\n", cfg.AISettings.AutoResolveMode)
		}

		if autoResolveMode == "disabled" && cfg.AISettings.AutoResolveThreads {
			// Legacy support: AutoResolveThreads=true maps to "immediate" mode
			autoResolveMode = "immediate"
		}
	}

	// Create callback for thread resolution if needed
	var callback func(*storage.Task) error
	if autoResolveMode != "disabled" && newStatus == "done" {
		callback = func(task *storage.Task) error {
			// Only resolve threads for tasks with source comment IDs
			// (embedded comments from Codex won't have thread IDs)
			if task.SourceCommentID == 0 {
				return nil
			}

			// For "complete" mode, check if all tasks for this comment are done
			if autoResolveMode == "complete" {
				allDone, err := storageManager.AreAllCommentTasksCompleted(task.PRNumber, task.SourceCommentID)
				if err != nil {
					return fmt.Errorf("failed to check comment completion status: %w", err)
				}
				if !allDone {
					// Not all tasks are done yet, skip resolution
					return nil
				}
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
