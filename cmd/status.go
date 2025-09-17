package cmd

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
	"reviewtask/internal/tui"
	"reviewtask/internal/ui"
)

var (
	statusShowAll    bool
	statusSpecificPR int
	statusBranch     string
	statusWatch      bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current task status and statistics",
	Long: `Display current tasks, next tasks to work on, and overall statistics.

By default, shows tasks for the current branch. Use flags to show all PRs 
or filter by specific criteria.

Output Modes:
- AI Mode (default): Clean, parseable text format for automation
- Human Mode (--watch): Rich TUI dashboard with real-time updates

Shows:
- Current tasks (doing status)
- Next tasks (todo status, sorted by priority)
- Task statistics (status breakdown, priority breakdown, completion rate)

Examples:
  reviewtask status             # AI mode: simple text output
  reviewtask status --all       # Show all PRs tasks
  reviewtask status --pr 123    # Show PR #123 tasks
  reviewtask status --branch feature/xyz # Show specific branch tasks
  reviewtask status -w          # Human mode: rich TUI dashboard
  reviewtask status --watch --all # TUI dashboard for all PRs`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	storageManager := storage.NewManager()

	// Check for watch mode
	if statusWatch {
		return runHumanMode(storageManager)
	}

	// Default: AI Mode
	return runAIMode(storageManager)
}

// runAIMode implements simple, parseable text output for automation
func runAIMode(storageManager *storage.Manager) error {
	// Determine which tasks to load based on flags
	var allTasks []storage.Task
	var err error
	var contextDescription string

	if statusSpecificPR > 0 {
		// --pr flag
		allTasks, err = storageManager.GetTasksByPR(statusSpecificPR)
		contextDescription = fmt.Sprintf("PR #%d", statusSpecificPR)
	} else if statusBranch != "" {
		// --branch flag
		prNumbers, err := storageManager.GetPRsForBranch(statusBranch)
		if err != nil {
			return fmt.Errorf("failed to get PRs for branch '%s': %w", statusBranch, err)
		}

		for _, prNumber := range prNumbers {
			tasks, err := storageManager.GetTasksByPR(prNumber)
			if err != nil {
				return fmt.Errorf("failed to get tasks for PR %d: %w", prNumber, err)
			}
			allTasks = append(allTasks, tasks...)
		}
		contextDescription = fmt.Sprintf("branch '%s'", statusBranch)
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

// runHumanMode implements rich TUI dashboard with real-time updates
func runHumanMode(storageManager *storage.Manager) error {
	// Import the TUI dashboard
	model := tui.NewModel(storageManager, statusShowAll, statusSpecificPR, statusBranch)

	// Create and run the bubbletea program
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

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
	// Add flags
	statusCmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
	statusCmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
	statusCmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Human mode: rich TUI dashboard with real-time updates")
}
