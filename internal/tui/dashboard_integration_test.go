package tui

import (
	"strings"
	"testing"
	"time"

	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
)

// TestDashboardFullRender tests the complete dashboard rendering
func TestDashboardFullRender(t *testing.T) {
	// Create a model with test data
	model := Model{
		tasks: []storage.Task{
			{
				ID:          "task1",
				PRNumber:    42,
				Priority:    "high",
				Status:      "doing",
				Description: "現在作業中の日本語タスク",
			},
			{
				ID:          "task2",
				PRNumber:    42,
				Priority:    "medium",
				Status:      "todo",
				Description: "次のタスク with English",
			},
			{
				ID:          "task3",
				PRNumber:    42,
				Priority:    "low",
				Status:      "done",
				Description: "完了したタスク",
			},
		},
		stats: tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":    1,
				"doing":   1,
				"done":    1,
				"pending": 0,
				"cancel":  0,
			},
		},
		width:      80,
		height:     30,
		lastUpdate: time.Now(),
	}

	// Render the full dashboard
	output := model.View()

	// Basic structure tests
	if !strings.Contains(output, "ReviewTask Status Dashboard") {
		t.Error("Dashboard should contain title")
	}

	if !strings.Contains(output, "Progress Overview") {
		t.Error("Dashboard should contain progress section")
	}

	if !strings.Contains(output, "Task Summary") {
		t.Error("Dashboard should contain task summary")
	}

	if !strings.Contains(output, "Current Task") {
		t.Error("Dashboard should contain current task section")
	}

	if !strings.Contains(output, "Next Tasks") {
		t.Error("Dashboard should contain next tasks section")
	}

	// Test Japanese content is displayed
	if !strings.Contains(output, "現在作業中の日本語タスク") {
		t.Error("Dashboard should display Japanese text in current task")
	}

	if !strings.Contains(output, "次のタスク with English") {
		t.Error("Dashboard should display mixed language text")
	}

	// Test that content sections don't have box borders
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Skip header and footer lines
		if i < 5 || i > len(lines)-3 {
			continue
		}

		// Check for box border characters in content areas
		if strings.Contains(line, "│ ┌") || strings.Contains(line, "│ └") {
			if strings.Contains(lines[i-1], "Task Summary") ||
				strings.Contains(lines[i-1], "Current Task") ||
				strings.Contains(lines[i-1], "Next Tasks") {
				t.Errorf("Content section at line %d should not have box borders: %s", i, line)
			}
		}
	}
}

// TestDashboardErrorState tests dashboard rendering when there's an error
func TestDashboardErrorState(t *testing.T) {
	testError := strings.Join([]string{"test", "error"}, " ")

	model := Model{
		width:     80,
		height:    30,
		loadError: &testErrorType{msg: testError},
	}

	output := model.View()

	if !strings.Contains(output, "Error loading tasks") {
		t.Error("Dashboard should display error message")
	}

	if !strings.Contains(output, testError) {
		t.Error("Dashboard should display specific error details")
	}
}

// testErrorType is a simple error type for testing
type testErrorType struct {
	msg string
}

func (e *testErrorType) Error() string {
	return e.msg
}

// TestDashboardEmptyState tests dashboard with no tasks
func TestDashboardEmptyState(t *testing.T) {
	model := Model{
		tasks: []storage.Task{},
		stats: tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":    0,
				"doing":   0,
				"done":    0,
				"pending": 0,
				"cancel":  0,
			},
		},
		width:  80,
		height: 30,
	}

	output := model.View()

	// Check for empty state messages
	if !strings.Contains(output, "0%") {
		t.Error("Empty dashboard should show 0% progress")
	}

	if !strings.Contains(output, "アクティブなタスクはありません") {
		t.Error("Empty dashboard should show no active tasks message")
	}

	if !strings.Contains(output, "待機中のタスクはありません") {
		t.Error("Empty dashboard should show no pending tasks message")
	}
}

// BenchmarkDashboardRender benchmarks the dashboard rendering performance
func BenchmarkDashboardRender(b *testing.B) {
	// Create test data with many tasks
	var testTasks []storage.Task
	for i := 0; i < 50; i++ {
		testTasks = append(testTasks, storage.Task{
			ID:          string(rune('a' + i)),
			PRNumber:    1,
			Priority:    "medium",
			Status:      "todo",
			Description: "テストタスク番号" + string(rune('0'+i)),
		})
	}

	model := Model{
		tasks: testTasks,
		stats: tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":    50,
				"doing":   0,
				"done":    0,
				"pending": 0,
				"cancel":  0,
			},
		},
		width:      80,
		height:     30,
		lastUpdate: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

// TestMultibyteCharacterHandling tests specific multibyte character scenarios
func TestMultibyteCharacterHandling(t *testing.T) {
	testCases := []struct {
		name        string
		description string
		expectError bool
	}{
		{
			name:        "Emoji in description",
			description: "🚀 ロケット打ち上げタスク",
			expectError: false,
		},
		{
			name:        "Full-width characters",
			description: "ＡＢＣ　全角文字テスト",
			expectError: false,
		},
		{
			name:        "Mixed scripts",
			description: "日本語 English 中文 한글",
			expectError: false,
		},
		{
			name:        "Very long Japanese",
			description: "非常に長い日本語の説明文がここに入ります。これはテストのための長い文章です。",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := Model{
				tasks: []storage.Task{
					{
						ID:          "test1",
						PRNumber:    1,
						Priority:    "high",
						Status:      "doing",
						Description: tc.description,
					},
				},
				width: 80,
			}

			// This should not panic
			result := model.renderCurrentTask()

			if tc.expectError && strings.Contains(result, tc.description) {
				t.Errorf("Expected error handling for: %s", tc.description)
			}

			if !tc.expectError && !strings.Contains(result, tc.description) {
				// Check if it was truncated (should have "...")
				if !strings.Contains(result, "...") {
					t.Errorf("Expected description to be displayed or truncated: %s", tc.description)
				}
			}
		})
	}
}
