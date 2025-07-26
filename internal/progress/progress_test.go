package progress

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := New()

	// Test initial state
	assert.NotNil(t, m.stages)
	assert.NotNil(t, m.progressBars)
	assert.Len(t, m.stages, 3)
	assert.Len(t, m.progressBars, 3)
	assert.Equal(t, []string{"github", "analysis", "saving"}, m.stageOrder)

	// Check initial stages
	assert.Equal(t, "GitHub API", m.stages["github"].Name)
	assert.Equal(t, "pending", m.stages["github"].Status)
	assert.Equal(t, "AI Analysis", m.stages["analysis"].Name)
	assert.Equal(t, "pending", m.stages["analysis"].Status)
	assert.Equal(t, "Saving Data", m.stages["saving"].Name)
	assert.Equal(t, "pending", m.stages["saving"].Status)
}

func TestUpdateProgress(t *testing.T) {
	m := New()

	// Test progress update
	msg := progressMsg{
		stage:      "github",
		current:    5,
		total:      10,
		percentage: 0.5,
	}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	assert.Equal(t, 5, m.stages["github"].Current)
	assert.Equal(t, 10, m.stages["github"].Total)
	assert.Equal(t, 0.5, m.stages["github"].Percentage)
	assert.Equal(t, "in_progress", m.stages["github"].Status)
	assert.Equal(t, "github", m.activeStage)

	// Test completion
	msg = progressMsg{
		stage:      "github",
		current:    10,
		total:      10,
		percentage: 1.0,
	}

	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	assert.Equal(t, "completed", m.stages["github"].Status)
}

func TestUpdateStatus(t *testing.T) {
	m := New()

	// Test status update
	msg := statusMsg{
		stage:  "analysis",
		status: "error",
	}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	assert.Equal(t, "error", m.stages["analysis"].Status)
}

func TestUpdateStats(t *testing.T) {
	m := New()

	// Test stats update
	stats := Statistics{
		CommentsProcessed: 5,
		TotalComments:     10,
		TasksGenerated:    3,
		CurrentOperation:  "Processing review comment",
	}

	msg := statsMsg{stats: stats}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	assert.Equal(t, 5, m.stats.CommentsProcessed)
	assert.Equal(t, 10, m.stats.TotalComments)
	assert.Equal(t, 3, m.stats.TasksGenerated)
	assert.Equal(t, "Processing review comment", m.stats.CurrentOperation)
}

func TestWindowResize(t *testing.T) {
	m := New()

	// Test window resize
	msg := tea.WindowSizeMsg{
		Width:  120,
		Height: 30,
	}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 30, m.height)
}

func TestGetStatusIcon(t *testing.T) {
	m := New()

	tests := []struct {
		status string
		want   string
	}{
		{"completed", "✓"},
		{"in_progress", "●"},
		{"error", "✗"},
		{"pending", "○"},
		{"unknown", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := m.getStatusIcon(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m0s"},
		{90 * time.Second, "1m30s"},
		{125 * time.Second, "2m5s"},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			got := formatDuration(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNonTTYView(t *testing.T) {
	m := New()
	m.isTTY = false

	// Set some progress
	m.stages["github"].Status = "in_progress"
	m.stages["github"].Current = 1
	m.stages["github"].Total = 2
	m.stages["github"].Percentage = 0.5

	m.stats.CurrentOperation = "Fetching data"

	view := m.View()

	assert.Contains(t, view, "GitHub API: 1/2 (50%)")
	assert.Contains(t, view, "Current: Fetching data")
}

func TestView(t *testing.T) {
	m := New()
	m.isTTY = true

	// Set some progress
	m.stages["github"].Status = "completed"
	m.stages["github"].Current = 2
	m.stages["github"].Total = 2
	m.stages["github"].Percentage = 1.0

	m.stages["analysis"].Status = "in_progress"
	m.stages["analysis"].Current = 5
	m.stages["analysis"].Total = 10
	m.stages["analysis"].Percentage = 0.5

	m.stats.CommentsProcessed = 5
	m.stats.TotalComments = 10
	m.stats.TasksGenerated = 3
	m.stats.CurrentOperation = "Processing comment from @reviewer"
	m.stats.ElapsedTime = 45 * time.Second

	view := m.View()

	// Check that view contains expected elements
	assert.Contains(t, view, "Fetching PR Review Data...")
	assert.Contains(t, view, "GitHub API")
	assert.Contains(t, view, "AI Analysis")
	assert.Contains(t, view, "Saving Data")
	assert.Contains(t, view, "Processing comment from @reviewer")
	assert.Contains(t, view, "45s")
	assert.Contains(t, view, "Comments: 5/10")
	assert.Contains(t, view, "Tasks: 3")
}

func TestCommandFactories(t *testing.T) {
	t.Run("UpdateProgress", func(t *testing.T) {
		cmd := UpdateProgress("github", 5, 10)
		msg := cmd()

		progressMsg, ok := msg.(progressMsg)
		assert.True(t, ok)
		assert.Equal(t, "github", progressMsg.stage)
		assert.Equal(t, 5, progressMsg.current)
		assert.Equal(t, 10, progressMsg.total)
		assert.Equal(t, 0.5, progressMsg.percentage)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		cmd := UpdateStatus("analysis", "error")
		msg := cmd()

		statusMsg, ok := msg.(statusMsg)
		assert.True(t, ok)
		assert.Equal(t, "analysis", statusMsg.stage)
		assert.Equal(t, "error", statusMsg.status)
	})

	t.Run("UpdateStats", func(t *testing.T) {
		stats := Statistics{
			CommentsProcessed: 10,
			TotalComments:     20,
			TasksGenerated:    5,
			CurrentOperation:  "Test operation",
		}

		cmd := UpdateStats(stats)
		msg := cmd()

		statsMsg, ok := msg.(statsMsg)
		assert.True(t, ok)
		assert.Equal(t, stats, statsMsg.stats)
	})
}
