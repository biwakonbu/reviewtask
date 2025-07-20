package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
)

var (
	statusShowAll    bool
	statusSpecificPR int
	statusBranch     string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current task status and statistics",
	Long: `Display current tasks, next tasks to work on, and overall statistics.

By default, shows tasks for the current branch. Use flags to show all PRs 
or filter by specific criteria.

Shows:
- Current tasks (doing status)
- Next tasks (todo status, sorted by priority)
- Task statistics (status breakdown, priority breakdown, completion rate)

Examples:
  reviewtask status             # Show current branch tasks
  reviewtask status --all       # Show all PRs tasks
  reviewtask status --pr 123    # Show PR #123 tasks
  reviewtask status --branch feature/xyz # Show specific branch tasks`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	storageManager := storage.NewManager()

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
				continue // Skip PRs that can't be read
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
				continue // Skip PRs that can't be read
			}
			allTasks = append(allTasks, tasks...)
		}
		contextDescription = fmt.Sprintf("current branch '%s'", currentBranch)
	}

	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(allTasks) == 0 {
		fmt.Printf("No tasks found for %s. Run 'reviewtask' to fetch and generate tasks from PR reviews.\n", contextDescription)
		return nil
	}

	fmt.Printf("📋 Task Status for %s\n\n", contextDescription)

	// Display current tasks (doing status)
	fmt.Println("🔄 Current Tasks (doing):")
	doingTasks := filterTasksByStatus(allTasks, "doing")
	if len(doingTasks) == 0 {
		fmt.Println("  No tasks currently in progress")
	} else {
		for _, task := range doingTasks {
			fmt.Printf("  • %s [%s] - %s:%d\n", task.Description, task.Priority, task.File, task.Line)
		}
	}
	fmt.Println()

	// Display next tasks (todo status, sorted by priority)
	fmt.Println("📋 Next Tasks (todo, by priority):")
	todoTasks := filterTasksByStatus(allTasks, "todo")
	sortTasksByPriority(todoTasks)

	if len(todoTasks) == 0 {
		fmt.Println("  No pending tasks")
	} else {
		// Show top 5 tasks
		maxDisplay := 5
		if len(todoTasks) < maxDisplay {
			maxDisplay = len(todoTasks)
		}

		for i := 0; i < maxDisplay; i++ {
			task := todoTasks[i]
			fmt.Printf("  %d. %s [%s] - %s:%d\n", i+1, task.Description, task.Priority, task.File, task.Line)
		}

		if len(todoTasks) > maxDisplay {
			fmt.Printf("  ... and %d more tasks\n", len(todoTasks)-maxDisplay)
		}
	}
	fmt.Println()

	// Display statistics
	fmt.Println("📊 Statistics:")
	stats := calculateTaskStats(allTasks)

	// Status breakdown
	fmt.Println("  Status breakdown:")
	fmt.Printf("    todo: %d, doing: %d, done: %d, pending: %d, cancel: %d\n",
		stats.StatusCounts["todo"], stats.StatusCounts["doing"], stats.StatusCounts["done"],
		stats.StatusCounts["pending"], stats.StatusCounts["cancel"])

	// Priority breakdown
	fmt.Println("  Priority breakdown:")
	fmt.Printf("    critical: %d, high: %d, medium: %d, low: %d\n",
		stats.PriorityCounts["critical"], stats.PriorityCounts["high"],
		stats.PriorityCounts["medium"], stats.PriorityCounts["low"])

	// PR breakdown
	if len(stats.PRCounts) > 0 {
		fmt.Println("  PR breakdown:")
		for pr, count := range stats.PRCounts {
			fmt.Printf("    PR-%d: %d tasks\n", pr, count)
		}
	}

	// Completion rate
	total := len(allTasks)
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total) * 100
	fmt.Printf("  Completion rate: %.1f%% (%d/%d)\n", completionRate, completed, total)

	return nil
}

func filterTasksByStatus(tasks []storage.Task, status string) []storage.Task {
	var filtered []storage.Task
	for _, task := range tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func sortTasksByPriority(tasks []storage.Task) {
	// Simple priority sorting: critical > high > medium > low
	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	// Bubble sort for simplicity
	for i := 0; i < len(tasks)-1; i++ {
		for j := 0; j < len(tasks)-i-1; j++ {
			if priorityOrder[tasks[j].Priority] > priorityOrder[tasks[j+1].Priority] {
				tasks[j], tasks[j+1] = tasks[j+1], tasks[j]
			}
		}
	}
}

type TaskStats struct {
	StatusCounts   map[string]int
	PriorityCounts map[string]int
	PRCounts       map[int]int
}

func calculateTaskStats(tasks []storage.Task) TaskStats {
	stats := TaskStats{
		StatusCounts:   make(map[string]int),
		PriorityCounts: make(map[string]int),
		PRCounts:       make(map[int]int),
	}

	for _, task := range tasks {
		stats.StatusCounts[task.Status]++
		stats.PriorityCounts[task.Priority]++
		stats.PRCounts[task.PRNumber]++
	}

	return stats
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Add flags
	statusCmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
	statusCmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
	statusCmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")
}
