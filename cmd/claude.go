package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var claudeCmd = &cobra.Command{
	Use:   "claude [TARGET]",
	Short: "Output command templates for Claude Code to .claude/commands directory",
	Long: `Output command templates for Claude Code to .claude/commands directory for better organization and discoverability.

Available targets:
  pr-review    Output PR review workflow command template to .claude/commands/pr-review/

Examples:
  reviewtask claude pr-review    # Output review-task-workflow command template for Claude Code

DEPRECATION WARNING:
  This command is deprecated. Please use 'reviewtask prompt claude' instead.
  The 'reviewtask claude' command will be removed in a future major version.
  
  Migration examples:
    Old: reviewtask claude pr-review
    New: reviewtask prompt claude pr-review`,
	Args:       cobra.ExactArgs(1),
	RunE:       runClaude,
	Deprecated: "use 'reviewtask prompt claude' instead. This command will be removed in a future major version.",
}

func init() {
	// Command registration moved to root.go
}

func runClaude(cmd *cobra.Command, args []string) error {
	// Show deprecation warning
	fmt.Fprintf(os.Stderr, "⚠️  DEPRECATION WARNING: The 'reviewtask claude' command is deprecated.\n")
	fmt.Fprintf(os.Stderr, "⚠️  Please use 'reviewtask prompt claude %s' instead.\n", args[0])
	fmt.Fprintf(os.Stderr, "⚠️  This command will be removed in a future major version.\n\n")

	target := args[0]

	switch target {
	case "pr-review":
		return outputClaudePRReviewCommands()
	default:
		return fmt.Errorf("unknown target: %s\n\nAvailable targets:\n  pr-review    Output PR review workflow command template", target)
	}
}

func outputClaudePRReviewCommands() error {
	// Create the output directory
	outputDir := ".claude/commands/pr-review"
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

	fmt.Printf("✓ Created Claude Code command template at %s\n", workflowPath)
	fmt.Println()
	fmt.Println("Claude Code commands have been organized in .claude/commands/pr-review/")
	fmt.Println("You can now use the /review-task-workflow command in Claude Code.")
	fmt.Println()
	fmt.Println("Future expansion possibilities:")
	fmt.Println("  reviewtask claude pr-review    # Current functionality")
	fmt.Println("  reviewtask vscode pr-review    # Future: VSCode extensions")
	fmt.Println("  reviewtask cursor pr-review    # Future: Cursor rules")

	return nil
}
