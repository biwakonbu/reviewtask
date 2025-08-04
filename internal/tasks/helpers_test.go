package tasks

import (
	"reflect"
	"testing"

	"reviewtask/internal/storage"
)

// TestFilterTasksByStatus tests task filtering by status
func TestFilterTasksByStatus(t *testing.T) {
	tasks := []storage.Task{
		{ID: "1", Status: "todo", Priority: "high", PRNumber: 100},
		{ID: "2", Status: "doing", Priority: "medium", PRNumber: 100},
		{ID: "3", Status: "done", Priority: "low", PRNumber: 100},
		{ID: "4", Status: "todo", Priority: "critical", PRNumber: 101},
		{ID: "5", Status: "pending", Priority: "high", PRNumber: 101},
	}

	tests := []struct {
		name         string
		status       string
		expectedIDs  []string
	}{
		{
			name:        "todoタスクのフィルタリング",
			status:      "todo",
			expectedIDs: []string{"1", "4"},
		},
		{
			name:        "doingタスクのフィルタリング",
			status:      "doing",
			expectedIDs: []string{"2"},
		},
		{
			name:        "doneタスクのフィルタリング",
			status:      "done",
			expectedIDs: []string{"3"},
		},
		{
			name:        "存在しないステータス",
			status:      "invalid",
			expectedIDs: []string{},
		},
		{
			name:        "空文字列でのフィルタリング",
			status:      "",
			expectedIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterTasksByStatus(tasks, tt.status)
			
			if len(filtered) != len(tt.expectedIDs) {
				t.Errorf("Expected %d tasks, got %d", len(tt.expectedIDs), len(filtered))
			}

			for i, task := range filtered {
				if i < len(tt.expectedIDs) && task.ID != tt.expectedIDs[i] {
					t.Errorf("Expected task ID %s at position %d, got %s", tt.expectedIDs[i], i, task.ID)
				}
			}
		})
	}
}

// TestSortTasksByPriority tests priority-based sorting
func TestSortTasksByPriority(t *testing.T) {
	scenarios := []struct {
		name        string
		tasks       []storage.Task
		expectedOrder []string // Expected IDs in order
	}{
		{
			name: "混在した優先度のソート",
			tasks: []storage.Task{
				{ID: "1", Priority: "low"},
				{ID: "2", Priority: "critical"},
				{ID: "3", Priority: "medium"},
				{ID: "4", Priority: "high"},
			},
			expectedOrder: []string{"2", "4", "3", "1"}, // critical, high, medium, low
		},
		{
			name: "同じ優先度のタスク",
			tasks: []storage.Task{
				{ID: "1", Priority: "high"},
				{ID: "2", Priority: "high"},
				{ID: "3", Priority: "high"},
			},
			expectedOrder: []string{"1", "2", "3"}, // Order preserved for same priority
		},
		{
			name: "既にソート済み",
			tasks: []storage.Task{
				{ID: "1", Priority: "critical"},
				{ID: "2", Priority: "high"},
				{ID: "3", Priority: "medium"},
				{ID: "4", Priority: "low"},
			},
			expectedOrder: []string{"1", "2", "3", "4"},
		},
		{
			name: "逆順からのソート",
			tasks: []storage.Task{
				{ID: "1", Priority: "low"},
				{ID: "2", Priority: "medium"},
				{ID: "3", Priority: "high"},
				{ID: "4", Priority: "critical"},
			},
			expectedOrder: []string{"4", "3", "2", "1"},
		},
		{
			name:          "空のタスクリスト",
			tasks:         []storage.Task{},
			expectedOrder: []string{},
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			// Copy tasks to avoid modifying original
			tasksCopy := make([]storage.Task, len(tt.tasks))
			copy(tasksCopy, tt.tasks)

			SortTasksByPriority(tasksCopy)

			if len(tasksCopy) != len(tt.expectedOrder) {
				t.Errorf("Expected %d tasks, got %d", len(tt.expectedOrder), len(tasksCopy))
			}

			for i, task := range tasksCopy {
				if i < len(tt.expectedOrder) && task.ID != tt.expectedOrder[i] {
					t.Errorf("Expected task %s at position %d, got %s", tt.expectedOrder[i], i, task.ID)
				}
			}
		})
	}
}

// TestGenerateTaskID tests task ID generation
func TestGenerateTaskID(t *testing.T) {
	tests := []struct {
		name     string
		task     storage.Task
		expected string
	}{
		{
			name:     "PR番号1",
			task:     storage.Task{PRNumber: 1},
			expected: "TSK-001",
		},
		{
			name:     "PR番号100",
			task:     storage.Task{PRNumber: 100},
			expected: "TSK-100",
		},
		{
			name:     "PR番号999",
			task:     storage.Task{PRNumber: 999},
			expected: "TSK-999",
		},
		{
			name:     "PR番号0",
			task:     storage.Task{PRNumber: 0},
			expected: "TSK-000",
		},
		{
			name:     "大きなPR番号",
			task:     storage.Task{PRNumber: 12345},
			expected: "TSK-12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateTaskID(tt.task)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestCalculateTaskStats tests statistics calculation
func TestCalculateTaskStats(t *testing.T) {
	scenarios := []struct {
		name             string
		tasks            []storage.Task
		expectedStats    TaskStats
	}{
		{
			name: "複数のタスクの統計",
			tasks: []storage.Task{
				{Status: "todo", Priority: "high", PRNumber: 100},
				{Status: "todo", Priority: "high", PRNumber: 100},
				{Status: "doing", Priority: "medium", PRNumber: 101},
				{Status: "done", Priority: "low", PRNumber: 100},
				{Status: "pending", Priority: "critical", PRNumber: 102},
			},
			expectedStats: TaskStats{
				StatusCounts: map[string]int{
					"todo":    2,
					"doing":   1,
					"done":    1,
					"pending": 1,
				},
				PriorityCounts: map[string]int{
					"high":     2,
					"medium":   1,
					"low":      1,
					"critical": 1,
				},
				PRCounts: map[int]int{
					100: 3,
					101: 1,
					102: 1,
				},
			},
		},
		{
			name: "空のタスクリスト",
			tasks: []storage.Task{},
			expectedStats: TaskStats{
				StatusCounts:   map[string]int{},
				PriorityCounts: map[string]int{},
				PRCounts:       map[int]int{},
			},
		},
		{
			name: "単一タスク",
			tasks: []storage.Task{
				{Status: "todo", Priority: "high", PRNumber: 123},
			},
			expectedStats: TaskStats{
				StatusCounts:   map[string]int{"todo": 1},
				PriorityCounts: map[string]int{"high": 1},
				PRCounts:       map[int]int{123: 1},
			},
		},
		{
			name: "同じPRの複数タスク",
			tasks: []storage.Task{
				{Status: "todo", Priority: "high", PRNumber: 200},
				{Status: "doing", Priority: "medium", PRNumber: 200},
				{Status: "done", Priority: "low", PRNumber: 200},
			},
			expectedStats: TaskStats{
				StatusCounts: map[string]int{
					"todo":  1,
					"doing": 1,
					"done":  1,
				},
				PriorityCounts: map[string]int{
					"high":   1,
					"medium": 1,
					"low":    1,
				},
				PRCounts: map[int]int{
					200: 3,
				},
			},
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			stats := CalculateTaskStats(tt.tasks)

			// Check status counts
			if !reflect.DeepEqual(stats.StatusCounts, tt.expectedStats.StatusCounts) {
				t.Errorf("StatusCounts mismatch.\nExpected: %v\nGot: %v", 
					tt.expectedStats.StatusCounts, stats.StatusCounts)
			}

			// Check priority counts
			if !reflect.DeepEqual(stats.PriorityCounts, tt.expectedStats.PriorityCounts) {
				t.Errorf("PriorityCounts mismatch.\nExpected: %v\nGot: %v", 
					tt.expectedStats.PriorityCounts, stats.PriorityCounts)
			}

			// Check PR counts
			if !reflect.DeepEqual(stats.PRCounts, tt.expectedStats.PRCounts) {
				t.Errorf("PRCounts mismatch.\nExpected: %v\nGot: %v", 
					tt.expectedStats.PRCounts, stats.PRCounts)
			}
		})
	}
}

// TestTaskWorkflowScenarios tests complete task workflows
func TestTaskWorkflowScenarios(t *testing.T) {
	scenarios := []struct {
		name  string
		steps []func(tasks []storage.Task) []storage.Task
		initial []storage.Task
		verify func(t *testing.T, final []storage.Task)
	}{
		{
			name: "優先度別タスク処理フロー",
			initial: []storage.Task{
				{ID: "1", Status: "todo", Priority: "low", PRNumber: 100},
				{ID: "2", Status: "todo", Priority: "critical", PRNumber: 100},
				{ID: "3", Status: "todo", Priority: "high", PRNumber: 100},
			},
			steps: []func(tasks []storage.Task) []storage.Task{
				func(tasks []storage.Task) []storage.Task {
					// Sort by priority
					SortTasksByPriority(tasks)
					return tasks
				},
				func(tasks []storage.Task) []storage.Task {
					// Process critical task first
					if len(tasks) > 0 {
						tasks[0].Status = "doing"
					}
					return tasks
				},
			},
			verify: func(t *testing.T, final []storage.Task) {
				if len(final) != 3 {
					t.Errorf("Expected 3 tasks, got %d", len(final))
				}
				if final[0].Priority != "critical" || final[0].Status != "doing" {
					t.Error("Critical task should be first and in doing status")
				}
			},
		},
		{
			name: "ステータス別フィルタリングフロー",
			initial: []storage.Task{
				{ID: "1", Status: "todo", Priority: "high", PRNumber: 100},
				{ID: "2", Status: "doing", Priority: "high", PRNumber: 100},
				{ID: "3", Status: "done", Priority: "high", PRNumber: 100},
				{ID: "4", Status: "todo", Priority: "low", PRNumber: 100},
			},
			steps: []func(tasks []storage.Task) []storage.Task{
				func(tasks []storage.Task) []storage.Task {
					// Get only todo tasks
					return FilterTasksByStatus(tasks, "todo")
				},
				func(tasks []storage.Task) []storage.Task {
					// Sort remaining tasks
					SortTasksByPriority(tasks)
					return tasks
				},
			},
			verify: func(t *testing.T, final []storage.Task) {
				if len(final) != 2 {
					t.Errorf("Expected 2 todo tasks, got %d", len(final))
				}
				for _, task := range final {
					if task.Status != "todo" {
						t.Errorf("All tasks should have todo status, got %s", task.Status)
					}
				}
				if len(final) > 0 && final[0].Priority != "high" {
					t.Error("High priority task should be first")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tasks := make([]storage.Task, len(scenario.initial))
			copy(tasks, scenario.initial)

			for _, step := range scenario.steps {
				tasks = step(tasks)
			}

			scenario.verify(t, tasks)
		})
	}
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("nil tasks slice", func(t *testing.T) {
		var tasks []storage.Task
		
		filtered := FilterTasksByStatus(tasks, "todo")
		if len(filtered) != 0 {
			t.Error("Expected empty slice for nil input")
		}

		SortTasksByPriority(tasks) // Should not panic

		stats := CalculateTaskStats(tasks)
		if stats.StatusCounts == nil || len(stats.StatusCounts) != 0 {
			t.Error("Expected empty maps for nil input")
		}
	})

	t.Run("invalid priority values", func(t *testing.T) {
		tasks := []storage.Task{
			{ID: "1", Priority: "invalid"},
			{ID: "2", Priority: ""},
			{ID: "3", Priority: "unknown"},
		}

		// Should not panic with invalid priorities
		SortTasksByPriority(tasks)
	})

	t.Run("duplicate task IDs", func(t *testing.T) {
		tasks := []storage.Task{
			{ID: "1", Status: "todo", Priority: "high"},
			{ID: "1", Status: "doing", Priority: "low"},
			{ID: "1", Status: "done", Priority: "medium"},
		}

		stats := CalculateTaskStats(tasks)
		if stats.StatusCounts["todo"] != 1 {
			t.Error("Should count tasks with duplicate IDs")
		}
	})
}

// TestPerformance tests performance with large datasets
func TestPerformance(t *testing.T) {
	t.Run("大量タスクのソート", func(t *testing.T) {
		// Create 10000 tasks
		var tasks []storage.Task
		priorities := []string{"critical", "high", "medium", "low"}
		
		for i := 0; i < 10000; i++ {
			tasks = append(tasks, storage.Task{
				ID:       string(rune(i)),
				Status:   "todo",
				Priority: priorities[i%4],
				PRNumber: i % 100,
			})
		}

		// This should complete quickly
		SortTasksByPriority(tasks)

		// Verify first task is critical priority
		if tasks[0].Priority != "critical" {
			t.Error("First task should have critical priority after sorting")
		}
	})

	t.Run("大量タスクの統計計算", func(t *testing.T) {
		var tasks []storage.Task
		statuses := []string{"todo", "doing", "done", "pending"}
		priorities := []string{"critical", "high", "medium", "low"}
		
		for i := 0; i < 10000; i++ {
			tasks = append(tasks, storage.Task{
				ID:       string(rune(i)),
				Status:   statuses[i%4],
				Priority: priorities[i%4],
				PRNumber: i % 50,
			})
		}

		stats := CalculateTaskStats(tasks)

		// Verify stats are calculated correctly
		totalTasks := 0
		for _, count := range stats.StatusCounts {
			totalTasks += count
		}
		if totalTasks != 10000 {
			t.Errorf("Expected 10000 tasks in stats, got %d", totalTasks)
		}
	})
}