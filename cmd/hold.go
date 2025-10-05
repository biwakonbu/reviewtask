package cmd

import (
	"github.com/spf13/cobra"
)

var holdCmd = &cobra.Command{
	Use:   "hold <task-id>",
	Short: "Put a task on hold",
	Long: `Mark a task as "pending" to indicate it needs evaluation or is blocked.

This is equivalent to 'reviewtask update <task-id> pending', but more intuitive.

Use this when:
- You need to evaluate whether to address the feedback
- The task is blocked by external dependencies
- You need to discuss the approach with reviewers

Examples:
  reviewtask hold task-1
  reviewtask hold abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runHold,
}

func runHold(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Delegate to update command with "pending" status
	return runUpdate(cmd, []string{taskID, "pending"})
}

func init() {
	rootCmd.AddCommand(holdCmd)
}
