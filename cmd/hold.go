package cmd

import (
	"fmt"

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
  reviewtask hold abc123
  reviewtask hold task-1 --reason "Waiting for external API approval"`,
	Args: cobra.ExactArgs(1),
	RunE: runHold,
}

func init() {
	holdCmd.Flags().String("reason", "", "Reason for putting the task on hold")
}

func runHold(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Get the reason flag
	reason, _ := cmd.Flags().GetString("reason")

	fmt.Printf("⏸️  Putting task '%s' on hold...\n", taskID)

	// If reason provided, show it in the output
	if reason != "" {
		fmt.Printf("📝 Reason: %s\n", reason)
	}

	// Delegate to update command with "pending" status
	err := runUpdate(cmd, []string{taskID, "pending"})
	if err != nil {
		return err
	}

	fmt.Printf("✅ Task '%s' is now on hold (pending)\n", taskID)

	// Provide guidance based on whether reason was provided
	if reason != "" {
		fmt.Printf("💡 Tip: Use 'reviewtask start %s' when you're ready to resume work\n", taskID)
	} else {
		fmt.Printf("💡 Tip: Use 'reviewtask start %s' when you're ready to resume work\n", taskID)
		fmt.Printf("   Consider using --reason flag next time to document why it was put on hold\n")
	}

	return nil
}
