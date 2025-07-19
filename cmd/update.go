package cmd

import (
	"fmt"

	"gh-review-task/internal/storage"
	"github.com/spf13/cobra"
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
  gh-review-task update task-1 doing
  gh-review-task update task-2 done
  gh-review-task update task-3 cancel`,
	Args: cobra.ExactArgs(2),
	RunE: runUpdate,
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

	// Update task status
	err := storageManager.UpdateTaskStatus(taskID, newStatus)
	if err != nil {
		if err == storage.ErrTaskNotFound {
			return fmt.Errorf("task '%s' not found", taskID)
		}
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("✓ Updated task '%s' status to '%s'\n", taskID, newStatus)

	return nil
}
