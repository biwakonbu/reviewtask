package cmd

import (
	"fmt"
	"strconv"

	"reviewtask/internal/ai"
	"reviewtask/internal/storage"
	"github.com/spf13/cobra"
)

var (
	showAllPRs bool
	specificPR int
	branchName string
)

var statsCmd = &cobra.Command{
	Use:   "stats [PR_NUMBER]",
	Short: "Show task statistics by comment",
	Long: `Display detailed statistics about tasks generated from PR review comments.
Shows both overall statistics and per-comment breakdown of task status.

By default, shows statistics for the current branch. Use flags to show all PRs 
or filter by specific criteria.

Examples:
  reviewtask stats           # Show stats for current branch
  reviewtask stats --all     # Show stats for all PRs
  reviewtask stats --pr 123  # Show stats for PR #123
  reviewtask stats --branch feature/xyz  # Show stats for specific branch
  reviewtask stats 123       # Show stats for PR #123 (positional)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	// Initialize storage manager
	storageManager := storage.NewManager()
	statsManager := ai.NewStatisticsManager(storageManager)

	// Determine what statistics to show based on flags and arguments
	var stats *storage.TaskStatistics
	var err error

	// Priority: positional argument > --pr flag > --branch flag > --all flag > current branch (default)
	if len(args) > 0 {
		// Positional argument for PR number
		prNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}
		stats, err = statsManager.GenerateTaskStatistics(prNumber)
		if err != nil {
			return fmt.Errorf("failed to generate statistics for PR #%d: %w", prNumber, err)
		}
	} else if specificPR > 0 {
		// --pr flag
		stats, err = statsManager.GenerateTaskStatistics(specificPR)
		if err != nil {
			return fmt.Errorf("failed to generate statistics for PR #%d: %w", specificPR, err)
		}
	} else if branchName != "" {
		// --branch flag
		stats, err = statsManager.GenerateBranchStatistics(branchName)
		if err != nil {
			return fmt.Errorf("failed to generate statistics for branch '%s': %w", branchName, err)
		}
	} else if showAllPRs {
		// --all flag - show all PRs statistics
		return showAllPRsStatistics(storageManager, statsManager)
	} else {
		// Default: current branch statistics
		stats, err = statsManager.GenerateCurrentBranchStatistics()
		if err != nil {
			return fmt.Errorf("failed to generate statistics for current branch: %w", err)
		}
	}

	// Display formatted statistics
	displayStatistics(stats)
	return nil
}

func showAllPRsStatistics(storageManager *storage.Manager, statsManager *ai.StatisticsManager) error {
	// Get all PR numbers
	prNumbers, err := storageManager.GetAllPRNumbers()
	if err != nil {
		return fmt.Errorf("failed to get PR numbers: %w", err)
	}

	if len(prNumbers) == 0 {
		fmt.Println("📊 No PRs found")
		return nil
	}

	fmt.Printf("📊 Task Statistics for All PRs (%d total)\n\n", len(prNumbers))

	var totalStats storage.StatusSummary
	for _, prNumber := range prNumbers {
		stats, err := statsManager.GenerateTaskStatistics(prNumber)
		if err != nil {
			fmt.Printf("⚠️ Failed to get stats for PR #%d: %v\n", prNumber, err)
			continue
		}

		// Aggregate totals
		totalStats.Done += stats.StatusSummary.Done
		totalStats.Doing += stats.StatusSummary.Doing
		totalStats.Todo += stats.StatusSummary.Todo
		totalStats.Pending += stats.StatusSummary.Pending
		totalStats.Cancelled += stats.StatusSummary.Cancelled

		fmt.Printf("PR #%d: %d tasks (%d done, %d doing, %d todo)\n",
			prNumber, stats.TotalTasks, stats.StatusSummary.Done,
			stats.StatusSummary.Doing, stats.StatusSummary.Todo)
	}

	fmt.Println("\nOverall Summary:")
	fmt.Printf("  ✅ Done: %d\n", totalStats.Done)
	fmt.Printf("  🔄 Doing: %d\n", totalStats.Doing)
	fmt.Printf("  📋 Todo: %d\n", totalStats.Todo)
	fmt.Printf("  ⏸️ Pending: %d\n", totalStats.Pending)
	fmt.Printf("  ❌ Cancelled: %d\n", totalStats.Cancelled)

	return nil
}

func displayStatistics(stats *storage.TaskStatistics) {
	// Display header based on whether it's PR-specific or branch-specific
	if stats.BranchName != "" {
		fmt.Printf("📊 Task Statistics for Branch '%s'\n\n", stats.BranchName)
	} else {
		fmt.Printf("📊 Task Statistics for PR #%d\n\n", stats.PRNumber)
	}

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
}

func init() {
	// Add flags
	statsCmd.Flags().BoolVar(&showAllPRs, "all", false, "Show statistics for all PRs")
	statsCmd.Flags().IntVar(&specificPR, "pr", 0, "Show statistics for specific PR number")
	statsCmd.Flags().StringVar(&branchName, "branch", "", "Show statistics for specific branch")
}
