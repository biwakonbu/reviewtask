package ai

import (
	"sync"
	"testing"
	"time"
)

func TestProgressReporter(t *testing.T) {
	t.Run("Basic Progress Tracking", func(t *testing.T) {
		progressUpdates := make([]struct{ current, total int }, 0)
		var mu sync.Mutex

		reporter := NewProgressReporter(3, func(current, total int) {
			mu.Lock()
			defer mu.Unlock()
			progressUpdates = append(progressUpdates, struct{ current, total int }{current, total})
		})

		// Simulate processing 3 comments
		reporter.ReportStepProgress(0, StepPreparePrompt)
		reporter.ReportStepProgress(0, StepCallClaude)
		reporter.ReportStepProgress(0, StepParseResponse)

		reporter.ReportStepProgress(1, StepPreparePrompt)
		reporter.ReportStepProgress(1, StepCallClaude)

		// Check progress updates
		mu.Lock()
		defer mu.Unlock()
		if len(progressUpdates) != 5 {
			t.Errorf("Expected 5 progress updates, got %d", len(progressUpdates))
		}

		// Check that progress is calculated based on weights
		lastUpdate := progressUpdates[len(progressUpdates)-1]
		expectedProgress := StepWeights[StepPreparePrompt] + StepWeights[StepCallClaude] +
			StepWeights[StepParseResponse] + StepWeights[StepPreparePrompt] + StepWeights[StepCallClaude]
		if lastUpdate.current != expectedProgress {
			t.Errorf("Expected current progress %d, got %d", expectedProgress, lastUpdate.current)
		}
	})

	t.Run("Concurrent Progress Updates", func(t *testing.T) {
		callCount := 0
		var mu sync.Mutex

		reporter := NewProgressReporter(10, func(current, total int) {
			mu.Lock()
			defer mu.Unlock()
			callCount++
		})

		// Simulate concurrent processing
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				reporter.ReportStepProgress(index, StepPreparePrompt)
				time.Sleep(10 * time.Millisecond)
				reporter.ReportStepProgress(index, StepCallClaude)
				reporter.ReportStepProgress(index, StepParseResponse)
				reporter.ReportCommentComplete(index)
			}(i)
		}

		wg.Wait()

		mu.Lock()
		defer mu.Unlock()
		// Each comment reports 3 steps
		if callCount != 30 {
			t.Errorf("Expected 30 progress callbacks, got %d", callCount)
		}
	})

	t.Run("Progress Percentage Calculation", func(t *testing.T) {
		reporter := NewProgressReporter(2, nil)

		// Process first comment completely
		reporter.ReportStepProgress(0, StepPreparePrompt)
		reporter.ReportStepProgress(0, StepCallClaude)
		reporter.ReportStepProgress(0, StepParseResponse)
		reporter.ReportStepProgress(0, StepValidateFormat)
		reporter.ReportStepProgress(0, StepValidateContent)
		reporter.ReportStepProgress(0, StepDeduplication)
		reporter.ReportCommentComplete(0)

		progress := reporter.GetProgress()
		// First comment should be 50% of total work
		if progress < 45 || progress > 55 {
			t.Errorf("Expected progress around 50%%, got %.2f%%", progress)
		}
	})
}
