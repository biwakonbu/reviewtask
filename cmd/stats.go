package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"gh-review-task/internal/ai"
	"gh-review-task/internal/github"
	"gh-review-task/internal/storage"
)

var statsCmd = &cobra.Command{
	Use:   "stats [PR_NUMBER]",
	Short: "Show task statistics by comment",
	Long: `Display detailed statistics about tasks generated from PR review comments.
Shows both overall statistics and per-comment breakdown of task status.

Examples:
  gh-review-task stats        # Show stats for current branch's PR
  gh-review-task stats 123    # Show stats for PR #123`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Initialize storage manager
	storageManager := storage.NewManager()

	// Determine PR number
	var prNumber int
	var err error
	if len(args) > 0 {
		prNumber, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}
	} else {
		// Get PR number from current branch
		ghClient, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("failed to initialize GitHub client: %w", err)
		}
		prNumber, err = ghClient.GetCurrentBranchPR(ctx)
		if err != nil {
			return fmt.Errorf("failed to get PR for current branch: %w", err)
		}
	}

	// Generate statistics
	statsManager := ai.NewStatisticsManager(storageManager)
	stats, err := statsManager.GenerateTaskStatistics(prNumber)
	if err != nil {
		return fmt.Errorf("failed to generate statistics: %w", err)
	}

	// Display formatted statistics
	fmt.Printf("📊 Task Statistics for PR #%d\n\n", prNumber)
	fmt.Printf("Total Comments: %d\n", stats.TotalComments)
	fmt.Printf("Total Tasks: %d\n\n", stats.TotalTasks)

	fmt.Println("Status Summary:")
	fmt.Printf("  ✅ Done: %d\n", stats.StatusSummary.Done)
	fmt.Printf("  🔄 Doing: %d\n", stats.StatusSummary.Doing)
	fmt.Printf("  📋 Todo: %d\n", stats.StatusSummary.Todo)
	fmt.Printf("  ⏸️ Pending: %d\n", stats.StatusSummary.Pending)
	fmt.Printf("  ❌ Cancelled: %d\n\n", stats.StatusSummary.Cancelled)

	if len(stats.CommentStats) > 0 {
		fmt.Println("By Comment:")
		for _, comment := range stats.CommentStats {
			fmt.Printf("  Comment #%d (%s:%d) - %d tasks\n", 
				comment.CommentID, comment.File, comment.Line, comment.TotalTasks)
			fmt.Printf("    Done: %d, Doing: %d, Todo: %d\n", 
				comment.CompletedTasks, comment.InProgressTasks, comment.PendingTasks)
			
			// Show first 50 characters of origin text for context
			originPreview := comment.OriginText
			if len(originPreview) > 50 {
				originPreview = originPreview[:50] + "..."
			}
			fmt.Printf("    Text: %s\n\n", originPreview)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(statsCmd)
}