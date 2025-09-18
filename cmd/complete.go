package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"reviewtask/internal/verification"
)

var completeCmd = &cobra.Command{
	Use:   "complete <task-id>",
	Short: "Complete task with verification",
	Long: `Complete a task after running verification checks to ensure the work is properly implemented.

This command performs the following steps:
1. Runs all configured verification checks (build, test, lint, etc.)
2. If all verifications pass, marks the task as 'done'
3. If any verification fails, provides detailed error information

Examples:
  reviewtask complete task-1
  reviewtask complete task-2 --verify
  reviewtask complete task-3 --skip-verification`,
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
