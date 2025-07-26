package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointOperations(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-checkpoint-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create manager with custom base directory
	manager := &Manager{
		baseDir: tempDir,
	}

	t.Run("SaveAndLoadCheckpoint", func(t *testing.T) {
		prNumber := 123
		checkpoint := &CheckpointState{
			PRNumber: prNumber,
			ProcessedComments: map[int64]string{
				100: "hash1",
				101: "hash2",
			},
			TotalComments:       10,
			ProcessedCount:      2,
			BatchSize:           5,
			StartedAt:           time.Now().Add(-10 * time.Minute),
			LastProcessedReview: 200,
			LastProcessedIndex:  1,
		}

		// Save checkpoint
		err := manager.SaveCheckpoint(prNumber, checkpoint)
		assert.NoError(t, err)

		// Verify file exists
		checkpointPath := filepath.Join(tempDir, "PR-123", "checkpoint.json")
		assert.FileExists(t, checkpointPath)

		// Load checkpoint
		loaded, err := manager.LoadCheckpoint(prNumber)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)

		// Verify loaded data
		assert.Equal(t, checkpoint.PRNumber, loaded.PRNumber)
		assert.Equal(t, checkpoint.ProcessedComments, loaded.ProcessedComments)
		assert.Equal(t, checkpoint.TotalComments, loaded.TotalComments)
		assert.Equal(t, checkpoint.ProcessedCount, loaded.ProcessedCount)
		assert.Equal(t, checkpoint.BatchSize, loaded.BatchSize)
		assert.Equal(t, checkpoint.LastProcessedReview, loaded.LastProcessedReview)
		assert.Equal(t, checkpoint.LastProcessedIndex, loaded.LastProcessedIndex)
		assert.True(t, loaded.LastProcessedAt.After(checkpoint.StartedAt))
	})

	t.Run("LoadNonExistentCheckpoint", func(t *testing.T) {
		checkpoint, err := manager.LoadCheckpoint(999)
		assert.NoError(t, err)
		assert.Nil(t, checkpoint)
	})

	t.Run("DeleteCheckpoint", func(t *testing.T) {
		prNumber := 456
		checkpoint := &CheckpointState{
			PRNumber:       prNumber,
			TotalComments:  5,
			ProcessedCount: 5,
		}

		// Save checkpoint
		err := manager.SaveCheckpoint(prNumber, checkpoint)
		assert.NoError(t, err)

		// Verify it exists
		loaded, err := manager.LoadCheckpoint(prNumber)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)

		// Delete checkpoint
		err = manager.DeleteCheckpoint(prNumber)
		assert.NoError(t, err)

		// Verify it's gone
		loaded, err = manager.LoadCheckpoint(prNumber)
		assert.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("DeleteNonExistentCheckpoint", func(t *testing.T) {
		err := manager.DeleteCheckpoint(789)
		assert.NoError(t, err) // Should not error on non-existent file
	})

	t.Run("CheckpointWithPartialTasks", func(t *testing.T) {
		prNumber := 789
		tasks := []Task{
			{
				ID:          "task-1",
				Description: "Fix bug",
				Priority:    "high",
				Status:      "todo",
			},
			{
				ID:          "task-2",
				Description: "Add test",
				Priority:    "medium",
				Status:      "todo",
			},
		}

		checkpoint := &CheckpointState{
			PRNumber:       prNumber,
			PartialTasks:   tasks,
			TotalComments:  10,
			ProcessedCount: 2,
		}

		// Save checkpoint with tasks
		err := manager.SaveCheckpoint(prNumber, checkpoint)
		assert.NoError(t, err)

		// Load and verify tasks
		loaded, err := manager.LoadCheckpoint(prNumber)
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Len(t, loaded.PartialTasks, 2)
		assert.Equal(t, tasks[0].ID, loaded.PartialTasks[0].ID)
		assert.Equal(t, tasks[1].Description, loaded.PartialTasks[1].Description)
	})
}

func TestIsCheckpointStale(t *testing.T) {
	tests := []struct {
		name       string
		checkpoint *CheckpointState
		maxAge     time.Duration
		expected   bool
	}{
		{
			name:       "NilCheckpoint",
			checkpoint: nil,
			maxAge:     time.Hour,
			expected:   true,
		},
		{
			name: "FreshCheckpoint",
			checkpoint: &CheckpointState{
				LastProcessedAt: time.Now(),
			},
			maxAge:   time.Hour,
			expected: false,
		},
		{
			name: "StaleCheckpoint",
			checkpoint: &CheckpointState{
				LastProcessedAt: time.Now().Add(-2 * time.Hour),
			},
			maxAge:   time.Hour,
			expected: true,
		},
		{
			name: "ExactlyAtMaxAge",
			checkpoint: &CheckpointState{
				LastProcessedAt: time.Now().Add(-time.Hour),
			},
			maxAge:   time.Hour,
			expected: true, // Should be considered stale at exactly max age
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCheckpointStale(tt.checkpoint, tt.maxAge)
			assert.Equal(t, tt.expected, result)
		})
	}
}
