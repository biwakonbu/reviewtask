package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// IncrementalOptions contains options for incremental processing
type IncrementalOptions struct {
	BatchSize       int
	Resume          bool
	FastMode        bool
	MaxTimeout      time.Duration
	ShowProgress    bool
	OnProgress      func(processed, total int)
	OnBatchComplete func(batchTasks []storage.Task)
}

// GenerateTasksIncremental processes reviews incrementally with checkpointing
func (a *Analyzer) GenerateTasksIncremental(reviews []github.Review, prNumber int, storageManager *storage.Manager, opts IncrementalOptions) ([]storage.Task, error) {
	// Clear validation feedback to ensure clean state
	a.clearValidationFeedback()

	if len(reviews) == 0 {
		return []storage.Task{}, nil
	}

	// Extract all comments with filtering
	allComments := a.extractComments(reviews)
	if len(allComments) == 0 {
		return []storage.Task{}, nil
	}

	// Load or create checkpoint
	checkpoint, err := a.loadOrCreateCheckpoint(prNumber, storageManager, allComments, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Filter out already processed comments if resuming
	remainingComments := a.filterProcessedComments(allComments, checkpoint)

	if opts.ShowProgress && checkpoint.ProcessedCount > 0 {
		printf("✅ Resuming from checkpoint: %d/%d comments already processed\n", checkpoint.ProcessedCount, checkpoint.TotalComments)
	}

	if len(remainingComments) == 0 {
		return checkpoint.PartialTasks, nil
	}

	// Process in batches with timeout and checkpointing
	ctx, cancel := context.WithTimeout(context.Background(), opts.MaxTimeout)
	defer cancel()

	allTasks := append([]storage.Task{}, checkpoint.PartialTasks...)

	for i := 0; i < len(remainingComments); i += opts.BatchSize {
		select {
		case <-ctx.Done():
			// Save checkpoint before timeout
			checkpoint.PartialTasks = allTasks
			if err := storageManager.SaveCheckpoint(prNumber, checkpoint); err != nil {
				printf("⚠️  Failed to save checkpoint: %v\n", err)
				return nil, fmt.Errorf("processing timed out after %v and failed to save checkpoint: %w", opts.MaxTimeout, err)
			}
			return nil, fmt.Errorf("processing timed out after %v. Use --resume to continue", opts.MaxTimeout)
		default:
		}

		// Calculate batch boundaries
		end := i + opts.BatchSize
		if end > len(remainingComments) {
			end = len(remainingComments)
		}

		batch := remainingComments[i:end]

		// No need for batch-level progress messages since we have real-time progress

		// Process batch
		batchTasks, err := a.processBatch(batch, opts)
		if err != nil {
			// Save checkpoint before continuing
			checkpoint.PartialTasks = allTasks
			if saveErr := storageManager.SaveCheckpoint(prNumber, checkpoint); saveErr != nil {
				printf("⚠️  Failed to save checkpoint: %v\n", saveErr)
				// For critical errors with checkpoint save failure, return both
				if isCriticalError(err) {
					return nil, fmt.Errorf("critical error: %w, and failed to save checkpoint: %w", err, saveErr)
				}
			}

			// For critical errors, return immediately
			if isCriticalError(err) {
				return nil, fmt.Errorf("critical error: %w. Run 'reviewtask' again to resume from checkpoint", err)
			}

			// For other errors, log and continue
			printf("⚠️  Some comments could not be processed: %v\n", err)
			continue
		}

		// Update checkpoint
		for _, commentCtx := range batch {
			checkpoint.ProcessedComments[commentCtx.Comment.ID] = a.calculateCommentHash(commentCtx.Comment)
			checkpoint.ProcessedCount++
		}

		allTasks = append(allTasks, batchTasks...)
		checkpoint.PartialTasks = allTasks

		// Save checkpoint after each batch
		if err := storageManager.SaveCheckpoint(prNumber, checkpoint); err != nil {
			printf("⚠️  Failed to save checkpoint: %v\n", err)
		}

		// Call progress callbacks
		if opts.OnProgress != nil {
			opts.OnProgress(checkpoint.ProcessedCount, checkpoint.TotalComments)
		}
		if opts.OnBatchComplete != nil {
			opts.OnBatchComplete(batchTasks)
		}

		// Add small delay to prevent API rate limiting
		if !opts.FastMode && i+opts.BatchSize < len(remainingComments) {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Delete checkpoint on successful completion
	if err := storageManager.DeleteCheckpoint(prNumber); err != nil {
		printf("⚠️  Failed to delete checkpoint: %v\n", err)
	}

	// Apply deduplication
	if a.config.AISettings.DeduplicationEnabled {
		deduped := a.deduplicateTasks(allTasks)
		if opts.ShowProgress && len(deduped) < len(allTasks) {
			printf("\n🔄 Deduplication: %d tasks → %d tasks (removed %d duplicates)\n",
				len(allTasks), len(deduped), len(allTasks)-len(deduped))
		}
		return deduped, nil
	}

	return allTasks, nil
}

// extractComments extracts all comments from reviews with filtering
func (a *Analyzer) extractComments(reviews []github.Review) []CommentContext {
	var allComments []CommentContext
	resolvedCommentCount := 0

	for _, review := range reviews {
		// Process review body as a comment if it exists
		if review.Body != "" {
			reviewBodyComment := github.Comment{
				ID:        review.ID,
				File:      "",
				Line:      0,
				Body:      review.Body,
				Author:    review.Reviewer,
				CreatedAt: review.SubmittedAt,
			}

			if !a.isCommentResolved(reviewBodyComment) {
				allComments = append(allComments, CommentContext{
					Comment:      reviewBodyComment,
					SourceReview: review,
				})
			} else {
				resolvedCommentCount++
			}
		}

		// Process individual inline comments
		for _, comment := range review.Comments {
			if a.isCommentResolved(comment) {
				resolvedCommentCount++
				continue
			}

			allComments = append(allComments, CommentContext{
				Comment:      comment,
				SourceReview: review,
			})
		}
	}

	if resolvedCommentCount > 0 {
		printf("📝 Filtered out %d resolved comments\n", resolvedCommentCount)
	}

	return allComments
}

// loadOrCreateCheckpoint loads existing checkpoint or creates new one
func (a *Analyzer) loadOrCreateCheckpoint(prNumber int, storageManager *storage.Manager, comments []CommentContext, opts IncrementalOptions) (*storage.CheckpointState, error) {
	if opts.Resume {
		checkpoint, err := storageManager.LoadCheckpoint(prNumber)
		if err != nil {
			return nil, err
		}

		if checkpoint != nil {
			// Check if checkpoint is still valid (not too old)
			if !storage.IsCheckpointStale(checkpoint, 24*time.Hour) {
				printf("✅ Resuming from checkpoint (processed %d/%d comments)\n",
					checkpoint.ProcessedCount, checkpoint.TotalComments)
				return checkpoint, nil
			}
			println("⚠️  Checkpoint is too old, starting fresh")
		}
	}

	// Create new checkpoint
	checkpoint := &storage.CheckpointState{
		PRNumber:          prNumber,
		ProcessedComments: make(map[int64]string),
		StartedAt:         time.Now(),
		TotalComments:     len(comments),
		ProcessedCount:    0,
		BatchSize:         opts.BatchSize,
		PartialTasks:      []storage.Task{},
	}

	return checkpoint, nil
}

// filterProcessedComments removes already processed comments based on checkpoint
func (a *Analyzer) filterProcessedComments(comments []CommentContext, checkpoint *storage.CheckpointState) []CommentContext {
	if len(checkpoint.ProcessedComments) == 0 {
		return comments
	}

	var remaining []CommentContext
	for _, commentCtx := range comments {
		hash, exists := checkpoint.ProcessedComments[commentCtx.Comment.ID]
		currentHash := a.calculateCommentHash(commentCtx.Comment)

		// Include if not processed or if content changed
		if !exists || hash != currentHash {
			remaining = append(remaining, commentCtx)
		}
	}

	return remaining
}

// processBatch processes a batch of comments with optimizations
func (a *Analyzer) processBatch(batch []CommentContext, opts IncrementalOptions) ([]storage.Task, error) {
	if len(batch) == 0 {
		return []storage.Task{}, nil
	}

	// Use fast mode optimizations
	if opts.FastMode {
		// Process with reduced validation and simpler prompts
		return a.processBatchFastMode(batch)
	}

	// Check if validation is enabled
	if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
		return a.processBatchWithValidation(batch)
	}

	// Standard parallel processing
	return a.processBatchStandard(batch)
}

// processBatchStandard processes batch with standard parallel processing
func (a *Analyzer) processBatchStandard(batch []CommentContext) ([]storage.Task, error) {
	type commentResult struct {
		tasks []TaskRequest
		err   error
		index int
	}

	results := make(chan commentResult, len(batch))
	var wg sync.WaitGroup

	// Process each comment in parallel
	for i, commentCtx := range batch {
		wg.Add(1)
		go func(index int, ctx CommentContext) {
			defer wg.Done()

			tasks, err := a.processComment(ctx)
			results <- commentResult{
				tasks: tasks,
				err:   err,
				index: index,
			}
		}(i, commentCtx)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allTasks []TaskRequest
	var errors []error

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Errorf("comment %d: %w", result.index, result.err))
		} else {
			allTasks = append(allTasks, result.tasks...)
		}
	}

	// Report errors but continue if we have some successful results
	if len(errors) > 0 {
		for _, err := range errors {
			printf("  ⚠️  %v\n", err)
		}
		if len(allTasks) == 0 {
			return nil, fmt.Errorf("all comment processing failed")
		}
	}

	// Convert to storage tasks
	return a.convertToStorageTasks(allTasks), nil
}

// processBatchWithValidation processes batch with validation enabled
func (a *Analyzer) processBatchWithValidation(batch []CommentContext) ([]storage.Task, error) {
	return a.processCommentsParallel(batch, a.processCommentWithValidation)
}

// processBatchFastMode processes batch with fast mode optimizations
func (a *Analyzer) processBatchFastMode(batch []CommentContext) ([]storage.Task, error) {
	// In fast mode, we skip validation and use simpler prompts
	var allTasks []TaskRequest

	// Process comments with minimal overhead
	for _, commentCtx := range batch {
		// Skip very short comments in fast mode
		if len(commentCtx.Comment.Body) < 20 {
			continue
		}

		// Use simplified prompt for speed
		prompt := a.buildFastModePrompt(commentCtx)
		tasks, err := a.callClaudeCode(prompt)
		if err != nil {
			printf("  ⚠️  Fast mode processing error: %v\n", err)
			continue
		}

		allTasks = append(allTasks, tasks...)
	}

	return a.convertToStorageTasks(allTasks), nil
}

// buildFastModePrompt creates a simplified prompt for fast processing
func (a *Analyzer) buildFastModePrompt(ctx CommentContext) string {
	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" {
		languageInstruction = fmt.Sprintf("Generate task descriptions in %s.\n", a.config.AISettings.UserLanguage)
	}

	// Build example task using proper JSON marshaling
	exampleTask := map[string]interface{}{
		"description":       "Task description",
		"origin_text":       ctx.Comment.Body,
		"priority":          "medium", // Use neutral priority to avoid bias
		"source_review_id":  ctx.SourceReview.ID,
		"source_comment_id": ctx.Comment.ID,
		"file":              ctx.Comment.File,
		"line":              ctx.Comment.Line,
		"task_index":        0,
	}

	exampleJSON, err := json.MarshalIndent([]interface{}{exampleTask}, "", "  ")
	if err != nil {
		// Fallback to simple format if marshaling fails
		exampleJSON = []byte(`[{"description": "Task description", "origin_text": "...", "priority": "medium", "task_index": 0}]`)
	}

	return fmt.Sprintf(`Analyze this GitHub PR comment and generate actionable tasks.

%s

Comment: %s
File: %s:%d

Return JSON array:
%s

Only create tasks for actionable items. Return empty array [] if no action needed.`,
		languageInstruction,
		ctx.Comment.Body,
		ctx.Comment.File,
		ctx.Comment.Line,
		string(exampleJSON))
}
