package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"reviewtask/internal/verification"
)

var completeCmd = &cobra.Command{
	Use:   "complete <task-id>",
	Short: "[DEPRECATED] Use 'reviewtask done' instead",
	Long: `⚠️  DEPRECATION NOTICE: This command will be removed in v3.0.0

The 'complete' command has been superseded by the 'done' command with full workflow automation.

Please use instead:
  reviewtask done <task-id>      # Complete task with full automation

The 'done' command provides:
- Verification (build/test/lint checks)
- Status update to "done"
- Auto-commit with structured message
- Review thread resolution
- Next task suggestion

To skip specific automation phases:
  reviewtask done <task-id> --skip-verification
  reviewtask done <task-id> --skip-commit
  reviewtask done <task-id> --skip-resolve
  reviewtask done <task-id> --skip-suggestion

Migration:
  reviewtask complete task-1                  →  reviewtask done task-1
  reviewtask complete task-1 --verify         →  reviewtask done task-1
  reviewtask complete task-1 --skip-verification  →  reviewtask done task-1 --skip-verification`,
	Args: cobra.ExactArgs(1),
	RunE: runComplete,
}

var (
	completeWithVerify bool
	completeSkipVerify bool
	completeVerbose    bool
)

func init() {
	completeCmd.Flags().BoolVar(&completeWithVerify, "verify", true, "Run verification checks before completion (default: true)")
	completeCmd.Flags().BoolVar(&completeSkipVerify, "skip-verification", false, "Skip verification checks and complete task directly")
	completeCmd.Flags().BoolVarP(&completeVerbose, "verbose", "v", false, "Show detailed verification output")
}

func runComplete(cmd *cobra.Command, args []string) error {
	// Display deprecation warning
	fmt.Println("⚠️  Warning: 'reviewtask complete' is deprecated")
	fmt.Println("⚠️  Use 'reviewtask done' instead for full workflow automation")
	fmt.Println()

	// Display AI provider info
	_, err := DisplayAIProviderIfNeeded()
	if err != nil {
		// Continue without config - complete can work without it
	}

	taskID := args[0]

	// Skip verification takes precedence
	if completeSkipVerify {
		return completeTaskDirectly(taskID)
	}

	// Default behavior is to verify unless explicitly disabled
	if completeWithVerify && !completeSkipVerify {
		return completeTaskWithVerification(taskID)
	}

	return completeTaskDirectly(taskID)
}

func completeTaskWithVerification(taskID string) error {
	verifier, err := verification.NewVerifier()
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	fmt.Printf("🔍 Running verification checks for task '%s'...\n", taskID)

	err = verifier.CompleteTaskWithVerification(taskID)
	if err != nil {
		fmt.Printf("❌ Task completion failed: %v\n", err)
		fmt.Println("\n💡 To see detailed verification results, run:")
		fmt.Printf("   reviewtask verify %s --verbose\n", taskID)
		fmt.Println("\n💡 To complete without verification, run:")
		fmt.Printf("   reviewtask complete %s --skip-verification\n", taskID)
		return err
	}

	fmt.Printf("✅ Task '%s' completed successfully!\n", taskID)
	fmt.Println("✅ All verification checks passed")
	return nil
}

func completeTaskDirectly(taskID string) error {
	// Use the existing update command logic for direct completion
	return runUpdate(nil, []string{taskID, "done"})
}
