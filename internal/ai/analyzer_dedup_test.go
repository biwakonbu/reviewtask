package ai

import (
	"testing"

	"github.com/google/uuid"
	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

func TestDeduplicateTasks(t *testing.T) {
	// Skip tests that require Claude CLI until mock is implemented
	t.Skip("Skipping AI deduplication tests - requires Claude CLI or mock implementation")
}

func TestDeduplicateTasksDisabled(t *testing.T) {
	// Create config with deduplication disabled
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
		AISettings: config.AISettings{
			MaxTasksPerComment:   2,
			DeduplicationEnabled: false, // Disabled
			SimilarityThreshold:  0.8,
		},
	}
	analyzer := NewAnalyzer(cfg)

	tasks := []storage.Task{
		{ID: uuid.New().String(), Description: "Fix bug 1", Priority: "high", SourceCommentID: 123},
		{ID: uuid.New().String(), Description: "Fix bug 2", Priority: "medium", SourceCommentID: 123},
		{ID: uuid.New().String(), Description: "Fix bug 3", Priority: "low", SourceCommentID: 123},
		{ID: uuid.New().String(), Description: "Fix bug 4", Priority: "critical", SourceCommentID: 123},
	}

	dedupedTasks := analyzer.deduplicateTasks(tasks)
	if len(dedupedTasks) != len(tasks) {
		t.Errorf("With deduplication disabled, should return all tasks: got %d, want %d", len(dedupedTasks), len(tasks))
	}
}

func TestSortTasksByPriority(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
		AISettings: config.AISettings{
			MaxTasksPerComment:   4,
			DeduplicationEnabled: true,
		},
	}
	analyzer := NewAnalyzer(cfg)

	tasks := []storage.Task{
		{ID: "1", Description: "Low task", Priority: "low", TaskIndex: 0},
		{ID: "2", Description: "Critical task", Priority: "critical", TaskIndex: 1},
		{ID: "3", Description: "Medium task", Priority: "medium", TaskIndex: 2},
		{ID: "4", Description: "High task", Priority: "high", TaskIndex: 3},
		{ID: "5", Description: "Another medium", Priority: "medium", TaskIndex: 4},
	}

	sorted := analyzer.sortTasksByPriority(tasks)

	// Check order
	expectedOrder := []string{"critical", "high", "medium", "medium", "low"}
	for i, task := range sorted {
		if task.Priority != expectedOrder[i] {
			t.Errorf("Task %d: expected priority %s, got %s", i, expectedOrder[i], task.Priority)
		}
	}

	// Verify tasks with same priority are ordered by task index
	if sorted[2].TaskIndex > sorted[3].TaskIndex {
		t.Errorf("Tasks with same priority should be ordered by task index")
	}
}

func TestCalculateSimilarity(t *testing.T) {
	cfg := &config.Config{}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		s1       string
		s2       string
		expected float64
		delta    float64
	}{
		{
			s1:       "Fix memory leak in parser",
			s2:       "Fix memory leak in parser",
			expected: 1.0,
			delta:    0.01,
		},
		{
			s1:       "Fix memory leak in parser",
			s2:       "Fix memory leak in the parser",
			expected: 0.833, // 5 common words out of 6 unique words
			delta:    0.01,
		},
		{
			s1:       "Add validation",
			s2:       "Add input validation",
			expected: 0.67, // 2 common words out of 3 unique words
			delta:    0.01,
		},
		{
			s1:       "Fix bug",
			s2:       "Update documentation",
			expected: 0.0, // No common words
			delta:    0.01,
		},
		{
			s1:       "",
			s2:       "",
			expected: 1.0, // Both empty
			delta:    0.01,
		},
		{
			s1:       "Fix bug",
			s2:       "",
			expected: 0.0, // One empty
			delta:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_vs_"+tt.s2, func(t *testing.T) {
			similarity := analyzer.calculateSimilarity(tt.s1, tt.s2)
			if similarity < tt.expected-tt.delta || similarity > tt.expected+tt.delta {
				t.Errorf("calculateSimilarity(%q, %q) = %f, want %f±%f", tt.s1, tt.s2, similarity, tt.expected, tt.delta)
			}
		})
	}
}

func TestDeduplicateSimilarTasks(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			SimilarityThreshold: 0.6, // Lower threshold to catch "Add validation" vs "Add input validation"
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name      string
		tasks     []storage.Task
		wantCount int
	}{
		{
			name: "Remove highly similar tasks",
			tasks: []storage.Task{
				{ID: "1", Description: "Fix memory leak in parser", Priority: "high"},
				{ID: "2", Description: "Fix memory leak in the parser", Priority: "medium"},
				{ID: "3", Description: "Update documentation", Priority: "low"},
			},
			wantCount: 2, // Should remove the similar one with lower priority
		},
		{
			name: "Keep dissimilar tasks",
			tasks: []storage.Task{
				{ID: "1", Description: "Fix memory leak", Priority: "high"},
				{ID: "2", Description: "Add validation", Priority: "high"},
				{ID: "3", Description: "Update tests", Priority: "high"},
			},
			wantCount: 3, // All different enough
		},
		{
			name: "Priority matters for deduplication",
			tasks: []storage.Task{
				{ID: "1", Description: "Add input validation", Priority: "low"},
				{ID: "2", Description: "Add validation", Priority: "critical"},
			},
			wantCount: 1, // Should keep the critical one
		},
		{
			name:      "Empty list",
			tasks:     []storage.Task{},
			wantCount: 0,
		},
		{
			name: "Single task",
			tasks: []storage.Task{
				{ID: "1", Description: "Fix bug", Priority: "high"},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deduped := analyzer.deduplicateSimilarTasks(tt.tasks)
			if len(deduped) != tt.wantCount {
				t.Errorf("deduplicateSimilarTasks() returned %d tasks, want %d", len(deduped), tt.wantCount)
				t.Logf("Returned tasks:")
				for i, task := range deduped {
					t.Logf("  %d: %s (priority: %s)", i+1, task.Description, task.Priority)
				}
			}
		})
	}
}

func TestGetPriorityValue(t *testing.T) {
	analyzer := NewAnalyzer(&config.Config{})

	tests := []struct {
		priority string
		expected int
	}{
		{"critical", 0},
		{"high", 1},
		{"medium", 2},
		{"low", 3},
		{"unknown", 4},
		{"", 4},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			value := analyzer.getPriorityValue(tt.priority)
			if value != tt.expected {
				t.Errorf("getPriorityValue(%q) = %d, want %d", tt.priority, value, tt.expected)
			}
		})
	}
}