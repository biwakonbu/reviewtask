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
			DefaultStatus:    "todo",
			AutoPrioritize:   false,
		},
		AISettings: config.AISettings{
			UserLanguage:        "English",
			OutputFormat:        "json",
			MaxRetries:          1, // Limit retries for testing
			ValidationEnabled:   &[]bool{false}[0], // Disable validation for integration test
			QualityThreshold:    0.8,
			DebugMode:          false,
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

		// Verify UUID version (should be v4)
		if parsedUUID.Version() != 4 {
			t.Errorf("Integration test task %d UUID '%s' is not version 4", i, task.ID)
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

// TestUUIDCollisionResistance tests that UUID generation is collision-resistant
func TestUUIDCollisionResistance(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Generate UUIDs multiple times with identical input data
	const numIterations = 100
	const numTasksPerIteration = 50

	allGeneratedIDs := make(map[string]bool)
	totalIDs := 0

	for iteration := 0; iteration < numIterations; iteration++ {
		// Create identical task requests for each iteration
		taskRequests := make([]TaskRequest, numTasksPerIteration)
		for i := 0; i < numTasksPerIteration; i++ {
			taskRequests[i] = TaskRequest{
				Description:     "Collision test task",
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

		// Check for collisions within this iteration
		iterationIDs := make(map[string]bool)
		for _, task := range storageTasks {
			if iterationIDs[task.ID] {
				t.Errorf("Collision test: duplicate UUID within iteration %d: %s", iteration, task.ID)
			}
			iterationIDs[task.ID] = true

			// Check for collisions across all iterations
			if allGeneratedIDs[task.ID] {
				t.Errorf("Collision test: UUID collision across iterations: %s", task.ID)
			}
			allGeneratedIDs[task.ID] = true
			totalIDs++
		}
	}

	expectedTotalIDs := numIterations * numTasksPerIteration
	if totalIDs != expectedTotalIDs {
		t.Errorf("Collision test: expected %d total IDs, got %d", expectedTotalIDs, totalIDs)
	}

	if len(allGeneratedIDs) != totalIDs {
		t.Errorf("Collision test: UUID uniqueness failed - expected %d unique IDs, got %d", 
			totalIDs, len(allGeneratedIDs))
	}

	t.Logf("Collision test: Generated %d unique UUIDs across %d iterations with identical input data", 
		totalIDs, numIterations)
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