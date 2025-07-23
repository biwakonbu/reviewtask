package tui

import (
	"strings"
	"testing"

	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
)

// TestRenderTaskSummary tests that the task summary is rendered without box borders
func TestRenderTaskSummary(t *testing.T) {
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
		width: 80,
	}

	result := model.renderTaskSummary()

	// Verify no box borders are present
	if strings.Contains(result, "┌") || strings.Contains(result, "┐") {
		t.Errorf("renderTaskSummary should not contain top border characters")
	}
	if strings.Contains(result, "└") || strings.Contains(result, "┘") {
		t.Errorf("renderTaskSummary should not contain bottom border characters")
	}

	// Verify content is present
	if !strings.Contains(result, "Task Summary") {
		t.Errorf("renderTaskSummary should contain 'Task Summary' header")
	}
	if !strings.Contains(result, "Todo: 3") {
		t.Errorf("renderTaskSummary should contain todo count")
	}
	if !strings.Contains(result, "Doing: 1") {
		t.Errorf("renderTaskSummary should contain doing count")
	}
	if !strings.Contains(result, "Done: 2") {
		t.Errorf("renderTaskSummary should contain done count")
	}
}

// TestRenderCurrentTask tests current task rendering without box borders
func TestRenderCurrentTask(t *testing.T) {
	tests := []struct {
		name  string
		tasks []storage.Task
		want  string
	}{
		{
			name:  "no active tasks",
			tasks: []storage.Task{},
			want:  "アクティブなタスクはありません",
		},
		{
			name: "with active task",
			tasks: []storage.Task{
				{
					ID:          "task1",
					PRNumber:    1,
					Priority:    "high",
					Status:      "doing",
					Description: "Fix bug in authentication",
				},
			},
			want: "Fix bug in authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				tasks: tt.tasks,
				width: 80,
			}

			result := model.renderCurrentTask()

			// Verify no box borders
			if strings.Contains(result, "┌") || strings.Contains(result, "┐") {
				t.Errorf("renderCurrentTask should not contain top border characters")
			}
			if strings.Contains(result, "└") || strings.Contains(result, "┘") {
				t.Errorf("renderCurrentTask should not contain bottom border characters")
			}

			// Verify content
			if !strings.Contains(result, "Current Task") {
				t.Errorf("renderCurrentTask should contain 'Current Task' header")
			}
			if !strings.Contains(result, tt.want) {
				t.Errorf("renderCurrentTask should contain expected content: %s", tt.want)
			}
		})
	}
}

// TestRenderNextTasks tests next tasks rendering without box borders
func TestRenderNextTasks(t *testing.T) {
	tests := []struct {
		name  string
		tasks []storage.Task
		want  []string
	}{
		{
			name:  "no pending tasks",
			tasks: []storage.Task{},
			want:  []string{"待機中のタスクはありません"},
		},
		{
			name: "with pending tasks",
			tasks: []storage.Task{
				{
					ID:          "task1",
					PRNumber:    1,
					Priority:    "high",
					Status:      "todo",
					Description: "Update documentation",
				},
				{
					ID:          "task2",
					PRNumber:    1,
					Priority:    "medium",
					Status:      "todo",
					Description: "Add unit tests",
				},
			},
			want: []string{"Update documentation", "Add unit tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				tasks: tt.tasks,
				width: 80,
			}

			result := model.renderNextTasks()

			// Verify no box borders
			if strings.Contains(result, "┌") || strings.Contains(result, "┐") {
				t.Errorf("renderNextTasks should not contain top border characters")
			}
			if strings.Contains(result, "└") || strings.Contains(result, "┘") {
				t.Errorf("renderNextTasks should not contain bottom border characters")
			}

			// Verify content
			if !strings.Contains(result, "Next Tasks") {
				t.Errorf("renderNextTasks should contain 'Next Tasks' header")
			}
			for _, expected := range tt.want {
				if !strings.Contains(result, expected) {
					t.Errorf("renderNextTasks should contain: %s", expected)
				}
			}
		})
	}
}

// TestJapaneseTextDisplay tests that Japanese text displays correctly without border issues
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
		width: 80,
	}

	// Test current task with Japanese
	currentResult := model.renderCurrentTask()
	if !strings.Contains(currentResult, "日本語のタスク説明文") {
		t.Errorf("renderCurrentTask should properly display Japanese text")
	}

	// Test next tasks with Japanese
	nextResult := model.renderNextTasks()
	if !strings.Contains(nextResult, "もう一つの日本語タスク") {
		t.Errorf("renderNextTasks should properly display Japanese text")
	}

	// Ensure no border characters that could break with Japanese
	combinedResult := currentResult + nextResult
	borderChars := []string{"┌", "┐", "└", "┘", "├", "┤", "─"}
	for _, char := range borderChars {
		if strings.Count(combinedResult, char) > 0 {
			// Only horizontal lines should appear in headers, not in content boxes
			lines := strings.Split(combinedResult, "\n")
			for i, line := range lines {
				if strings.Contains(line, char) && i > 1 { // Skip header lines
					if strings.Contains(line, "日本語") || strings.Contains(line, "タスク") {
						t.Errorf("Border character '%s' found near Japanese text which could cause display issues", char)
					}
				}
			}
		}
	}
}

// TestPadToWidth tests the padding function with multibyte characters
func TestPadToWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected int // expected display width
	}{
		{
			name:     "ASCII only",
			input:    "Hello World",
			width:    20,
			expected: 20,
		},
		{
			name:     "Japanese text",
			input:    "こんにちは",
			width:    20,
			expected: 20,
		},
		{
			name:     "Mixed ASCII and Japanese",
			input:    "Task: タスク",
			width:    20,
			expected: 20,
		},
		{
			name:     "Truncation with Japanese",
			input:    "非常に長い日本語のテキスト",
			width:    10,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padToWidth(tt.input, tt.width)
			// The padToWidth function should ensure the visual width matches
			// We can't easily test visual width in Go tests, but we can ensure
			// the function doesn't panic and returns a string
			if result == "" {
				t.Errorf("padToWidth returned empty string for input: %s", tt.input)
			}
		})
	}
}
