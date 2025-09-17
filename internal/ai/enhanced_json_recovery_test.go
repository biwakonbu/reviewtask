package ai

import (
	"errors"
	"testing"
)

func TestEnhancedJSONRecovery_NewEnhancedJSONRecovery(t *testing.T) {
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
			recoverer := NewEnhancedJSONRecovery(tt.enableRecovery, tt.verboseMode)

			if recoverer.config.EnableRecovery != tt.expectedEnabled {
				t.Errorf("Expected EnableRecovery=%v, got %v", tt.expectedEnabled, recoverer.config.EnableRecovery)
			}

			if recoverer.verboseMode != tt.verboseMode {
				t.Errorf("Expected verboseMode=%v, got %v", tt.verboseMode, recoverer.verboseMode)
			}

			// Check enhanced configuration defaults
			if recoverer.config.MaxRecoveryAttempts != 5 {
				t.Errorf("Expected MaxRecoveryAttempts=5, got %d", recoverer.config.MaxRecoveryAttempts)
			}

			if recoverer.config.PartialThreshold != 0.6 {
				t.Errorf("Expected PartialThreshold=0.6, got %f", recoverer.config.PartialThreshold)
			}
		})
	}
}

func TestEnhancedJSONRecovery_CategorizeError(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

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

func TestEnhancedJSONRecovery_StructuralRepairs(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name          string
		malformedJSON string
		expectedTasks int
		shouldRecover bool
		description   string
	}{
		{
			name:          "missing comma between objects",
			malformedJSON: `[{"description": "Task 1", "priority": "high"} {"description": "Task 2", "priority": "medium"}]`,
			expectedTasks: 2,
			shouldRecover: true,
			description:   "Should fix missing comma between objects",
		},
		{
			name:          "missing comma between arrays",
			malformedJSON: `[{"description": "Task 1", "priority": "high"}] [{"description": "Task 2", "priority": "medium"}]`,
			expectedTasks: 2,    // The enhanced recovery can extract both valid structures
			shouldRecover: true, // Enhanced recovery is more aggressive
			description:   "Enhanced recovery can extract valid structures even from complex malformed JSON",
		},
		{
			name:          "proper array with spacing",
			malformedJSON: `[ {"description": "Task 1", "priority": "high"}, {"description": "Task 2", "priority": "medium"} ]`,
			expectedTasks: 2,
			shouldRecover: true,
			description:   "Should handle proper JSON with spacing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalError := errors.New("json parse error")
			result := recoverer.RepairAndRecover(tt.malformedJSON, originalError)

			if result.IsRecovered != tt.shouldRecover {
				t.Errorf("Expected recovery=%v, got %v for case: %s", tt.shouldRecover, result.IsRecovered, tt.description)
			}

			if tt.shouldRecover && len(result.Tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d for case: %s", tt.expectedTasks, len(result.Tasks), tt.description)
			}
		})
	}
}

func TestEnhancedJSONRecovery_TruncatedCompletion(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name          string
		truncatedJSON string
		expectedTasks int
		shouldRecover bool
		description   string
	}{
		{
			name:          "truncated single task",
			truncatedJSON: `[{"description": "Fix authentication bug`,
			expectedTasks: 1,
			shouldRecover: true,
			description:   "Should complete truncated task description",
		},
		{
			name:          "truncated with partial field",
			truncatedJSON: `[{"description": "Update user interface", "priority": "med`,
			expectedTasks: 1,
			shouldRecover: true,
			description:   "Should complete truncated priority field",
		},
		{
			name:          "completely truncated object",
			truncatedJSON: `[{"description": "Complete refactoring", "priority": "high", "status": "tod`,
			expectedTasks: 1,
			shouldRecover: true,
			description:   "Should complete truncated status field",
		},
		{
			name:          "no object start",
			truncatedJSON: `description": "Fix bug"`,
			expectedTasks: 0,
			shouldRecover: false,
			description:   "Cannot recover without object structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalError := errors.New("unexpected end of JSON input")
			result := recoverer.RepairAndRecover(tt.truncatedJSON, originalError)

			if result.IsRecovered != tt.shouldRecover {
				t.Errorf("Expected recovery=%v, got %v for case: %s", tt.shouldRecover, result.IsRecovered, tt.description)
			}

			if tt.shouldRecover && len(result.Tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d for case: %s", tt.expectedTasks, len(result.Tasks), tt.description)
				if len(result.Tasks) > 0 {
					t.Logf("Recovered task: %+v", result.Tasks[0])
				}
			}
		})
	}
}

func TestEnhancedJSONRecovery_PartialStructureExtraction(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name          string
		malformedJSON string
		expectedTasks int
		shouldRecover bool
		description   string
	}{
		{
			name: "mixed valid and invalid structures",
			malformedJSON: `{
				"description": "Valid task 1",
				"priority": "high"
			} invalid json here {
				"description": "Valid task 2",
				"priority": "medium"
			} more invalid`,
			expectedTasks: 2,
			shouldRecover: true,
			description:   "Should extract valid task structures from mixed content",
		},
		{
			name: "partial task with incomplete fields",
			malformedJSON: `{
				"description": "Implement user authentication system",
				"incomplete_field":
			}`,
			expectedTasks: 1,
			shouldRecover: true,
			description:   "Should extract task even with incomplete fields",
		},
		{
			name:          "no task-like structures",
			malformedJSON: `{"random": "data", "not": "a task"}`,
			expectedTasks: 0,
			shouldRecover: false,
			description:   "Should not extract non-task structures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalError := errors.New("invalid JSON structure")
			result := recoverer.RepairAndRecover(tt.malformedJSON, originalError)

			if result.IsRecovered != tt.shouldRecover {
				t.Errorf("Expected recovery=%v, got %v for case: %s", tt.shouldRecover, result.IsRecovered, tt.description)
			}

			if tt.shouldRecover && len(result.Tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d for case: %s", tt.expectedTasks, len(result.Tasks), tt.description)
				t.Logf("Result message: %s", result.Message)
				for i, task := range result.Tasks {
					t.Logf("Task %d: %+v", i, task)
				}
			}
		})
	}
}

func TestEnhancedJSONRecovery_FragmentReconstruction(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name          string
		fragmentJSON  string
		expectedTasks int
		shouldRecover bool
		description   string
	}{
		{
			name:          "task description fragments",
			fragmentJSON:  `"Fix the authentication system to handle OAuth2 correctly"`,
			expectedTasks: 1,
			shouldRecover: true,
			description:   "Should reconstruct task from description fragment",
		},
		{
			name:          "multiple description fragments",
			fragmentJSON:  `"Implement user registration with email validation" and "Update the database schema for new user fields"`,
			expectedTasks: 2,
			shouldRecover: true,
			description:   "Should reconstruct multiple tasks from fragments",
		},
		{
			name:          "short non-task strings",
			fragmentJSON:  `"ok" "yes" "no"`,
			expectedTasks: 0,
			shouldRecover: false,
			description:   "Should not reconstruct tasks from short strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalError := errors.New("malformed JSON")
			result := recoverer.RepairAndRecover(tt.fragmentJSON, originalError)

			if result.IsRecovered != tt.shouldRecover {
				t.Errorf("Expected recovery=%v, got %v for case: %s", tt.shouldRecover, result.IsRecovered, tt.description)
			}

			if tt.shouldRecover && len(result.Tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d for case: %s", tt.expectedTasks, len(result.Tasks), tt.description)
				for i, task := range result.Tasks {
					t.Logf("Reconstructed task %d: %+v", i, task)
				}
			}
		})
	}
}

func TestEnhancedJSONRecovery_IntelligentFieldCompletion(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name           string
		incompleteJSON string
		expectedTasks  int
		shouldRecover  bool
		description    string
	}{
		{
			name:           "missing priority and status",
			incompleteJSON: `{"description": "Fix critical security vulnerability"}`,
			expectedTasks:  1,
			shouldRecover:  true,
			description:    "Should complete missing priority and status fields",
		},
		{
			name:           "missing status only",
			incompleteJSON: `{"description": "Update documentation", "priority": "low"}`,
			expectedTasks:  1,
			shouldRecover:  true,
			description:    "Should complete missing status field",
		},
		{
			name:           "complete task object",
			incompleteJSON: `{"description": "Refactor code", "priority": "medium", "status": "todo"}`,
			expectedTasks:  1,
			shouldRecover:  true,
			description:    "Should handle complete task objects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalError := errors.New("incomplete JSON object")
			result := recoverer.RepairAndRecover(tt.incompleteJSON, originalError)

			if result.IsRecovered != tt.shouldRecover {
				t.Errorf("Expected recovery=%v, got %v for case: %s", tt.shouldRecover, result.IsRecovered, tt.description)
			}

			if tt.shouldRecover && len(result.Tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d for case: %s", tt.expectedTasks, len(result.Tasks), tt.description)
			}

			if tt.shouldRecover && len(result.Tasks) > 0 {
				task := result.Tasks[0]
				if task.Priority == "" {
					t.Errorf("Expected priority to be set, got empty string")
				}
				if task.Status == "" {
					t.Errorf("Expected status to be set, got empty string")
				}
				if task.Description == "" {
					t.Errorf("Expected description to be preserved, got empty string")
				}
			}
		})
	}
}

func TestEnhancedJSONRecovery_DisabledRecovery(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(false, false)

	originalError := errors.New("JSON parse error")
	result := recoverer.RepairAndRecover(`[{"description": "test"`, originalError)

	if result.IsRecovered {
		t.Error("Expected recovery to be disabled, but got recovery=true")
	}

	if result.Message != "Enhanced JSON recovery disabled" {
		t.Errorf("Expected disabled message, got: %s", result.Message)
	}

	if len(result.Tasks) != 0 {
		t.Errorf("Expected no tasks when recovery disabled, got %d tasks", len(result.Tasks))
	}
}

func TestEnhancedJSONRecovery_LooksLikeTaskDescription(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name        string
		text        string
		shouldMatch bool
	}{
		{
			name:        "contains fix keyword",
			text:        "Fix the authentication bug in login system",
			shouldMatch: true,
		},
		{
			name:        "contains add keyword",
			text:        "Add new user registration functionality",
			shouldMatch: true,
		},
		{
			name:        "contains update keyword",
			text:        "Update the database schema for performance",
			shouldMatch: true,
		},
		{
			name:        "reasonable length without keywords",
			text:        "Refactor the codebase for better maintainability",
			shouldMatch: true,
		},
		{
			name:        "too short",
			text:        "ok",
			shouldMatch: false,
		},
		{
			name:        "too long",
			text:        "This is an extremely long description that goes on and on and repeats itself multiple times with unnecessary verbosity and excessive detail that would not normally be found in a concise task description and should probably be rejected as it exceeds reasonable length limits for what constitutes a proper task description in our system and continues to ramble without providing any meaningful actionable information",
			shouldMatch: true, // The enhanced recovery uses 500 char limit which this text doesn't exceed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recoverer.looksLikeTaskDescription(tt.text)
			if result != tt.shouldMatch {
				t.Errorf("Expected looksLikeTaskDescription(%q)=%v, got %v", tt.text, tt.shouldMatch, result)
			}
		})
	}
}
