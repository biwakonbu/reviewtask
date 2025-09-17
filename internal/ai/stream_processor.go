package ai

import (
	"fmt"
	"strings"
	"sync"

	"reviewtask/internal/storage"
)

// StreamProcessor handles comment processing with streaming results
type StreamProcessor struct {
	analyzer    *Analyzer
	writeWorker *storage.WriteWorker
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(analyzer *Analyzer) *StreamProcessor {
	return &StreamProcessor{
		analyzer: analyzer,
	}
}

// SetWriteWorker sets the write worker for real-time saving
func (sp *StreamProcessor) SetWriteWorker(worker *storage.WriteWorker) {
	sp.writeWorker = worker
}

// ProcessCommentsStream processes comments with streaming results
// This allows successful tasks to be written incrementally, and failed comments to be tracked separately
func (sp *StreamProcessor) ProcessCommentsStream(comments []CommentContext, processor func(CommentContext) ([]TaskRequest, error)) ([]storage.Task, error) {
	if !sp.analyzer.config.AISettings.StreamProcessingEnabled {
		// Fallback to traditional parallel processing
		return sp.analyzer.processCommentsParallel(comments, processor)
	}

	if sp.analyzer.config.AISettings.VerboseMode {
		fmt.Printf("Processing %d comments with streaming mode...\n", len(comments))
	}

	type streamResult struct {
		tasks   []TaskRequest
		err     error
		index   int
		context CommentContext
	}

	results := make(chan streamResult, len(comments))
	var wg sync.WaitGroup

	// Process each comment in parallel (same as before)
	for i, commentCtx := range comments {
		wg.Add(1)
		go func(index int, ctx CommentContext) {
			defer wg.Done()

			tasks, err := processor(ctx)
			results <- streamResult{
				tasks:   tasks,
				err:     err,
				index:   index,
				context: ctx,
			}
		}(i, commentCtx)
	}

	// Signal completion by closing the channel after all work is done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Stream processing: handle results as they arrive
	var allTasks []TaskRequest
	var successfulResults []streamResult
	var failedResults []streamResult

	processed := 0
	for result := range results {

		processed++

		if result.err != nil {
			// Record failed result
			failedResults = append(failedResults, result)

			// Record error in error tracker
			if sp.analyzer.errorTracker != nil {
				errorType := sp.categorizeError(result.err)
				sp.analyzer.errorTracker.RecordCommentError(result.context, errorType, result.err.Error(), 0, false, 0, 0)
			}

			if sp.analyzer.config.AISettings.VerboseMode {
				fmt.Printf("  ❌ Comment %d failed: %v\n", result.context.Comment.ID, result.err)
			}
		} else {
			// Record successful result
			successfulResults = append(successfulResults, result)
			allTasks = append(allTasks, result.tasks...)

			if sp.analyzer.config.AISettings.VerboseMode {
				fmt.Printf("  ✅ Comment %d processed: %d tasks generated\n", result.context.Comment.ID, len(result.tasks))
			}
		}

		// Show progress
		if sp.analyzer.config.AISettings.VerboseMode && processed%5 == 0 {
			fmt.Printf("  📊 Progress: %d/%d comments processed (%d successful, %d failed)\n",
				processed, len(comments), len(successfulResults), len(failedResults))
		}
	}

	// Final progress report
	if sp.analyzer.config.AISettings.VerboseMode {
		fmt.Printf("  📊 Final: %d/%d comments processed (%d successful, %d failed)\n",
			processed, len(comments), len(successfulResults), len(failedResults))
	}

	// Report detailed error summary if any failures occurred
	if len(failedResults) > 0 {
		if sp.analyzer.config.AISettings.VerboseMode {
			fmt.Printf("  ⚠️  %d comment(s) failed to process:\n", len(failedResults))
			for _, failed := range failedResults {
				fmt.Printf("    • Comment %d: %v\n", failed.context.Comment.ID, failed.err)
			}
		}

		// Show error summary
		if sp.analyzer.errorTracker != nil && sp.analyzer.config.AISettings.VerboseMode {
			sp.analyzer.errorTracker.PrintErrorSummary()
		}

		// Return error only if ALL processing failed
		if len(allTasks) == 0 {
			return nil, fmt.Errorf("all comment processing failed (%d errors)", len(failedResults))
		}
	}

	// Convert to storage tasks
	storageTasks := sp.analyzer.convertToStorageTasks(allTasks)

	// Apply deduplication
	dedupedTasks := sp.analyzer.deduplicateTasks(storageTasks)

	if sp.analyzer.config.AISettings.DeduplicationEnabled && len(dedupedTasks) < len(storageTasks) && sp.analyzer.config.AISettings.VerboseMode {
		fmt.Printf("  🔄 Deduplication: %d tasks → %d tasks (removed %d duplicates)\n",
			len(storageTasks), len(dedupedTasks), len(storageTasks)-len(dedupedTasks))
	}

	return dedupedTasks, nil
}

// ProcessCommentsWithRealtimeSaving processes comments in parallel with real-time task saving
func (sp *StreamProcessor) ProcessCommentsWithRealtimeSaving(comments []CommentContext, storageManager *storage.Manager, prNumber int) ([]storage.Task, error) {
	if sp.analyzer.config.AISettings.VerboseMode {
		fmt.Printf("Processing %d comments in parallel with real-time saving...\n", len(comments))
	}

	// Create and start write worker if not provided
	if sp.writeWorker == nil {
		sp.writeWorker = storage.NewWriteWorker(storageManager, 100, sp.analyzer.config.AISettings.VerboseMode)
		if err := sp.writeWorker.Start(); err != nil {
			return nil, fmt.Errorf("failed to start write worker: %w", err)
		}
		defer sp.writeWorker.Stop()
	}

	type result struct {
		tasks   []TaskRequest
		err     error
		context CommentContext
	}

	results := make(chan result, len(comments))
	var wg sync.WaitGroup

	// Process all comments in parallel
	for _, commentCtx := range comments {
		wg.Add(1)
		go func(ctx CommentContext) {
			defer wg.Done()

			tasks, err := sp.analyzer.processComment(ctx)
			results <- result{
				tasks:   tasks,
				err:     err,
				context: ctx,
			}
		}(commentCtx)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results as they arrive
	var allTasks []storage.Task
	failedComments := make([]storage.FailedComment, 0)

	for res := range results {
		if res.err != nil {
			// Track failed comment
			failedComment := storage.FailedComment{
				CommentID:  res.context.Comment.ID,
				ReviewID:   res.context.SourceReview.ID,
				PRNumber:   prNumber,
				File:       res.context.Comment.File,
				Line:       res.context.Comment.Line,
				Author:     res.context.Comment.Author,
				Body:       res.context.Comment.Body,
				URL:        res.context.Comment.URL,
				Error:      res.err.Error(),
				ErrorType:  sp.categorizeError(res.err),
				RetryCount: 0,
			}
			failedComments = append(failedComments, failedComment)

			if sp.analyzer.config.AISettings.VerboseMode {
				fmt.Printf("  ❌ Comment %d failed: %v\n", res.context.Comment.ID, res.err)
			}
		} else {
			// Convert to storage tasks
			storageTasks := sp.analyzer.convertToStorageTasks(res.tasks)

			// Set PR number for each task
			for i := range storageTasks {
				storageTasks[i].PRNumber = prNumber
			}

			// Queue tasks for real-time saving
			for _, task := range storageTasks {
				if err := sp.writeWorker.QueueTask(task); err != nil {
					if sp.analyzer.config.AISettings.VerboseMode {
						fmt.Printf("  ⚠️  Failed to queue task %s: %v\n", task.ID, err)
					}
				}
			}

			allTasks = append(allTasks, storageTasks...)

			if sp.analyzer.config.AISettings.VerboseMode {
				fmt.Printf("  ✅ Comment %d processed: %d tasks generated and queued\n",
					res.context.Comment.ID, len(res.tasks))
			}
		}
	}

	// Save failed comments
	for _, failedComment := range failedComments {
		if err := storageManager.SaveFailedComment(failedComment); err != nil {
			if sp.analyzer.config.AISettings.VerboseMode {
				fmt.Printf("  ⚠️  Failed to save failed comment %d: %v\n", failedComment.CommentID, err)
			}
		}
	}

	// Wait for all writes to complete
	sp.writeWorker.WaitForCompletion()

	// Check for write errors
	writeErrors := sp.writeWorker.GetErrors()
	if len(writeErrors) > 0 {
		if sp.analyzer.config.AISettings.VerboseMode {
			fmt.Printf("  ⚠️  %d tasks failed to write\n", len(writeErrors))
			for _, we := range writeErrors {
				fmt.Printf("    • Task %s: %v\n", we.Task.ID, we.Error)
			}
		}
	}

	// Apply deduplication
	dedupedTasks := sp.analyzer.deduplicateTasks(allTasks)

	if sp.analyzer.config.AISettings.VerboseMode {
		fmt.Printf("📊 Final: %d tasks generated, %d failed comments\n",
			len(dedupedTasks), len(failedComments))
		if len(failedComments) > 0 {
			fmt.Printf("  Failed comments saved for retry in failed_comments.json\n")
		}
	}

	return dedupedTasks, nil
}

// categorizeError categorizes errors for better tracking
func (sp *StreamProcessor) categorizeError(err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "json") || strings.Contains(errStr, "JSON") {
		return "json_parse"
	} else if strings.Contains(errStr, "API") || strings.Contains(errStr, "execution failed") {
		return "api_failure"
	} else if strings.Contains(errStr, "context") || strings.Contains(errStr, "size") || strings.Contains(errStr, "limit") {
		return "context_overflow"
	} else if strings.Contains(errStr, "timeout") {
		return "timeout"
	}

	return "processing_failed"
}
