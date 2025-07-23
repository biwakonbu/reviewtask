package cmd

import (
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [PR_NUMBER]",
	Short: "Fetch GitHub Pull Request reviews and generate tasks",
	Long: `Fetch GitHub Pull Request reviews, save them locally,
and use AI to analyze review content for task generation.

Examples:
  reviewtask fetch        # Check reviews for current branch's PR
  reviewtask fetch 123    # Check reviews for PR #123`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReviewTask,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}
