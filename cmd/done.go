package cmd

import (
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <task-id>",
	Short: "Mark a task as completed",
	Long: `Mark a task as "done" to indicate it's been completed.

This is equivalent to 'reviewtask update <task-id> done', but more intuitive.

Note: This command currently provides basic completion behavior only.
Future enhancements will add automation features like:
- Verification of changes
- Automatic commit
- Thread resolution
- Next task suggestion

Examples:
  reviewtask done task-1
  reviewtask done abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runDone,
}

func runDone(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Delegate to update command with "done" status
	// Note: runUpdate already handles thread auto-resolution if configured
	return runUpdate(cmd, []string{taskID, "done"})
}
