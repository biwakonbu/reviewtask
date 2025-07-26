package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
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

// Model represents the TUI dashboard state
type Model struct {
	storageManager *storage.Manager
	tasks          []storage.Task
	stats          tasks.TaskStats
	width          int
	height         int
	lastUpdate     time.Time
	showAll        bool
	specificPR     int
	branch         string
	loadError      error
}

// NewModel creates a new TUI dashboard model
func NewModel(storageManager *storage.Manager, showAll bool, specificPR int, branch string) Model {
	return Model{
		storageManager: storageManager,
		showAll:        showAll,
		specificPR:     specificPR,
		branch:         branch,
		lastUpdate:     time.Now(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTasks,
		tickCmd(),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tasksLoadedMsg:
		if msg.err != nil {
			// Store error for display
			m.loadError = msg.err
		} else {
			m.tasks = msg.tasks
			m.stats = msg.stats
			m.loadError = nil
		}

	case tickMsg:
		return m, tea.Batch(
			m.loadTasks,
			tickCmd(),
		)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the TUI dashboard
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Build the dashboard
	var sections []string

	// Calculate stats
	total := len(m.tasks)
	completed := m.stats.StatusCounts["done"] + m.stats.StatusCounts["cancel"]
	completionRate := float64(0)
	if total > 0 {
		completionRate = float64(completed) / float64(total) * 100
	}

	// Title
	if total == 0 {
		sections = append(sections, "ReviewTask Status - 0% Complete")
	} else {
		sections = append(sections, fmt.Sprintf("ReviewTask Status - %.1f%% Complete (%d/%d)", completionRate, completed, total))
	}
	sections = append(sections, "")

	// Error display
	if m.loadError != nil {
		sections = append(sections, fmt.Sprintf("⚠️  Error loading tasks: %s", m.loadError.Error()))
		sections = append(sections, "")
	}

	// Progress bar with colors based on task status
	progressBar := generateColoredProgressBar(m.stats, 80)
	sections = append(sections, fmt.Sprintf("Progress: %s", progressBar))
	sections = append(sections, "")

	// Task Summary
	sections = append(sections, "Task Summary:")
	sections = append(sections, fmt.Sprintf("  todo: %d    doing: %d    done: %d    pending: %d    cancel: %d",
		m.stats.StatusCounts["todo"], m.stats.StatusCounts["doing"], m.stats.StatusCounts["done"],
		m.stats.StatusCounts["pending"], m.stats.StatusCounts["cancel"]))
	sections = append(sections, "")

	// Current Task
	sections = append(sections, "Current Task:")
	doingTasks := tasks.FilterTasksByStatus(m.tasks, "doing")
	if len(doingTasks) == 0 {
		sections = append(sections, "  アクティブなタスクはありません - すべて完了しています！")
	} else {
		task := doingTasks[0]
		sections = append(sections, fmt.Sprintf("  1. %s  %s    %s", tasks.GenerateTaskID(task), strings.ToUpper(task.Priority), task.Description))
	}
	sections = append(sections, "")

	// Next Tasks
	sections = append(sections, "Next Tasks (up to 5):")
	todoTasks := tasks.FilterTasksByStatus(m.tasks, "todo")
	tasks.SortTasksByPriority(todoTasks)

	if len(todoTasks) == 0 {
		sections = append(sections, "  待機中のタスクはありません")
	} else {
		maxDisplay := 5
		if len(todoTasks) < maxDisplay {
			maxDisplay = len(todoTasks)
		}
		for i := 0; i < maxDisplay; i++ {
			task := todoTasks[i]
			sections = append(sections, fmt.Sprintf("  %d. %s  %s    %s", i+1, tasks.GenerateTaskID(task), strings.ToUpper(task.Priority), task.Description))
		}
	}
	sections = append(sections, "")

	// Footer
	sections = append(sections, fmt.Sprintf("Last updated: %s", m.lastUpdate.Format("15:04:05")))

	return strings.Join(sections, "\n")
}

// Messages

type tasksLoadedMsg struct {
	tasks []storage.Task
	stats tasks.TaskStats
	err   error
}

type tickMsg time.Time

// Commands

func (m Model) loadTasks() tea.Msg {
	// Load tasks based on flags (same logic as AI mode)
	var allTasks []storage.Task
	var err error

	if m.specificPR > 0 {
		allTasks, err = m.storageManager.GetTasksByPR(m.specificPR)
		if err != nil {
			return tasksLoadedMsg{tasks: []storage.Task{}, stats: tasks.TaskStats{}, err: err}
		}
	} else if m.branch != "" {
		prNumbers, err := m.storageManager.GetPRsForBranch(m.branch)
		if err != nil {
			return tasksLoadedMsg{tasks: []storage.Task{}, stats: tasks.TaskStats{}, err: err}
		}
		for _, prNumber := range prNumbers {
			tasks, err := m.storageManager.GetTasksByPR(prNumber)
			if err != nil {
				// Log individual PR errors but continue processing others
				continue
			}
			allTasks = append(allTasks, tasks...)
		}
	} else if m.showAll {
		allTasks, err = m.storageManager.GetAllTasks()
		if err != nil {
			return tasksLoadedMsg{tasks: []storage.Task{}, stats: tasks.TaskStats{}, err: err}
		}
	} else {
		currentBranch, err := m.storageManager.GetCurrentBranch()
		if err != nil {
			return tasksLoadedMsg{tasks: []storage.Task{}, stats: tasks.TaskStats{}, err: err}
		}
		prNumbers, err := m.storageManager.GetPRsForBranch(currentBranch)
		if err != nil {
			return tasksLoadedMsg{tasks: []storage.Task{}, stats: tasks.TaskStats{}, err: err}
		}
		for _, prNumber := range prNumbers {
			tasks, err := m.storageManager.GetTasksByPR(prNumber)
			if err != nil {
				// Log individual PR errors but continue processing others
				continue
			}
			allTasks = append(allTasks, tasks...)
		}
	}

	stats := tasks.CalculateTaskStats(allTasks)

	return tasksLoadedMsg{
		tasks: allTasks,
		stats: stats,
		err:   nil,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
