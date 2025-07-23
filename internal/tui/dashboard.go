package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
)

// UI layout constants
const (
	dashboardTitlePrefix   = "┌─ ReviewTask Status Dashboard "
	dashboardBorderPadding = 2
	progressBarPadding     = 10
	taskBoxWidth           = 75 // Width of task content boxes
	taskBoxPadding         = 6  // Padding for task box content
	footerPadding          = 58 // Padding for footer text
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

	// Styles
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230"))

	// Build the dashboard
	var sections []string

	// Header
	headerWidth := m.width - dashboardBorderPadding
	titleLength := len(dashboardTitlePrefix) + 1 // +1 for closing "┐"
	header := borderStyle.Width(headerWidth).Render(
		titleStyle.Render(dashboardTitlePrefix + strings.Repeat("─", m.width-titleLength-1) + "┐"),
	)
	sections = append(sections, header)

	// Error display
	if m.loadError != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		errorMsg := fmt.Sprintf("│ ⚠️  Error loading tasks: %s", m.loadError.Error())
		errorLine := errorStyle.Render(padToWidth(errorMsg, m.width-3)) + " │"
		sections = append(sections, errorLine)
		sections = append(sections, "│"+strings.Repeat(" ", m.width-2)+"│")
	}

	// Progress Overview
	sections = append(sections, m.renderProgressSection())

	// Task Summary
	sections = append(sections, m.renderTaskSummary())

	// Current Task
	sections = append(sections, m.renderCurrentTask())

	// Next Tasks
	sections = append(sections, m.renderNextTasks())

	// Footer
	footer := fmt.Sprintf("│ Press Ctrl+C to exit%sLast updated: %s │",
		strings.Repeat(" ", footerPadding-24), // 24 is the length of "Press Ctrl+C to exit" + "Last updated: "
		m.lastUpdate.Format("15:04"))
	sections = append(sections, footer)

	// Bottom border
	sections = append(sections, "└"+strings.Repeat("─", m.width-dashboardBorderPadding)+"┘")

	return strings.Join(sections, "\n")
}

func (m Model) renderProgressSection() string {
	if len(m.tasks) == 0 {
		return m.renderEmptyProgress()
	}

	completed := m.stats.StatusCounts["done"] + m.stats.StatusCounts["cancel"]
	total := len(m.tasks)
	percentage := float64(completed) / float64(total) * 100

	// Progress bar
	barWidth := m.width - progressBarPadding
	filledWidth := int(float64(barWidth) * percentage / 100)
	emptyWidth := barWidth - filledWidth

	progressBar := strings.Repeat("█", filledWidth) + strings.Repeat("░", emptyWidth)

	return fmt.Sprintf(`│                                                                               │
│ Progress Overview                                                             │
│ %s │
│ [%s] %.0f%%   │
│                                                                               │`,
		progressBar, progressBar, percentage)
}

func (m Model) renderEmptyProgress() string {
	barWidth := m.width - progressBarPadding
	progressBar := strings.Repeat("░", barWidth)

	return fmt.Sprintf(`│                                                                               │
│ Progress Overview                                                             │
│ %s │
│ [%s] 0%%    │
│                                                                               │`,
		progressBar, strings.Repeat(" ", barWidth))
}

func (m Model) renderTaskSummary() string {
	summary := fmt.Sprintf("  Todo: %d    Doing: %d    Done: %d    Pending: %d    Cancel: %d              ",
		m.stats.StatusCounts["todo"],
		m.stats.StatusCounts["doing"],
		m.stats.StatusCounts["done"],
		m.stats.StatusCounts["pending"],
		m.stats.StatusCounts["cancel"])

	return fmt.Sprintf(`│ Task Summary                                                                  │
│ ┌%s┐   │
│ │%s│   │
│ └%s┘   │
│                                                                               │`,
		strings.Repeat("─", taskBoxWidth-2),
		summary,
		strings.Repeat("─", taskBoxWidth-2))
}

func (m Model) renderCurrentTask() string {
	doingTasks := tasks.FilterTasksByStatus(m.tasks, "doing")

	content := "│ アクティブなタスクはありません - すべて完了しています！                     │"
	if len(doingTasks) > 0 {
		task := doingTasks[0]
		taskLine := fmt.Sprintf("1. %s  %s    %s", tasks.GenerateTaskID(task), strings.ToUpper(task.Priority), task.Description)
		content = fmt.Sprintf("│ %s", padToWidth(taskLine, m.width-taskBoxPadding)) + " │"
	}

	return fmt.Sprintf(`│ Current Task                                                                  │
│ ┌%s┐   │
%s   │
│ └%s┘   │
│                                                                               │`,
		strings.Repeat("─", taskBoxWidth-2),
		content,
		strings.Repeat("─", taskBoxWidth-2))
}

func (m Model) renderNextTasks() string {
	todoTasks := tasks.FilterTasksByStatus(m.tasks, "todo")
	tasks.SortTasksByPriority(todoTasks)

	var taskLines []string
	if len(todoTasks) == 0 {
		taskLines = append(taskLines, "│ │ 待機中のタスクはありません                                               │   │")
	} else {
		maxDisplay := 5
		if len(todoTasks) < maxDisplay {
			maxDisplay = len(todoTasks)
		}

		for i := 0; i < maxDisplay; i++ {
			task := todoTasks[i]
			taskLine := fmt.Sprintf("%d. %s  %s    %s", i+1, tasks.GenerateTaskID(task), strings.ToUpper(task.Priority), task.Description)
			line := fmt.Sprintf("│ │ %s", padToWidth(taskLine, m.width-progressBarPadding)) + " │   │"
			taskLines = append(taskLines, line)
		}
	}

	content := strings.Join(taskLines, "\n")

	return fmt.Sprintf(`│ Next Tasks (up to 5)                                                         │
│ ┌%s┐   │
%s
│ └%s┘   │
│                                                                               │`,
		strings.Repeat("─", taskBoxWidth-2),
		content,
		strings.Repeat("─", taskBoxWidth-2))
}

// Helper functions

// padToWidth pads or truncates a string to fit the specified width
// accounting for multibyte characters
func padToWidth(s string, width int) string {
	currentWidth := runewidth.StringWidth(s)
	if currentWidth > width {
		// Truncate with ellipsis
		truncated := truncateString(s, width-3)
		return truncated + "..."
	}
	// Pad with spaces
	return s + strings.Repeat(" ", width-currentWidth)
}

// truncateString truncates a string to the specified display width
// accounting for multibyte characters
func truncateString(s string, width int) string {
	var result []rune
	currentWidth := 0

	for _, r := range s {
		rWidth := runewidth.RuneWidth(r)
		if currentWidth+rWidth > width {
			break
		}
		result = append(result, r)
		currentWidth += rWidth
	}

	return string(result)
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
