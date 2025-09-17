package ai

import (
	"encoding/json"
	"os"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"testing"
)

// TestMultiInstructionCommentProcessing tests that comments with multiple
// unrelated issues are appropriately handled (consolidated vs split)
//
// This is a golden test similar to other AI-dependent tests that requires:
// - Manual execution with environment variable
// - cursor-agent authentication
// - Not run in CI/automated environments
//
// Usage:
//   UPDATE_GOLDEN=1 SKIP_CURSOR_AUTH_CHECK=true go test -v ./internal/ai -run TestMultiInstructionCommentProcessing
func TestMultiInstructionCommentProcessing(t *testing.T) {
	// Skip by default - only run when UPDATE_GOLDEN=1 like other golden tests
	if os.Getenv("UPDATE_GOLDEN") != "1" {
		t.Skip("Skipping multi-instruction test - set UPDATE_GOLDEN=1 to run")
	}

	// Load test data
	testData, err := os.ReadFile("testdata/multi_instruction_comments.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	var reviewData struct {
		Reviews []github.Review `json:"reviews"`
	}
	if err := json.Unmarshal(testData, &reviewData); err != nil {
		t.Fatalf("Failed to parse test data: %v", err)
	}

	// Setup test config with cursor provider
	cfg := &config.Config{
		AISettings: config.AISettings{
			AIProvider:   "cursor",
			VerboseMode:  false, // Reduce noise in tests
			Model:        "auto",
			UserLanguage: "English",
		},
	}

	// Force cursor provider via environment
	os.Setenv("REVIEWTASK_AI_PROVIDER", "cursor")
	defer os.Unsetenv("REVIEWTASK_AI_PROVIDER")

	// Initialize analyzer
	analyzer := NewAnalyzer(cfg)

	// Generate tasks
	tasks, err := analyzer.GenerateTasks(reviewData.Reviews)
	if err != nil {
		t.Fatalf("Failed to generate tasks: %v", err)
	}

	// Analyze results
	if len(tasks) == 0 {
		t.Fatal("No tasks generated")
	}

	// Count tasks per comment
	commentTaskCount := make(map[int64]int)
	for _, task := range tasks {
		commentTaskCount[task.SourceCommentID]++
	}

	// Test expectations
	totalComments := len(reviewData.Reviews[0].Comments)
	totalTasks := len(tasks)
	averageTasksPerComment := float64(totalTasks) / float64(totalComments)

	t.Logf("Test Results:")
	t.Logf("  Total Comments: %d", totalComments)
	t.Logf("  Total Tasks Generated: %d", totalTasks)
	t.Logf("  Average Tasks per Comment: %.1f", averageTasksPerComment)

	// Verify that we don't have excessive task splitting
	if averageTasksPerComment > 2.0 {
		t.Errorf("Too many tasks per comment: %.1f (should be <= 2.0)", averageTasksPerComment)
	}

	// Verify we have reasonable consolidation (not too aggressive)
	if averageTasksPerComment < 0.8 {
		t.Errorf("Too few tasks per comment: %.1f (should be >= 0.8)", averageTasksPerComment)
	}

	// Check individual comment processing
	for _, comment := range reviewData.Reviews[0].Comments {
		taskCount := commentTaskCount[comment.ID]
		t.Logf("Comment %d: %d tasks", comment.ID, taskCount)

		// Single focused issue (comment 99999003) should remain single
		if comment.ID == 99999003 {
			if taskCount != 1 {
				t.Errorf("Single focused comment %d should generate exactly 1 task, got %d", comment.ID, taskCount)
			}
		}

		// Multi-issue comments should be consolidated appropriately
		if comment.ID == 99999001 || comment.ID == 99999002 {
			if taskCount < 1 {
				t.Errorf("Multi-issue comment %d should generate at least 1 task, got %d", comment.ID, taskCount)
			}
			if taskCount > 3 {
				t.Errorf("Multi-issue comment %d generated too many tasks: %d (should be <= 3)", comment.ID, taskCount)
			}
		}
	}

	// Verify task quality - all tasks should have valid descriptions and priorities
	for i, task := range tasks {
		if task.Description == "" {
			t.Errorf("Task %d has empty description", i)
		}
		if task.Priority == "" {
			t.Errorf("Task %d has empty priority", i)
		}
		if task.SourceCommentID == 0 {
			t.Errorf("Task %d has invalid source comment ID", i)
		}
	}

	t.Logf("✅ Multi-instruction comment processing test passed")
}