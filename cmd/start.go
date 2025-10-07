package cmd

import (
	"fmt"

	"reviewtask/internal/guidance"
	"reviewtask/internal/storage"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <task-id>",
	Short: "Start working on a task",
	Long: `Mark a task as "doing" to indicate you're actively working on it.

This is equivalent to 'reviewtask update <task-id> doing', but more intuitive.

Examples:
  reviewtask start task-1
  reviewtask start abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Validate that task is not already doing or done
	storageManager := storage.NewManager()

	// Get all tasks to find the specific task
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// Find the specific task
	var targetTask *storage.Task
	for _, task := range allTasks {
		if task.ID == taskID {
			targetTask = &task
			break
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task '%s' not found", taskID)
	}

	// Validate current status
	if targetTask.Status == "doing" {
		return fmt.Errorf("task '%s' is already in progress (doing). Use 'reviewtask done %s' to complete it first, or 'reviewtask hold %s' if you need to pause it", taskID, taskID, taskID)
	}
	if targetTask.Status == "done" {
		return fmt.Errorf("task '%s' is already completed (done). Create a new task if you need to work on this again, or use 'reviewtask update %s doing' if you need to reopen it", taskID, taskID)
	}

	fmt.Printf("🚀 Starting work on task '%s'...\n", taskID)

	// Delegate to update command with "doing" status
	err = runUpdate(cmd, []string{taskID, "doing"})
	if err != nil {
		return err
	}

	fmt.Printf("✅ Task '%s' is now in progress!\n", taskID)
	fmt.Printf("💡 Tip: Use 'reviewtask done %s' when you complete this task\n", taskID)

	// Context-aware guidance
	detector := guidance.NewDetector(storageManager)
	ctx, err := detector.DetectContext()
	if err == nil {
		guide := ctx.Generate()
		fmt.Print(guide.Format())
	}

	return nil
}
