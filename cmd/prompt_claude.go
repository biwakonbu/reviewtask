package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var promptClaudeCmd = &cobra.Command{
	Use:   "claude [TARGET]",
	Short: "Output command templates for Claude Code to .claude/commands directory",
	Long: `Output command templates for Claude Code to .claude/commands directory for better organization and discoverability.

Available targets:
  pr-review    Output PR review workflow command template to .claude/commands/pr-review/

Examples:
  reviewtask prompt claude pr-review    # Output review-task-workflow command template for Claude Code`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptClaude,
}

func init() {
	promptCmd.AddCommand(promptClaudeCmd)
}

func runPromptClaude(cmd *cobra.Command, args []string) error {
	target := args[0]

	switch target {
	case "pr-review":
		return outputClaudePRReviewCommands()
	default:
		return fmt.Errorf("unknown target: %s\n\nAvailable targets:\n  pr-review    Output PR review workflow command template", target)
	}
}

// outputClaudePRReviewCommands is moved from claude.go and reused here
// This ensures consistent functionality between the old and new command structures
