package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FailedComment represents a comment that failed to be processed
type FailedComment struct {
	CommentID  int64     `json:"comment_id"`
	ReviewID   int64     `json:"review_id"`
	PRNumber   int       `json:"pr_number"`
	File       string    `json:"file"`
	Line       int       `json:"line"`
	Author     string    `json:"author"`
	Body       string    `json:"body"`
	URL        string    `json:"url"`
	Error      string    `json:"error"`
	ErrorType  string    `json:"error_type"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int       `json:"retry_count"`
	LastRetry  time.Time `json:"last_retry"`
	NextRetry  time.Time `json:"next_retry"`
	IsResolved bool      `json:"is_resolved"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
}

// FailedCommentsFile represents the persistent storage for failed comments
type FailedCommentsFile struct {
	Version        string          `json:"version"`
	LastUpdated    time.Time       `json:"last_updated"`
	FailedComments []FailedComment `json:"failed_comments"`
	Statistics     FailureStats    `json:"statistics"`
}

// FailureStats tracks failure statistics
type FailureStats struct {
	TotalFailures      int            `json:"total_failures"`
	ResolvedCount      int            `json:"resolved_count"`
	PendingCount       int            `json:"pending_count"`
	ByErrorType        map[string]int `json:"by_error_type"`
	LastRetryRun       time.Time      `json:"last_retry_run,omitempty"`
	NextScheduledRetry time.Time      `json:"next_scheduled_retry,omitempty"`
}

// SaveFailedComment saves a failed comment for later retry
func (m *Manager) SaveFailedComment(comment FailedComment) error {
	filePath := filepath.Join(m.baseDir, "failed_comments.json")

	// Load existing failed comments
	failedFile, err := m.LoadFailedComments()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing failed comments: %w", err)
	}

	if failedFile == nil {
		failedFile = &FailedCommentsFile{
			Version:        "1.0",
			LastUpdated:    time.Now(),
			FailedComments: []FailedComment{},
			Statistics: FailureStats{
				ByErrorType: make(map[string]int),
			},
		}
	}

	// Check if comment already exists
	found := false
	for i, existing := range failedFile.FailedComments {
		if existing.CommentID == comment.CommentID {
			// Update existing comment
			comment.RetryCount = existing.RetryCount + 1
			comment.LastRetry = time.Now()
			// Calculate next retry with exponential backoff
			comment.NextRetry = calculateNextRetry(comment.RetryCount)
			failedFile.FailedComments[i] = comment
			found = true
			break
		}
	}

	if !found {
		// Add new failed comment
		comment.Timestamp = time.Now()
		comment.RetryCount = 0
		comment.NextRetry = calculateNextRetry(0)
		failedFile.FailedComments = append(failedFile.FailedComments, comment)
	}

	// Update statistics
	failedFile.LastUpdated = time.Now()
	failedFile.Statistics = m.calculateFailureStats(failedFile.FailedComments)

	// Save to file
	data, err := json.MarshalIndent(failedFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal failed comments: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write failed comments file: %w", err)
	}

	return nil
}

// LoadFailedComments loads all failed comments from storage
func (m *Manager) LoadFailedComments() (*FailedCommentsFile, error) {
	filePath := filepath.Join(m.baseDir, "failed_comments.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read failed comments file: %w", err)
	}

	var failedFile FailedCommentsFile
	if err := json.Unmarshal(data, &failedFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal failed comments: %w", err)
	}

	return &failedFile, nil
}

// GetRetryableComments returns comments that are ready for retry
func (m *Manager) GetRetryableComments() ([]FailedComment, error) {
	failedFile, err := m.LoadFailedComments()
	if err != nil {
		return nil, err
	}

	if failedFile == nil {
		return []FailedComment{}, nil
	}

	var retryable []FailedComment
	now := time.Now()

	for _, comment := range failedFile.FailedComments {
		if !comment.IsResolved && now.After(comment.NextRetry) {
			retryable = append(retryable, comment)
		}
	}

	return retryable, nil
}

// MarkCommentResolved marks a failed comment as successfully resolved
func (m *Manager) MarkCommentResolved(commentID int64) error {
	failedFile, err := m.LoadFailedComments()
	if err != nil {
		return err
	}

	if failedFile == nil {
		return fmt.Errorf("no failed comments file found")
	}

	found := false
	for i, comment := range failedFile.FailedComments {
		if comment.CommentID == commentID {
			failedFile.FailedComments[i].IsResolved = true
			failedFile.FailedComments[i].ResolvedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("comment %d not found in failed comments", commentID)
	}

	// Update statistics
	failedFile.LastUpdated = time.Now()
	failedFile.Statistics = m.calculateFailureStats(failedFile.FailedComments)

	// Save updated file
	data, err := json.MarshalIndent(failedFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal failed comments: %w", err)
	}

	if err := os.WriteFile(filepath.Join(m.baseDir, "failed_comments.json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write failed comments file: %w", err)
	}

	return nil
}

// ClearResolvedComments removes resolved comments from the failed list
func (m *Manager) ClearResolvedComments() error {
	failedFile, err := m.LoadFailedComments()
	if err != nil {
		return err
	}

	if failedFile == nil {
		return nil
	}

	var unresolved []FailedComment
	for _, comment := range failedFile.FailedComments {
		if !comment.IsResolved {
			unresolved = append(unresolved, comment)
		}
	}

	failedFile.FailedComments = unresolved
	failedFile.LastUpdated = time.Now()
	failedFile.Statistics = m.calculateFailureStats(failedFile.FailedComments)

	// Save updated file
	data, err := json.MarshalIndent(failedFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal failed comments: %w", err)
	}

	if err := os.WriteFile(filepath.Join(m.baseDir, "failed_comments.json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write failed comments file: %w", err)
	}

	return nil
}

// calculateFailureStats computes statistics from failed comments
func (m *Manager) calculateFailureStats(comments []FailedComment) FailureStats {
	stats := FailureStats{
		TotalFailures: len(comments),
		ByErrorType:   make(map[string]int),
	}

	var nextRetry time.Time
	for _, comment := range comments {
		if comment.IsResolved {
			stats.ResolvedCount++
		} else {
			stats.PendingCount++
			if nextRetry.IsZero() || comment.NextRetry.Before(nextRetry) {
				nextRetry = comment.NextRetry
			}
		}

		if comment.ErrorType != "" {
			stats.ByErrorType[comment.ErrorType]++
		}
	}

	stats.NextScheduledRetry = nextRetry
	return stats
}

// calculateNextRetry calculates the next retry time with exponential backoff
func calculateNextRetry(retryCount int) time.Time {
	// Exponential backoff: 1min, 5min, 15min, 30min, 1hr, 2hr, 4hr, 8hr, then 24hr
	delays := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		2 * time.Hour,
		4 * time.Hour,
		8 * time.Hour,
		24 * time.Hour,
	}

	var delay time.Duration
	if retryCount < len(delays) {
		delay = delays[retryCount]
	} else {
		delay = 24 * time.Hour // Max delay
	}

	return time.Now().Add(delay)
}
