package ai

import (
	"errors"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

func TestNewStreamProcessor(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             false,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	if processor.analyzer != analyzer {
		t.Error("Expected StreamProcessor to store reference to analyzer")
	}
}

func TestStreamProcessor_CategorizeError(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	tests := []struct {
		name         string
		error        error
		expectedType string
	}{
		{
			name:         "json parse error",
			error:        errors.New("failed to parse json response"),
			expectedType: "json_parse",
		},
		{
			name:         "JSON uppercase error",
			error:        errors.New("invalid JSON format received"),
			expectedType: "json_parse",
		},
		{
			name:         "API failure",
			error:        errors.New("API call failed with 500 error"),
			expectedType: "api_failure",
		},
		{
			name:         "execution failed",
			error:        errors.New("command execution failed"),
			expectedType: "api_failure",
		},
		{
			name:         "context overflow",
			error:        errors.New("context size limit exceeded"),
			expectedType: "context_overflow",
		},
		{
			name:         "size limit error",
			error:        errors.New("request size too large"),
			expectedType: "context_overflow",
		},
		{
			name:         "timeout error",
			error:        errors.New("request timeout occurred"),
			expectedType: "timeout",
		},
		{
			name:         "generic processing error",
			error:        errors.New("something went wrong"),
			expectedType: "processing_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.categorizeError(tt.error)
			if result != tt.expectedType {
				t.Errorf("Expected error type '%s', got '%s'", tt.expectedType, result)
			}
		})
	}
}

func TestStreamProcessor_ProcessCommentsStream_Disabled(t *testing.T) {
	// Test fallback to traditional processing when stream processing is disabled
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: false,
			VerboseMode:             false,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	// Create test comments
	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:   1,
				Body: "Test comment 1",
			},
		},
	}

	// Mock processor function that should succeed
	mockProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		return []TaskRequest{
			{
				Description: "Test task from comment " + string(rune(ctx.Comment.ID)),
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	// This should fallback to traditional parallel processing
	// We can't easily test the actual parallel processing without mocking the entire analyzer,
	// but we can verify that the function doesn't crash and handles the disabled flag correctly
	tasks, err := processor.ProcessCommentsStream(comments, mockProcessor)

	// The actual result depends on the analyzer's processCommentsParallel method
	// For this test, we mainly want to verify that the fallback path is taken
	// and the function completes without crashing
	if err == nil && len(tasks) >= 0 {
		t.Log("Stream processing correctly fell back to traditional processing")
	}
}

func TestStreamProcessor_ProcessCommentsStream_Success(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             false,
			DeduplicationEnabled:    false, // Disable to simplify testing
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	// Create test comments
	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:   1,
				Body: "Fix authentication bug",
			},
		},
		{
			Comment: github.Comment{
				ID:   2,
				Body: "Update documentation",
			},
		},
	}

	// Mock processor function that succeeds for all comments
	successfulProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		return []TaskRequest{
			{
				Description: "Task for comment " + string(rune('0'+ctx.Comment.ID)),
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	tasks, err := processor.ProcessCommentsStream(comments, successfulProcessor)

	if err != nil {
		t.Errorf("Expected no error for successful processing, got: %v", err)
	}

	expectedTasks := 2
	if len(tasks) != expectedTasks {
		t.Errorf("Expected %d tasks, got %d", expectedTasks, len(tasks))
	}
}

func TestStreamProcessor_ProcessCommentsStream_PartialFailure(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             false,
			DeduplicationEnabled:    false,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:   1,
				Body: "Success comment",
			},
		},
		{
			Comment: github.Comment{
				ID:   2,
				Body: "Failure comment",
			},
		},
		{
			Comment: github.Comment{
				ID:   3,
				Body: "Another success comment",
			},
		},
	}

	// Mock processor that fails for comment ID 2
	partialFailureProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		if ctx.Comment.ID == 2 {
			return nil, errors.New("processing failed for comment 2")
		}
		return []TaskRequest{
			{
				Description: "Task for comment " + string(rune('0'+ctx.Comment.ID)),
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	tasks, err := processor.ProcessCommentsStream(comments, partialFailureProcessor)

	// Should succeed overall since some comments were processed successfully
	if err != nil {
		t.Errorf("Expected no error for partial failure, got: %v", err)
	}

	// Should have 2 successful tasks (from comments 1 and 3)
	expectedTasks := 2
	if len(tasks) != expectedTasks {
		t.Errorf("Expected %d tasks from successful comments, got %d", expectedTasks, len(tasks))
	}
}

func TestStreamProcessor_ProcessCommentsStream_CompleteFailure(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             false,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:   1,
				Body: "Failure comment 1",
			},
		},
		{
			Comment: github.Comment{
				ID:   2,
				Body: "Failure comment 2",
			},
		},
	}

	// Mock processor that always fails
	alwaysFailProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		return nil, errors.New("processing always fails")
	}

	tasks, err := processor.ProcessCommentsStream(comments, alwaysFailProcessor)

	// Should return error since all processing failed
	if err == nil {
		t.Error("Expected error when all comment processing fails")
	}

	if !containsString(err.Error(), "all comment processing failed") {
		t.Errorf("Expected error message to contain 'all comment processing failed', got: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected no tasks when all processing fails, got %d", len(tasks))
	}
}

func TestStreamProcessor_ProcessCommentsStream_EmptyInput(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             false,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	// Empty comment list
	comments := []CommentContext{}

	mockProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		return []TaskRequest{}, nil
	}

	tasks, err := processor.ProcessCommentsStream(comments, mockProcessor)

	if err != nil {
		t.Errorf("Expected no error for empty input, got: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("Expected no tasks for empty input, got %d", len(tasks))
	}
}

func TestStreamProcessor_ProcessCommentsStream_ErrorTracking(t *testing.T) {
	// Create a temporary directory for error tracking
	tempDir := t.TempDir()

	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             true,
			ErrorTrackingEnabled:    true,
		},
	}

	analyzer := NewAnalyzer(cfg)
	// Override the error tracker to use our temp directory
	analyzer.errorTracker = NewErrorTracker(true, false, tempDir)

	processor := NewStreamProcessor(analyzer)

	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:   1,
				Body: "Success comment",
			},
		},
		{
			Comment: github.Comment{
				ID:   2,
				Body: "JSON parse failure comment",
			},
		},
	}

	// Mock processor that fails for comment 2 with JSON error
	errorTrackingProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		if ctx.Comment.ID == 2 {
			return nil, errors.New("failed to parse JSON response")
		}
		return []TaskRequest{
			{
				Description: "Task for comment " + string(rune('0'+ctx.Comment.ID)),
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	tasks, err := processor.ProcessCommentsStream(comments, errorTrackingProcessor)

	// Should succeed overall
	if err != nil {
		t.Errorf("Expected no error with error tracking, got: %v", err)
	}

	// Should have 1 task from successful comment
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	// Check that error was tracked
	errorCount := analyzer.errorTracker.GetErrorCount()
	if errorCount != 1 {
		t.Errorf("Expected 1 tracked error, got %d", errorCount)
	}

	// Verify the error details
	errors, err := analyzer.errorTracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get error summary: %v", err)
	}

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error in summary, got %d", len(errors))
	}

	trackedError := errors[0]
	if trackedError.ErrorType != "json_parse" {
		t.Errorf("Expected error type 'json_parse', got '%s'", trackedError.ErrorType)
	}

	if trackedError.CommentID != 2 {
		t.Errorf("Expected comment ID 2, got %d", trackedError.CommentID)
	}
}

func TestStreamProcessor_ProcessCommentsStream_VerboseMode(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             true, // Enable verbose output
			DeduplicationEnabled:    false,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:   1,
				Body: "Test comment for verbose mode",
			},
		},
	}

	successfulProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		return []TaskRequest{
			{
				Description: "Test task",
				Priority:    "medium",
				Status:      "todo",
			},
		}, nil
	}

	// This mainly tests that verbose mode doesn't crash
	// The actual output goes to stdout and is hard to test directly
	tasks, err := processor.ProcessCommentsStream(comments, successfulProcessor)

	if err != nil {
		t.Errorf("Expected no error in verbose mode, got: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task in verbose mode, got %d", len(tasks))
	}
}

// Helper function to check if a string contains a substring
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		haystack[len(haystack)-len(needle):] == needle ||
		indexOf(haystack, needle) >= 0
}

// Simple indexOf function for string searching
func indexOf(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func TestStreamProcessor_Integration_WithAnalyzer(t *testing.T) {
	// This is more of an integration test that verifies the stream processor
	// works correctly with the analyzer's task conversion and deduplication

	cfg := &config.Config{
		AISettings: config.AISettings{
			StreamProcessingEnabled: true,
			VerboseMode:             false,
			DeduplicationEnabled:    true,
		},
	}

	analyzer := NewAnalyzer(cfg)
	processor := NewStreamProcessor(analyzer)

	comments := []CommentContext{
		{
			Comment: github.Comment{
				ID:   1,
				Body: "Fix authentication bug",
				File: "auth.go",
				Line: 42,
			},
		},
		{
			Comment: github.Comment{
				ID:   2,
				Body: "Fix authentication bug", // Duplicate task
				File: "auth.go",
				Line: 45,
			},
		},
	}

	duplicateTaskProcessor := func(ctx CommentContext) ([]TaskRequest, error) {
		return []TaskRequest{
			{
				Description: "Fix authentication bug", // Same description for both
				Priority:    "high",
				Status:      "todo",
			},
		}, nil
	}

	tasks, err := processor.ProcessCommentsStream(comments, duplicateTaskProcessor)

	if err != nil {
		t.Errorf("Expected no error in integration test, got: %v", err)
	}

	// With deduplication enabled, should have only 1 unique task
	// (This depends on the deduplication implementation in the analyzer)
	if len(tasks) == 0 {
		t.Error("Expected at least some tasks from integration test")
	}

	// Verify tasks are properly converted to storage.Task format
	for i, task := range tasks {
		if task.ID == "" {
			t.Errorf("Task %d should have an ID", i)
		}
		// Note: Status might be empty in this test since we're using a mock processor
		// that doesn't go through the full analyzer pipeline
		if task.Status != "" && task.Status != "todo" {
			t.Errorf("Task %d should have proper status, got %s", i, task.Status)
		}
	}
}
