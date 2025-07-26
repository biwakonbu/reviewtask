package cmd

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
	"reviewtask/internal/tui"
)

var (
	statusShowAll    bool
	statusSpecificPR int
	statusBranch     string
	statusWatch      bool
)

// Progress bar color styles for different task states
var (
	todoProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")) // Gray for TODO

	doingProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")) // Yellow for DOING

	doneProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")) // Green for DONE

	pendingProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")) // Red for PENDING

	emptyProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")) // Dark gray for empty
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
	fmt.Println("ReviewTask Status - 0% Complete")
	fmt.Println()
	emptyBar := strings.Repeat("░", 80)
	fmt.Printf("Progress: %s\n", emptyProgressStyle.Render(emptyBar))
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
	stats := tasks.CalculateTaskStats(allTasks)
	total := len(allTasks)
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total) * 100

	fmt.Printf("ReviewTask Status - %.1f%% Complete (%d/%d) - %s\n", completionRate, completed, total, contextDescription)
	fmt.Println()

	// Progress bar with colors based on task status
	progressBar := generateColoredProgressBar(stats, 80)
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

// generateColoredProgressBar creates a progress bar with colors representing different task states
func generateColoredProgressBar(stats tasks.TaskStats, width int) string {
	total := stats.StatusCounts["todo"] + stats.StatusCounts["doing"] +
		stats.StatusCounts["done"] + stats.StatusCounts["pending"] + stats.StatusCounts["cancel"]

	if total == 0 {
		// Empty progress bar
		emptyBar := strings.Repeat("░", width)
		return emptyProgressStyle.Render(emptyBar)
	}

	// Calculate completion rate
	completed := stats.StatusCounts["done"] + stats.StatusCounts["cancel"]
	completionRate := float64(completed) / float64(total)

	// Calculate widths based on completion vs remaining
	filledWidth := int(completionRate * float64(width))
	emptyWidth := width - filledWidth

	// For filled portion, show proportional colors for done/cancel
	var segments []string

	if filledWidth > 0 {
		// Within filled portion, show proportions of done vs cancel
		if completed > 0 {
			doneInFilled := int(float64(stats.StatusCounts["done"]) / float64(completed) * float64(filledWidth))
			cancelInFilled := filledWidth - doneInFilled

			if doneInFilled > 0 {
				segments = append(segments, doneProgressStyle.Render(strings.Repeat("█", doneInFilled)))
			}
			if cancelInFilled > 0 {
				segments = append(segments, emptyProgressStyle.Render(strings.Repeat("█", cancelInFilled)))
			}
		}
	}

	// For empty portion, show remaining work with status colors
	if emptyWidth > 0 {
		remaining := stats.StatusCounts["todo"] + stats.StatusCounts["doing"] + stats.StatusCounts["pending"]
		if remaining > 0 {
			// Proportional representation of remaining work
			doingInEmpty := int(float64(stats.StatusCounts["doing"]) / float64(remaining) * float64(emptyWidth))
			pendingInEmpty := int(float64(stats.StatusCounts["pending"]) / float64(remaining) * float64(emptyWidth))
			todoInEmpty := emptyWidth - doingInEmpty - pendingInEmpty

			if doingInEmpty > 0 {
				segments = append(segments, doingProgressStyle.Render(strings.Repeat("░", doingInEmpty)))
			}
			if pendingInEmpty > 0 {
				segments = append(segments, pendingProgressStyle.Render(strings.Repeat("░", pendingInEmpty)))
			}
			if todoInEmpty > 0 {
				segments = append(segments, todoProgressStyle.Render(strings.Repeat("░", todoInEmpty)))
			}
		} else {
			// Just empty gray
			segments = append(segments, emptyProgressStyle.Render(strings.Repeat("░", emptyWidth)))
		}
	}

	return strings.Join(segments, "")
}

func init() {
	// Add flags
	statusCmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
	statusCmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
	statusCmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")
	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Human mode: rich TUI dashboard with real-time updates")
}
