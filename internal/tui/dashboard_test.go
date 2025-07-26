package tui

import (
	"strings"
	"testing"
	"time"

	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
)

// TestDashboardView tests that the dashboard is rendered in simple format
func TestDashboardView(t *testing.T) {
	model := Model{
		stats: tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":    3,
				"doing":   1,
				"done":    2,
				"pending": 1,
				"cancel":  0,
			},
		},
		tasks: []storage.Task{
			{Status: "done"},
			{Status: "done"},
			{Status: "doing"},
			{Status: "todo"},
			{Status: "todo"},
			{Status: "todo"},
			{Status: "pending"},
		},
		width:      80,
		lastUpdate: time.Now(),
	}

	result := model.View()

	// Verify no box borders are present
	if strings.Contains(result, "┌") || strings.Contains(result, "┐") ||
		strings.Contains(result, "└") || strings.Contains(result, "┘") ||
		strings.Contains(result, "│") {
		t.Errorf("Dashboard should not contain any border characters")
	}

	// Verify content is present
	if !strings.Contains(result, "ReviewTask Status") {
		t.Errorf("Dashboard should contain 'ReviewTask Status' header")
	}
	if !strings.Contains(result, "Task Summary:") {
		t.Errorf("Dashboard should contain 'Task Summary:' header")
	}
	if !strings.Contains(result, "todo: 3") {
		t.Errorf("Dashboard should contain todo count")
	}
	if !strings.Contains(result, "doing: 1") {
		t.Errorf("Dashboard should contain doing count")
	}
	if !strings.Contains(result, "done: 2") {
		t.Errorf("Dashboard should contain done count")
	}
	if !strings.Contains(result, "Current Task:") {
		t.Errorf("Dashboard should contain 'Current Task:' header")
	}
	if !strings.Contains(result, "Next Tasks") {
		t.Errorf("Dashboard should contain 'Next Tasks' header")
	}
	if !strings.Contains(result, "Last updated:") {
		t.Errorf("Dashboard should contain last updated timestamp")
	}
}

// TestDashboardViewEmpty tests empty state
func TestDashboardViewEmpty(t *testing.T) {
	model := Model{
		stats: tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":    0,
				"doing":   0,
				"done":    0,
				"pending": 0,
				"cancel":  0,
			},
		},
		tasks:      []storage.Task{},
		width:      80,
		lastUpdate: time.Now(),
	}

	result := model.View()

	// Verify no box borders
	if strings.Contains(result, "┌") || strings.Contains(result, "┐") ||
		strings.Contains(result, "└") || strings.Contains(result, "┘") ||
		strings.Contains(result, "│") {
		t.Errorf("Dashboard should not contain any border characters")
	}

	// Verify empty state messages
	if !strings.Contains(result, "ReviewTask Status - 0% Complete") {
		t.Error("Empty dashboard should show 0% complete")
	}
	if !strings.Contains(result, "アクティブなタスクはありません") {
		t.Errorf("Empty dashboard should show no active tasks message")
	}
	if !strings.Contains(result, "待機中のタスクはありません") {
		t.Errorf("Empty dashboard should show no pending tasks message")
	}
}

// TestJapaneseTextDisplay tests that Japanese text displays correctly
func TestJapaneseTextDisplay(t *testing.T) {
	model := Model{
		tasks: []storage.Task{
			{
				ID:          "task1",
				PRNumber:    1,
				Priority:    "high",
				Status:      "doing",
				Description: "日本語のタスク説明文",
			},
			{
				ID:          "task2",
				PRNumber:    1,
				Priority:    "medium",
				Status:      "todo",
				Description: "もう一つの日本語タスク",
			},
		},
		stats: tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":    1,
				"doing":   1,
				"done":    0,
				"pending": 0,
				"cancel":  0,
			},
		},
		width:      80,
		lastUpdate: time.Now(),
	}

	result := model.View()

	// Verify Japanese text is present
	if !strings.Contains(result, "日本語のタスク説明文") {
		t.Errorf("Dashboard should properly display Japanese text in current task")
	}
	if !strings.Contains(result, "もう一つの日本語タスク") {
		t.Errorf("Dashboard should properly display Japanese text in next tasks")
	}

	// Ensure no border characters
	borderChars := []string{"┌", "┐", "└", "┘", "├", "┤", "─", "│"}
	for _, char := range borderChars {
		if strings.Contains(result, char) {
			t.Errorf("Dashboard should not contain border character '%s'", char)
		}
	}
}

// TestDashboardViewWithError tests error display
func TestDashboardViewWithError(t *testing.T) {
	model := Model{
		stats:      tasks.TaskStats{StatusCounts: map[string]int{}},
		tasks:      []storage.Task{},
		width:      80,
		lastUpdate: time.Now(),
		loadError:  &testError{msg: "Failed to load tasks"},
	}

	result := model.View()

	// Verify error message is displayed
	if !strings.Contains(result, "Error loading tasks") {
		t.Errorf("Dashboard should display error message")
	}
	if !strings.Contains(result, "Failed to load tasks") {
		t.Errorf("Dashboard should display specific error details")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestGenerateColoredProgressBarTUI tests the TUI version of colored progress bar generation
func TestGenerateColoredProgressBarTUI(t *testing.T) {
	testCases := []struct {
		name             string
		stats            tasks.TaskStats
		width            int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Empty stats",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{},
			},
			width:            10,
			shouldContain:    []string{"░"},
			shouldNotContain: []string{"█"},
		},
		{
			name: "All done tasks",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"done": 5,
				},
			},
			width:            10,
			shouldContain:    []string{"█"},
			shouldNotContain: []string{"░"},
		},
		{
			name: "Mixed states with colors",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"done":    1,
					"doing":   1,
					"todo":    1,
					"pending": 1,
					"cancel":  1,
				},
			},
			width:            10,
			shouldContain:    []string{"█", "░"},
			shouldNotContain: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generateColoredProgressBar(tc.stats, tc.width)

			// Check that result is not empty
			if result == "" {
				t.Error("Expected non-empty progress bar")
			}

			// Check for expected characters
			for _, expected := range tc.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected progress bar to contain '%s', got: %s", expected, result)
				}
			}

			// Check for unexpected characters
			for _, unexpected := range tc.shouldNotContain {
				if strings.Contains(result, unexpected) {
					t.Errorf("Expected progress bar NOT to contain '%s', got: %s", unexpected, result)
				}
			}
		})
	}
}
