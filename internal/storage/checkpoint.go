package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CheckpointState represents the state of incremental processing
type CheckpointState struct {
	PRNumber            int              `json:"pr_number"`
	ProcessedComments   map[int64]string `json:"processed_comments"` // comment_id -> hash
	LastProcessedAt     time.Time        `json:"last_processed_at"`
	TotalComments       int              `json:"total_comments"`
	ProcessedCount      int              `json:"processed_count"`
	BatchSize           int              `json:"batch_size"`
	StartedAt           time.Time        `json:"started_at"`
	PartialTasks        []Task           `json:"partial_tasks,omitempty"`
	LastProcessedReview int64            `json:"last_processed_review_id,omitempty"`
	LastProcessedIndex  int              `json:"last_processed_index,omitempty"`
}

// SaveCheckpoint saves the current processing checkpoint
func (m *Manager) SaveCheckpoint(prNumber int, checkpoint *CheckpointState) error {
	dir := m.getPRDir(prNumber)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	checkpointPath := filepath.Join(dir, "checkpoint.json")
	checkpoint.LastProcessedAt = time.Now()

	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(checkpointPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	return nil
}

// LoadCheckpoint loads the processing checkpoint if it exists
func (m *Manager) LoadCheckpoint(prNumber int) (*CheckpointState, error) {
	checkpointPath := filepath.Join(m.getPRDir(prNumber), "checkpoint.json")

	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No checkpoint exists
		}
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	var checkpoint CheckpointState
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// DeleteCheckpoint removes the checkpoint file after successful completion
func (m *Manager) DeleteCheckpoint(prNumber int) error {
	checkpointPath := filepath.Join(m.getPRDir(prNumber), "checkpoint.json")

	if err := os.Remove(checkpointPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	return nil
}

// IsCheckpointStale checks if a checkpoint is too old to be used
func IsCheckpointStale(checkpoint *CheckpointState, maxAge time.Duration) bool {
	if checkpoint == nil {
		return true
	}
	return time.Since(checkpoint.LastProcessedAt) > maxAge
}
