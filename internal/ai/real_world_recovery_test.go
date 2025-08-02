package ai

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// TestRealWorldClaudeAPIFailures tests recovery mechanisms against actual patterns
// observed from Claude API failures as described in Issue #140
func TestRealWorldClaudeAPIFailures(t *testing.T) {

	// Real-world test scenarios based on Issue #140
	testCases := []struct {
		name              string
		truncatedJSON     string
		originalError     string
		expectedRecover   bool
		minTasksRecovered int
		description       string
	}{
		{
			name: "Claude API sudden truncation mid-array",
			truncatedJSON: `[
				{
					"description": "Fix memory leak in HTTP client",
					"origin_text": "The HTTP client is not properly closing connections, causing memory leaks over time",
					"priority": "high",
					"source_review_id": 123456,
					"source_comment_id": 789012,
					"file": "internal/client/http.go",
					"line": 45,
					"task_index": 0
				},
				{
					"description": "Add proper error handling for database timeouts",
					"origin_text": "Database operations should have timeout handling",
					"priority": "medium",
					"source_review_id": 123456,
					"source_comment_id": 789013,
					"file": "internal/db/connection.go",
					"line": 78,
					"task_index": 1
				},
				{
					"description": "Implement retry logic for failed API calls",
					"origin_text": "API calls should be retried with exponential backoff when they fail",
					"priority": "high",
					"source_review_id": 123456,
					"source_comment_id": 789014,
					"file": "internal/api/client.go",
					"line": 123,
					"task_in`, // Suddenly cut off during field name
			originalError:     "unexpected end of JSON input",
			expectedRecover:   true,
			minTasksRecovered: 2, // Should recover the first 2 complete tasks
			description:       "Simulates real Claude API truncation during large response generation",
		},
		{
			name: "Claude API truncation with malformed trailing JSON",
			truncatedJSON: `[
				{
					"description": "Update documentation for new API endpoints",
					"origin_text": "The API documentation needs to be updated to reflect the new endpoints",
					"priority": "low",
					"source_review_id": 123456,
					"source_comment_id": 789015,
					"file": "docs/api.md",
					"line": 12,
					"task_index": 0
				},
				{
					"description": "Add validation for user input",
					"origin_text": "User input should be validated before processing",
					"priority": "critical",
					"source_review_id": 123456,
					"source_comment_id": 789016,
					"file": "internal/validation/input.go",
					"line": 67,
					"task_index": 1
				},
				{
					"description": "Fix broken unit test",
					"origin_text": "TestUserValidation is failing due to mock setup",
					"priority": "medium",,
					"source_comment_id": 789017,
					"file": "tests/validation_test.go"
					"task_index": 2
				`,
			originalError:     "invalid character ',' looking for beginning of object key string",
			expectedRecover:   true,
			minTasksRecovered: 2, // Should recover despite malformed JSON in third object
			description:       "Tests recovery from malformed JSON with syntax errors",
		},
		{
			name:              "Claude API response with markdown wrapper truncation",
			truncatedJSON:     "```json\n[\n\t{\n\t\t\"description\": \"Refactor authentication module\",\n\t\t\"origin_text\": \"The authentication module needs refactoring for better security\",\n\t\t\"priority\": \"high\",\n\t\t\"source_review_id\": 123456,\n\t\t\"source_comment_id\": 789018,\n\t\t\"file\": \"internal/auth/handler.go\",\n\t\t\"line\": 89,\n\t\t\"task_index\": 0\n\t}\n]", // Missing closing ```
			originalError:     "unexpected end of JSON input",
			expectedRecover:   true,
			minTasksRecovered: 1,
			description:       "Tests handling of markdown code blocks that get truncated",
		},
		{
			name:              "Large response truncation at token limit",
			truncatedJSON:     generateLargeTruncatedResponse(),
			originalError:     "unexpected end of JSON input",
			expectedRecover:   true,
			minTasksRecovered: 8, // Should recover most of the tasks before truncation
			description:       "Simulates truncation due to Claude API token limits",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing scenario: %s", tc.description)

			// Test JSON recovery directly
			recoverer := NewJSONRecoverer(true, true) // Verbose mode for debugging
			result := recoverer.RecoverJSON(tc.truncatedJSON, errors.New(tc.originalError))

			if result.IsRecovered != tc.expectedRecover {
				t.Errorf("Expected recovery success: %v, got: %v", tc.expectedRecover, result.IsRecovered)
				t.Logf("Recovery message: %s", result.Message)
				return
			}

			if tc.expectedRecover {
				if len(result.Tasks) < tc.minTasksRecovered {
					t.Errorf("Expected at least %d recovered tasks, got %d", tc.minTasksRecovered, len(result.Tasks))
					t.Logf("Recovered tasks: %+v", result.Tasks)
				}

				// Verify task quality
				for i, task := range result.Tasks {
					if task.Description == "" {
						t.Errorf("Task %d has empty description", i)
					}
					if task.OriginText == "" {
						t.Errorf("Task %d has empty origin text", i)
					}
					if task.Priority == "" {
						t.Errorf("Task %d has empty priority", i)
					}
				}

				t.Logf("Successfully recovered %d tasks from truncated response", len(result.Tasks))
			}
		})
	}
}

// TestEndToEndRecoveryWithAnalyzer tests the complete recovery flow
// through the actual analyzer code path that was failing in Issue #140
func TestEndToEndRecoveryWithAnalyzer(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			EnableJSONRecovery: true,
			VerboseMode:        true,
		},
	}

	// Create a mock Claude client that returns truncated responses
	mockClient := NewMockClaudeClient()

	// Simulate the exact failure pattern from Issue #140
	mockClient.Responses["Test comment that causes truncation"] = `[
		{
			"description": "Fix the critical bug in the payment processor",
			"origin_text": "Test comment that causes truncation",
			"priority": "critical",
			"source_review_id": 100,
			"source_comment_id": 200,
			"file": "payment/processor.go",
			"line": 45,
			"task_index": 0
		},
		{
			"description": "Add comprehensive logging",
			"origin_text": "Test comment that causes truncation",
			"priority": "medium",
			"source_review_id": 100,
			"source_comment_id": 200,
			"file": "logging/logger.go",
			"line": 23,
			"task_in` // Truncated mid-field

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create test reviews that match the failure pattern
	reviews := []github.Review{
		{
			ID:       100,
			Reviewer: "test-reviewer",
			Comments: []github.Comment{
				{
					ID:     200,
					File:   "payment/processor.go",
					Line:   45,
					Body:   "Test comment that causes truncation",
					Author: "test-reviewer",
				},
			},
		},
	}

	// This should NOT fail even with truncated response
	tasks, err := analyzer.GenerateTasks(reviews)

	if err != nil {
		t.Errorf("Expected analyzer to handle truncated response gracefully, got error: %v", err)
		return
	}

	if len(tasks) == 0 {
		t.Error("Expected at least some tasks to be recovered from truncated response")
		return
	}

	// Should recover at least the first complete task
	if len(tasks) < 1 {
		t.Error("Expected at least 1 task to be recovered")
	}

	// Verify the recovered task is valid
	task := tasks[0]
	if task.Description != "Fix the critical bug in the payment processor" {
		t.Errorf("Expected specific task description, got: %s", task.Description)
	}

	t.Logf("Successfully recovered %d tasks from truncated Claude API response", len(tasks))
}

// TestRetryStrategyWithRealPatterns tests retry logic with realistic failure patterns
func TestRetryStrategyWithRealPatterns(t *testing.T) {
	strategy := NewRetryStrategy(true) // Verbose mode

	realWorldErrors := []struct {
		name             string
		error            error
		promptSize       int
		responseSize     int
		expectedStrategy string
		description      string
	}{
		{
			name:             "Large prompt causes truncation",
			error:            errors.New("unexpected end of JSON input"),
			promptSize:       35000,                    // Large prompt
			responseSize:     1200,                     // Small response suggests truncation
			expectedStrategy: "reduce_prompt_moderate", // Large prompt > threshold triggers moderate reduction
			description:      "Large prompts often cause Claude to truncate responses",
		},
		{
			name:             "Severe truncation with high score",
			error:            errors.New("unexpected end of JSON input"),
			promptSize:       40000,                      // Very large prompt
			responseSize:     200,                        // Very small response -> high truncation score
			expectedStrategy: "reduce_prompt_aggressive", // High truncation score triggers aggressive reduction
			description:      "Severe truncation patterns require aggressive prompt reduction",
		},
		{
			name:             "Medium prompt with JSON errors",
			error:            errors.New("invalid character '}' looking for beginning of object key string"),
			promptSize:       18000,
			responseSize:     3400,
			expectedStrategy: "simple_retry",
			description:      "Malformed JSON might be temporary, try simple retry first",
		},
		{
			name:             "Rate limiting during high usage",
			error:            errors.New("rate limit exceeded"),
			promptSize:       15000,
			responseSize:     0,
			expectedStrategy: "exponential_backoff",
			description:      "Rate limits require exponential backoff strategy",
		},
	}

	for _, testCase := range realWorldErrors {
		t.Run(testCase.name, func(t *testing.T) {
			t.Logf("Testing pattern: %s", testCase.description)

			retryAttempt, shouldRetry := strategy.ShouldRetry(0, testCase.error, testCase.promptSize, testCase.responseSize)

			if !shouldRetry {
				t.Error("Expected retry to be recommended for real-world error pattern")
				return
			}

			if retryAttempt.Strategy != testCase.expectedStrategy {
				t.Errorf("Expected strategy %s, got %s", testCase.expectedStrategy, retryAttempt.Strategy)
			}

			t.Logf("Correctly identified strategy: %s", retryAttempt.Strategy)
		})
	}
}

// generateLargeTruncatedResponse creates a large response that gets truncated
// to simulate Claude API token limit truncations
func generateLargeTruncatedResponse() string {
	var builder strings.Builder
	builder.WriteString("[\n")

	// Generate 10 complete tasks
	for i := 0; i < 10; i++ {
		if i > 0 {
			builder.WriteString(",\n")
		}

		taskLetter := string(rune('A' + i))
		moduleNum := strconv.Itoa(i)
		reviewID := strconv.Itoa(123450 + i)
		commentID := strconv.Itoa(678900 + i)
		lineNum := strconv.Itoa(i * 10)
		taskIndex := strconv.Itoa(i)

		builder.WriteString(`	{
		"description": "Task ` + taskLetter + `: Fix critical issue in module ` + moduleNum + `",
		"origin_text": "This is a detailed comment about issue ` + taskLetter + ` that requires attention. The issue involves complex logic that needs to be refactored for better performance and maintainability. This comment contains a lot of detail to simulate real-world review comments that can be quite lengthy.",
		"priority": "high",
		"source_review_id": ` + reviewID + `,
		"source_comment_id": ` + commentID + `,
		"file": "module` + moduleNum + `/handler.go",
		"line": ` + lineNum + `,
		"task_index": ` + taskIndex + `
	}`)
	}

	// Add a truncated final task to simulate token limit cutoff
	builder.WriteString(`,
	{
		"description": "Final task that gets truncated",
		"origin_text": "This task gets cut off because Claude API reaches its token limit while generating`)
	// Suddenly truncated here - no closing bracket

	return builder.String()
}
