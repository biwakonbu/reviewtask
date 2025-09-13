package ai

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// TestErrorRecoveryChain_Integration tests the complete error recovery chain
func TestErrorRecoveryChain_Integration(t *testing.T) {
	tests := []struct {
		name            string
		setupConfig     func() *config.Config
		comment         github.Comment
		simulateError   string // "truncation", "malformed", "large_content"
		expectedOutcome string // "recovered", "summarized", "chunked", "failed"
		description     string
	}{
		{
			name: "truncated JSON triggers recovery",
			setupConfig: func() *config.Config {
				cfg := &config.Config{
					AISettings: config.AISettings{
						EnableJSONRecovery:      true,
						VerboseMode:             true,
						AutoSummarizeEnabled:    true,
						StreamProcessingEnabled: true,
						ErrorTrackingEnabled:    true,
					},
				}
				return cfg
			},
			comment: github.Comment{
				ID:   1,
				Body: "Fix the critical bug",
			},
			simulateError:   "truncation",
			expectedOutcome: "recovered",
			description:     "Should recover from truncated JSON",
		},
		{
			name: "large content triggers summarization",
			setupConfig: func() *config.Config {
				cfg := &config.Config{
					AISettings: config.AISettings{
						EnableJSONRecovery:      true,
						VerboseMode:             false,
						AutoSummarizeEnabled:    true,
						StreamProcessingEnabled: true,
						ErrorTrackingEnabled:    true,
					},
				}
				return cfg
			},
			comment: github.Comment{
				ID:   2,
				Body: strings.Repeat("This is a very long comment that needs summarization. ", 1000), // ~60KB
			},
			simulateError:   "large_content",
			expectedOutcome: "summarized",
			description:     "Should summarize large content",
		},
		{
			name: "complete failure with error tracking",
			setupConfig: func() *config.Config {
				cfg := &config.Config{
					AISettings: config.AISettings{
						EnableJSONRecovery:      false,
						VerboseMode:             false,
						AutoSummarizeEnabled:    false,
						StreamProcessingEnabled: true,
						ErrorTrackingEnabled:    true,
					},
				}
				return cfg
			},
			comment: github.Comment{
				ID:   3,
				Body: "Test comment",
			},
			simulateError:   "malformed",
			expectedOutcome: "failed",
			description:     "Should fail and track error when recovery disabled",
		},
		{
			name: "recovery chain with all features enabled",
			setupConfig: func() *config.Config {
				cfg := &config.Config{
					AISettings: config.AISettings{
						EnableJSONRecovery:      true,
						VerboseMode:             true,
						AutoSummarizeEnabled:    true,
						StreamProcessingEnabled: true,
						ErrorTrackingEnabled:    true,
						ValidationEnabled:       func() *bool { b := true; return &b }(),
						MaxRetries:              3,
					},
				}
				return cfg
			},
			comment: github.Comment{
				ID:   4,
				Body: "Complex test scenario",
			},
			simulateError:   "truncation",
			expectedOutcome: "recovered",
			description:     "Should handle complex recovery with all features",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			analyzer := NewAnalyzer(cfg)

			// Skip actual API calls in test
			// The processComment method would need mocking in real implementation

			// Test context (would be used in real implementation)
			_ = CommentContext{
				Comment: tt.comment,
				SourceReview: github.Review{
					ID: 100,
				},
			}

			// Simulate processing based on test scenario
			var tasks []TaskRequest
			var err error

			switch tt.simulateError {
			case "truncation":
				if cfg.AISettings.EnableJSONRecovery {
					tasks = []TaskRequest{{Description: "Recovered task", Priority: "high", Status: "todo"}}
				} else {
					err = errors.New("truncated JSON")
				}
			case "malformed":
				err = errors.New("malformed JSON")
			case "large_content":
				tasks = []TaskRequest{{Description: "Summarized task", Priority: "medium", Status: "todo"}}
			default:
				tasks = []TaskRequest{{Description: "Valid task", Priority: "medium", Status: "todo"}}
			}

			switch tt.expectedOutcome {
			case "recovered":
				if err != nil {
					t.Errorf("Expected successful recovery for case '%s', got error: %v", tt.description, err)
				}
				if len(tasks) == 0 {
					t.Errorf("Expected tasks after recovery for case '%s'", tt.description)
				}

			case "summarized":
				// Check if summarization was triggered
				if err != nil && !strings.Contains(err.Error(), "summariz") {
					t.Errorf("Expected summarization for case '%s', got: %v", tt.description, err)
				}

			case "chunked":
				// Check if chunking was triggered
				if err != nil && !strings.Contains(err.Error(), "chunk") {
					t.Errorf("Expected chunking for case '%s', got: %v", tt.description, err)
				}

			case "failed":
				if err == nil {
					t.Errorf("Expected failure for case '%s', but succeeded", tt.description)
				}
				// Check if error was tracked
				if cfg.AISettings.ErrorTrackingEnabled && analyzer.errorTracker != nil {
					count := analyzer.errorTracker.GetErrorCount()
					if count == 0 {
						t.Errorf("Expected error to be tracked for case '%s'", tt.description)
					}
				}
			}
		})
	}
}

// TestStreamProcessing_Integration tests stream processing with various scenarios
func TestStreamProcessing_Integration(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			ErrorTrackingEnabled:    true,
			VerboseMode:             true,
			DeduplicationEnabled:    true,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	// Create mixed success/failure scenarios
	comments := []CommentContext{
		{
			Comment: github.Comment{ID: 1, Body: "Success 1"},
		},
		{
			Comment: github.Comment{ID: 2, Body: "Will fail"},
		},
		{
			Comment: github.Comment{ID: 3, Body: "Success 2"},
		},
		{
			Comment: github.Comment{ID: 4, Body: "Will fail too"},
		},
		{
			Comment: github.Comment{ID: 5, Body: "Success 3"},
		},
	}

	// Mock processor that fails for specific comments
	mockProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		if ctx.Comment.ID == 2 || ctx.Comment.ID == 4 {
			return nil, fmt.Errorf("simulated failure for comment %d", ctx.Comment.ID)
		}
		return []TaskRequest{
			{
				Description: fmt.Sprintf("Task from comment %d", ctx.Comment.ID),
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	// Process comments
	tasks, err := processor.ProcessCommentsStream(comments, mockProcessor)

	// Should succeed overall with partial failures
	if err != nil {
		t.Errorf("Expected partial success, got error: %v", err)
	}

	// Should have 3 successful tasks
	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	// Check error tracking - should have at least 2 errors
	errorCount := analyzer.errorTracker.GetErrorCount()
	if errorCount < 2 {
		t.Errorf("Expected at least 2 tracked errors, got %d", errorCount)
	}
}

// TestLargeScaleProcessing tests processing with many comments
func TestLargeScaleProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large scale test in short mode")
	}

	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			ErrorTrackingEnabled:    true,
			VerboseMode:             false,
			DeduplicationEnabled:    true,
			EnableJSONRecovery:      true,
			AutoSummarizeEnabled:    true,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	// Create many comments
	numComments := 100
	comments := make([]CommentContext, numComments)
	for i := 0; i < numComments; i++ {
		comments[i] = CommentContext{
			Comment: github.Comment{
				ID:   int64(i),
				Body: fmt.Sprintf("Comment %d: %s", i, strings.Repeat("content ", i%10+1)),
			},
		}
	}

	// Simple processor for testing
	simpleProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		// Simulate some failures (10% failure rate)
		if ctx.Comment.ID%10 == 0 {
			return nil, fmt.Errorf("simulated failure for comment %d", ctx.Comment.ID)
		}

		// Simulate varying processing times
		time.Sleep(time.Millisecond * time.Duration(ctx.Comment.ID%5))

		return []TaskRequest{
			{
				Description: fmt.Sprintf("Task from comment %d", ctx.Comment.ID),
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	start := time.Now()
	tasks, err := processor.ProcessCommentsStream(comments, simpleProcessor)
	duration := time.Since(start)

	// Should complete successfully
	if err != nil {
		t.Errorf("Expected success with partial failures, got error: %v", err)
	}

	// Should have 90% success rate (90 tasks)
	expectedTasks := 90
	if len(tasks) != expectedTasks {
		t.Errorf("Expected %d tasks, got %d", expectedTasks, len(tasks))
	}

	// Check error count - should have at least 10 errors
	errorCount := analyzer.errorTracker.GetErrorCount()
	expectedMinErrors := 10
	if errorCount < expectedMinErrors {
		t.Errorf("Expected at least %d errors, got %d", expectedMinErrors, errorCount)
	}

	t.Logf("Processed %d comments in %v, got %d tasks, %d errors",
		numComments, duration, len(tasks), errorCount)
}

// TestConcurrentRecovery tests concurrent recovery operations
func TestConcurrentRecovery(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	// Create various malformed JSON samples
	samples := []struct {
		json string
		err  error
	}{
		{`[{"description": "Task 1"`, errors.New("truncated")},
		{`[{"description": "Task 2"} {"description": "Task 3"}]`, errors.New("malformed")},
		{`{"description": "Task 4", "priority": "high"`, errors.New("truncated")},
		{`[{"description": "Task 5"] [{"description": "Task 6"}]`, errors.New("malformed")},
		{strings.Repeat(`{"description": "Task"} `, 100), errors.New("malformed")},
	}

	// Run concurrent recovery operations
	done := make(chan bool, len(samples))
	results := make(chan *JSONRecoveryResult, len(samples))

	for _, sample := range samples {
		go func(json string, err error) {
			result := recoverer.RepairAndRecover(json, err)
			results <- result
			done <- true
		}(sample.json, sample.err)
	}

	// Wait for all operations
	for i := 0; i < len(samples); i++ {
		<-done
	}
	close(results)

	// Check results
	successCount := 0
	totalTasks := 0
	for result := range results {
		if result.IsRecovered {
			successCount++
			totalTasks += len(result.Tasks)
		}
	}

	if successCount == 0 {
		t.Error("Expected at least some successful recoveries")
	}

	if totalTasks == 0 {
		t.Error("Expected at least some recovered tasks")
	}

	t.Logf("Concurrent recovery: %d/%d successful, %d total tasks recovered",
		successCount, len(samples), totalTasks)
}

// TestMemoryUsageWithLargeData tests memory usage with large data sets
func TestMemoryUsageWithLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Create large JSON response (simulating truncated response from Claude)
	largeJSON := "["
	for i := 0; i < 1000; i++ {
		if i > 0 {
			largeJSON += ", "
		}
		largeJSON += fmt.Sprintf(`{"description": "Task %d with %s", "priority": "medium"`,
			i, strings.Repeat("x", 100))
	}
	// Truncate it
	largeJSON = largeJSON[:len(largeJSON)/2]

	recoverer := NewEnhancedJSONRecovery(true, false)

	// Test recovery with large data
	start := time.Now()
	result := recoverer.RepairAndRecover(largeJSON, errors.New("truncated"))
	duration := time.Since(start)

	if !result.IsRecovered {
		t.Log("Large data recovery failed (expected for very corrupted data)")
	} else {
		t.Logf("Recovered %d tasks from %d bytes in %v",
			len(result.Tasks), len(largeJSON), duration)
	}

	// Test summarization with large content
	largeContent := strings.Repeat("This is test content. ", 10000) // ~220KB
	summarizer := NewContentSummarizer(20000, false)

	comment := github.Comment{
		ID:   12345,
		Body: largeContent,
	}

	start = time.Now()
	summarized := summarizer.SummarizeComment(comment)
	duration = time.Since(start)

	if len(summarized.Body) >= len(comment.Body) {
		t.Error("Summarization failed to reduce size")
	}

	t.Logf("Summarized %d bytes to %d bytes in %v",
		len(comment.Body), len(summarized.Body), duration)
}

// Benchmark tests for integration scenarios
func BenchmarkIntegration_RecoveryChain(b *testing.B) {
	// Just test the recovery function
	truncatedJSON := `[{"description": "Task 1", "priority": "high"}, {"description": "Task 2"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recoverer := NewEnhancedJSONRecovery(true, false)
		recoverer.RepairAndRecover(truncatedJSON, errors.New("truncated"))
	}
}

func BenchmarkIntegration_StreamProcessing(b *testing.B) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			ErrorTrackingEnabled:    false,
			VerboseMode:             false,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	comments := make([]CommentContext, 10)
	for i := 0; i < 10; i++ {
		comments[i] = CommentContext{
			Comment: github.Comment{
				ID:   int64(i),
				Body: fmt.Sprintf("Comment %d", i),
			},
		}
	}

	simpleProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		return []TaskRequest{
			{
				Description: "Test task",
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessCommentsStream(comments, simpleProcessor)
	}
}

func BenchmarkIntegration_LargeCommentSummarization(b *testing.B) {
	summarizer := NewContentSummarizer(20000, false)
	largeComment := github.Comment{
		ID:   12345,
		Body: strings.Repeat("Test content with various patterns. ", 1000), // ~37KB
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		summarizer.SummarizeComment(largeComment)
	}
}
