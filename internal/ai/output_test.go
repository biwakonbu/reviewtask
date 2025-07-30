package ai

import (
	"testing"
)

func TestIsErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		// Emoji indicators
		{
			name:     "Warning emoji",
			message:  "⚠️ This is a warning",
			expected: true,
		},
		{
			name:     "Error emoji",
			message:  "❌ This failed",
			expected: true,
		},
		// Case-insensitive keyword matching
		{
			name:     "Lowercase error",
			message:  "an error occurred",
			expected: true,
		},
		{
			name:     "Uppercase ERROR",
			message:  "ERROR: Something went wrong",
			expected: true,
		},
		{
			name:     "Mixed case Error",
			message:  "Error in processing",
			expected: true,
		},
		{
			name:     "Lowercase failed",
			message:  "operation failed",
			expected: true,
		},
		{
			name:     "Uppercase FAILED",
			message:  "TEST FAILED",
			expected: true,
		},
		{
			name:     "Warning keyword",
			message:  "Warning: deprecated function",
			expected: true,
		},
		{
			name:     "Exception keyword",
			message:  "Exception thrown in handler",
			expected: true,
		},
		// Non-error messages
		{
			name:     "Normal message",
			message:  "Processing complete",
			expected: false,
		},
		{
			name:     "Success message",
			message:  "All tests passed",
			expected: false,
		},
		{
			name:     "Info message",
			message:  "Starting process",
			expected: false,
		},
		// Edge cases
		{
			name:     "Empty string",
			message:  "",
			expected: false,
		},
		{
			name:     "Keyword as part of word",
			message:  "The warrior succeeded",
			expected: false,
		},
		{
			name:     "Keyword at boundary",
			message:  "It is an error to proceed",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isErrorMessage(tt.message)
			if result != tt.expected {
				t.Errorf("isErrorMessage(%q) = %v, want %v", tt.message, result, tt.expected)
			}
		})
	}
}
