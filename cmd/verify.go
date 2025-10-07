package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"reviewtask/internal/ui"
	"reviewtask/internal/verification"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <task-id>",
	Short: "Verify task completion requirements",
	Long: `Verify that a task meets completion requirements before marking it as done.

This command runs configured verification checks such as:
  - Build verification (compile/build checks)
  - Test execution (run relevant tests)
  - Lint/format checks (code quality standards)
  - Custom verification (project-specific commands)

Examples:
  reviewtask verify task-1
  reviewtask verify task-2 --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

var verifyVerbose bool

func init() {
	verifyCmd.Flags().BoolVarP(&verifyVerbose, "verbose", "v", false, "Show detailed verification output")
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Display AI provider info
	_, err := DisplayAIProviderIfNeeded()
	if err != nil {
		// Continue without config - verify can work without it
	}

	taskID := args[0]

	verifier, err := verification.NewVerifier()
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	fmt.Println(ui.SectionDivider(fmt.Sprintf("Verifying Task %s", taskID)))
	fmt.Println()

	results, err := verifier.VerifyTask(taskID)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println(ui.Warning("No verification checks configured"))
		return nil
	}

	fmt.Println(ui.SectionDivider("Verification Results"))
	allPassed := true
	for _, result := range results {
		status := ui.SymbolSuccess
		if !result.Success {
			status = ui.SymbolError
			allPassed = false
		}

		fmt.Printf("%s %s: %s (%.2fs)\n", status, strings.ToUpper(string(result.Type)), result.Message, result.Duration.Seconds())

		if verifyVerbose && result.Output != "" {
			fmt.Printf("  Command: %s\n", result.Command)
			fmt.Printf("  Output:\n%s\n", indentOutput(result.Output))
		}
	}

	fmt.Println()
	fmt.Println(ui.SectionDivider("Next Steps"))
	if allPassed {
		fmt.Println(ui.Success(fmt.Sprintf("All verification checks passed for task '%s'", taskID)))
		fmt.Println(ui.Next("Complete this task"))
		fmt.Printf("  reviewtask done %s\n", taskID)
	} else {
		fmt.Println(ui.Error(fmt.Sprintf("Some verification checks failed for task '%s'", taskID)))
		fmt.Println(ui.Warning("Please fix the issues above before completing the task"))
		return fmt.Errorf("verification checks failed")
	}
	fmt.Println()

	return nil
}

// indentOutput adds indentation to each line of output for better readability
func indentOutput(output string) string {
	if output == "" {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var indentedLines []string
	for _, line := range lines {
		indentedLines = append(indentedLines, "     "+line)
	}
	return strings.Join(indentedLines, "\n")
}
