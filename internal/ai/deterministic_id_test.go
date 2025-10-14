package ai

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"reviewtask/internal/config"
)

// TestGenerateDeterministicTaskID_Idempotency verifies that the same comment ID
// and task index always produce the same task ID (idempotency guarantee).
func TestGenerateDeterministicTaskID_Idempotency(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Generate ID multiple times with same inputs
	commentID := int64(12345)
	taskIndex := 0

	id1 := analyzer.generateDeterministicTaskID(commentID, taskIndex)
	id2 := analyzer.generateDeterministicTaskID(commentID, taskIndex)
	id3 := analyzer.generateDeterministicTaskID(commentID, taskIndex)

	// All IDs should be identical
	assert.Equal(t, id1, id2, "Same inputs should produce same ID")
	assert.Equal(t, id2, id3, "Same inputs should produce same ID")
	assert.Equal(t, id1, id3, "Same inputs should produce same ID")

	// Verify it's a valid UUID
	parsedUUID, err := uuid.Parse(id1)
	assert.NoError(t, err, "Generated ID should be valid UUID")
	assert.Equal(t, uuid.Version(5), parsedUUID.Version(), "Should be UUID v5 (deterministic)")
}

// TestGenerateDeterministicTaskID_Uniqueness verifies that different inputs
// produce different task IDs (uniqueness guarantee).
func TestGenerateDeterministicTaskID_Uniqueness(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Test different comment IDs
	id1 := analyzer.generateDeterministicTaskID(12345, 0)
	id2 := analyzer.generateDeterministicTaskID(67890, 0)
	assert.NotEqual(t, id1, id2, "Different comment IDs should produce different IDs")

	// Test different task indexes
	id3 := analyzer.generateDeterministicTaskID(12345, 0)
	id4 := analyzer.generateDeterministicTaskID(12345, 1)
	assert.NotEqual(t, id3, id4, "Different task indexes should produce different IDs")

	// Test both different
	id5 := analyzer.generateDeterministicTaskID(12345, 0)
	id6 := analyzer.generateDeterministicTaskID(67890, 1)
	assert.NotEqual(t, id5, id6, "Different inputs should produce different IDs")
}

// TestGenerateDeterministicTaskID_RFCCompliance verifies UUID RFC 4122 compliance.
func TestGenerateDeterministicTaskID_RFCCompliance(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	testCases := []struct {
		commentID int64
		taskIndex int
	}{
		{12345, 0},
		{67890, 1},
		{99999, 10},
		{1, 0},
		{1000000, 999},
	}

	for _, tc := range testCases {
		id := analyzer.generateDeterministicTaskID(tc.commentID, tc.taskIndex)

		// Verify valid UUID format
		parsedUUID, err := uuid.Parse(id)
		assert.NoError(t, err, "ID should be valid UUID for comment=%d, task=%d", tc.commentID, tc.taskIndex)

		// Verify UUID version 5 (SHA-1 based, deterministic)
		assert.Equal(t, uuid.Version(5), parsedUUID.Version(),
			"Should be UUID v5 for comment=%d, task=%d", tc.commentID, tc.taskIndex)

		// Verify UUID variant (RFC 4122)
		assert.Equal(t, uuid.RFC4122, parsedUUID.Variant(),
			"Should be RFC 4122 variant for comment=%d, task=%d", tc.commentID, tc.taskIndex)
	}
}

// TestConvertToStorageTasks_DeterministicIDs verifies that convertToStorageTasks
// uses deterministic ID generation.
func TestConvertToStorageTasks_DeterministicIDs(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
		AISettings: config.AISettings{
			UserLanguage: "English",
		},
	}
	analyzer := NewAnalyzer(cfg)

	taskRequests := []TaskRequest{
		{
			Description:     "Task 1",
			OriginText:      "Original comment 1",
			Priority:        "high",
			SourceReviewID:  100,
			SourceCommentID: 12345,
			TaskIndex:       0,
		},
		{
			Description:     "Task 2",
			OriginText:      "Original comment 1",
			Priority:        "medium",
			SourceReviewID:  100,
			SourceCommentID: 12345,
			TaskIndex:       1,
		},
	}

	// Convert multiple times
	storageTasks1 := analyzer.convertToStorageTasks(taskRequests)
	storageTasks2 := analyzer.convertToStorageTasks(taskRequests)

	// IDs should be identical across conversions
	assert.Equal(t, storageTasks1[0].ID, storageTasks2[0].ID,
		"Same comment+index should produce same ID")
	assert.Equal(t, storageTasks1[1].ID, storageTasks2[1].ID,
		"Same comment+index should produce same ID")

	// Different task indexes should have different IDs
	assert.NotEqual(t, storageTasks1[0].ID, storageTasks1[1].ID,
		"Different task indexes should produce different IDs")
}

// TestDeterministicID_MultipleRunsPreventsD uplicates verifies the fix for Issue #247:
// Running reviewtask multiple times should not create duplicate tasks.
func TestDeterministicID_MultipleRunsPreventsDuplicates(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Simulate same comment being processed multiple times
	commentID := int64(12345)
	taskIndex := 0

	// Run 1: Generate task ID
	id1 := analyzer.generateDeterministicTaskID(commentID, taskIndex)

	// Run 2: Generate task ID again (simulating reviewtask running again)
	id2 := analyzer.generateDeterministicTaskID(commentID, taskIndex)

	// Run 3: Generate task ID again
	id3 := analyzer.generateDeterministicTaskID(commentID, taskIndex)

	// All runs should produce the same ID, allowing WriteWorker to deduplicate
	assert.Equal(t, id1, id2, "Multiple runs should produce same ID (prevents duplicates)")
	assert.Equal(t, id2, id3, "Multiple runs should produce same ID (prevents duplicates)")
	assert.Equal(t, id1, id3, "Multiple runs should produce same ID (prevents duplicates)")
}

// TestDeterministicID_StabilityAcrossLargeRange verifies ID generation stability
// across a realistic range of comment IDs and task indexes.
func TestDeterministicID_StabilityAcrossLargeRange(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Test with realistic comment ID ranges
	commentIDs := []int64{1, 1000, 10000, 100000, 1000000, 9999999999}
	taskIndexes := []int{0, 1, 5, 10, 50, 100}

	idMap := make(map[string]bool)

	for _, commentID := range commentIDs {
		for _, taskIndex := range taskIndexes {
			id := analyzer.generateDeterministicTaskID(commentID, taskIndex)

			// Verify uniqueness
			key := id
			if idMap[key] {
				t.Errorf("Duplicate ID detected: comment=%d, task=%d, id=%s", commentID, taskIndex, id)
			}
			idMap[key] = true

			// Verify valid UUID
			_, err := uuid.Parse(id)
			assert.NoError(t, err, "Invalid UUID for comment=%d, task=%d", commentID, taskIndex)

			// Verify idempotency by generating again
			id2 := analyzer.generateDeterministicTaskID(commentID, taskIndex)
			assert.Equal(t, id, id2, "Same inputs should always produce same ID")
		}
	}

	// Verify we generated correct number of unique IDs
	expectedCount := len(commentIDs) * len(taskIndexes)
	assert.Equal(t, expectedCount, len(idMap), "Should have generated %d unique IDs", expectedCount)
}
