package cmd

import (
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

	// Delegate to update command with "doing" status
	return runUpdate(cmd, []string{taskID, "doing"})
}

func init() {
	rootCmd.AddCommand(startCmd)
}
