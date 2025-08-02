package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// Helper function to create a mock client that simulates different failure scenarios
func createFailingMockClient(failureType string, maxFailures int) *MockClaudeClient {
	client := NewMockClaudeClient()
	
	switch failureType {
	case "truncated_json":
		// Return incomplete JSON that should trigger recovery
		client.Responses["There is a bug here"] = `[{"description":"Fix the bug","origin_text":"There is a bug here","priority":"high","source_review_id":123,"source_comment_id":456,"file":"test.go","line":10,"task_index":0},{"description":"Add test","origin_text":"Need more tests","priority":"medium","source_review_id":123,"source_comment_id":457,"file":"test.go","line":20,"task_index":1`
	case "malformed_json":
		// Return JSON with syntax errors
		client.Responses["There is a bug here"] = `[{"description":"Fix the bug","origin_text":"There is a bug here","priority":"high","source_review_id":123,"source_comment_id":456,"file":"test.go","line":10,"task_index":0,},]`
	case "api_error":
		client.Error = errors.New("Claude API execution failed: rate limit exceeded")
	case "prompt_too_large":
		client.Error = errors.New("prompt size exceeds maximum limit")
	case "network_timeout":
		client.Error = errors.New("network timeout occurred")
	}
	
	return client
}

func TestRecoveryMechanism_TruncatedJSONRecovery(t *testing.T) {
	// Setup
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	mockClient := createFailingMockClient("truncated_json", 1)
	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create test reviews
	reviews := []github.Review{
		{
			ID:       123,
			Body:     "This PR has some issues",
			Reviewer: "reviewer1",
			State:    "CHANGES_REQUESTED",
			Comments: []github.Comment{
				{
					ID:     456,
					File:   "test.go",
					Line:   10,
					Body:   "There is a bug here",
					Author: "reviewer1",
				},
			},
		},
	}

	// Execute
	tasks, err := analyzer.GenerateTasks(reviews)

	// Verify
	if err != nil {
		t.Errorf("Expected successful recovery, got error: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks to be recovered from truncated JSON")
	}

	// Should have made at least one API call
	if mockClient.CallCount == 0 {
		t.Error("Expected at least one API call")
	}

	// Verify task content
	if len(tasks) > 0 {
		task := tasks[0]
		if task.Description != "Fix the bug" {
			t.Errorf("Expected description 'Fix the bug', got '%s'", task.Description)
		}
		if task.OriginText != "There is a bug here" {
			t.Errorf("Expected origin text 'There is a bug here', got '%s'", task.OriginText)
		}
	}
}

func TestRecoveryMechanism_MalformedJSONRecovery(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	mockClient := createFailingMockClient("malformed_json", 1)
	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID:   123,
			Body: "Review body",
			Comments: []github.Comment{
				{
					ID:   456,
					Body: "There is a bug here",
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)

	if err != nil {
		t.Errorf("Expected successful recovery, got error: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks to be recovered from malformed JSON")
	}
}

func TestRecoveryMechanism_RetryWithPromptReduction(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	mockClient := createFailingMockClient("prompt_too_large", 2)

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create large review to trigger prompt size issues
	largeBody := strings.Repeat("This is a very long review comment with lots of text. ", 1000)
	
	reviews := []github.Review{
		{
			ID:   123,
			Body: largeBody,
			Comments: []github.Comment{
				{
					ID:   456,
					Body: largeBody,
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)

	if err != nil {
		t.Errorf("Expected successful retry with prompt reduction, got error: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks after prompt size reduction")
	}

	// Should have made multiple attempts
	if mockClient.callCount < 2 {
		t.Errorf("Expected at least 2 API calls (retries), got %d", mockClient.callCount)
	}
}

func TestRecoveryMechanism_NetworkTimeoutRetry(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	mockClient := NewMockClaudeClient()
	mockClient.shouldFail = true
	mockClient.failureType = "network_timeout"
	mockClient.maxCalls = 2

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID: 123,
			Comments: []github.Comment{
				{
					ID:   456,
					Body: "Network issue test",
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)

	if err != nil {
		t.Errorf("Expected successful retry after network timeout, got error: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks after network retry")
	}

	// Should have made multiple attempts
	if mockClient.callCount < 2 {
		t.Errorf("Expected at least 2 API calls (retries), got %d", mockClient.callCount)
	}
}

func TestRecoveryMechanism_CompleteFailure(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	mockClient := NewMockClaudeClient()
	mockClient.shouldFail = true
	mockClient.failureType = "api_error"
	mockClient.maxCalls = 10 // Always fail

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID: 123,
			Comments: []github.Comment{
				{
					ID:   456,
					Body: "This should fail completely",
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)

	if err == nil {
		t.Error("Expected error when recovery completely fails")
	}

	if len(tasks) != 0 {
		t.Errorf("Expected no tasks when completely failed, got %d", len(tasks))
	}

	// Should have made multiple retry attempts
	if mockClient.callCount < 3 {
		t.Errorf("Expected multiple retry attempts, got %d", mockClient.callCount)
	}
}

func TestRecoveryMechanism_DisabledRecovery(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: false, // Disabled
			VerboseMode:        false,
		},
	}

	mockClient := NewMockClaudeClient()
	mockClient.shouldFail = true
	mockClient.failureType = "truncated_json"
	mockClient.maxCalls = 10

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID: 123,
			Comments: []github.Comment{
				{
					ID:   456,
					Body: "Should fail without recovery",
				},
			},
		},
	}

	tasks, err := analyzer.GenerateTasks(reviews)

	if err == nil {
		t.Error("Expected error when recovery is disabled")
	}

	if len(tasks) != 0 {
		t.Errorf("Expected no tasks when recovery disabled, got %d", len(tasks))
	}

	// Should only make one attempt when recovery is disabled
	if mockClient.callCount != 1 {
		t.Errorf("Expected exactly 1 API call when recovery disabled, got %d", mockClient.callCount)
	}
}

func TestRecoveryMechanism_ResponseMonitoringIntegration(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	mockClient := NewMockClaudeClient()
	mockClient.shouldFail = true
	mockClient.failureType = "truncated_json"
	mockClient.maxCalls = 1

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID: 123,
			Comments: []github.Comment{
				{
					ID:   456,
					Body: "Test monitoring integration",
				},
			},
		},
	}

	// Execute with response monitoring
	tasks, err := analyzer.GenerateTasks(reviews)

	if err != nil {
		t.Errorf("Expected successful recovery, got error: %v", err)
	}

	// Verify that response monitor was used
	if analyzer.responseMonitor == nil {
		t.Error("Expected response monitor to be initialized")
	}

	// Check that events would be recorded (we can't easily verify the actual recording
	// without exposing internal state, but we can verify the monitor exists)
	monitor := analyzer.responseMonitor
	if monitor.config == nil {
		t.Error("Expected response monitor config to be initialized")
	}

	if !monitor.config.EnableMonitoring {
		t.Error("Expected monitoring to be enabled by default")
	}
}

func TestRecoveryMechanism_PartialRecoveryScenarios(t *testing.T) {
	tests := []struct {
		name              string
		response          string
		expectedTaskCount int
		shouldSucceed     bool
	}{
		{
			name: "multiple complete tasks with truncation",
			response: `{"type":"completion","subtype":"text","is_error":false,"result":"[{\"description\":\"Fix bug 1\",\"origin_text\":\"Bug 1\",\"priority\":\"high\",\"source_review_id\":123,\"source_comment_id\":456,\"file\":\"test.go\",\"line\":10,\"task_index\":0},{\"description\":\"Fix bug 2\",\"origin_text\":\"Bug 2\",\"priority\":\"medium\",\"source_review_id\":123,\"source_comment_id\":457,\"file\":\"test.go\",\"line\":20,\"task_index\":1},{\"description\":\"Incomplete"}`,
			expectedTaskCount: 2,
			shouldSucceed:     true,
		},
		{
			name: "single complete task with truncation",
			response: `{"type":"completion","subtype":"text","is_error":false,"result":"[{\"description\":\"Fix the bug\",\"origin_text\":\"Bug here\",\"priority\":\"high\",\"source_review_id\":123,\"source_comment_id\":456,\"file\":\"test.go\",\"line\":10,\"task_index\":0},{\"desc"}`,
			expectedTaskCount: 1,
			shouldSucceed:     true,
		},
		{
			name:              "completely corrupted response",
			response:          `{"type":"completion","subtype":"text","is_error":false,"result":"garbage data with no valid JSON"}`,
			expectedTaskCount: 0,
			shouldSucceed:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					EnableJSONRecovery: true,
					VerboseMode:        false,
				},
			}

			mockClient := NewMockClaudeClient()
			mockClient.responses = []string{tt.response}

			analyzer := NewAnalyzerWithClient(cfg, mockClient)

			reviews := []github.Review{
				{
					ID: 123,
					Comments: []github.Comment{
						{
							ID:   456,
							Body: "Test partial recovery",
						},
					},
				},
			}

			tasks, err := analyzer.GenerateTasks(reviews)

			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if len(tasks) != tt.expectedTaskCount {
					t.Errorf("Expected %d tasks, got %d", tt.expectedTaskCount, len(tasks))
				}
			} else {
				if err == nil && len(tasks) > 0 {
					t.Error("Expected failure but got successful result")
				}
			}
		})
	}
}

func TestRecoveryMechanism_PerformanceImpact(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        false,
		},
	}

	// Test that recovery doesn't significantly impact performance for successful cases
	mockClient := NewMockClaudeClient()
	mockClient.shouldFail = false // Always succeed

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	reviews := []github.Review{
		{
			ID: 123,
			Comments: []github.Comment{
				{
					ID:   456,
					Body: "Performance test",
				},
			},
		},
	}

	// Measure time for successful execution
	start := time.Now()
	tasks, err := analyzer.GenerateTasks(reviews)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks from successful execution")
	}

	// Should complete quickly (less than 1 second for simple case)
	if duration > time.Second {
		t.Errorf("Expected quick execution, took %v", duration)
	}

	// Should only make one API call for successful case
	if mockClient.callCount != 1 {
		t.Errorf("Expected exactly 1 API call for successful case, got %d", mockClient.callCount)
	}
}

func TestRecoveryMechanism_ConfigurationRespected(t *testing.T) {
	tests := []struct {
		name                string
		enableJSONRecovery  bool
		verboseMode         bool
		expectedRetryCount  int
	}{
		{
			name:               "recovery enabled, verbose mode",
			enableJSONRecovery: true,
			verboseMode:        true,
			expectedRetryCount: 3, // Should retry
		},
		{
			name:               "recovery enabled, quiet mode",
			enableJSONRecovery: true,
			verboseMode:        false,
			expectedRetryCount: 3, // Should retry
		},
		{
			name:               "recovery disabled",
			enableJSONRecovery: false,
			verboseMode:        false,
			expectedRetryCount: 1, // Should not retry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					EnableJSONRecovery: tt.enableJSONRecovery,
					VerboseMode:        tt.verboseMode,
				},
			}

			mockClient := NewMockClaudeClient()
			mockClient.shouldFail = true
			mockClient.failureType = "truncated_json"
			mockClient.maxCalls = 10 // Always fail

			analyzer := NewAnalyzerWithClient(cfg, mockClient)

			reviews := []github.Review{
				{
					ID: 123,
					Comments: []github.Comment{
						{
							ID:   456,
							Body: "Configuration test",
						},
					},
				},
			}

			// Execute (will fail, but we're testing configuration behavior)
			analyzer.GenerateTasks(reviews)

			// Check that retry behavior respects configuration
			if tt.enableJSONRecovery {
				if mockClient.callCount < tt.expectedRetryCount {
					t.Errorf("Expected at least %d retries when recovery enabled, got %d", 
						tt.expectedRetryCount, mockClient.callCount)
				}
			} else {
				if mockClient.callCount > 1 {
					t.Errorf("Expected no retries when recovery disabled, got %d calls", 
						mockClient.callCount)
				}
			}
		})
	}
}