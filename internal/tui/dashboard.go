package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"reviewtask/internal/storage"
)

// Model represents the TUI dashboard state
type Model struct {
	storageManager *storage.Manager
	tasks          []storage.Task
	stats          TaskStats
	width          int
	height         int
	lastUpdate     time.Time
	showAll        bool
	specificPR     int
	branch         string
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
		m.tasks = msg.tasks
		m.stats = msg.stats

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
	header := borderStyle.Width(m.width - 2).Render(
		titleStyle.Render("┌─ ReviewTask Status Dashboard " + strings.Repeat("─", m.width-34) + "┐"),
	)
	sections = append(sections, header)

	// Progress Overview
	sections = append(sections, m.renderProgressSection())

	// Task Summary
	sections = append(sections, m.renderTaskSummary())

	// Current Task
	sections = append(sections, m.renderCurrentTask())

	// Next Tasks
	sections = append(sections, m.renderNextTasks())

	// Footer
	footer := fmt.Sprintf("│ Press Ctrl+C to exit                                    Last updated: %s │",
		m.lastUpdate.Format("15:04"))
	sections = append(sections, footer)

	// Bottom border
	sections = append(sections, "└"+strings.Repeat("─", m.width-2)+"┘")

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
	barWidth := m.width - 10
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
	barWidth := m.width - 10
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
│ ┌─────────────────────────────────────────────────────────────────────────┐   │
│ │%s│   │
│ └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                               │`, summary)
}

func (m Model) renderCurrentTask() string {
	doingTasks := filterTasksByStatus(m.tasks, "doing")

	content := "│ アクティブなタスクはありません - すべて完了しています！                     │"
	if len(doingTasks) > 0 {
		task := doingTasks[0]
		taskLine := fmt.Sprintf("1. %s  %s    %s", generateTaskID(task), strings.ToUpper(task.Priority), task.Description)
		content = fmt.Sprintf("│ %s", padToWidth(taskLine, m.width-6)) + " │"
	}

	return fmt.Sprintf(`│ Current Task                                                                  │
│ ┌─────────────────────────────────────────────────────────────────────────┐   │
%s   │
│ └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                               │`, content)
}

func (m Model) renderNextTasks() string {
	todoTasks := filterTasksByStatus(m.tasks, "todo")
	sortTasksByPriority(todoTasks)

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
			taskLine := fmt.Sprintf("%d. %s  %s    %s", i+1, generateTaskID(task), strings.ToUpper(task.Priority), task.Description)
			line := fmt.Sprintf("│ │ %s", padToWidth(taskLine, m.width-10)) + " │   │"
			taskLines = append(taskLines, line)
		}
	}

	content := strings.Join(taskLines, "\n")

	return fmt.Sprintf(`│ Next Tasks (up to 5)                                                         │
│ ┌─────────────────────────────────────────────────────────────────────────┐   │
%s
│ └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                               │`, content)
}

// Helper functions

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
	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	for i := 0; i < len(tasks)-1; i++ {
		for j := 0; j < len(tasks)-i-1; j++ {
			if priorityOrder[tasks[j].Priority] > priorityOrder[tasks[j+1].Priority] {
				tasks[j], tasks[j+1] = tasks[j+1], tasks[j]
			}
		}
	}
}

func generateTaskID(task storage.Task) string {
	return fmt.Sprintf("TSK-%03d", task.PRNumber)
}

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
	stats TaskStats
}

type tickMsg time.Time

// Commands

func (m Model) loadTasks() tea.Msg {
	// Load tasks based on flags (same logic as AI mode)
	var allTasks []storage.Task
	var err error

	if m.specificPR > 0 {
		allTasks, err = m.storageManager.GetTasksByPR(m.specificPR)
	} else if m.branch != "" {
		prNumbers, err := m.storageManager.GetPRsForBranch(m.branch)
		if err == nil {
			for _, prNumber := range prNumbers {
				tasks, err := m.storageManager.GetTasksByPR(prNumber)
				if err == nil {
					allTasks = append(allTasks, tasks...)
				}
			}
		}
	} else if m.showAll {
		allTasks, err = m.storageManager.GetAllTasks()
	} else {
		currentBranch, err := m.storageManager.GetCurrentBranch()
		if err == nil {
			prNumbers, err := m.storageManager.GetPRsForBranch(currentBranch)
			if err == nil {
				for _, prNumber := range prNumbers {
					tasks, err := m.storageManager.GetTasksByPR(prNumber)
					if err == nil {
						allTasks = append(allTasks, tasks...)
					}
				}
			}
		}
	}

	if err != nil {
		allTasks = []storage.Task{}
	}

	stats := calculateTaskStats(allTasks)

	return tasksLoadedMsg{
		tasks: allTasks,
		stats: stats,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// TaskStats represents statistics about tasks
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
