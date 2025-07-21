package ai

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// TestDeduplicationIntegration tests the complete deduplication flow
func TestDeduplicationIntegration(t *testing.T) {
	// Create config with deduplication enabled
	validationTrue := true
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:  "todo",
			AutoPrioritize: true,
		},
		AISettings: config.AISettings{
			UserLanguage:         "English",
			OutputFormat:         "json",
			MaxRetries:           1,
			ValidationEnabled:    &validationTrue,
			QualityThreshold:     0.8,
			DebugMode:            true, // Enable debug for test
			MaxTasksPerComment:   2,
			DeduplicationEnabled: true,
			SimilarityThreshold:  0.7,
		},
	}

	analyzer := NewAnalyzer(cfg)

	// Test scenario: Simulate AI generating multiple duplicate tasks per comment
	tests := []struct {
		name          string
		inputTasks    []TaskRequest
		expectedCount int
		expectedTasks map[string]bool // Expected task descriptions
		description   string
	}{
		{
			name: "Multiple duplicates from single comment",
			inputTasks: []TaskRequest{
				// Comment 2218346566 - binary filename issues (15 variations in original bug report)
				{Description: "Fix binary filename format issue", Priority: "critical", SourceCommentID: 2218346566, TaskIndex: 0},
				{Description: "Fix binary filename format", Priority: "high", SourceCommentID: 2218346566, TaskIndex: 1},
				{Description: "Fix the binary filename format issue", Priority: "high", SourceCommentID: 2218346566, TaskIndex: 2},
				{Description: "Correct binary filename format", Priority: "medium", SourceCommentID: 2218346566, TaskIndex: 3},
				{Description: "Fix binary file naming format", Priority: "medium", SourceCommentID: 2218346566, TaskIndex: 4},
				// Different task that should survive
				{Description: "Add validation for binary names", Priority: "critical", SourceCommentID: 2218346566, TaskIndex: 5},
			},
			expectedCount: 2, // Should keep only 2 tasks per comment
			expectedTasks: map[string]bool{
				"Fix binary filename format issue": true, // Critical priority, first
				"Add validation for binary names":  true, // Different task, critical priority
			},
			description: "Should reduce 6 tasks to 2, keeping highest priority similar tasks",
		},
		{
			name: "Multiple comments with duplicates",
			inputTasks: []TaskRequest{
				// Comment 1 - validation issues (8 variations in original)
				{Description: "Add validation for --version argument", Priority: "high", SourceCommentID: 2218346551, TaskIndex: 0},
				{Description: "Add argument validation for --version", Priority: "high", SourceCommentID: 2218346551, TaskIndex: 1},
				{Description: "Validate --version argument", Priority: "medium", SourceCommentID: 2218346551, TaskIndex: 2},
				{Description: "Add --version validation", Priority: "medium", SourceCommentID: 2218346551, TaskIndex: 3},
				// Comment 2 - checksum issues (5 variations in original)
				{Description: "Fix checksum verification on macOS", Priority: "critical", SourceCommentID: 2218552490, TaskIndex: 0},
				{Description: "Fix macOS checksum verification", Priority: "critical", SourceCommentID: 2218552490, TaskIndex: 1},
				{Description: "Repair checksum check on macOS", Priority: "high", SourceCommentID: 2218552490, TaskIndex: 2},
			},
			expectedCount: 2, // Only 2 unique comments, so max 2 tasks (1 per comment after dedup)
			expectedTasks: map[string]bool{
				"Add validation for --version argument": true,
				"Fix checksum verification on macOS":    true,
			},
			description: "Should deduplicate within each comment, keeping highest priority",
		},
		{
			name: "Tasks with varying similarity levels",
			inputTasks: []TaskRequest{
				// Similar but not identical from comment 1
				{Description: "Fix error handling in parser", Priority: "high", SourceCommentID: 12345, TaskIndex: 0},
				{Description: "Fix error handling in the parser", Priority: "high", SourceCommentID: 12345, TaskIndex: 1},
				{Description: "Fix parser error handling", Priority: "medium", SourceCommentID: 12345, TaskIndex: 2},
				// Completely different from comment 2
				{Description: "Update test coverage", Priority: "medium", SourceCommentID: 12346, TaskIndex: 0},
			},
			expectedCount: 2,
			expectedTasks: map[string]bool{
				"Fix error handling in parser": true, // First high priority from comment 1
				"Update test coverage":         true, // From comment 2
			},
			description: "Should detect and remove similar tasks while keeping distinct ones",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert TaskRequests to storage.Tasks (simulating AI output)
			storageTasks := analyzer.convertToStorageTasks(tt.inputTasks)

			// Apply deduplication
			dedupedTasks := analyzer.deduplicateTasks(storageTasks)

			// Check task count
			if len(dedupedTasks) != tt.expectedCount {
				t.Errorf("%s: got %d tasks, expected %d", tt.description, len(dedupedTasks), tt.expectedCount)
				t.Logf("Tasks returned:")
				for i, task := range dedupedTasks {
					t.Logf("  %d: %s (priority: %s, comment: %d)",
						i+1, task.Description, task.Priority, task.SourceCommentID)
				}
			}

			// Verify expected tasks are present
			foundTasks := make(map[string]bool)
			for _, task := range dedupedTasks {
				foundTasks[task.Description] = true
			}

			for expectedDesc := range tt.expectedTasks {
				if !foundTasks[expectedDesc] {
					t.Errorf("Expected task not found: %s", expectedDesc)
				}
			}

			// Verify all tasks have valid UUIDs
			for _, task := range dedupedTasks {
				if _, err := uuid.Parse(task.ID); err != nil {
					t.Errorf("Task has invalid UUID: %s", task.ID)
				}
			}
		})
	}
}

// TestDeduplicationPerformance tests deduplication with large numbers of tasks
func TestDeduplicationPerformance(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
		AISettings: config.AISettings{
			MaxTasksPerComment:   2,
			DeduplicationEnabled: true,
			SimilarityThreshold:  0.7,
		},
	}

	analyzer := NewAnalyzer(cfg)

	// Generate a large number of tasks simulating the bug report scenario
	// 40 tasks from 16 comments with heavy duplication
	var tasks []storage.Task
	commentIDs := []int64{
		2218346566, 2218346551, 2218552499, 2218552490,
		2218346569, 2218257366, 2218257367, 2218257368,
	}

	taskTemplates := []string{
		"Fix %s issue in %s",
		"Resolve %s problem in %s",
		"Address %s in %s",
		"Update %s for %s",
		"Improve %s handling in %s",
	}

	issues := []string{"validation", "checksum", "binary naming", "error handling"}
	components := []string{"installer", "parser", "validator", "main function"}

	// Generate tasks with realistic duplication patterns
	taskID := 0
	for _, commentID := range commentIDs {
		// Generate 5-15 tasks per comment (simulating bug report)
		numTasks := 5 + (int(commentID) % 10)
		issue := issues[commentID%int64(len(issues))]
		component := components[commentID%int64(len(components))]

		for i := 0; i < numTasks; i++ {
			template := taskTemplates[i%len(taskTemplates)]
			task := storage.Task{
				ID:              uuid.New().String(),
				Description:     fmt.Sprintf(template, issue, component),
				Priority:        []string{"critical", "high", "medium", "low"}[i%4],
				SourceCommentID: commentID,
				TaskIndex:       i,
				Status:          "todo",
			}
			tasks = append(tasks, task)
			taskID++
		}
	}

	startCount := len(tasks)
	t.Logf("Starting with %d tasks from %d comments", startCount, len(commentIDs))

	// Time the deduplication
	start := time.Now()
	dedupedTasks := analyzer.deduplicateTasks(tasks)
	duration := time.Since(start)

	endCount := len(dedupedTasks)
	reductionPercent := float64(startCount-endCount) / float64(startCount) * 100

	t.Logf("Deduplication completed in %v", duration)
	t.Logf("Reduced %d tasks to %d tasks (%.1f%% reduction)", startCount, endCount, reductionPercent)

	// Verify results
	expectedMaxTasks := len(commentIDs) * cfg.AISettings.MaxTasksPerComment
	if endCount > expectedMaxTasks {
		t.Errorf("Too many tasks after deduplication: got %d, expected max %d", endCount, expectedMaxTasks)
	}

	// Verify no comment has more than MaxTasksPerComment
	tasksByComment := make(map[int64]int)
	for _, task := range dedupedTasks {
		tasksByComment[task.SourceCommentID]++
	}

	for commentID, count := range tasksByComment {
		if count > cfg.AISettings.MaxTasksPerComment {
			t.Errorf("Comment %d has %d tasks, exceeds max of %d", commentID, count, cfg.AISettings.MaxTasksPerComment)
		}
	}

	// Performance check - should complete quickly even with many tasks
	if duration > 100*time.Millisecond {
		t.Logf("Warning: Deduplication took longer than expected: %v", duration)
	}
}

// TestPromptChangesReduceDuplication verifies that prompt changes result in fewer tasks
func TestPromptChangesReduceDuplication(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
		AISettings: config.AISettings{
			UserLanguage:         "English",
			MaxTasksPerComment:   2,
			DeduplicationEnabled: true,
			SimilarityThreshold:  0.7,
		},
	}

	analyzer := NewAnalyzer(cfg)

	// Verify that buildCommentPrompt includes the new guidelines
	ctx := CommentContext{
		Comment: github.Comment{
			ID:     12345,
			Author: "reviewer",
			Body:   "Please fix the validation, error handling, and add tests",
			File:   "main.go",
			Line:   42,
		},
		SourceReview: github.Review{
			ID:       67890,
			Reviewer: "reviewer",
			State:    "CHANGES_REQUESTED",
		},
	}

	prompt := analyzer.buildCommentPrompt(ctx)

	// Check that prompt contains new guidelines
	expectedPhrases := []string{
		"Create MINIMAL tasks",
		"MAXIMUM 2 tasks per comment",
		"Prioritize creating ONE comprehensive task",
		"Combine related suggestions into a single actionable task",
		"Focus on the primary intent",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("Prompt missing expected phrase: %s", phrase)
		}
	}

	// Verify old splitting instructions are removed
	unwantedPhrases := []string{
		"SPLIT multiple issues in a single comment into separate tasks",
		"Each issue should become a separate task",
	}

	for _, phrase := range unwantedPhrases {
		if strings.Contains(prompt, phrase) {
			t.Errorf("Prompt contains unwanted phrase: %s", phrase)
		}
	}
}
