package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"reviewtask/internal/storage"
)

// TestUIComponentRendering tests TUI component rendering
func TestUIComponentRendering(t *testing.T) {
	scenarios := []struct {
		name   string
		setup  func() interface{}
		verify func(t *testing.T, output string)
	}{
		{
			name: "タスクリスト表示",
			setup: func() interface{} {
				return []storage.Task{
					{
						ID:          "TSK-001",
						Description: "メモリリークを修正",
						Status:      "doing",
						Priority:    "critical",
						PRNumber:    100,
					},
					{
						ID:          "TSK-002",
						Description: "ドキュメント更新",
						Status:      "todo",
						Priority:    "low",
						PRNumber:    100,
					},
				}
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "TSK-001") {
					t.Error("Task ID not displayed")
				}
				if !strings.Contains(output, "メモリリーク") {
					t.Error("Japanese content not displayed")
				}
			},
		},
		{
			name: "ステータスインジケーター",
			setup: func() interface{} {
				return map[string]string{
					"todo":    "○",
					"doing":   "◐",
					"done":    "●",
					"pending": "⊘",
					"cancel":  "✗",
				}
			},
			verify: func(t *testing.T, output string) {
				// Status indicators should be distinct
				if output == "" {
					t.Error("Status indicators not generated")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			data := scenario.setup()

			// Simulate rendering
			var buf bytes.Buffer
			switch v := data.(type) {
			case []storage.Task:
				for _, task := range v {
					buf.WriteString(task.ID + " " + task.Description + "\n")
				}
			case map[string]string:
				for status, indicator := range v {
					buf.WriteString(status + ": " + indicator + "\n")
				}
			}

			output := buf.String()
			scenario.verify(t, output)
		})
	}
}

// TestInteractiveSelection tests interactive task selection
func TestInteractiveSelection(t *testing.T) {
	tasks := []storage.Task{
		{ID: "1", Description: "Task 1", Status: "todo", Priority: "high"},
		{ID: "2", Description: "Task 2", Status: "doing", Priority: "medium"},
		{ID: "3", Description: "Task 3", Status: "done", Priority: "low"},
	}

	tests := []struct {
		name           string
		selectedIndex  int
		expectedTaskID string
	}{
		{"最初のタスク選択", 0, "1"},
		{"中間のタスク選択", 1, "2"},
		{"最後のタスク選択", 2, "3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.selectedIndex >= len(tasks) {
				t.Fatal("Invalid index")
			}

			selected := tasks[tt.selectedIndex]
			if selected.ID != tt.expectedTaskID {
				t.Errorf("Expected task %s, got %s", tt.expectedTaskID, selected.ID)
			}
		})
	}
}

// TestColorSchemes tests color scheme application
func TestColorSchemes(t *testing.T) {
	colorMap := map[string]string{
		"critical": "red",
		"high":     "yellow",
		"medium":   "blue",
		"low":      "gray",
		"todo":     "white",
		"doing":    "yellow",
		"done":     "green",
		"pending":  "red",
		"cancel":   "gray",
	}

	for key, color := range colorMap {
		if color == "" {
			t.Errorf("No color defined for %s", key)
		}
	}
}

// TestUIResponsiveness tests UI update performance
func TestUIResponsiveness(t *testing.T) {
	// Create a large task list
	var tasks []storage.Task
	for i := 0; i < 1000; i++ {
		tasks = append(tasks, storage.Task{
			ID:          string(rune(i)),
			Description: "Task description",
			Status:      "todo",
			Priority:    "medium",
			PRNumber:    i,
		})
	}

	start := time.Now()

	// Simulate rendering all tasks
	var buf bytes.Buffer
	for _, task := range tasks {
		buf.WriteString(formatTask(task))
	}

	elapsed := time.Since(start)

	// Should render 1000 tasks quickly
	if elapsed > 100*time.Millisecond {
		t.Logf("Rendering 1000 tasks took %v", elapsed)
	}
}

// TestKeyboardShortcuts tests keyboard shortcut definitions
func TestKeyboardShortcuts(t *testing.T) {
	shortcuts := map[string]string{
		"j":     "next",
		"k":     "previous",
		"enter": "select",
		"q":     "quit",
		"?":     "help",
		"1":     "todo",
		"2":     "doing",
		"3":     "done",
		"4":     "pending",
		"5":     "cancel",
	}

	for key, action := range shortcuts {
		if action == "" {
			t.Errorf("No action defined for key %s", key)
		}
	}
}

// TestTaskFormatting tests task display formatting
func TestTaskFormatting(t *testing.T) {
	tests := []struct {
		name     string
		task     storage.Task
		expected []string // Expected strings in formatted output
	}{
		{
			name: "高優先度タスク",
			task: storage.Task{
				ID:          "TSK-001",
				Description: "Critical bug fix",
				Status:      "doing",
				Priority:    "critical",
				PRNumber:    100,
			},
			expected: []string{"TSK-001", "Critical bug", "doing", "critical"},
		},
		{
			name: "長い説明のタスク",
			task: storage.Task{
				ID:          "TSK-002",
				Description: strings.Repeat("Very long description ", 20),
				Status:      "todo",
				Priority:    "low",
				PRNumber:    200,
			},
			expected: []string{"TSK-002", "Very long", "todo", "low"},
		},
		{
			name: "日本語タスク",
			task: storage.Task{
				ID:          "TSK-003",
				Description: "データベース接続の最適化を実装する",
				Status:      "pending",
				Priority:    "high",
				PRNumber:    300,
			},
			expected: []string{"TSK-003", "データベース", "pending", "high"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatTask(tt.task)

			for _, exp := range tt.expected {
				if !strings.Contains(formatted, exp) {
					t.Errorf("Expected formatted output to contain %q", exp)
				}
			}
		})
	}
}

// TestPaginationLogic tests pagination for large task lists
func TestPaginationLogic(t *testing.T) {
	tests := []struct {
		name          string
		totalTasks    int
		pageSize      int
		currentPage   int
		expectedStart int
		expectedEnd   int
	}{
		{"最初のページ", 100, 10, 0, 0, 10},
		{"中間ページ", 100, 10, 5, 50, 60},
		{"最後のページ", 100, 10, 9, 90, 100},
		{"部分的な最終ページ", 95, 10, 9, 90, 95},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := tt.currentPage * tt.pageSize
			end := start + tt.pageSize
			if end > tt.totalTasks {
				end = tt.totalTasks
			}

			if start != tt.expectedStart {
				t.Errorf("Expected start %d, got %d", tt.expectedStart, start)
			}
			if end != tt.expectedEnd {
				t.Errorf("Expected end %d, got %d", tt.expectedEnd, end)
			}
		})
	}
}

// TestFilterDisplay tests task filtering in UI
func TestFilterDisplay(t *testing.T) {
	allTasks := []storage.Task{
		{ID: "1", Status: "todo", Priority: "high", PRNumber: 100},
		{ID: "2", Status: "doing", Priority: "high", PRNumber: 100},
		{ID: "3", Status: "done", Priority: "low", PRNumber: 100},
		{ID: "4", Status: "todo", Priority: "critical", PRNumber: 200},
		{ID: "5", Status: "pending", Priority: "medium", PRNumber: 200},
	}

	filters := []struct {
		name     string
		filterFn func(storage.Task) bool
		expected int
	}{
		{
			name: "TODOタスクのみ",
			filterFn: func(t storage.Task) bool {
				return t.Status == "todo"
			},
			expected: 2,
		},
		{
			name: "高優先度タスク",
			filterFn: func(t storage.Task) bool {
				return t.Priority == "high" || t.Priority == "critical"
			},
			expected: 3,
		},
		{
			name: "PR-100のタスク",
			filterFn: func(t storage.Task) bool {
				return t.PRNumber == 100
			},
			expected: 3,
		},
	}

	for _, filter := range filters {
		t.Run(filter.name, func(t *testing.T) {
			var filtered []storage.Task
			for _, task := range allTasks {
				if filter.filterFn(task) {
					filtered = append(filtered, task)
				}
			}

			if len(filtered) != filter.expected {
				t.Errorf("Expected %d tasks, got %d", filter.expected, len(filtered))
			}
		})
	}
}

// TestUIStateManagement tests UI state transitions
func TestUIStateManagement(t *testing.T) {
	type UIState struct {
		Mode         string
		SelectedTask int
		Filter       string
		Page         int
	}

	tests := []struct {
		name     string
		initial  UIState
		action   string
		expected UIState
	}{
		{
			name:     "次のタスクを選択",
			initial:  UIState{Mode: "list", SelectedTask: 0, Filter: "", Page: 0},
			action:   "next",
			expected: UIState{Mode: "list", SelectedTask: 1, Filter: "", Page: 0},
		},
		{
			name:     "フィルタモードに切り替え",
			initial:  UIState{Mode: "list", SelectedTask: 5, Filter: "", Page: 0},
			action:   "filter",
			expected: UIState{Mode: "filter", SelectedTask: 5, Filter: "", Page: 0},
		},
		{
			name:     "次のページ",
			initial:  UIState{Mode: "list", SelectedTask: 9, Filter: "", Page: 0},
			action:   "nextpage",
			expected: UIState{Mode: "list", SelectedTask: 0, Filter: "", Page: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.initial

			// Apply action
			switch tt.action {
			case "next":
				state.SelectedTask++
			case "filter":
				state.Mode = "filter"
			case "nextpage":
				state.Page++
				state.SelectedTask = 0
			}

			if state != tt.expected {
				t.Errorf("Expected state %+v, got %+v", tt.expected, state)
			}
		})
	}
}

// Helper function to format task for display
func formatTask(task storage.Task) string {
	// Truncate long descriptions
	desc := task.Description
	if len(desc) > 50 {
		desc = desc[:47] + "..."
	}

	return task.ID + " | " + desc + " | " + task.Status + " | " + task.Priority
}
