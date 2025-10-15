package ai

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"reviewtask/internal/config"
)

// TestTaskGenerationIntegrationWithUUIDs tests the complete task generation workflow
// to ensure UUID generation works correctly in the full pipeline
func TestTaskGenerationIntegrationWithUUIDs(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:  "todo",
			AutoPrioritize: false,
		},
		AISettings: config.AISettings{
			UserLanguage:      "English",
			OutputFormat:      "json",
			MaxRetries:        1,                 // Limit retries for testing
			ValidationEnabled: &[]bool{false}[0], // Disable validation for integration test
			QualityThreshold:  0.8,
			VerboseMode:       false,
		},
	}

	analyzer := NewAnalyzer(cfg)

	// Test the conversion process directly (since we can't mock Claude Code CLI easily)
	mockTaskRequests := []TaskRequest{
		{
			Description:     "Improve error handling in main function",
			OriginText:      "This function needs better error handling",
			Priority:        "high",
			SourceReviewID:  12345,
			SourceCommentID: 67890,
			File:            "main.go",
			Line:            42,
			TaskIndex:       0,
		},
		{
			Description:     "Add input validation to utility function",
			OriginText:      "Consider adding input validation here",
			Priority:        "medium",
			SourceReviewID:  12345,
			SourceCommentID: 67891,
			File:            "utils.go",
			Line:            15,
			TaskIndex:       0,
		},
	}

	// Convert to storage tasks
	storageTasks := analyzer.convertToStorageTasks(mockTaskRequests)

	// Integration test assertions
	if len(storageTasks) != 2 {
		t.Errorf("Expected 2 tasks from integration test, got %d", len(storageTasks))
	}

	// Verify each task has a unique, valid UUID
	seenIDs := make(map[string]bool)
	for i, task := range storageTasks {
		// Verify UUID format and validity
		parsedUUID, err := uuid.Parse(task.ID)
		if err != nil {
			t.Errorf("Integration test task %d has invalid UUID '%s': %v", i, task.ID, err)
		}

		// Verify UUID version (should be v5 for deterministic generation - Issue #247)
		if parsedUUID.Version() != 5 {
			t.Errorf("Integration test task %d UUID '%s' is not version 5, got version %d", i, task.ID, parsedUUID.Version())
		}

		// Verify uniqueness
		if seenIDs[task.ID] {
			t.Errorf("Integration test found duplicate UUID '%s'", task.ID)
		}
		seenIDs[task.ID] = true

		// Verify task contains expected data from mock reviews
		if task.SourceReviewID != 12345 {
			t.Errorf("Integration test task %d has wrong SourceReviewID: expected 12345, got %d",
				i, task.SourceReviewID)
		}

		// Verify timestamps are set
		if task.CreatedAt == "" || task.UpdatedAt == "" {
			t.Errorf("Integration test task %d missing timestamps", i)
		}

		// Verify status is set to default
		if task.Status != cfg.TaskSettings.DefaultStatus {
			t.Errorf("Integration test task %d has wrong status: expected '%s', got '%s'",
				i, cfg.TaskSettings.DefaultStatus, task.Status)
		}
	}
}

// TestUUIDGenerationPerformance tests UUID generation performance at scale
func TestUUIDGenerationPerformance(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Create a large number of task requests
	const numTasks = 10000
	taskRequests := make([]TaskRequest, numTasks)
	for i := 0; i < numTasks; i++ {
		taskRequests[i] = TaskRequest{
			Description:     "Performance test task",
			OriginText:      "Original comment for performance testing",
			Priority:        "medium",
			SourceReviewID:  12345,
			SourceCommentID: int64(67890 + i),
			File:            "test.go",
			Line:            42,
			TaskIndex:       0,
		}
	}

	// Measure conversion time
	startTime := time.Now()
	storageTasks := analyzer.convertToStorageTasks(taskRequests)
	duration := time.Since(startTime)

	// Performance assertions
	if len(storageTasks) != numTasks {
		t.Errorf("Performance test: expected %d tasks, got %d", numTasks, len(storageTasks))
	}

	// Should complete within reasonable time (adjust threshold as needed)
	if duration > 5*time.Second {
		t.Errorf("Performance test: UUID generation took too long: %v", duration)
	}

	// Verify all UUIDs are unique and valid
	seenIDs := make(map[string]bool)
	for i, task := range storageTasks {
		if seenIDs[task.ID] {
			t.Errorf("Performance test: duplicate UUID found at index %d: %s", i, task.ID)
		}
		seenIDs[task.ID] = true

		_, err := uuid.Parse(task.ID)
		if err != nil {
			t.Errorf("Performance test: invalid UUID at index %d: %s", i, task.ID)
		}
	}

	t.Logf("Performance test: Generated %d unique UUIDs in %v", numTasks, duration)
}

// TestUUIDDeterministicGeneration tests that UUID generation is deterministic (Issue #247)
// Same comment ID + task index should always produce the same UUID across multiple runs
func TestUUIDDeterministicGeneration(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Generate UUIDs multiple times with identical input data
	const numIterations = 100
	const numTasksPerIteration = 50

	// Store first iteration results as baseline
	var baselineIDs []string

	for iteration := 0; iteration < numIterations; iteration++ {
		// Create identical task requests for each iteration
		taskRequests := make([]TaskRequest, numTasksPerIteration)
		for i := 0; i < numTasksPerIteration; i++ {
			taskRequests[i] = TaskRequest{
				Description:     "Deterministic test task",
				OriginText:      "Identical original comment text",
				Priority:        "medium",
				SourceReviewID:  12345,
				SourceCommentID: 67890, // Same comment ID for all
				File:            "test.go",
				Line:            42,
				TaskIndex:       i,
			}
		}

		// Convert to storage tasks
		storageTasks := analyzer.convertToStorageTasks(taskRequests)

		if iteration == 0 {
			// Store baseline IDs from first iteration
			for _, task := range storageTasks {
				baselineIDs = append(baselineIDs, task.ID)
			}
		} else {
			// Verify all subsequent iterations produce identical IDs
			if len(storageTasks) != len(baselineIDs) {
				t.Fatalf("Iteration %d produced %d tasks, expected %d",
					iteration, len(storageTasks), len(baselineIDs))
			}

			for i, task := range storageTasks {
				if task.ID != baselineIDs[i] {
					t.Errorf("Iteration %d, task %d: ID mismatch. Expected %s, got %s (deterministic generation failed)",
						iteration, i, baselineIDs[i], task.ID)
				}
			}
		}

		// Verify all IDs are valid UUID v5
		for i, task := range storageTasks {
			parsedUUID, err := uuid.Parse(task.ID)
			if err != nil {
				t.Errorf("Iteration %d, task %d has invalid UUID '%s': %v", iteration, i, task.ID, err)
			}
			if parsedUUID.Version() != 5 {
				t.Errorf("Iteration %d, task %d has wrong UUID version: expected v5, got v%d",
					iteration, i, parsedUUID.Version())
			}
		}
	}

	t.Logf("Deterministic generation test: Verified %d iterations produced identical UUIDs for identical inputs",
		numIterations)
}

// TestTaskGenerationWithVariousCommentStructures tests UUID generation with different comment structures
func TestTaskGenerationWithVariousCommentStructures(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Test various comment structures that might occur in real usage
	testCases := []struct {
		name  string
		tasks []TaskRequest
	}{
		{
			name: "Single comment, single task",
			tasks: []TaskRequest{
				{
					Description:     "Fix the bug",
					OriginText:      "There's a bug here",
					Priority:        "high",
					SourceReviewID:  12345,
					SourceCommentID: 67890,
					File:            "main.go",
					Line:            42,
					TaskIndex:       0,
				},
			},
		},
		{
			name: "Single comment, multiple tasks",
			tasks: []TaskRequest{
				{
					Description:     "Fix memory leak",
					OriginText:      "This code has a memory leak and also needs better error handling",
					Priority:        "critical",
					SourceReviewID:  12345,
					SourceCommentID: 67891,
					File:            "memory.go",
					Line:            15,
					TaskIndex:       0,
				},
				{
					Description:     "Improve error handling",
					OriginText:      "This code has a memory leak and also needs better error handling",
					Priority:        "high",
					SourceReviewID:  12345,
					SourceCommentID: 67891,
					File:            "memory.go",
					Line:            15,
					TaskIndex:       1,
				},
			},
		},
		{
			name: "Multiple comments from same review",
			tasks: []TaskRequest{
				{
					Description:     "Add documentation",
					OriginText:      "This function needs documentation",
					Priority:        "medium",
					SourceReviewID:  12345,
					SourceCommentID: 67892,
					File:            "utils.go",
					Line:            25,
					TaskIndex:       0,
				},
				{
					Description:     "Fix naming convention",
					OriginText:      "Variable name should follow camelCase",
					Priority:        "low",
					SourceReviewID:  12345,
					SourceCommentID: 67893,
					File:            "utils.go",
					Line:            30,
					TaskIndex:       0,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storageTasks := analyzer.convertToStorageTasks(tc.tasks)

			if len(storageTasks) != len(tc.tasks) {
				t.Errorf("Test case '%s': expected %d tasks, got %d",
					tc.name, len(tc.tasks), len(storageTasks))
			}

			// Verify all UUIDs are unique and valid
			seenIDs := make(map[string]bool)
			for i, task := range storageTasks {
				// Verify UUID validity
				_, err := uuid.Parse(task.ID)
				if err != nil {
					t.Errorf("Test case '%s': task %d has invalid UUID '%s': %v",
						tc.name, i, task.ID, err)
				}

				// Verify uniqueness
				if seenIDs[task.ID] {
					t.Errorf("Test case '%s': duplicate UUID found: %s", tc.name, task.ID)
				}
				seenIDs[task.ID] = true
			}
		})
	}
}
