package cmd

import (
	"github.com/spf13/cobra"
)

// FetchOptions contains options for the fetch command
type FetchOptions struct {
	BatchSize    int
	Resume       bool
	FastMode     bool
	MaxTimeout   int
	ShowProgress bool
}

var fetchOptions FetchOptions

var fetchCmd = &cobra.Command{
	Use:   "fetch [PR_NUMBER]",
	Short: "Fetch GitHub Pull Request reviews and generate tasks",
	Long: `Fetch GitHub Pull Request reviews, save them locally,
and use AI to analyze review content for task generation.

Examples:
  reviewtask fetch                   # Check reviews for current branch's PR
  reviewtask fetch 123               # Check reviews for PR #123
  reviewtask fetch --batch-size=10   # Process 10 comments at a time
  reviewtask fetch --resume          # Resume from last checkpoint
  reviewtask fetch --fast-mode       # Skip non-essential processing`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReviewTask,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	
	// Add flags for incremental processing and performance
	fetchCmd.Flags().IntVar(&fetchOptions.BatchSize, "batch-size", 5, "Number of comments to process in each batch")
	fetchCmd.Flags().BoolVar(&fetchOptions.Resume, "resume", false, "Resume from last checkpoint")
	fetchCmd.Flags().BoolVar(&fetchOptions.FastMode, "fast-mode", false, "Skip non-essential processing for speed")
	fetchCmd.Flags().IntVar(&fetchOptions.MaxTimeout, "timeout", 300, "Maximum timeout in seconds for the operation")
	fetchCmd.Flags().BoolVar(&fetchOptions.ShowProgress, "progress", true, "Show progress during processing")
}
