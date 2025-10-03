package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cursorAllFlag    bool
	cursorStdoutFlag bool
)

var cursorCmd = &cobra.Command{
	Use:   "cursor [TARGET]",
	Short: "Output command templates for Cursor IDE to .cursor/commands directory",
	Long: `Output command templates for Cursor IDE to .cursor/commands directory for better organization and discoverability.

Available targets:
  pr-review      Output PR review workflow command template
  issue-to-pr    Output Issue-to-PR workflow command template
  label-issues   Output Label Issues workflow command template

Examples:
  reviewtask cursor pr-review          # Output review-task-workflow command template for Cursor IDE
  reviewtask cursor issue-to-pr        # Output issue-to-pr workflow command template
  reviewtask cursor label-issues       # Output label-issues workflow command template
  reviewtask cursor --all              # Output all available command templates
  reviewtask cursor pr-review --stdout # Output to standard output instead of files
  reviewtask cursor --all --stdout     # Output all templates to standard output`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCursor,
}

func init() {
	// Command registration will be done in root.go
	cursorCmd.Flags().BoolVar(&cursorAllFlag, "all", false, "Generate all available command templates")
	cursorCmd.Flags().BoolVar(&cursorStdoutFlag, "stdout", false, "Output to standard output instead of files")
}

func runCursor(cmd *cobra.Command, args []string) error {
	// If --all flag is set, generate all templates
	if cursorAllFlag {
		return outputAllCursorCommands(cmd.OutOrStdout())
	}

	// Require target argument if not using --all
	if len(args) == 0 {
		return fmt.Errorf("target argument required when not using --all flag\n\nAvailable targets:\n  pr-review      Output PR review workflow command template\n  issue-to-pr    Output Issue-to-PR workflow command template\n  label-issues   Output Label Issues workflow command template")
	}

	target := args[0]

	// If --stdout flag is set, output to stdout
	if cursorStdoutFlag {
		return outputCursorCommandToStdout(cmd.OutOrStdout(), target)
	}

	// Default: output to files
	switch target {
	case "pr-review":
		return outputCursorPRReviewCommands(cmd.OutOrStdout())
	case "issue-to-pr":
		return outputCursorIssueToPRCommands(cmd.OutOrStdout())
	case "label-issues":
		return outputCursorLabelIssuesCommands(cmd.OutOrStdout())
	default:
		return fmt.Errorf("unknown target: %s\n\nAvailable targets:\n  pr-review      Output PR review workflow command template\n  issue-to-pr    Output Issue-to-PR workflow command template\n  label-issues   Output Label Issues workflow command template", target)
	}
}

func outputCursorPRReviewCommands(out io.Writer) error {
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

	fmt.Fprintf(out, "✓ Created Cursor IDE command template at %s\n", workflowPath)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Cursor IDE commands have been organized in .cursor/commands/pr-review/")
	fmt.Fprintln(out, "You can now use the /review-task-workflow command in Cursor IDE.")

	return nil
}

func outputCursorIssueToPRCommands(out io.Writer) error {
	// Create the output directory
	outputDir := ".cursor/commands/issue-to-pr"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Use the template content
	workflowTemplate := getIssueToPRTemplate()

	// Write the workflow template
	workflowPath := filepath.Join(outputDir, "issue-to-pr.md")
	if err := os.WriteFile(workflowPath, []byte(workflowTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write workflow template: %w", err)
	}

	fmt.Fprintf(out, "✓ Created Cursor IDE command template at %s\n", workflowPath)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Cursor IDE commands have been organized in .cursor/commands/issue-to-pr/")
	fmt.Fprintln(out, "You can now use the /issue-to-pr command in Cursor IDE.")

	return nil
}

func outputCursorLabelIssuesCommands(out io.Writer) error {
	// Create the output directory
	outputDir := ".cursor/commands/label-issues"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Use the template content
	workflowTemplate := getLabelIssuesTemplate()

	// Write the workflow template
	workflowPath := filepath.Join(outputDir, "label-issues.md")
	if err := os.WriteFile(workflowPath, []byte(workflowTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write workflow template: %w", err)
	}

	fmt.Fprintf(out, "✓ Created Cursor IDE command template at %s\n", workflowPath)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Cursor IDE commands have been organized in .cursor/commands/label-issues/")
	fmt.Fprintln(out, "You can now use the /label-issues command in Cursor IDE.")

	return nil
}

func outputAllCursorCommands(out io.Writer) error {
	// If stdout flag is set, output all to stdout
	if cursorStdoutFlag {
		// Output all templates to stdout with clear separators
		fmt.Fprintln(out, "# === PR REVIEW WORKFLOW ===")
		fmt.Fprintln(out, getPRReviewPromptTemplate())
		fmt.Fprintln(out, "\n# === ISSUE TO PR WORKFLOW ===")
		fmt.Fprintln(out, getIssueToPRTemplate())
		fmt.Fprintln(out, "\n# === LABEL ISSUES WORKFLOW ===")
		fmt.Fprintln(out, getLabelIssuesTemplate())
		return nil
	}

	// Generate all command templates to files
	if err := outputCursorPRReviewCommands(out); err != nil {
		return err
	}
	if err := outputCursorIssueToPRCommands(out); err != nil {
		return err
	}
	if err := outputCursorLabelIssuesCommands(out); err != nil {
		return err
	}

	fmt.Fprintln(out, "\n✅ All Cursor IDE command templates have been generated successfully!")
	return nil
}

func outputCursorCommandToStdout(out io.Writer, target string) error {
	var template string

	switch target {
	case "pr-review":
		template = getPRReviewPromptTemplate()
	case "issue-to-pr":
		template = getIssueToPRTemplate()
	case "label-issues":
		template = getLabelIssuesTemplate()
	default:
		return fmt.Errorf("unknown target: %s", target)
	}

	fmt.Fprint(out, template)
	return nil
}
