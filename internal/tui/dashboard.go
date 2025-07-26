package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
	"reviewtask/internal/ui"
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
	progressBar := ui.GenerateColoredProgressBar(m.stats, 80)
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
