package ai

import (
	"reviewtask/internal/config"
	"testing"
)

func TestConsolidateTasksIfNeeded(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			VerboseMode: false, // Disable verbose output for tests
		},
	}
	analyzer := &Analyzer{config: cfg}

	testCases := []struct {
		name     string
		input    []SimpleTaskRequest
		expected int // Expected number of tasks after consolidation
	}{
		{
			name: "Single task - no consolidation needed",
			input: []SimpleTaskRequest{
				{Description: "Fix authentication bug", Priority: "high"},
			},
			expected: 1,
		},
		{
			name:     "Empty input",
			input:    []SimpleTaskRequest{},
			expected: 0,
		},
		{
			name: "Code duplication tasks - should consolidate",
			input: []SimpleTaskRequest{
				{Description: "Remove duplicated Execute method from RealCursorClient", Priority: "high"},
				{Description: "Refactor RealCursorClient to delegate to BaseCLIClient.Execute", Priority: "high"},
				{Description: "Update callers and tests to use delegated behavior", Priority: "medium"},
				{Description: "Delete redundant code block in cursor_client.go lines 245-320", Priority: "medium"},
			},
			expected: 1,
		},
		{
			name: "Implementation tasks - should consolidate",
			input: []SimpleTaskRequest{
				{Description: "Implement missing getCursorRulesTemplate function", Priority: "high"},
				{Description: "Add getCursorRulesTemplate to cmd package", Priority: "medium"},
				{Description: "Export the same signature used by cursor.go", Priority: "low"},
				{Description: "Add unit tests covering both templates", Priority: "medium"},
			},
			expected: 1,
		},
		{
			name: "Configuration tasks - should consolidate",
			input: []SimpleTaskRequest{
				{Description: "Add environment variable override for REVIEWTASK_AI_PROVIDER", Priority: "medium"},
				{Description: "Read os.Getenv first before falling back to config", Priority: "medium"},
				{Description: "Ensure the file imports os and strings", Priority: "low"},
			},
			expected: 1,
		},
		{
			name: "Mixed unrelated tasks - should still consolidate (same comment)",
			input: []SimpleTaskRequest{
				{Description: "Fix documentation", Priority: "low"},
				{Description: "Update tests", Priority: "medium"},
			},
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.consolidateTasksIfNeeded(tc.input)

			if len(result) != tc.expected {
				t.Errorf("Expected %d tasks after consolidation, got %d", tc.expected, len(result))
			}

			// If consolidation happened, verify the result
			if len(tc.input) > 1 && len(result) == 1 {
				// Should have taken highest priority
				expectedPriority := "low"
				priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}

				for _, task := range tc.input {
					if priorityOrder[task.Priority] > priorityOrder[expectedPriority] {
						expectedPriority = task.Priority
					}
				}

				if result[0].Priority != expectedPriority {
					t.Errorf("Expected priority %s, got %s", expectedPriority, result[0].Priority)
				}

				// Description should be non-empty
				if result[0].Description == "" {
					t.Error("Consolidated task description should not be empty")
				}
			}
		})
	}
}

func TestCreateUnifiedTaskDescription(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			VerboseMode: false,
		},
	}
	analyzer := &Analyzer{config: cfg}

	testCases := []struct {
		name     string
		input    []string
		contains []string // Strings that should be contained in the result
	}{
		{
			name:     "Single description",
			input:    []string{"Fix authentication bug"},
			contains: []string{"Fix authentication bug"},
		},
		{
			name: "Code duplication pattern",
			input: []string{
				"Remove duplicated Execute method",
				"Refactor RealCursorClient to delegate",
			},
			contains: []string{"Remove code duplication", "refactoring", "CursorClient"},
		},
		{
			name: "Implementation pattern",
			input: []string{
				"Implement missing getCursorRulesTemplate function",
				"Add unit tests covering both templates",
			},
			contains: []string{"Implement", "comprehensive implementation"},
		},
		{
			name: "Configuration pattern",
			input: []string{
				"Add environment variable override",
				"Read os.Getenv first",
			},
			contains: []string{"configuration"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.createUnifiedTaskDescription(tc.input)

			if result == "" {
				t.Error("Unified description should not be empty")
			}

			for _, should_contain := range tc.contains {
				if result == should_contain || containsIgnoreCase(result, should_contain) {
					// Found at least one expected string
					return
				}
			}

			// If we reach here, none of the expected strings were found
			if len(tc.contains) > 0 {
				t.Errorf("Result '%s' should contain at least one of %v", result, tc.contains)
			}
		})
	}
}

// Helper function for case-insensitive substring check
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		findIgnoreCase(s, substr) >= 0
}

func findIgnoreCase(s, substr string) int {
	s_lower := toLower(s)
	substr_lower := toLower(substr)

	for i := 0; i <= len(s_lower)-len(substr_lower); i++ {
		if s_lower[i:i+len(substr_lower)] == substr_lower {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	result := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else {
			result += string(r)
		}
	}
	return result
}
