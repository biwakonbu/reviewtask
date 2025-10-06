package cmd

import (
	"fmt"

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

	fmt.Printf("🚀 Starting work on task '%s'...\n", taskID)

	// Delegate to update command with "doing" status
	// The update command will handle validation and provide appropriate error messages
	err := runUpdate(cmd, []string{taskID, "doing"})
	if err != nil {
		// If the task is already doing or done, update command will return appropriate error
		return err
	}

	fmt.Printf("✅ Task '%s' is now in progress!\n", taskID)
	fmt.Printf("💡 Tip: Use 'reviewtask done %s' when you complete this task\n", taskID)

	return nil
}
