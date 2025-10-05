package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
	"reviewtask/internal/ui"
)

var (
	statusShowAll bool
	statusShort   bool
)

var statusCmd = &cobra.Command{
	Use:   "status [PR_NUMBER]",
	Short: "Show current task status and statistics",
	Long: `Display current tasks, next tasks to work on, and overall statistics.

By default, shows tasks for the current branch. Provide a PR number to show
tasks for a specific PR, or use --all to show all PRs.

Shows:
- Current tasks (doing status)
- Next tasks (todo status, sorted by priority)
- Task statistics (status breakdown, priority breakdown, completion rate)

Examples:
  reviewtask status             # Show tasks for current branch
  reviewtask status 123         # Show tasks for PR #123
  reviewtask status --all       # Show tasks for all PRs
  reviewtask status --short     # Brief output format`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Display AI provider info
	_, err := DisplayAIProviderIfNeeded()
	if err != nil {
		// Continue without config - status can work without it
	}

	// Parse PR number from arguments if provided
	var statusSpecificPR int
	if len(args) > 0 {
		prNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}
		statusSpecificPR = prNumber
	}

	storageManager := storage.NewManager()

	// Default: AI Mode
	return runAIMode(storageManager, statusSpecificPR)
}

// runAIMode implements simple, parseable text output for automation
func runAIMode(storageManager *storage.Manager, specificPR int) error {
	// Determine which tasks to load based on arguments and flags
	var allTasks []storage.Task
	var err error
	var contextDescription string

	if specificPR > 0 {
		// PR number provided as argument
		allTasks, err = storageManager.GetTasksByPR(specificPR)
		contextDescription = fmt.Sprintf("PR #%d", specificPR)
	} else if statusShowAll {
		// --all flag
		allTasks, err = storageManager.GetAllTasks()
		contextDescription = "all PRs"
	} else {
		// Default: current branch
		currentBranch, err := storageManager.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		prNumbers, err := storageManager.GetPRsForBranch(currentBranch)
		if err != nil {
			return fmt.Errorf("failed to get PRs for current branch '%s': %w", currentBranch, err)
		}

		for _, prNumber := range prNumbers {
			tasks, err := storageManager.GetTasksByPR(prNumber)
			if err != nil {
				return fmt.Errorf("failed to get tasks for PR %d: %w", prNumber, err)
			}
			allTasks = append(allTasks, tasks...)
		}
		contextDescription = fmt.Sprintf("current branch '%s'", currentBranch)
	}

	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(allTasks) == 0 {
		return displayAIModeEmpty()
	}

	return displayAIModeContent(allTasks, contextDescription)
}

// displayAIModeEmpty shows empty state in AI mode format
func displayAIModeEmpty() error {
	storageManager := storage.NewManager()

	// Check for incomplete analysis before showing empty state
	if err := displayIncompleteAnalysis(storageManager); err != nil {
		// Non-fatal error, continue with empty display
		fmt.Printf("⚠️  Warning: Failed to check for incomplete analysis: %v\n\n", err)
	}

	fmt.Println("ReviewTask Status - 0% Complete")
	fmt.Println()
	emptyBar := strings.Repeat("░", 80)
	fmt.Printf("Progress: %s\n", ui.EmptyProgressStyle.Render(emptyBar))
	fmt.Println()
	fmt.Println("Task Summary:")
	fmt.Println("  todo: 0    doing: 0    done: 0    pending: 0    cancel: 0")
	fmt.Println()
	fmt.Println("Current Task:")
	fmt.Println("  No active tasks - all completed!")
	fmt.Println()
	fmt.Println("Next Tasks:")
	fmt.Println("  No pending tasks")
	fmt.Println()
	fmt.Printf("Last updated: %s\n", time.Now().Format("15:04:05"))
	return nil
}

// displayAIModeContent shows tasks in AI mode format
func displayAIModeContent(allTasks []storage.Task, contextDescription string) error {
	storageManager := storage.NewManager()

	// Check for incomplete analysis before showing task content
	if err := displayIncompleteAnalysis(storageManager); err != nil {
		// Non-fatal error, continue with task display
		fmt.Printf("⚠️  Warning: Failed to check for incomplete analysis: %v\n\n", err)
	}

	stats := tasks.CalculateTaskStats(allTasks)
	total := len(allTasks)
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total) * 100

	fmt.Printf("ReviewTask Status - %.1f%% Complete (%d/%d) - %s\n", completionRate, completed, total, contextDescription)
	fmt.Println()

	// Progress bar with colors based on task status
	progressBar := ui.GenerateColoredProgressBar(stats, 80)
	fmt.Printf("Progress: %s\n", progressBar)
	fmt.Println()

	// Task Summary
	fmt.Println("Task Summary:")
	fmt.Printf("  todo: %d    doing: %d    done: %d    pending: %d    cancel: %d\n",
		stats.StatusCounts["todo"], stats.StatusCounts["doing"], stats.StatusCounts["done"],
		stats.StatusCounts["pending"], stats.StatusCounts["cancel"])
	fmt.Println()

	// Current Task (single active task)
	fmt.Println("Current Task:")
	doingTasks := tasks.FilterTasksByStatus(allTasks, "doing")
	if len(doingTasks) == 0 {
		fmt.Println("  No active tasks")
	} else {
		// Show first doing task with work order format: 着手順, ID, Priority, Title
		task := doingTasks[0]
		fmt.Printf("  1. %s  %s    %s\n", task.ID, strings.ToUpper(task.Priority), task.Description)
	}
	fmt.Println()

	// Next Tasks (up to 5)
	fmt.Println("Next Tasks (up to 5):")
	todoTasks := tasks.FilterTasksByStatus(allTasks, "todo")
	tasks.SortTasksByPriority(todoTasks)

	if len(todoTasks) == 0 {
		fmt.Println("  No pending tasks")
	} else {
		// Show top 5 tasks with work order format: 着手順, ID, Priority, Title
		maxDisplay := 5
		if len(todoTasks) < maxDisplay {
			maxDisplay = len(todoTasks)
		}

		for i := 0; i < maxDisplay; i++ {
			task := todoTasks[i]
			fmt.Printf("  %d. %s  %s    %s\n", i+1, task.ID, strings.ToUpper(task.Priority), task.Description)
		}
	}
	fmt.Println()

	// Last updated timestamp
	fmt.Printf("Last updated: %s\n", time.Now().Format("15:04:05"))

	return nil
}

// displayIncompleteAnalysis checks and displays any PRs with incomplete analysis
func displayIncompleteAnalysis(storageManager *storage.Manager) error {
	// Get all PR numbers
	allPRs, err := storageManager.GetAllPRNumbers()
	if err != nil {
		return fmt.Errorf("failed to get PR numbers: %w", err)
	}

	var incompletePRs []incompleteAnalysisInfo
	for _, prNumber := range allPRs {
		// Check if reviews exist
		reviewsExist, err := storageManager.ReviewsExist(prNumber)
		if err != nil {
			continue // Skip on error
		}
		if !reviewsExist {
			continue // No reviews, skip
		}

		// Check if checkpoint exists (indicating incomplete analysis)
		checkpointExists, err := storageManager.CheckpointExists(prNumber)
		if err != nil {
			continue // Skip on error
		}
		if !checkpointExists {
			continue // No checkpoint, analysis is complete or not started
		}

		// Load checkpoint to get progress info
		checkpoint, err := storageManager.LoadCheckpoint(prNumber)
		if err != nil || checkpoint == nil {
			continue // Skip on error
		}

		incompletePRs = append(incompletePRs, incompleteAnalysisInfo{
			PRNumber:        prNumber,
			ProcessedCount:  checkpoint.ProcessedCount,
			TotalComments:   checkpoint.TotalComments,
			LastProcessedAt: checkpoint.LastProcessedAt,
		})
	}

	if len(incompletePRs) > 0 {
		fmt.Println("📊 Incomplete Analysis:")
		for _, info := range incompletePRs {
			remaining := info.TotalComments - info.ProcessedCount
			percentage := float64(remaining) / float64(info.TotalComments) * 100
			fmt.Printf("  PR #%d: %d/%d comments processed, %d remaining (%.1f%% pending)\n",
				info.PRNumber, info.ProcessedCount, info.TotalComments, remaining, percentage)
			fmt.Printf("    🔄 Continue with: reviewtask analyze %d\n", info.PRNumber)
		}
		fmt.Println()
	}

	// Check for unresolved comments
	if err := displayUnresolvedComments(storageManager); err != nil {
		// Non-fatal error
		fmt.Printf("⚠️  Warning: Failed to check for unresolved comments: %v\n\n", err)
	}

	return nil
}

// displayUnresolvedComments checks and displays PRs with unresolved review threads
func displayUnresolvedComments(storageManager *storage.Manager) error {
	// Get all PR numbers
	allPRs, err := storageManager.GetAllPRNumbers()
	if err != nil {
		return fmt.Errorf("failed to get PR numbers: %w", err)
	}

	type unresolvedInfo struct {
		PRNumber        int
		UnresolvedCount int
		TotalComments   int
		LastCheckedAt   string
	}

	var prsWithUnresolved []unresolvedInfo
	for _, prNumber := range allPRs {
		// Load reviews
		reviews, err := storageManager.LoadReviews(prNumber)
		if err != nil {
			continue // Skip on error
		}

		totalComments := 0
		unresolvedCount := 0
		lastChecked := ""

		for _, review := range reviews {
			for _, comment := range review.Comments {
				// Skip embedded comments without IDs
				if comment.ID == 0 {
					continue
				}

				totalComments++

				// Check if thread is unresolved
				if !comment.GitHubThreadResolved {
					unresolvedCount++
				}

				// Track latest check time
				if comment.LastCheckedAt != "" && (lastChecked == "" || comment.LastCheckedAt > lastChecked) {
					lastChecked = comment.LastCheckedAt
				}
			}
		}

		if unresolvedCount > 0 {
			prsWithUnresolved = append(prsWithUnresolved, unresolvedInfo{
				PRNumber:        prNumber,
				UnresolvedCount: unresolvedCount,
				TotalComments:   totalComments,
				LastCheckedAt:   lastChecked,
			})
		}
	}

	if len(prsWithUnresolved) > 0 {
		fmt.Println("💬 Unresolved Review Threads:")
		for _, info := range prsWithUnresolved {
			percentage := float64(info.UnresolvedCount) / float64(info.TotalComments) * 100
			fmt.Printf("  PR #%d: %d/%d comments unresolved (%.1f%%)\n",
				info.PRNumber, info.UnresolvedCount, info.TotalComments, percentage)
			if info.LastCheckedAt != "" {
				fmt.Printf("    Last checked: %s\n", info.LastCheckedAt)
			}
			fmt.Printf("    📝 Address feedback and resolve threads to complete review\n")
		}
		fmt.Println()
	}

	return nil
}

// incompleteAnalysisInfo holds information about a PR with incomplete analysis
type incompleteAnalysisInfo struct {
	PRNumber        int
	ProcessedCount  int
	TotalComments   int
	LastProcessedAt time.Time
}

func init() {
	// Add flags - only --all and --short for v3.0.0
	statusCmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
	statusCmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")
}
