package ai

import (
	"errors"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

func TestJSONRecovery_Integration(t *testing.T) {
	// Test the JSON recovery mechanism with a simple truncated response
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	// Create analyzer without Claude client (will use manual JSON recovery)
	analyzer := NewAnalyzer(cfg)

	// Test truncated JSON recovery directly
	truncatedJSON := `[
		{
			"description": "Fix the bug",
			"origin_text": "There is a bug here",
			"priority": "high",
			"source_review_id": 123,
			"source_comment_id": 456,
			"file": "test.go",
			"line": 10,
			"task_index": 0
		},
		{
			"description": "Add test",
			"origin_text": "Need more tests",
			"priority": "medium"` // Truncated here

	recoverer := NewJSONRecoverer(true, false)
	result := recoverer.RecoverJSON(truncatedJSON, errors.New("unexpected end of JSON input"))

	if !result.IsRecovered {
		t.Error("Expected JSON recovery to succeed")
	}

	if len(result.Tasks) == 0 {
		t.Error("Expected at least one task to be recovered")
	}

	// Verify recovered task
	if len(result.Tasks) > 0 {
		task := result.Tasks[0]
		if task.Description != "Fix the bug" {
			t.Errorf("Expected description 'Fix the bug', got '%s'", task.Description)
		}
		if task.OriginText != "There is a bug here" {
			t.Errorf("Expected origin text 'There is a bug here', got '%s'", task.OriginText)
		}
		if task.Priority != "high" {
			t.Errorf("Expected priority 'high', got '%s'", task.Priority)
		}
	}
}

func TestRetryStrategy_Integration(t *testing.T) {
	strategy := NewRetryStrategy(false)

	// Test error categorization
	tests := []struct {
		name         string
		error        error
		expectedType string
	}{
		{
			name:         "JSON truncation",
			error:        errors.New("unexpected end of JSON input"),
			expectedType: "json_truncation",
		},
		{
			name:         "Prompt size limit",
			error:        errors.New("prompt size exceeds maximum limit"),
			expectedType: "prompt_size_limit",
		},
		{
			name:         "Rate limit",
			error:        errors.New("rate limit exceeded"),
			expectedType: "rate_limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := strategy.categorizeRetryError(tt.error)
			if errorType != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, errorType)
			}

			// Test retry decision
			retryAttempt, shouldRetry := strategy.ShouldRetry(0, tt.error, 15000, 5000)
			if !shouldRetry {
				t.Error("Expected retry to be recommended")
			}

			if retryAttempt == nil {
				t.Error("Expected retry attempt details")
			} else {
				if retryAttempt.AttemptNumber != 1 {
					t.Errorf("Expected attempt number 1, got %d", retryAttempt.AttemptNumber)
				}
			}
		})
	}
}

func TestResponseMonitor_Integration(t *testing.T) {
	monitor := NewResponseMonitor(false)

	// Test event recording and analysis
	events := []ResponseEvent{
		{
			PromptSize:      10000,
			ResponseSize:    5000,
			ProcessingTime:  2000,
			Success:         true,
			RecoveryUsed:    false,
			TasksExtracted:  2,
		},
		{
			PromptSize:      20000,
			ResponseSize:    1000,
			ProcessingTime:  8000,
			Success:         false,
			ErrorType:       "json_truncation",
			RecoveryUsed:    true,
			TasksExtracted:  0,
		},
		{
			PromptSize:      15000,
			ResponseSize:    4000,
			ProcessingTime:  3000,
			Success:         true,
			RecoveryUsed:    true,
			TasksExtracted:  1,
		},
	}

	// Disable file I/O for this test
	monitor.config.CollectionEnabled = false

	// Test direct analytics calculation
	analytics := monitor.calculateAnalytics(events)

	if analytics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", analytics.TotalRequests)
	}

	expectedSuccessRate := 2.0 / 3.0
	if analytics.SuccessRate < expectedSuccessRate-0.01 || analytics.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("Expected success rate ~%.2f, got %.2f", expectedSuccessRate, analytics.SuccessRate)
	}

	expectedRecoveryRate := 2.0 / 3.0
	if analytics.RecoveryRate < expectedRecoveryRate-0.01 || analytics.RecoveryRate > expectedRecoveryRate+0.01 {
		t.Errorf("Expected recovery rate ~%.2f, got %.2f", expectedRecoveryRate, analytics.RecoveryRate)
	}

	if analytics.ErrorDistribution["json_truncation"] != 1 {
		t.Errorf("Expected 1 json_truncation error, got %d", analytics.ErrorDistribution["json_truncation"])
	}
}

func TestCompleteRecoveryFlow_Integration(t *testing.T) {
	// Test the complete flow with a mock scenario
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	// Use mock client that succeeds
	mockClient := NewMockClaudeClient()
	mockClient.Responses["There is a bug"] = `[
		{
			"description": "Fix the bug",
			"origin_text": "There is a bug",
			"priority": "high",
			"source_review_id": 123,
			"source_comment_id": 456,
			"file": "test.go",
			"line": 10,
			"task_index": 0
		}
	]`

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID:       123,
			Reviewer: "reviewer1",
			Comments: []github.Comment{
				{
					ID:     456,
					File:   "test.go",
					Line:   10,
					Body:   "There is a bug",
					Author: "reviewer1",
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)

	if err != nil {
		t.Errorf("Expected successful task generation, got error: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected at least one task")
	}

	// Verify task content
	if len(tasks) > 0 {
		task := tasks[0]
		if task.Description != "Fix the bug" {
			t.Errorf("Expected description 'Fix the bug', got '%s'", task.Description)
		}
		if task.Priority != "high" {
			t.Errorf("Expected priority 'high', got '%s'", task.Priority)
		}
	}

	// Verify response monitoring was used
	if analyzer.responseMonitor == nil {
		t.Error("Expected response monitor to be initialized")
	}
}

func TestConfigurationOverrides_Integration(t *testing.T) {
	tests := []struct {
		name               string
		enableJSONRecovery bool
		expectedBehavior   string
	}{
		{
			name:               "recovery enabled",
			enableJSONRecovery: true,
			expectedBehavior:   "should_create_recoverer",
		},
		{
			name:               "recovery disabled",
			enableJSONRecovery: false,
			expectedBehavior:   "should_not_recover",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					EnableJSONRecovery: tt.enableJSONRecovery,
					VerboseMode:        false,
				},
			}

			analyzer := NewAnalyzer(cfg)

			// Test that configuration is respected
			recoverer := NewJSONRecoverer(tt.enableJSONRecovery, false)

			if recoverer.config.EnableRecovery != tt.enableJSONRecovery {
				t.Errorf("Expected EnableRecovery=%v, got %v", 
					tt.enableJSONRecovery, recoverer.config.EnableRecovery)
			}

			// Test actual recovery behavior
			result := recoverer.RecoverJSON("invalid json", errors.New("test error"))

			if tt.enableJSONRecovery {
				if result.Message == "JSON recovery disabled" {
					t.Error("Expected recovery to be enabled")
				}
			} else {
				if result.Message != "JSON recovery disabled" {
					t.Error("Expected recovery to be disabled")
				}
			}
		})
	}
}