package ai

import (
	"context"
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

	if opts.ShowProgress {
		fmt.Printf("📊 Progress: %d/%d comments already processed\n", checkpoint.ProcessedCount, checkpoint.TotalComments)
		if len(remainingComments) == 0 {
			fmt.Println("✅ All comments already processed!")
			return checkpoint.PartialTasks, nil
		}
		fmt.Printf("🔄 Processing %d remaining comments in batches of %d\n", len(remainingComments), opts.BatchSize)
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
				fmt.Printf("⚠️  Failed to save checkpoint: %v\n", err)
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

		if opts.ShowProgress {
			fmt.Printf("\n🔄 Processing batch %d-%d of %d comments...\n", i+1, end, len(remainingComments))
		}

		// Process batch
		batchTasks, err := a.processBatch(batch, opts)
		if err != nil {
			fmt.Printf("⚠️  Batch processing error: %v\n", err)
			// Continue with next batch on error
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
			fmt.Printf("⚠️  Failed to save checkpoint: %v\n", err)
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
		fmt.Printf("⚠️  Failed to delete checkpoint: %v\n", err)
	}

	// Apply deduplication
	if a.config.AISettings.DeduplicationEnabled {
		deduped := a.deduplicateTasks(allTasks)
		if opts.ShowProgress && len(deduped) < len(allTasks) {
			fmt.Printf("\n🔄 Deduplication: %d tasks → %d tasks (removed %d duplicates)\n",
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
		fmt.Printf("📝 Filtered out %d resolved comments\n", resolvedCommentCount)
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
				fmt.Printf("✅ Resuming from checkpoint (processed %d/%d comments)\n",
					checkpoint.ProcessedCount, checkpoint.TotalComments)
				return checkpoint, nil
			}
			fmt.Println("⚠️  Checkpoint is too old, starting fresh")
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
			fmt.Printf("  ⚠️  %v\n", err)
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
			fmt.Printf("  ⚠️  Fast mode processing error: %v\n", err)
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

	return fmt.Sprintf(`Analyze this GitHub PR comment and generate actionable tasks.

%s

Comment: %s
File: %s:%d

Return JSON array:
[{
  "description": "Task description",
  "origin_text": "%s",
  "priority": "high",
  "source_review_id": %d,
  "source_comment_id": %d,
  "file": "%s",
  "line": %d,
  "task_index": 0
}]

Only create tasks for actionable items. Return empty array [] if no action needed.`,
		languageInstruction,
		ctx.Comment.Body,
		ctx.Comment.File,
		ctx.Comment.Line,
		ctx.Comment.Body,
		ctx.SourceReview.ID,
		ctx.Comment.ID,
		ctx.Comment.File,
		ctx.Comment.Line)
}
