package ui

import (
	"strings"
	"testing"

	"reviewtask/internal/tasks"
)

// TestGenerateColoredProgressBar tests progress bar generation
func TestGenerateColoredProgressBar(t *testing.T) {
	scenarios := []struct {
		name          string
		stats         tasks.TaskStats
		width         int
		validateFunc  func(t *testing.T, result string)
	}{
		{
			name: "空のタスク統計",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{},
			},
			width: 50,
			validateFunc: func(t *testing.T, result string) {
				// Should return empty progress bar
				if !strings.Contains(result, "░") {
					t.Error("Expected empty progress bar with ░ characters")
				}
			},
		},
		{
			name: "全てのステータスが混在",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo":    5,
					"doing":   3,
					"done":    10,
					"pending": 2,
					"cancel":  0,
				},
			},
			width: 80,
			validateFunc: func(t *testing.T, result string) {
				// Should contain progress bar characters
				if !strings.ContainsAny(result, "█▓▒░") {
					t.Error("Progress bar should contain bar characters")
				}
			},
		},
		{
			name: "完了率100%",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"done": 10,
				},
			},
			width: 50,
			validateFunc: func(t *testing.T, result string) {
				// Should contain filled characters
				if !strings.ContainsAny(result, "█▓") {
					t.Error("Completed progress bar should contain filled characters")
				}
			},
		},
		{
			name: "完了率0%",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo": 10,
				},
			},
			width: 50,
			validateFunc: func(t *testing.T, result string) {
				// Should contain empty or partial characters
				if !strings.ContainsAny(result, "░▒") {
					t.Error("Empty progress bar should contain empty characters")
				}
			},
		},
		{
			name: "幅が0の場合",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo": 5,
				},
			},
			width: 0,
			validateFunc: func(t *testing.T, result string) {
				if result != "" {
					t.Error("Expected empty string for width 0")
				}
			},
		},
		{
			name: "負の幅",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo": 5,
				},
			},
			width: -10,
			validateFunc: func(t *testing.T, result string) {
				if result != "" {
					t.Error("Expected empty string for negative width")
				}
			},
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateColoredProgressBar(tt.stats, tt.width)
			tt.validateFunc(t, result)
		})
	}
}

// TestProgressBarFunctionality tests that progress bar works correctly
func TestProgressBarFunctionality(t *testing.T) {
	tests := []struct {
		name  string
		stats tasks.TaskStats
		width int
	}{
		{
			name: "正常な進捗バー生成",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo":  25,
					"doing": 25,
					"done":  25,
					"pending": 25,
				},
			},
			width: 100,
		},
		{
			name: "偏った分布",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo":  5,
					"doing": 1,
					"done":  90,
					"pending": 4,
				},
			},
			width: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateColoredProgressBar(tt.stats, tt.width)
			
			// For positive width, should return non-empty result
			if tt.width > 0 && result == "" {
				t.Error("Expected non-empty progress bar for positive width")
			}
			
			// Should contain progress bar characters
			if tt.width > 0 && !strings.ContainsAny(result, "█▓▒░") {
				t.Error("Progress bar should contain bar characters")
			}
		})
	}
}

// TestProgressBarStyles tests that styles are applied correctly
func TestProgressBarStyles(t *testing.T) {
	t.Run("スタイル定義テスト", func(t *testing.T) {
		// Test that styles are defined
		styles := []struct {
			name  string
			style interface{}
		}{
			{"TodoProgressStyle", TodoProgressStyle},
			{"DoingProgressStyle", DoingProgressStyle},
			{"DoneProgressStyle", DoneProgressStyle},
			{"PendingProgressStyle", PendingProgressStyle},
			{"EmptyProgressStyle", EmptyProgressStyle},
		}

		for _, s := range styles {
			if s.style == nil {
				t.Errorf("Style %s is not defined", s.name)
			}
		}
	})
}

// TestProgressBarWorkflow tests complete progress bar workflows
func TestProgressBarWorkflow(t *testing.T) {
	scenarios := []struct {
		name   string
		stages []tasks.TaskStats
		width  int
	}{
		{
			name: "タスク進行シミュレーション",
			stages: []tasks.TaskStats{
				// Stage 1: All todo
				{
					StatusCounts: map[string]int{
						"todo": 10,
					},
				},
				// Stage 2: Some in progress
				{
					StatusCounts: map[string]int{
						"todo":  6,
						"doing": 4,
					},
				},
				// Stage 3: Some completed
				{
					StatusCounts: map[string]int{
						"todo":  3,
						"doing": 3,
						"done":  4,
					},
				},
				// Stage 4: Most completed
				{
					StatusCounts: map[string]int{
						"todo": 1,
						"done": 9,
					},
				},
				// Stage 5: All done
				{
					StatusCounts: map[string]int{
						"done": 10,
					},
				},
			},
			width: 50,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for i, stage := range scenario.stages {
				bar := GenerateColoredProgressBar(stage, scenario.width)
				
				// Verify bar is generated
				if bar == "" && scenario.width > 0 {
					t.Errorf("Stage %d: Expected non-empty progress bar", i)
				}

				// Should contain progress bar characters
				if scenario.width > 0 && !strings.ContainsAny(bar, "█▓▒░") {
					t.Errorf("Stage %d: Progress bar should contain bar characters", i)
				}
			}
		})
	}
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		stats tasks.TaskStats
		width int
	}{
		{
			name: "nil StatusCounts map",
			stats: tasks.TaskStats{
				StatusCounts: nil,
			},
			width: 50,
		},
		{
			name: "極端に大きい値",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo":  999999,
					"doing": 999999,
					"done":  999999,
				},
			},
			width: 50,
		},
		{
			name: "負の値（無効だが処理される）",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"todo": -10,
					"done": 10,
				},
			},
			width: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := GenerateColoredProgressBar(tt.stats, tt.width)
			
			// If width is positive, check for valid output
			if tt.width > 0 && result != "" {
				// Should contain some progress bar characters
				if !strings.ContainsAny(result, "█▓▒░") {
					t.Log("Progress bar may not contain expected characters for edge case")
				}
			}
		})
	}
}

// TestPerformance tests performance with various sizes
func TestPerformance(t *testing.T) {
	t.Run("大量タスクの進捗バー", func(t *testing.T) {
		stats := tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":    100000,
				"doing":   50000,
				"done":    150000,
				"pending": 25000,
				"cancel":  25000,
			},
		}

		// Should complete quickly even with large numbers
		result := GenerateColoredProgressBar(stats, 100)
		
		if result == "" {
			t.Error("Expected non-empty result for large task counts")
		}

		// Should contain progress bar characters
		if !strings.ContainsAny(result, "█▓▒░") {
			t.Error("Progress bar should contain bar characters")
		}
	})

	t.Run("非常に広い進捗バー", func(t *testing.T) {
		stats := tasks.TaskStats{
			StatusCounts: map[string]int{
				"todo":  10,
				"doing": 5,
				"done":  15,
			},
		}

		// Test with very wide progress bar
		result := GenerateColoredProgressBar(stats, 10000)
		
		if result == "" {
			t.Error("Expected non-empty result for wide progress bar")
		}

		// Should contain progress bar characters
		if !strings.ContainsAny(result, "█▓▒░") {
			t.Error("Progress bar should contain bar characters")
		}
	})
}