package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [PR_NUMBER]",
	Short: "[DEPRECATED] Use 'reviewtask [PR_NUMBER]' instead",
	Long: `⚠️  DEPRECATION NOTICE: This command will be removed in v3.0.0

The 'fetch' command has been integrated into the main reviewtask command.

Please use instead:
  reviewtask              # Analyze current branch's PR
  reviewtask 123          # Analyze PR #123

The integrated command provides the same functionality with:
- Automatic fetch + analyze workflow
- AI-powered task generation
- No need to run separate commands

Migration:
  reviewtask fetch     →  reviewtask
  reviewtask fetch 123 →  reviewtask 123`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("⚠️  Warning: 'reviewtask fetch' is deprecated")
		fmt.Println("⚠️  Use 'reviewtask [PR_NUMBER]' instead for the integrated workflow")
		fmt.Println()
		return runReviewTask(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}
