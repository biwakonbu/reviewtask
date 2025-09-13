package ai

import (
	"fmt"
	"strings"
	"sync"

	"reviewtask/internal/storage"
)

// StreamProcessor handles comment processing with streaming results
type StreamProcessor struct {
	analyzer *Analyzer
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(analyzer *Analyzer) *StreamProcessor {
	return &StreamProcessor{
		analyzer: analyzer,
	}
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
		done    bool
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
				done:    false,
			}
		}(i, commentCtx)
	}

	// Signal completion
	go func() {
		wg.Wait()
		results <- streamResult{done: true}
		close(results)
	}()

	// Stream processing: handle results as they arrive
	var allTasks []TaskRequest
	var successfulResults []streamResult
	var failedResults []streamResult

	processed := 0
	for result := range results {
		if result.done {
			break
		}

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
