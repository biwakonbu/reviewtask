package cmd

import (
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Output command templates for AI providers",
	Long: `Output command templates for various AI providers to their respective directories.

This command provides a unified interface for generating AI-specific command templates
that can be used with different AI providers and tools.

Available subcommands:
  claude       Output command templates for Claude Code

Future AI provider support planned:
  openai       Output command templates for OpenAI (future)
  gemini       Output command templates for Gemini (future)

Examples:
  reviewtask prompt claude pr-review    # Output review workflow for Claude Code
  reviewtask prompt --help             # Show all available AI providers`,
}

func init() {
	// Subcommands are registered individually in their respective files
}
