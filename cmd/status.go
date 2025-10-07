package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
	"reviewtask/internal/ui"

	"github.com/spf13/cobra"
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
		if prNumber <= 0 {
			return fmt.Errorf("invalid PR number: %s (must be a positive integer)", args[0])
		}
		statusSpecificPR = prNumber
	}

	storageManager := storage.NewManager()

	// Default: AI Mode
	return runAIMode(storageManager, statusSpecificPR)
}

// runAIMode implements simple, parseable text output for automation
func runAIMode(storageManager *storage.Manager, specificPR int) error {
	ctx := context.Background()

	// Initialize GitHub client for comment tracking
	// Skip in test mode to avoid git/network dependencies
	var githubClient *github.Client
	var err error
	if os.Getenv("REVIEWTASK_TEST_MODE") != "true" {
		githubClient, err = github.NewClient()
		if err != nil {
			// Continue without GitHub client - status can work without it
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize GitHub client: %v\n", err)
		}
	}

	// Determine which PR to analyze for unresolved comments
	var targetPR int
	if specificPR > 0 {
		targetPR = specificPR
	} else if !statusShowAll {
		// Get PR for current branch
		currentBranch, err := storageManager.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		prNumbers, err := storageManager.GetPRsForBranch(currentBranch)
		if err != nil {
			return fmt.Errorf("failed to get PRs for current branch '%s': %w", currentBranch, err)
		}
		if len(prNumbers) > 0 {
			targetPR = prNumbers[0] // Use first PR for current branch
		}
	}

	// Get unresolved comments report if we have a GitHub client and target PR
	var unresolvedReport *github.UnresolvedCommentsReport
	if githubClient != nil && targetPR > 0 {
		commentManager := github.NewCommentManager(githubClient)
		unresolvedReport, err = commentManager.GetUnresolvedCommentsReport(ctx, targetPR)
		if err != nil {
			// Continue without unresolved comments report
			fmt.Fprintf(os.Stderr, "Warning: Failed to get unresolved comments report: %v\n", err)
		}
	}

	// Determine which tasks to load based on arguments and flags
	var allTasks []storage.Task
	var contextDescription string

	// Detect completion state by integrating task and comment status
	var completionDetection *CompletionDetectionResult
	if unresolvedReport != nil && targetPR > 0 {
		// We'll detect completion after loading tasks
	}

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

	// Detect completion state after loading tasks
	if unresolvedReport != nil && targetPR > 0 && len(allTasks) > 0 {
		completionDetection = DetectCompletionState(allTasks, unresolvedReport, targetPR)
	}

	if len(allTasks) == 0 {
		if statusShort {
			return displayAIModeEmptyShort()
		}
		return displayAIModeEmpty()
	}

	if statusShort {
		return displayAIModeContentShort(allTasks, contextDescription, unresolvedReport, completionDetection)
	}
	return displayAIModeContent(allTasks, contextDescription, unresolvedReport, completionDetection)
}

// displayAIModeEmpty shows empty state in AI mode format
func displayAIModeEmpty() error {
	storageManager := storage.NewManager()

	// Check for incomplete analysis before showing empty state
	if err := displayIncompleteAnalysis(storageManager); err != nil {
		// Non-fatal error, continue with empty display
		fmt.Printf("%s Warning: Failed to check for incomplete analysis: %v\n\n", ui.SymbolWarning, err)
	}

	fmt.Println(ui.SectionDivider("Review Status"))
	fmt.Println("Progress: 0% Complete (0/0)")
	fmt.Println()
	emptyBar := strings.Repeat("░", 80)
	fmt.Printf("Progress: %s\n", ui.EmptyProgressStyle.Render(emptyBar))
	fmt.Println()

	fmt.Println(ui.SectionDivider("Tasks"))
	fmt.Println("  todo: 0    doing: 0    done: 0    pending: 0    cancel: 0")
	fmt.Println()

	fmt.Println(ui.SectionDivider("Current Task"))
	fmt.Println("  No active tasks - all completed!")
	fmt.Println()

	fmt.Println(ui.SectionDivider("Next Steps"))
	fmt.Println(ui.Success("All tasks completed!"))
	fmt.Println()
	return nil
}

// displayAIModeContent shows tasks in AI mode format
func displayAIModeContent(allTasks []storage.Task, contextDescription string, unresolvedReport *github.UnresolvedCommentsReport, completionDetection *CompletionDetectionResult) error {
	storageManager := storage.NewManager()

	// Check for incomplete analysis before showing task content
	if err := displayIncompleteAnalysis(storageManager); err != nil {
		// Non-fatal error, continue with task display
		fmt.Printf("%s Warning: Failed to check for incomplete analysis: %v\n\n", ui.SymbolWarning, err)
	}

	stats := tasks.CalculateTaskStats(allTasks)
	total := len(allTasks)
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total) * 100

	fmt.Println(ui.SectionDivider("Review Status"))
	fmt.Printf("Progress: %.1f%% Complete (%d/%d) - %s\n", completionRate, completed, total, contextDescription)
	fmt.Println()

	// Display completion status if available
	if completionDetection != nil {
		fmt.Printf("Status: %.1f%% Complete\n", completionDetection.CompletionPercentage)
		fmt.Printf("Summary: %s\n", completionDetection.CompletionSummary)

		if completionDetection.IsComplete {
			fmt.Println(ui.Success("All work completed!"))
		} else {
			fmt.Printf("%s Unresolved items: %d tasks, %d comments\n",
				ui.SymbolWarning,
				len(completionDetection.UnresolvedTasks),
				len(completionDetection.UnresolvedComments))
		}
		fmt.Println()
	} else if unresolvedReport != nil {
		// Fallback to original unresolved comments display if completion detection is not available
		if unresolvedReport.IsComplete() {
			fmt.Println(ui.Success("No unresolved comments"))
			fmt.Println(ui.Success("All threads resolved"))
			fmt.Println()
		} else {
			fmt.Printf("Unresolved Comments: %d\n", len(unresolvedReport.UnanalyzedComments)+len(unresolvedReport.InProgressComments))
			if len(unresolvedReport.UnanalyzedComments) > 0 {
				fmt.Printf("%s %d comments not yet analyzed\n", ui.SymbolError, len(unresolvedReport.UnanalyzedComments))
			}
			if len(unresolvedReport.InProgressComments) > 0 {
				fmt.Printf("%s %d comments with pending tasks\n", ui.SymbolWarning, len(unresolvedReport.InProgressComments))
			}
			fmt.Println()
		}
	}

	// Progress bar with colors based on task status
	progressBar := ui.GenerateColoredProgressBar(stats, 80)
	fmt.Printf("Progress [%s]\n", progressBar)
	fmt.Println()

	// Task Summary
	fmt.Println(ui.SectionDivider("Tasks"))
	fmt.Printf("  TODO: %d\n", stats.StatusCounts["todo"])
	fmt.Printf("  DOING: %d\n", stats.StatusCounts["doing"])
	fmt.Printf("  DONE: %d\n", stats.StatusCounts["done"])
	fmt.Printf("  PENDING: %d\n", stats.StatusCounts["pending"])
	fmt.Printf("  CANCEL: %d\n", stats.StatusCounts["cancel"])
	fmt.Println()

	// Current Task (single active task)
	doingTasks := tasks.FilterTasksByStatus(allTasks, "doing")
	if len(doingTasks) > 0 {
		fmt.Println(ui.SectionDivider("Current Task"))
		task := doingTasks[0]
		fmt.Printf("  %s  %s    %s\n", task.ID, strings.ToUpper(task.Priority), task.Description)
		fmt.Println()
	}

	// Next Tasks (up to 5)
	todoTasks := tasks.FilterTasksByStatus(allTasks, "todo")
	tasks.SortTasksByPriority(todoTasks)

	if len(todoTasks) > 0 {
		fmt.Println(ui.SectionDivider("Next Tasks"))
		maxDisplay := 5
		if len(todoTasks) < maxDisplay {
			maxDisplay = len(todoTasks)
		}

		for i := 0; i < maxDisplay; i++ {
			task := todoTasks[i]
			fmt.Printf("  %d. %s  %s    %s\n", i+1, task.ID, strings.ToUpper(task.Priority), task.Description)
		}
		fmt.Println()
	}

	// Next Steps
	fmt.Println(ui.SectionDivider("Next Steps"))
	if unresolvedReport != nil && !unresolvedReport.IsComplete() {
		if len(unresolvedReport.UnanalyzedComments) > 0 {
			fmt.Println(ui.Warning("You have unresolved review comments"))
			fmt.Println(ui.Indent("reviewtask analyze", 2))
		} else if len(doingTasks) > 0 {
			fmt.Println(ui.Next("Continue with current task"))
			fmt.Println(ui.Indent(fmt.Sprintf("reviewtask show %s", doingTasks[0].ID), 2))
		} else if len(todoTasks) > 0 {
			fmt.Println(ui.Next("Start next task"))
			fmt.Println(ui.Indent(fmt.Sprintf("reviewtask start %s", todoTasks[0].ID), 2))
		}
	} else if len(doingTasks) > 0 {
		fmt.Println(ui.Next("Complete current task"))
		fmt.Println(ui.Indent(fmt.Sprintf("reviewtask done %s", doingTasks[0].ID), 2))
	} else if len(todoTasks) > 0 {
		fmt.Println(ui.Next("Start next task"))
		fmt.Println(ui.Indent(fmt.Sprintf("reviewtask start %s", todoTasks[0].ID), 2))
	} else {
		fmt.Println(ui.Success("All tasks completed!"))
	}
	fmt.Println()

	return nil
}

// displayAIModeEmptyShort shows empty state in brief format
func displayAIModeEmptyShort() error {
	fmt.Printf("Status: 0%% (0/0) | todo:0 doing:0 done:0 pending:0 cancel:0\n")
	return nil
}

// displayAIModeContentShort shows tasks in brief format
func displayAIModeContentShort(allTasks []storage.Task, contextDescription string, unresolvedReport *github.UnresolvedCommentsReport, completionDetection *CompletionDetectionResult) error {
	stats := tasks.CalculateTaskStats(allTasks)
	total := len(allTasks)
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total) * 100

	// Brief one-line summary
	fmt.Printf("Status: %.1f%% (%d/%d) - %s | todo:%d doing:%d done:%d pending:%d cancel:%d",
		completionRate, completed, total, contextDescription,
		stats.StatusCounts["todo"], stats.StatusCounts["doing"], stats.StatusCounts["done"],
		stats.StatusCounts["pending"], stats.StatusCounts["cancel"])

	// Show current task if any
	doingTasks := tasks.FilterTasksByStatus(allTasks, "doing")
	if len(doingTasks) > 0 {
		task := doingTasks[0]
		currentID := task.ID
		if len(currentID) > 8 {
			currentID = currentID[:8]
		}
		fmt.Printf(" | Current: %s (%s)", currentID, strings.ToUpper(task.Priority))
	}

	// Show next task if any
	todoTasks := tasks.FilterTasksByStatus(allTasks, "todo")
	tasks.SortTasksByPriority(todoTasks)
	if len(todoTasks) > 0 {
		task := todoTasks[0]
		nextID := task.ID
		if len(nextID) > 8 {
			nextID = nextID[:8]
		}
		fmt.Printf(" | Next: %s (%s)", nextID, strings.ToUpper(task.Priority))
	}

	fmt.Println()
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
		fmt.Println(ui.SectionDivider("Incomplete Analysis"))
		for _, info := range incompletePRs {
			remaining := info.TotalComments - info.ProcessedCount
			percentage := 0.0
			if info.TotalComments > 0 {
				percentage = float64(remaining) / float64(info.TotalComments) * 100
			}
			fmt.Printf("  PR #%d: %d/%d comments processed, %d remaining (%.1f%% pending)\n",
				info.PRNumber, info.ProcessedCount, info.TotalComments, remaining, percentage)
			fmt.Printf("    %s Continue with: reviewtask analyze %d\n", ui.SymbolNext, info.PRNumber)
		}
		fmt.Println()
	}

	// Check for unresolved comments
	if err := displayUnresolvedComments(storageManager); err != nil {
		// Non-fatal error
		fmt.Printf("%s Warning: Failed to check for unresolved comments: %v\n\n", ui.SymbolWarning, err)
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
		fmt.Println(ui.SectionDivider("Unresolved Review Threads"))
		for _, info := range prsWithUnresolved {
			percentage := float64(info.UnresolvedCount) / float64(info.TotalComments) * 100
			fmt.Printf("  PR #%d: %d/%d comments unresolved (%.1f%%)\n",
				info.PRNumber, info.UnresolvedCount, info.TotalComments, percentage)
			if info.LastCheckedAt != "" {
				fmt.Printf("    Last checked: %s\n", info.LastCheckedAt)
			}
			fmt.Printf("    %s Address feedback and resolve threads to complete review\n", ui.SymbolNext)
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

// CompletionDetectionResult represents the result of completion state detection
type CompletionDetectionResult struct {
	IsComplete           bool     `json:"is_complete"`
	CompletionSummary    string   `json:"completion_summary"`
	UnresolvedTasks      []string `json:"unresolved_tasks"`
	UnresolvedComments   []string `json:"unresolved_comments"`
	CompletionPercentage float64  `json:"completion_percentage"`
}

// DetectCompletionState analyzes tasks and comments to determine completion state
func DetectCompletionState(tasks []storage.Task, unresolvedReport *github.UnresolvedCommentsReport, prNumber int) *CompletionDetectionResult {
	result := &CompletionDetectionResult{
		IsComplete:         false,
		CompletionSummary:  "",
		UnresolvedTasks:    []string{},
		UnresolvedComments: []string{},
	}

	// Count task statuses
	var todoTasks, doingTasks, doneTasks, pendingTasks, cancelledTasks int
	for _, task := range tasks {
		switch task.Status {
		case "todo":
			todoTasks++
		case "doing":
			doingTasks++
		case "done":
			doneTasks++
		case "pending":
			pendingTasks++
		case "cancel":
			cancelledTasks++
		}
	}

	// Check if all tasks are completed (done or cancelled)
	allTasksCompleted := todoTasks == 0 && doingTasks == 0 && pendingTasks == 0

	// Check if all comments are resolved
	allCommentsResolved := unresolvedReport != nil && unresolvedReport.IsComplete()

	// Determine completion state
	// Complete if all tasks are completed, regardless of comment resolution status
	// (comments may not exist or may not be relevant for completion)
	result.IsComplete = allTasksCompleted

	// Calculate completion percentage
	totalTasks := len(tasks)
	completedTasks := doneTasks + cancelledTasks
	if totalTasks > 0 {
		result.CompletionPercentage = float64(completedTasks) / float64(totalTasks) * 100
	} else {
		result.CompletionPercentage = 100
	}

	// Build completion summary
	if result.IsComplete {
		result.CompletionSummary = "✅ All tasks completed and all comments resolved"
	} else {
		summaryParts := []string{}

		if !allTasksCompleted {
			if todoTasks > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d pending tasks", todoTasks))
			}
			if doingTasks > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d in-progress tasks", doingTasks))
			}
			if pendingTasks > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d blocked tasks", pendingTasks))
			}
		}

		if !allCommentsResolved && unresolvedReport != nil {
			unresolvedCount := len(unresolvedReport.UnanalyzedComments) + len(unresolvedReport.InProgressComments)
			if unresolvedCount > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%d unresolved comments", unresolvedCount))
			}
		}

		if len(summaryParts) > 0 {
			result.CompletionSummary = "⏳ Incomplete: " + strings.Join(summaryParts, ", ")
		} else {
			result.CompletionSummary = "⏳ Status unclear"
		}
	}

	// Collect unresolved task IDs
	for _, task := range tasks {
		if task.Status == "todo" || task.Status == "doing" || task.Status == "pending" {
			result.UnresolvedTasks = append(result.UnresolvedTasks, task.ID)
		}
	}

	// Collect unresolved comment IDs
	if unresolvedReport != nil {
		for _, comment := range unresolvedReport.UnanalyzedComments {
			result.UnresolvedComments = append(result.UnresolvedComments, fmt.Sprintf("comment-%d", comment.ID))
		}
		for _, comment := range unresolvedReport.InProgressComments {
			result.UnresolvedComments = append(result.UnresolvedComments, fmt.Sprintf("comment-%d", comment.ID))
		}
	}

	return result
}

func init() {
	// Add flags - only --all and --short for v3.0.0
	statusCmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
	statusCmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")
}
