package cmd

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
	"reviewtask/internal/tui"
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
		return displayAIModeEmpty()
	}

	return displayAIModeContent(allTasks, contextDescription)
}

// displayAIModeEmpty shows empty state in AI mode format
func displayAIModeEmpty() error {
	fmt.Println("ReviewTask Status - 0% Complete")
	fmt.Println()
	fmt.Printf("Progress: %s\n", strings.Repeat("░", 80))
	fmt.Println()
	fmt.Println("Task Summary:")
	fmt.Println("  todo: 0    doing: 0    done: 0    pending: 0    cancel: 0")
	fmt.Println()
	fmt.Println("Current Task:")
	fmt.Println("  アクティブなタスクはありません - すべて完了しています！")
	fmt.Println()
	fmt.Println("Next Tasks:")
	fmt.Println("  待機中のタスクはありません")
	fmt.Println()
	fmt.Printf("Last updated: %s\n", time.Now().Format("15:04:05"))
	return nil
}

// displayAIModeContent shows tasks in AI mode format
func displayAIModeContent(allTasks []storage.Task, contextDescription string) error {
	stats := calculateTaskStats(allTasks)
	total := len(allTasks)
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total) * 100

	fmt.Printf("ReviewTask Status - %.1f%% Complete (%d/%d)\n", completionRate, completed, total)
	fmt.Println()

	// Progress bar
	progressWidth := 80
	filledWidth := int(float64(progressWidth) * completionRate / 100)
	emptyWidth := progressWidth - filledWidth
	progressBar := strings.Repeat("█", filledWidth) + strings.Repeat("░", emptyWidth)
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
	doingTasks := filterTasksByStatus(allTasks, "doing")
	if len(doingTasks) == 0 {
		fmt.Println("  アクティブなタスクはありません")
	} else {
		// Show first doing task with work order format: 着手順, ID, Priority, Title
		task := doingTasks[0]
		fmt.Printf("  1. %s  %s    %s\n", generateTaskID(task), strings.ToUpper(task.Priority), task.Description)
	}
	fmt.Println()

	// Next Tasks (up to 5)
	fmt.Println("Next Tasks (up to 5):")
	todoTasks := filterTasksByStatus(allTasks, "todo")
	sortTasksByPriority(todoTasks)

	if len(todoTasks) == 0 {
		fmt.Println("  待機中のタスクはありません")
	} else {
		// Show top 5 tasks with work order format: 着手順, ID, Priority, Title
		maxDisplay := 5
		if len(todoTasks) < maxDisplay {
			maxDisplay = len(todoTasks)
		}

		for i := 0; i < maxDisplay; i++ {
			task := todoTasks[i]
			fmt.Printf("  %d. %s  %s    %s\n", i+1, generateTaskID(task), strings.ToUpper(task.Priority), task.Description)
		}
	}
	fmt.Println()

	// Last updated timestamp
	fmt.Printf("Last updated: %s\n", time.Now().Format("15:04:05"))

	return nil
}

// generateTaskID creates a task ID in TSK-XXX format
func generateTaskID(task storage.Task) string {
	return fmt.Sprintf("TSK-%03d", task.PRNumber)
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
		// Normalize "cancelled" to "cancel" for backward compatibility
		status := task.Status
		if status == "cancelled" {
			status = "cancel"
		}
		stats.StatusCounts[status]++
		stats.PriorityCounts[task.Priority]++
		stats.PRCounts[task.PRNumber]++
	}

	return stats
}

func init() {
	// Add flags
	statusCmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
	statusCmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
	statusCmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Human mode: rich TUI dashboard with real-time updates")
}
