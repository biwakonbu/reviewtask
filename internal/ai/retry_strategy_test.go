package ai

import (
	"errors"
	"testing"
	"time"
)

func TestNewRetryStrategy(t *testing.T) {
	tests := []struct {
		name        string
		verboseMode bool
	}{
		{
			name:        "verbose mode enabled",
			verboseMode: true,
		},
		{
			name:        "verbose mode disabled",
			verboseMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewRetryStrategy(tt.verboseMode)

			if strategy.verboseMode != tt.verboseMode {
				t.Errorf("Expected verboseMode=%v, got %v", tt.verboseMode, strategy.verboseMode)
			}

			// Check default configuration
			if !strategy.config.EnableSmartRetry {
				t.Error("Expected EnableSmartRetry to be true")
			}

			if strategy.config.MaxRetries != 3 {
				t.Errorf("Expected MaxRetries=3, got %d", strategy.config.MaxRetries)
			}

			if strategy.config.BaseDelay != time.Second*2 {
				t.Errorf("Expected BaseDelay=2s, got %v", strategy.config.BaseDelay)
			}

			if strategy.config.TruncationThreshold != 20000 {
				t.Errorf("Expected TruncationThreshold=20000, got %d", strategy.config.TruncationThreshold)
			}

			if strategy.truncationPattern == nil {
				t.Error("Expected truncationPattern to be initialized")
			}
		})
	}
}

func TestRetryStrategy_CategorizeRetryError(t *testing.T) {
	strategy := NewRetryStrategy(false)

	tests := []struct {
		name         string
		error        error
		expectedType string
	}{
		{
			name:         "JSON truncation error",
			error:        errors.New("unexpected end of JSON input"),
			expectedType: "json_truncation",
		},
		{
			name:         "prompt size limit error",
			error:        errors.New("prompt size exceeds maximum limit"),
			expectedType: "prompt_size_limit",
		},
		{
			name:         "rate limit error",
			error:        errors.New("rate limit exceeded"),
			expectedType: "rate_limit",
		},
		{
			name:         "too many requests error",
			error:        errors.New("too many requests"),
			expectedType: "rate_limit",
		},
		{
			name:         "timeout error",
			error:        errors.New("timeout occurred"),
			expectedType: "timeout",
		},
		{
			name:         "deadline exceeded error",
			error:        errors.New("context deadline exceeded"),
			expectedType: "timeout",
		},
		{
			name:         "network error",
			error:        errors.New("network connection failed"),
			expectedType: "network_error",
		},
		{
			name:         "invalid character error",
			error:        errors.New("invalid character 'x' looking for beginning of value"),
			expectedType: "malformed_response",
		},
		{
			name:         "unknown error",
			error:        errors.New("some other error"),
			expectedType: "unknown_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := strategy.categorizeRetryError(tt.error)
			if errorType != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, errorType)
			}
		})
	}
}

func TestRetryStrategy_DetermineRetryStrategy(t *testing.T) {
	strategy := NewRetryStrategy(false)

	tests := []struct {
		name             string
		errorType        string
		truncationScore  float64
		promptSize       int
		expectedStrategy string
	}{
		{
			name:             "high truncation score",
			errorType:        "json_truncation",
			truncationScore:  0.8,
			promptSize:       15000,
			expectedStrategy: "reduce_prompt_aggressive",
		},
		{
			name:             "large prompt with truncation",
			errorType:        "json_truncation",
			truncationScore:  0.5,
			promptSize:       25000,
			expectedStrategy: "reduce_prompt_moderate",
		},
		{
			name:             "small prompt with truncation",
			errorType:        "json_truncation",
			truncationScore:  0.3,
			promptSize:       10000,
			expectedStrategy: "simple_retry",
		},
		{
			name:             "prompt size limit",
			errorType:        "prompt_size_limit",
			truncationScore:  0.0,
			promptSize:       35000,
			expectedStrategy: "reduce_prompt_aggressive",
		},
		{
			name:             "rate limit",
			errorType:        "rate_limit",
			truncationScore:  0.0,
			promptSize:       15000,
			expectedStrategy: "exponential_backoff",
		},
		{
			name:             "timeout with large prompt",
			errorType:        "timeout",
			truncationScore:  0.0,
			promptSize:       25000,
			expectedStrategy: "reduce_prompt_moderate",
		},
		{
			name:             "timeout with small prompt",
			errorType:        "timeout",
			truncationScore:  0.0,
			promptSize:       10000,
			expectedStrategy: "exponential_backoff",
		},
		{
			name:             "network error",
			errorType:        "network_error",
			truncationScore:  0.0,
			promptSize:       15000,
			expectedStrategy: "exponential_backoff",
		},
		{
			name:             "malformed response",
			errorType:        "malformed_response",
			truncationScore:  0.0,
			promptSize:       15000,
			expectedStrategy: "simple_retry",
		},
		{
			name:             "unknown error",
			errorType:        "unknown_error",
			truncationScore:  0.0,
			promptSize:       15000,
			expectedStrategy: "simple_retry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategyType := strategy.determineRetryStrategy(tt.errorType, tt.truncationScore, tt.promptSize)
			if strategyType != tt.expectedStrategy {
				t.Errorf("Expected strategy %s, got %s", tt.expectedStrategy, strategyType)
			}
		})
	}
}

func TestRetryStrategy_CalculateDelay(t *testing.T) {
	strategy := NewRetryStrategy(false)

	tests := []struct {
		name        string
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "first retry (attempt 0)",
			attempt:     0,
			expectedMin: time.Second * 2,
			expectedMax: time.Second * 2,
		},
		{
			name:        "second retry (attempt 1)",
			attempt:     1,
			expectedMin: time.Second * 4,
			expectedMax: time.Second * 4,
		},
		{
			name:        "third retry (attempt 2)",
			attempt:     2,
			expectedMin: time.Second * 8,
			expectedMax: time.Second * 8,
		},
		{
			name:        "many retries (should cap at max)",
			attempt:     10,
			expectedMin: time.Second * 30,
			expectedMax: time.Second * 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := strategy.calculateDelay(tt.attempt)
			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("Expected delay between %v and %v, got %v", tt.expectedMin, tt.expectedMax, delay)
			}
		})
	}
}

func TestRetryStrategy_ShouldRetry(t *testing.T) {
	tests := []struct {
		name                  string
		enableRetry           bool
		maxRetries            int
		attempt               int
		shouldRetry           bool
		expectedAttemptNumber int
	}{
		{
			name:                  "first attempt, retry enabled",
			enableRetry:           true,
			maxRetries:            3,
			attempt:               0,
			shouldRetry:           true,
			expectedAttemptNumber: 1,
		},
		{
			name:                  "second attempt, retry enabled",
			enableRetry:           true,
			maxRetries:            3,
			attempt:               1,
			shouldRetry:           true,
			expectedAttemptNumber: 2,
		},
		{
			name:        "max attempts reached",
			enableRetry: true,
			maxRetries:  3,
			attempt:     3,
			shouldRetry: false,
		},
		{
			name:        "retry disabled",
			enableRetry: false,
			maxRetries:  3,
			attempt:     0,
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewRetryStrategy(false)
			strategy.config.EnableSmartRetry = tt.enableRetry
			strategy.config.MaxRetries = tt.maxRetries

			err := errors.New("test error")
			retryAttempt, shouldRetry := strategy.ShouldRetry(tt.attempt, err, 10000, 5000)

			if shouldRetry != tt.shouldRetry {
				t.Errorf("Expected shouldRetry=%v, got %v", tt.shouldRetry, shouldRetry)
			}

			if tt.shouldRetry {
				if retryAttempt == nil {
					t.Error("Expected retryAttempt to be non-nil when shouldRetry is true")
					return
				}

				if retryAttempt.AttemptNumber != tt.expectedAttemptNumber {
					t.Errorf("Expected attempt number %d, got %d", tt.expectedAttemptNumber, retryAttempt.AttemptNumber)
				}

				if retryAttempt.PromptSize != 10000 {
					t.Errorf("Expected prompt size 10000, got %d", retryAttempt.PromptSize)
				}

				if retryAttempt.ResponseSize != 5000 {
					t.Errorf("Expected response size 5000, got %d", retryAttempt.ResponseSize)
				}

				if retryAttempt.Error != "test error" {
					t.Errorf("Expected error 'test error', got '%s'", retryAttempt.Error)
				}
			}
		})
	}
}

func TestRetryStrategy_AdjustPromptForRetry(t *testing.T) {
	strategy := NewRetryStrategy(false)
	originalPrompt := `System prompt here

PR Reviews to analyze:

Review 1:
This is a very long review with lots of text that should be reduced when we need to retry...
` + string(make([]byte, 10000)) // Add padding to make it large

	tests := []struct {
		name              string
		strategy_type     string
		expectedReduction bool
		maxExpectedSize   int
	}{
		{
			name:              "aggressive reduction",
			strategy_type:     "reduce_prompt_aggressive",
			expectedReduction: true,
			maxExpectedSize:   len(originalPrompt) / 2,
		},
		{
			name:              "moderate reduction",
			strategy_type:     "reduce_prompt_moderate",
			expectedReduction: true,
			maxExpectedSize:   int(float64(len(originalPrompt)) * 0.7),
		},
		{
			name:              "simple retry - no reduction",
			strategy_type:     "simple_retry",
			expectedReduction: false,
			maxExpectedSize:   len(originalPrompt),
		},
		{
			name:              "exponential backoff - no reduction",
			strategy_type:     "exponential_backoff",
			expectedReduction: false,
			maxExpectedSize:   len(originalPrompt),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryAttempt := &RetryAttempt{
				Strategy: tt.strategy_type,
			}

			adjustedPrompt := strategy.AdjustPromptForRetry(originalPrompt, retryAttempt)

			if tt.expectedReduction {
				if len(adjustedPrompt) >= len(originalPrompt) {
					t.Errorf("Expected prompt reduction, but size increased or stayed same: %d -> %d",
						len(originalPrompt), len(adjustedPrompt))
				}

				if len(adjustedPrompt) > tt.maxExpectedSize {
					t.Errorf("Expected max size %d, got %d", tt.maxExpectedSize, len(adjustedPrompt))
				}

				// Should preserve system prompt
				if !containsSystemPrompt(adjustedPrompt) {
					t.Error("Expected system prompt to be preserved")
				}
			} else {
				if adjustedPrompt != originalPrompt {
					t.Error("Expected no prompt adjustment for this strategy")
				}
			}
		})
	}
}

func TestTruncationPatternDetector_AnalyzeResponse(t *testing.T) {
	detector := NewTruncationPatternDetector(false)

	tests := []struct {
		name          string
		promptSize    int
		responseSize  int
		errorType     string
		expectedScore float64
		tolerance     float64
	}{
		{
			name:          "JSON truncation with large prompt",
			promptSize:    35000,
			responseSize:  500,
			errorType:     "json_truncation",
			expectedScore: 0.8, // 0.4 (error) + 0.3 (size) + 0.1 (response)
			tolerance:     0.1,
		},
		{
			name:          "JSON truncation with medium prompt",
			promptSize:    25000,
			responseSize:  500,
			errorType:     "json_truncation",
			expectedScore: 0.7, // 0.4 (error) + 0.2 (size) + 0.1 (response)
			tolerance:     0.1,
		},
		{
			name:          "No truncation with small prompt",
			promptSize:    10000,
			responseSize:  5000,
			errorType:     "unknown",
			expectedScore: 0.0,
			tolerance:     0.1,
		},
		{
			name:          "Large prompt without JSON error",
			promptSize:    35000,
			responseSize:  10000,
			errorType:     "network_error",
			expectedScore: 0.3, // 0.3 (size only)
			tolerance:     0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := detector.AnalyzeResponse(tt.promptSize, tt.responseSize, tt.errorType)

			if score < tt.expectedScore-tt.tolerance || score > tt.expectedScore+tt.tolerance {
				t.Errorf("Expected score around %f (±%f), got %f", tt.expectedScore, tt.tolerance, score)
			}
		})
	}
}

func TestTruncationPatternDetector_RecordTruncation(t *testing.T) {
	detector := NewTruncationPatternDetector(false)

	// Record some events
	events := []struct {
		promptSize   int
		responseSize int
		errorType    string
	}{
		{10000, 5000, "json_truncation"},
		{20000, 3000, "json_truncation"},
		{30000, 1000, "json_truncation"},
	}

	for _, event := range events {
		detector.RecordTruncation(event.promptSize, event.responseSize, event.errorType)
	}

	if len(detector.truncationEvents) != len(events) {
		t.Errorf("Expected %d events recorded, got %d", len(events), len(detector.truncationEvents))
	}

	// Test size limits (should keep only last 20)
	for i := 0; i < 25; i++ {
		detector.RecordTruncation(1000, 500, "test")
	}

	if len(detector.truncationEvents) > 20 {
		t.Errorf("Expected max 20 events, got %d", len(detector.truncationEvents))
	}
}

func TestTruncationPatternDetector_GetTruncationStats(t *testing.T) {
	detector := NewTruncationPatternDetector(false)

	// Test empty stats
	stats := detector.GetTruncationStats()
	if stats["total_events"] != 0 {
		t.Error("Expected empty stats for new detector")
	}

	// Add some events
	detector.RecordTruncation(10000, 5000, "json_truncation")
	detector.RecordTruncation(20000, 3000, "json_truncation")
	detector.RecordTruncation(15000, 4000, "timeout")

	stats = detector.GetTruncationStats()

	if stats["total_events"] != 3 {
		t.Errorf("Expected 3 total events, got %v", stats["total_events"])
	}

	if stats["average_prompt_size"] != 15000 {
		t.Errorf("Expected average prompt size 15000, got %v", stats["average_prompt_size"])
	}

	if stats["average_response_size"] != 4000 {
		t.Errorf("Expected average response size 4000, got %v", stats["average_response_size"])
	}

	errorDist, ok := stats["error_type_distribution"].(map[string]int)
	if !ok {
		t.Error("Expected error_type_distribution to be map[string]int")
	} else {
		if errorDist["json_truncation"] != 2 {
			t.Errorf("Expected 2 json_truncation errors, got %d", errorDist["json_truncation"])
		}
		if errorDist["timeout"] != 1 {
			t.Errorf("Expected 1 timeout error, got %d", errorDist["timeout"])
		}
	}
}

// Helper function to check if system prompt is preserved
func containsSystemPrompt(prompt string) bool {
	systemIndicators := []string{
		"System prompt",
		"You are an AI assistant",
		"CRITICAL: Return response as JSON",
		"Requirements:",
	}

	for _, indicator := range systemIndicators {
		if len(prompt) > 0 && prompt[:min(len(prompt), 1000)] != "" {
			// Check first 1000 chars for system prompt indicators
			if len(prompt) >= len(indicator) {
				for i := 0; i <= min(1000, len(prompt)-len(indicator)); i++ {
					if len(prompt) >= i+len(indicator) && prompt[i:i+len(indicator)] == indicator {
						return true
					}
				}
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
