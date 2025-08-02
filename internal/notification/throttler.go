package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

// Throttler manages comment rate limiting and batching
type Throttler struct {
	config          config.ThrottlingSettings
	recentComments  []CommentRecord
	batchQueue      map[string]*CommentBatch
	mu              sync.Mutex
	aiThrottler     *AIThrottler // Optional AI-powered throttling
}

// CommentRecord tracks a posted comment for rate limiting
type CommentRecord struct {
	TaskID        string
	PR            int
	ReviewerLogin string
	Type          string
	Timestamp     time.Time
}

// CommentBatch represents a batch of comments to be sent together
type CommentBatch struct {
	ID       string
	PR       int
	Comments []BatchedComment
	Created  time.Time
}

// BatchedComment represents a single comment in a batch
type BatchedComment struct {
	TaskID  string
	Type    string
	Content string
}

// BatchSuggestion provides information about batching
type BatchSuggestion struct {
	ShouldBatch bool
	BatchID     string
	Reason      string
}

// NewThrottler creates a new Throttler instance
func NewThrottler(config config.ThrottlingSettings) *Throttler {
	return &Throttler{
		config:         config,
		recentComments: make([]CommentRecord, 0),
		batchQueue:     make(map[string]*CommentBatch),
	}
}

// SetAIThrottler sets an AI throttler for enhanced decisions
func (t *Throttler) SetAIThrottler(aiThrottler *AIThrottler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.aiThrottler = aiThrottler
}

// ShouldPostNow determines if a comment should be posted immediately
func (t *Throttler) ShouldPostNow(task *storage.Task, notificationType string) (bool, *BatchSuggestion) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.config.Enabled {
		return true, nil
	}

	// Clean up old records
	t.cleanupOldRecords()

	// Check rate limiting
	if t.isRateLimited() {
		return false, &BatchSuggestion{
			ShouldBatch: t.config.BatchSimilarComments,
			BatchID:     t.getBatchID(task.PR),
			Reason:      "Rate limit exceeded",
		}
	}

	// Check if we should batch similar comments
	if t.config.BatchSimilarComments && t.shouldBatchComment(task, notificationType) {
		return false, &BatchSuggestion{
			ShouldBatch: true,
			BatchID:     t.getBatchID(task.PR),
			Reason:      "Batching similar comments",
		}
	}

	return true, nil
}

// RecordComment records a posted comment for rate limiting
func (t *Throttler) RecordComment(task *storage.Task, notificationType string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	record := CommentRecord{
		TaskID:        task.ID,
		PR:            task.PR,
		ReviewerLogin: task.ReviewerLogin,
		Type:          notificationType,
		Timestamp:     time.Now(),
	}

	t.recentComments = append(t.recentComments, record)
}

// AddToBatch adds a comment to the batch queue
func (t *Throttler) AddToBatch(task *storage.Task, comment string, notificationType string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	batchID := t.getBatchID(task.PR)
	batch, exists := t.batchQueue[batchID]
	if !exists {
		batch = &CommentBatch{
			ID:       batchID,
			PR:       task.PR,
			Comments: make([]BatchedComment, 0),
			Created:  time.Now(),
		}
		t.batchQueue[batchID] = batch
	}

	batch.Comments = append(batch.Comments, BatchedComment{
		TaskID:  task.ID,
		Type:    notificationType,
		Content: comment,
	})

	return nil
}

// GetReadyBatches returns batches that are ready to be sent
func (t *Throttler) GetReadyBatches() []*CommentBatch {
	t.mu.Lock()
	defer t.mu.Unlock()

	readyBatches := make([]*CommentBatch, 0)
	windowDuration := time.Duration(t.config.BatchWindowMinutes) * time.Minute

	for _, batch := range t.batchQueue {
		if time.Since(batch.Created) >= windowDuration {
			readyBatches = append(readyBatches, batch)
		}
	}

	return readyBatches
}

// ClearBatch removes a batch from the queue
func (t *Throttler) ClearBatch(batchID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.batchQueue, batchID)
}

// cleanupOldRecords removes comment records older than 1 hour
func (t *Throttler) cleanupOldRecords() {
	cutoff := time.Now().Add(-1 * time.Hour)
	newRecords := make([]CommentRecord, 0)

	for _, record := range t.recentComments {
		if record.Timestamp.After(cutoff) {
			newRecords = append(newRecords, record)
		}
	}

	t.recentComments = newRecords
}

// isRateLimited checks if we've exceeded the rate limit
func (t *Throttler) isRateLimited() bool {
	return len(t.recentComments) >= t.config.MaxCommentsPerHour
}

// shouldBatchComment determines if a comment should be batched
func (t *Throttler) shouldBatchComment(task *storage.Task, notificationType string) bool {
	// Count recent comments to the same PR
	prCommentCount := 0
	cutoff := time.Now().Add(-30 * time.Minute)

	for _, record := range t.recentComments {
		if record.PR == task.PR && record.Timestamp.After(cutoff) {
			prCommentCount++
		}
	}

	// Batch if we've sent multiple comments to the same PR recently
	return prCommentCount >= 3
}

// getBatchID generates a unique batch ID for a PR
func (t *Throttler) getBatchID(pr int) string {
	return fmt.Sprintf("%s-pr-%d", time.Now().Format("2006-01-02"), pr)
}

// OptimizeBatches uses AI to reorganize pending batches for optimal delivery
func (t *Throttler) OptimizeBatches(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.aiThrottler == nil {
		return nil // No AI throttler available
	}

	// Collect all pending comments
	var pendingComments []PendingComment
	for _, batch := range t.batchQueue {
		for _, comment := range batch.Comments {
			pendingComments = append(pendingComments, PendingComment{
				Task: &storage.Task{
					ID: comment.TaskID,
					PR: batch.PR,
				},
				Comment:          comment.Content,
				NotificationType: comment.Type,
				CreatedAt:        batch.Created,
			})
		}
	}

	if len(pendingComments) == 0 {
		return nil
	}

	// Get AI batching decision
	decision, err := t.aiThrottler.AnalyzeBatchingStrategy(ctx, pendingComments, t.recentComments)
	if err != nil {
		return fmt.Errorf("AI batch optimization failed: %w", err)
	}

	// Reorganize batches based on AI decision
	if decision.ShouldBatch {
		// Clear existing batches
		t.batchQueue = make(map[string]*CommentBatch)

		// Create new optimized batches
		for _, group := range decision.BatchGroups {
			batch := &CommentBatch{
				ID:       group.GroupID,
				PR:       0, // Will be set from first task
				Comments: make([]BatchedComment, 0),
				Created:  time.Now(),
			}

			// Add tasks to this batch
			for _, taskID := range group.TaskIDs {
				// Find the pending comment for this task
				for _, pc := range pendingComments {
					if pc.Task.ID == taskID {
						if batch.PR == 0 {
							batch.PR = pc.Task.PR
						}
						batch.Comments = append(batch.Comments, BatchedComment{
							TaskID:  taskID,
							Type:    pc.NotificationType,
							Content: pc.Comment,
						})
						break
					}
				}
			}

			if len(batch.Comments) > 0 {
				t.batchQueue[batch.ID] = batch
			}
		}
	}

	return nil
}

// GetAIBatchRecommendation gets AI recommendation for a specific comment
func (t *Throttler) GetAIBatchRecommendation(ctx context.Context, task *storage.Task, notificationType string) (bool, time.Duration, error) {
	t.mu.Lock()
	aiThrottler := t.aiThrottler
	recentComments := make([]CommentRecord, len(t.recentComments))
	copy(recentComments, t.recentComments)
	t.mu.Unlock()

	if aiThrottler == nil {
		return true, 0, nil // No AI available, send immediately
	}

	// Get reviewer-specific activity
	var reviewerActivity []CommentRecord
	for _, rec := range recentComments {
		if rec.ReviewerLogin == task.ReviewerLogin {
			reviewerActivity = append(reviewerActivity, rec)
		}
	}

	return aiThrottler.OptimizeCommentTiming(ctx, task, notificationType, reviewerActivity)
}