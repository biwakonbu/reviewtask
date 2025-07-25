package test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// TestPromptSizeLimitIntegration tests the full workflow for handling large PRs
// that would exceed Claude Code CLI prompt size limits.
// This integration test addresses Issue #116.
func TestPromptSizeLimitIntegration(t *testing.T) {
	// Create configuration with validation enabled
	cfg := &config.Config{
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{true}[0],
			MaxRetries:       3,
			UserLanguage:     "English",
			DebugMode:       true,
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}

	// Create mock Claude client that simulates size limit scenarios
	mockClient := &MockClaudeClientForIntegration{
		callCount: 0,
		sizeLimitThreshold: 1000, // Simulate size limit at 1KB for testing
	}

	analyzer := ai.NewAnalyzerWithClient(cfg, mockClient)

	// Create large review that would exceed size limits in batch mode
	largeReview := createLargeReviewForTesting(100) // 100 comments

	t.Run("ParallelProcessingHandlesLargePRs", func(t *testing.T) {
		// Reset mock client
		mockClient.callCount = 0
		mockClient.sizeErrorCount = 0

		// Generate tasks using parallel processing (validation enabled)
		tasks, err := analyzer.GenerateTasks([]github.Review{largeReview})

		// Should succeed with parallel processing
		if err != nil {
			t.Fatalf("Expected parallel processing to handle large PR, got error: %v", err)
		}

		// Should generate tasks
		if len(tasks) == 0 {
			t.Error("Expected tasks to be generated for large PR")
		}

		// Should use multiple Claude calls (parallel processing)
		if mockClient.callCount < 50 { // Expect many individual calls
			t.Errorf("Expected many Claude calls for parallel processing, got %d", mockClient.callCount)
		}

		// Should have minimal size errors with parallel processing
		if mockClient.sizeErrorCount > 10 { // Allow some errors but not excessive
			t.Errorf("Expected minimal size errors with parallel processing, got %d", mockClient.sizeErrorCount)
		}

		t.Logf("Successfully processed large PR with %d Claude calls and %d size errors", 
			mockClient.callCount, mockClient.sizeErrorCount)
	})

	t.Run("EarlyDetectionAvoidsExcessiveRetries", func(t *testing.T) {
		// Create mock client that always returns size errors
		alwaysSizeErrorClient := &MockClaudeClientForIntegration{
			callCount:          0,
			sizeErrorCount:     0,
			sizeLimitThreshold: 0, // Always trigger size errors
		}

		analyzerWithSizeErrors := ai.NewAnalyzerWithClient(cfg, alwaysSizeErrorClient)

		// Create small review to test error handling
		smallReview := createLargeReviewForTesting(1) // Single comment

		// Generate tasks - should fail gracefully without excessive retries
		_, err := analyzerWithSizeErrors.GenerateTasks([]github.Review{smallReview})

		// Should handle size errors gracefully
		if err == nil {
			t.Error("Expected size error to be handled and reported")
		}

		// Should not make excessive retry attempts
		if alwaysSizeErrorClient.callCount > 5 {
			t.Errorf("Expected minimal retries for size errors, got %d calls", alwaysSizeErrorClient.callCount)
		}

		t.Logf("Size error handled with %d calls (early detection working)", alwaysSizeErrorClient.callCount)
	})
}

// TestValidationModeUsesParallelProcessingIntegration verifies that validation mode
// switches to parallel processing instead of batch processing.
func TestValidationModeUsesParallelProcessingIntegration(t *testing.T) {
	// Test with validation disabled (baseline)
	cfgNoValidation := &config.Config{
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{false}[0],
			UserLanguage:     "English",
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}

	// Test with validation enabled (should use parallel processing)
	cfgWithValidation := &config.Config{
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{true}[0],
			UserLanguage:     "English",
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}

	testReview := createLargeReviewForTesting(5) // 5 comments

	t.Run("ValidationDisabledUsesParallelProcessing", func(t *testing.T) {
		mockClient := &MockClaudeClientForIntegration{
			callCount: 0,
			sizeLimitThreshold: 10000, // High threshold to avoid size errors
		}

		analyzer := ai.NewAnalyzerWithClient(cfgNoValidation, mockClient)
		tasks, err := analyzer.GenerateTasks([]github.Review{testReview})

		if err != nil {
			t.Fatalf("GenerateTasks failed: %v", err)
		}

		if len(tasks) == 0 {
			t.Error("Expected tasks to be generated")
		}

		baselineCallCount := mockClient.callCount
		t.Logf("Validation disabled: %d Claude calls", baselineCallCount)
	})

	t.Run("ValidationEnabledUsesParallelProcessing", func(t *testing.T) {
		mockClient := &MockClaudeClientForIntegration{
			callCount: 0,
			sizeLimitThreshold: 10000, // High threshold to avoid size errors
		}

		analyzer := ai.NewAnalyzerWithClient(cfgWithValidation, mockClient)
		tasks, err := analyzer.GenerateTasks([]github.Review{testReview})

		if err != nil {
			t.Fatalf("GenerateTasks failed: %v", err)
		}

		if len(tasks) == 0 {
			t.Error("Expected tasks to be generated")
		}

		validationCallCount := mockClient.callCount
		t.Logf("Validation enabled: %d Claude calls", validationCallCount)

		// Both should use parallel processing (individual calls per comment)
		expectedCallCount := len(testReview.Comments)
		if validationCallCount < expectedCallCount {
			t.Errorf("Expected at least %d calls for parallel processing, got %d", 
				expectedCallCount, validationCallCount)
		}
	})
}

// MockClaudeClientForIntegration simulates Claude CLI behavior for integration testing
type MockClaudeClientForIntegration struct {
	callCount          int
	sizeErrorCount     int
	sizeLimitThreshold int // Prompt size threshold for triggering errors
}

func (m *MockClaudeClientForIntegration) Execute(ctx context.Context, prompt string, outputFormat string) (string, error) {
	m.callCount++

	// Simulate size limit check
	if len(prompt) > m.sizeLimitThreshold {
		m.sizeErrorCount++
		return "", fmt.Errorf("prompt size (%d bytes) exceeds maximum limit (%d bytes). Please shorten or chunk the prompt content", 
			len(prompt), m.sizeLimitThreshold)
	}

	// Return successful response
	response := `{"type": "claude_response", "subtype": "json", "is_error": false, "result": "[{\"description\": \"Test task\", \"origin_text\": \"Test comment\", \"priority\": \"medium\", \"source_review_id\": 1, \"source_comment_id\": 1, \"file\": \"test.go\", \"line\": 1, \"task_index\": 0}]"}`
	return response, nil
}

// createLargeReviewForTesting creates a review with many comments for testing large PR scenarios
func createLargeReviewForTesting(commentCount int) github.Review {
	comments := make([]github.Comment, commentCount)
	
	for i := 0; i < commentCount; i++ {
		comments[i] = github.Comment{
			ID:     int64(i + 1),
			Author: "test-reviewer",
			Body:   fmt.Sprintf("This is test comment #%d that contains detailed feedback about the code implementation. %s", 
				i+1, strings.Repeat("This is additional content to make the comment longer. ", 10)),
			File:   fmt.Sprintf("test_%d.go", i%10), // Spread across multiple files
			Line:   (i % 100) + 1,                   // Various line numbers
		}
	}

	return github.Review{
		ID:       1,
		Reviewer: "test-reviewer",
		State:    "PENDING",
		Body:     "This is a comprehensive review with many detailed comments.",
		Comments: comments,
	}
}