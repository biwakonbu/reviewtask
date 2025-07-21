package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CommentHistory tracks the history of review comments to detect edits and deletions
type CommentHistory struct {
	CommentID         int64     `json:"comment_id"`
	ReviewID          int64     `json:"review_id"`
	OriginalText      string    `json:"original_text"`
	CurrentText       string    `json:"current_text"`
	FirstSeen         time.Time `json:"first_seen"`
	LastModified      time.Time `json:"last_modified"`
	IsDeleted         bool      `json:"is_deleted"`
	SemanticHash      string    `json:"semantic_hash"`      // AI-generated semantic fingerprint
	TextHash          string    `json:"text_hash"`          // SHA256 hash of current text
	ModificationCount int       `json:"modification_count"` // Number of times modified
}

// CommentHistoryManager manages the history of review comments
type CommentHistoryManager struct {
	baseDir string
	prDir   string
}

// NewCommentHistoryManager creates a new comment history manager
func NewCommentHistoryManager(prNumber int) *CommentHistoryManager {
	baseDir := ".pr-review"
	prDir := filepath.Join(baseDir, fmt.Sprintf("PR-%d", prNumber))

	return &CommentHistoryManager{
		baseDir: baseDir,
		prDir:   prDir,
	}
}

// LoadHistory loads the comment history for a PR
func (m *CommentHistoryManager) LoadHistory() (map[int64]*CommentHistory, error) {
	historyFile := filepath.Join(m.prDir, "comment_history.json")

	// If file doesn't exist, return empty map
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		return make(map[int64]*CommentHistory), nil
	}

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read comment history: %w", err)
	}

	var history map[int64]*CommentHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse comment history: %w", err)
	}

	return history, nil
}

// SaveHistory saves the comment history for a PR
func (m *CommentHistoryManager) SaveHistory(history map[int64]*CommentHistory) error {
	// Ensure directory exists
	if err := os.MkdirAll(m.prDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	historyFile := filepath.Join(m.prDir, "comment_history.json")

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comment history: %w", err)
	}

	if err := os.WriteFile(historyFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write comment history: %w", err)
	}

	return nil
}

// CalculateTextHash generates a SHA256 hash of the comment text
func CalculateTextHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// DetectChanges analyzes comment changes and returns change information
type CommentChange struct {
	Type             string // "new", "modified", "deleted", "unchanged"
	CommentID        int64
	PreviousText     string
	CurrentText      string
	IsSemanticChange bool // To be determined by AI
}

// AnalyzeCommentChanges compares current comments with history
func (m *CommentHistoryManager) AnalyzeCommentChanges(currentComments map[int64]string, history map[int64]*CommentHistory) []CommentChange {
	var changes []CommentChange

	// Check for new, modified, or unchanged comments
	for commentID, currentText := range currentComments {
		currentHash := CalculateTextHash(currentText)

		if historyEntry, exists := history[commentID]; exists {
			if historyEntry.IsDeleted {
				// Comment was deleted but now exists again
				changes = append(changes, CommentChange{
					Type:         "new", // Treat as new since it was deleted
					CommentID:    commentID,
					PreviousText: historyEntry.CurrentText,
					CurrentText:  currentText,
				})
			} else if historyEntry.TextHash != currentHash {
				// Comment was modified
				changes = append(changes, CommentChange{
					Type:         "modified",
					CommentID:    commentID,
					PreviousText: historyEntry.CurrentText,
					CurrentText:  currentText,
				})
			} else {
				// Comment unchanged
				changes = append(changes, CommentChange{
					Type:         "unchanged",
					CommentID:    commentID,
					PreviousText: historyEntry.CurrentText,
					CurrentText:  currentText,
				})
			}
		} else {
			// New comment
			changes = append(changes, CommentChange{
				Type:        "new",
				CommentID:   commentID,
				CurrentText: currentText,
			})
		}
	}

	// Check for deleted comments
	for commentID, historyEntry := range history {
		if !historyEntry.IsDeleted {
			if _, exists := currentComments[commentID]; !exists {
				changes = append(changes, CommentChange{
					Type:         "deleted",
					CommentID:    commentID,
					PreviousText: historyEntry.CurrentText,
				})
			}
		}
	}

	return changes
}

// UpdateHistory updates the comment history based on detected changes
func (m *CommentHistoryManager) UpdateHistory(changes []CommentChange, history map[int64]*CommentHistory) map[int64]*CommentHistory {
	now := time.Now()

	for _, change := range changes {
		switch change.Type {
		case "new":
			history[change.CommentID] = &CommentHistory{
				CommentID:         change.CommentID,
				OriginalText:      change.CurrentText,
				CurrentText:       change.CurrentText,
				FirstSeen:         now,
				LastModified:      now,
				IsDeleted:         false,
				TextHash:          CalculateTextHash(change.CurrentText),
				ModificationCount: 0,
			}

		case "modified":
			if entry, exists := history[change.CommentID]; exists {
				entry.CurrentText = change.CurrentText
				entry.LastModified = now
				entry.TextHash = CalculateTextHash(change.CurrentText)
				entry.ModificationCount++
				entry.IsDeleted = false
			}

		case "deleted":
			if entry, exists := history[change.CommentID]; exists {
				entry.IsDeleted = true
				entry.LastModified = now
			}
		}
	}

	return history
}
