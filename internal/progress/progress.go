package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"os"
)

// Stage represents a single stage in the fetch process
type Stage struct {
	Name       string
	Current    int
	Total      int
	Status     string
	Percentage float64
}

// Model represents the progress tracking model
type Model struct {
	stages       map[string]*Stage
	stageOrder   []string
	activeStage  string
	startTime    time.Time
	lastUpdate   time.Time
	progressBars map[string]progress.Model
	width        int
	height       int
	isTTY        bool
	stats        Statistics
	errorQueue   []string
	maxErrors    int
	interrupted  bool // Track if user pressed Ctrl-C
}

// Statistics represents real-time statistics
type Statistics struct {
	CommentsProcessed int
	TotalComments     int
	TasksGenerated    int
	CurrentOperation  string
	ElapsedTime       time.Duration
}

// Messages
type progressMsg struct {
	stage      string
	current    int
	total      int
	percentage float64
}

type statusMsg struct {
	stage  string
	status string
}

type statsMsg struct {
	stats Statistics
}

type errorMsg struct {
	message string
}

type tickMsg time.Time

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	stageNameStyle = lipgloss.NewStyle().
			Width(20).
			Foreground(lipgloss.Color("4"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6"))

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5"))
)

// New creates a new progress model
func New() Model {
	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	m := Model{
		stages:       make(map[string]*Stage),
		progressBars: make(map[string]progress.Model),
		stageOrder:   []string{"github", "analysis", "saving"},
		startTime:    time.Now(),
		lastUpdate:   time.Now(),
		isTTY:        isTTY,
		width:        80,
		height:       10,
		errorQueue:   make([]string, 0),
		maxErrors:    5, // Limit displayed errors to prevent overflow
	}

	// Initialize stages
	m.stages["github"] = &Stage{
		Name:   "GitHub API",
		Status: "pending",
	}
	m.stages["analysis"] = &Stage{
		Name:   "AI Analysis",
		Status: "pending",
	}
	m.stages["saving"] = &Stage{
		Name:   "Saving Data",
		Status: "pending",
	}

	// Initialize progress bars with different colors
	m.progressBars["github"] = progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)
	m.progressBars["analysis"] = progress.New(
		progress.WithGradient("#5A56E0", "#EE6FF8"),
		progress.WithWidth(40),
	)
	m.progressBars["saving"] = progress.New(
		progress.WithGradient("#F793A8", "#F7DC6F"),
		progress.WithWidth(40),
	)

	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.progressBars["github"].Init(),
		m.progressBars["analysis"].Init(),
		m.progressBars["saving"].Init(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Mark as user interrupted and quit
			m.interrupted = true
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case progressMsg:
		if stage, ok := m.stages[msg.stage]; ok {
			stage.Current = msg.current
			stage.Total = msg.total
			stage.Percentage = msg.percentage
			if stage.Status == "pending" && msg.current > 0 {
				stage.Status = "in_progress"
			}
			if msg.current >= msg.total && msg.total > 0 {
				stage.Status = "completed"
			}
		}
		m.activeStage = msg.stage

		// Update progress bar
		if bar, ok := m.progressBars[msg.stage]; ok {
			cmd := bar.SetPercent(msg.percentage)
			newModel, _ := bar.Update(msg)
			if progressBar, ok := newModel.(progress.Model); ok {
				m.progressBars[msg.stage] = progressBar
			}
			return m, cmd
		}
		return m, nil

	case statusMsg:
		if stage, ok := m.stages[msg.stage]; ok {
			stage.Status = msg.status
		}
		return m, nil

	case statsMsg:
		m.stats = msg.stats
		m.stats.ElapsedTime = time.Since(m.startTime)
		return m, nil

	case errorMsg:
		// Add error to queue, keeping only the most recent ones
		m.errorQueue = append(m.errorQueue, msg.message)
		if len(m.errorQueue) > m.maxErrors {
			m.errorQueue = m.errorQueue[1:]
		}
		return m, nil

	case tickMsg:
		m.stats.ElapsedTime = time.Since(m.startTime)
		return m, tickCmd()

	case progress.FrameMsg:
		var cmds []tea.Cmd
		for key, bar := range m.progressBars {
			newModel, cmd := bar.Update(msg)
			if progressBar, ok := newModel.(progress.Model); ok {
				m.progressBars[key] = progressBar
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// View renders the progress display
func (m Model) View() string {
	if !m.isTTY {
		// Non-TTY fallback
		return m.nonTTYView()
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Fetching PR Review Data...") + "\n")

	// Progress bars
	for _, stageKey := range m.stageOrder {
		stage := m.stages[stageKey]
		bar := m.progressBars[stageKey]

		// Stage name with status icon
		icon := m.getStatusIcon(stage.Status)
		name := fmt.Sprintf("%s %s", icon, stage.Name)
		b.WriteString(stageNameStyle.Render(name))

		// Progress bar
		b.WriteString(bar.View())

		// Percentage and count
		if stage.Total > 0 {
			info := fmt.Sprintf(" %3.0f%% (%d/%d)", stage.Percentage*100, stage.Current, stage.Total)
			b.WriteString(statusStyle.Render(info))
		} else {
			b.WriteString(statusStyle.Render(" pending"))
		}

		b.WriteString("\n")
	}

	// Current operation with spinner
	if m.stats.CurrentOperation != "" {
		spinner := m.getSpinner()
		b.WriteString("\n" + spinnerStyle.Render(spinner) + " " + m.stats.CurrentOperation + "\n")
	}

	// Statistics
	b.WriteString("\n" + m.renderStats())

	// Queued errors (displayed at the end to avoid interference)
	if len(m.errorQueue) > 0 {
		b.WriteString("\n\n")
		for _, errMsg := range m.errorQueue {
			b.WriteString(statusStyle.Render("⚠️  "+errMsg) + "\n")
		}
	}

	return b.String()
}

// nonTTYView provides a simple view for non-TTY environments
func (m Model) nonTTYView() string {
	var b strings.Builder

	for _, stageKey := range m.stageOrder {
		stage := m.stages[stageKey]
		if stage.Status == "in_progress" {
			b.WriteString(fmt.Sprintf("%s: %d/%d (%.0f%%)\n",
				stage.Name, stage.Current, stage.Total, stage.Percentage*100))
		}
	}

	if m.stats.CurrentOperation != "" {
		b.WriteString(fmt.Sprintf("Current: %s\n", m.stats.CurrentOperation))
	}

	return b.String()
}

// getStatusIcon returns an icon based on the stage status
func (m Model) getStatusIcon(status string) string {
	switch status {
	case "completed":
		return "✓"
	case "in_progress":
		return "●"
	case "error":
		return "✗"
	default:
		return "○"
	}
}

// getSpinner returns a spinner character
func (m Model) getSpinner() string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	elapsed := time.Since(m.startTime).Milliseconds()
	return spinners[(elapsed/100)%int64(len(spinners))]
}

// renderStats renders the statistics section
func (m Model) renderStats() string {
	var parts []string

	if m.stats.ElapsedTime > 0 {
		parts = append(parts, fmt.Sprintf("⏱  %s", formatDuration(m.stats.ElapsedTime)))
	}

	if m.stats.TotalComments > 0 {
		parts = append(parts, fmt.Sprintf("💬 Comments: %d/%d",
			m.stats.CommentsProcessed, m.stats.TotalComments))
	}

	if m.stats.TasksGenerated > 0 {
		parts = append(parts, fmt.Sprintf("✅ Tasks: %d", m.stats.TasksGenerated))
	}

	return statsStyle.Render(strings.Join(parts, " | "))
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// tickCmd returns a command that sends a tick message
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// UpdateProgress updates the progress for a specific stage
func UpdateProgress(stage string, current, total int) tea.Cmd {
	percentage := 0.0
	if total > 0 {
		percentage = float64(current) / float64(total)
	}
	return func() tea.Msg {
		return progressMsg{
			stage:      stage,
			current:    current,
			total:      total,
			percentage: percentage,
		}
	}
}

// UpdateStatus updates the status for a specific stage
func UpdateStatus(stage, status string) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{
			stage:  stage,
			status: status,
		}
	}
}

// UpdateStats updates the statistics
func UpdateStats(stats Statistics) tea.Cmd {
	return func() tea.Msg {
		return statsMsg{stats: stats}
	}
}

// AddError adds an error message to the progress display queue
func AddError(message string) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{message: message}
	}
}
