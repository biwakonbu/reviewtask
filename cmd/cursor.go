package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var cursorCmd = &cobra.Command{
	Use:   "cursor [TARGET]",
	Short: "Output command templates for Cursor IDE to .cursor/commands directory",
	Long: `Output command templates for Cursor IDE to .cursor/commands directory for better organization and discoverability.

Available targets:
  pr-review    Output PR review workflow command template to .cursor/commands/pr-review/

Examples:
  reviewtask cursor pr-review    # Output review-task-workflow command template for Cursor IDE`,
	Args: cobra.ExactArgs(1),
	RunE: runCursor,
}

func init() {
	// Command registration will be done in root.go
}

func runCursor(cmd *cobra.Command, args []string) error {
	target := args[0]

	switch target {
	case "pr-review":
		return outputCursorPRReviewCommands()
	default:
		return fmt.Errorf("unknown target: %s\n\nAvailable targets:\n  pr-review    Output PR review workflow command template", target)
	}
}

func outputCursorPRReviewCommands() error {
	// Create the output directory
	outputDir := ".cursor/commands/pr-review"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Use the shared template from prompt_stdout.go
	workflowTemplate := getPRReviewPromptTemplate()

	// Write the workflow template
	workflowPath := filepath.Join(outputDir, "review-task-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write workflow template: %w", err)
	}

	fmt.Printf("✓ Created Cursor IDE command template at %s\n", workflowPath)
	fmt.Println()
	fmt.Println("Cursor IDE commands have been organized in .cursor/commands/pr-review/")
	fmt.Println("You can now use the /review-task-workflow command in Cursor IDE.")

	return nil
}
