package ai

import (
	"errors"
	"testing"
)

func TestJSONRecoverer_NewJSONRecoverer(t *testing.T) {
	tests := []struct {
		name            string
		enableRecovery  bool
		verboseMode     bool
		expectedEnabled bool
	}{
		{
			name:            "recovery enabled, verbose mode",
			enableRecovery:  true,
			verboseMode:     true,
			expectedEnabled: true,
		},
		{
			name:            "recovery disabled, verbose mode",
			enableRecovery:  false,
			verboseMode:     true,
			expectedEnabled: false,
		},
		{
			name:            "recovery enabled, quiet mode",
			enableRecovery:  true,
			verboseMode:     false,
			expectedEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recoverer := NewJSONRecoverer(tt.enableRecovery, tt.verboseMode)

			if recoverer.config.EnableRecovery != tt.expectedEnabled {
				t.Errorf("Expected EnableRecovery=%v, got %v", tt.expectedEnabled, recoverer.config.EnableRecovery)
			}

			if recoverer.verboseMode != tt.verboseMode {
				t.Errorf("Expected verboseMode=%v, got %v", tt.verboseMode, recoverer.verboseMode)
			}

			// Check default configuration
			if recoverer.config.MaxRecoveryAttempts != 3 {
				t.Errorf("Expected MaxRecoveryAttempts=3, got %d", recoverer.config.MaxRecoveryAttempts)
			}

			if recoverer.config.PartialThreshold != 0.7 {
				t.Errorf("Expected PartialThreshold=0.7, got %f", recoverer.config.PartialThreshold)
			}
		})
	}
}

func TestJSONRecoverer_CategorizeError(t *testing.T) {
	recoverer := NewJSONRecoverer(true, false)

	tests := []struct {
		name         string
		error        error
		expectedType string
	}{
		{
			name:         "unexpected end of JSON input",
			error:        errors.New("unexpected end of JSON input"),
			expectedType: "truncation",
		},
		{
			name:         "unexpected end of input",
			error:        errors.New("unexpected end of input"),
			expectedType: "truncation",
		},
		{
			name:         "invalid character",
			error:        errors.New("invalid character '}' looking for beginning of object key string"),
			expectedType: "malformed",
		},
		{
			name:         "cannot unmarshal",
			error:        errors.New("cannot unmarshal string into Go value of type int"),
			expectedType: "type_mismatch",
		},
		{
			name:         "unknown error",
			error:        errors.New("some other error"),
			expectedType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := recoverer.categorizeError(tt.error)
			if errorType != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, errorType)
			}
		})
	}
}

func TestJSONRecoverer_RecoverJSON_Disabled(t *testing.T) {
	recoverer := NewJSONRecoverer(false, false)
	err := errors.New("test error")

	result := recoverer.RecoverJSON("invalid json", err)

	if result.IsRecovered {
		t.Error("Expected recovery to be disabled")
	}

	if result.Message != "JSON recovery disabled" {
		t.Errorf("Expected disabled message, got: %s", result.Message)
	}
}

func TestJSONRecoverer_TryRecoverPartialArray(t *testing.T) {
	recoverer := NewJSONRecoverer(true, false)

	tests := []struct {
		name          string
		input         string
		expectedTasks int
		shouldSucceed bool
	}{
		{
			name: "complete truncated array",
			input: `[
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
					"priority": "medium",
					"source_review_id": 123,
					"source_comment_id": 457,
					"file": "test.go",
					"line": 20,
					"task_index": 1
				}`,
			expectedTasks: 2,
			shouldSucceed: true,
		},
		{
			name: "partial object at end",
			input: `[
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
					"priority": "medium",
					"source_review_id": 123,
					"source_comment_id": 457,
					"file": "test.go",
					"line": 20,
					"task_index":`,
			expectedTasks: 1,
			shouldSucceed: true,
		},
		{
			name:          "no array start",
			input:         `{"description": "test"}`,
			expectedTasks: 0,
			shouldSucceed: false,
		},
		{
			name:          "empty input",
			input:         "",
			expectedTasks: 0,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := recoverer.tryRecoverPartialArray(tt.input)

			if tt.shouldSucceed {
				if tasks == nil {
					t.Error("Expected recovery to succeed but got nil")
					return
				}
				if len(tasks) != tt.expectedTasks {
					t.Errorf("Expected %d tasks, got %d", tt.expectedTasks, len(tasks))
				}
			} else {
				if tasks != nil {
					t.Errorf("Expected recovery to fail but got %d tasks", len(tasks))
				}
			}
		})
	}
}

func TestJSONRecoverer_ExtractCompleteObjects(t *testing.T) {
	recoverer := NewJSONRecoverer(true, false)

	tests := []struct {
		name          string
		input         string
		expectedTasks int
	}{
		{
			name: "multiple complete objects",
			input: `Some text before
				{
					"description": "Fix the bug",
					"origin_text": "There is a bug here",
					"priority": "high",
					"source_review_id": 123,
					"source_comment_id": 456,
					"file": "test.go",
					"line": 10,
					"task_index": 0
				}
				More text
				{
					"description": "Add test",
					"origin_text": "Need more tests", 
					"priority": "medium",
					"source_review_id": 123,
					"source_comment_id": 457,
					"file": "test.go",
					"line": 20,
					"task_index": 1
				}`,
			expectedTasks: 2,
		},
		{
			name: "object with nested braces",
			input: `{
				"description": "Fix object with {nested} braces",
				"origin_text": "Code has {bad: 'syntax'}",
				"priority": "high",
				"source_review_id": 123,
				"source_comment_id": 456,
				"file": "test.go",
				"line": 10,
				"task_index": 0
			}`,
			expectedTasks: 1,
		},
		{
			name:          "no complete objects",
			input:         `{"description": "incomplete`,
			expectedTasks: 0,
		},
		{
			name: "invalid object content",
			input: `{
				"description": "",
				"origin_text": "",
				"priority": "",
				"source_comment_id": 0
			}`,
			expectedTasks: 0, // Should be filtered out as invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := recoverer.extractCompleteObjects(tt.input)

			if len(tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d", tt.expectedTasks, len(tasks))
			}
		})
	}
}

func TestJSONRecoverer_CleanMalformedJSON(t *testing.T) {
	recoverer := NewJSONRecoverer(true, false)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove markdown code blocks",
			input:    "```json\n[{\"test\": \"value\"}]\n```",
			expected: "[{\"test\": \"value\"}]",
		},
		{
			name:     "fix trailing comma in array",
			input:    "[{\"test\": \"value\"},]",
			expected: "[{\"test\": \"value\"}]",
		},
		{
			name:     "fix trailing comma in object",
			input:    "{\"test\": \"value\",}",
			expected: "{\"test\": \"value\"}",
		},
		{
			name:     "fix unquoted field names",
			input:    "{description: \"test\", priority: \"high\"}",
			expected: "{\"description\": \"test\", \"priority\": \"high\"}",
		},
		{
			name:     "multiple fixes",
			input:    "```json\n{description: \"test\",}\n```",
			expected: "{\"description\": \"test\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recoverer.cleanMalformedJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestJSONRecoverer_IsValidTaskRequest(t *testing.T) {
	recoverer := NewJSONRecoverer(true, false)

	tests := []struct {
		name     string
		task     TaskRequest
		expected bool
	}{
		{
			name: "valid task",
			task: TaskRequest{
				Description:     "Fix the bug",
				OriginText:      "There is a bug",
				Priority:        "high",
				SourceCommentID: 123,
			},
			expected: true,
		},
		{
			name: "missing description",
			task: TaskRequest{
				OriginText:      "There is a bug",
				Priority:        "high",
				SourceCommentID: 123,
			},
			expected: false,
		},
		{
			name: "missing origin text",
			task: TaskRequest{
				Description:     "Fix the bug",
				Priority:        "high",
				SourceCommentID: 123,
			},
			expected: false,
		},
		{
			name: "missing priority",
			task: TaskRequest{
				Description:     "Fix the bug",
				OriginText:      "There is a bug",
				SourceCommentID: 123,
			},
			expected: false,
		},
		{
			name: "missing comment ID",
			task: TaskRequest{
				Description: "Fix the bug",
				OriginText:  "There is a bug",
				Priority:    "high",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recoverer.isValidTaskRequest(tt.task)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestJSONRecoverer_FindCompleteJSONObjects(t *testing.T) {
	recoverer := NewJSONRecoverer(true, false)

	tests := []struct {
		name          string
		input         string
		expectedCount int
	}{
		{
			name:          "simple object",
			input:         `{"key": "value"}`,
			expectedCount: 1,
		},
		{
			name:          "object with string containing braces",
			input:         `{"description": "Fix {this} issue", "key": "value"}`,
			expectedCount: 1,
		},
		{
			name:          "object with escaped quotes",
			input:         `{"description": "Fix \"quoted\" text", "key": "value"}`,
			expectedCount: 1,
		},
		{
			name:          "multiple objects",
			input:         `{"first": "object"} some text {"second": "object"}`,
			expectedCount: 2,
		},
		{
			name:          "incomplete object",
			input:         `{"key": "value"`,
			expectedCount: 0,
		},
		{
			name:          "nested objects",
			input:         `{"outer": {"inner": "value"}}`,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := recoverer.findCompleteJSONObjects(tt.input)
			if len(objects) != tt.expectedCount {
				t.Errorf("Expected %d objects, got %d", tt.expectedCount, len(objects))
			}
		})
	}
}

func TestJSONRecoverer_RecoverJSON_Integration(t *testing.T) {
	recoverer := NewJSONRecoverer(true, false)

	tests := []struct {
		name              string
		input             string
		error             error
		expectedRecovered bool
		expectedTaskCount int
		expectedErrorType string
	}{
		{
			name: "successful truncation recovery",
			input: `[
				{
					"description": "Fix the bug",
					"origin_text": "There is a bug here",
					"priority": "high",
					"source_review_id": 123,
					"source_comment_id": 456,
					"file": "test.go",
					"line": 10,
					"task_index": 0
				}`,
			error:             errors.New("unexpected end of JSON input"),
			expectedRecovered: true,
			expectedTaskCount: 1,
			expectedErrorType: "truncation",
		},
		{
			name: "successful malformed recovery",
			input: `[
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
			]`,
			error:             errors.New("invalid character ']' looking for beginning of object key string"),
			expectedRecovered: true,
			expectedTaskCount: 1,
			expectedErrorType: "malformed",
		},
		{
			name:              "failed recovery - no valid data",
			input:             "completely invalid data",
			error:             errors.New("unexpected end of JSON input"),
			expectedRecovered: false,
			expectedTaskCount: 0,
			expectedErrorType: "truncation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recoverer.RecoverJSON(tt.input, tt.error)

			if result.IsRecovered != tt.expectedRecovered {
				t.Errorf("Expected IsRecovered=%v, got %v", tt.expectedRecovered, result.IsRecovered)
			}

			if len(result.Tasks) != tt.expectedTaskCount {
				t.Errorf("Expected %d tasks, got %d", tt.expectedTaskCount, len(result.Tasks))
			}

			if result.ErrorType != tt.expectedErrorType {
				t.Errorf("Expected error type %s, got %s", tt.expectedErrorType, result.ErrorType)
			}

			if result.OriginalSize != len(tt.input) {
				t.Errorf("Expected original size %d, got %d", len(tt.input), result.OriginalSize)
			}
		})
	}
}
