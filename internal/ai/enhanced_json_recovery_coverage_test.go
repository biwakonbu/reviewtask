package ai

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestCompleteTaskObject tests the completeTaskObject method with various edge cases
func TestEnhancedJSONRecovery_CompleteTaskObject(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name        string
		partialObj  string
		expectValid bool
		checkFields map[string]string
		description string
	}{
		{
			name:        "complete valid object",
			partialObj:  `{"description": "Fix bug", "priority": "high", "status": "doing"}`,
			expectValid: true,
			checkFields: map[string]string{
				"description": "Fix bug",
				"priority":    "high",
				"status":      "doing",
			},
			description: "Should handle already complete objects",
		},
		{
			name:        "missing priority and status",
			partialObj:  `{"description": "Update docs"}`,
			expectValid: true,
			checkFields: map[string]string{
				"description": "Update docs",
				"priority":    "medium", // default
				"status":      "todo",   // default
			},
			description: "Should add default values for missing fields",
		},
		{
			name:        "missing closing brace",
			partialObj:  `{"description": "Test task", "priority": "low"`,
			expectValid: true,
			checkFields: map[string]string{
				"description": "Test task",
				"priority":    "low",
				"status":      "todo", // default
			},
			description: "Should fix missing closing brace",
		},
		{
			name:        "missing opening brace",
			partialObj:  `"description": "Another task", "priority": "high"}`,
			expectValid: true,
			checkFields: map[string]string{
				"description": "Another task",
				"priority":    "high",
				"status":      "todo", // default
			},
			description: "Should fix missing opening brace",
		},
		{
			name:        "trailing comma",
			partialObj:  `{"description": "Task with comma", "priority": "medium",}`,
			expectValid: false, // completeTaskObject returns empty for invalid JSON
			checkFields: nil,
			description: "Should handle trailing comma",
		},
		{
			name:        "empty object",
			partialObj:  `{}`,
			expectValid: true,
			checkFields: map[string]string{
				"priority": "medium", // default
				"status":   "todo",   // default
			},
			description: "Should add all defaults for empty object",
		},
		{
			name:        "invalid JSON structure",
			partialObj:  `this is not json at all`,
			expectValid: false,
			checkFields: nil,
			description: "Should return empty string for invalid JSON",
		},
		{
			name:        "nested object (unsupported)",
			partialObj:  `{"description": "Task", "meta": {"author": "test"}}`,
			expectValid: true,
			checkFields: map[string]string{
				"description": "Task",
				"priority":    "medium", // default
				"status":      "todo",   // default
			},
			description: "Should handle nested objects gracefully",
		},
		{
			name:        "array instead of object",
			partialObj:  `["not", "an", "object"]`,
			expectValid: false,
			checkFields: nil,
			description: "Should reject arrays",
		},
		{
			name:        "null values",
			partialObj:  `{"description": null, "priority": "high"}`,
			expectValid: true,
			checkFields: map[string]string{
				"priority": "high",
				"status":   "todo", // default
			},
			description: "Should handle null values",
		},
		{
			name:        "unicode in description",
			partialObj:  `{"description": "修复漏洞 🐛"}`,
			expectValid: true,
			checkFields: map[string]string{
				"description": "修复漏洞 🐛",
				"priority":    "medium", // default
				"status":      "todo",   // default
			},
			description: "Should handle unicode characters",
		},
		{
			name:        "escaped quotes in description",
			partialObj:  `{"description": "Fix \"quoted\" text"}`,
			expectValid: true,
			checkFields: map[string]string{
				"description": `Fix "quoted" text`,
				"priority":    "medium", // default
				"status":      "todo",   // default
			},
			description: "Should handle escaped quotes",
		},
		{
			name:        "very long description",
			partialObj:  `{"description": "` + strings.Repeat("very long ", 100) + `"}`,
			expectValid: true,
			checkFields: map[string]string{
				"description": strings.Repeat("very long ", 100),
				"priority":    "medium", // default
				"status":      "todo",   // default
			},
			description: "Should handle very long descriptions",
		},
		{
			name:        "only whitespace",
			partialObj:  `   `,
			expectValid: true, // Function returns default fields for whitespace
			checkFields: map[string]string{
				"priority": "medium",
				"status":   "todo",
			},
			description: "Should return default fields for whitespace-only input",
		},
		{
			name:        "empty string",
			partialObj:  ``,
			expectValid: true, // Function returns default fields for empty string
			checkFields: map[string]string{
				"priority": "medium",
				"status":   "todo",
			},
			description: "Should return default fields for empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recoverer.completeTaskObject(tt.partialObj)

			if tt.expectValid {
				if result == "" {
					t.Errorf("Expected valid JSON for case '%s', got empty string", tt.description)
					return
				}

				// Parse the result
				var taskMap map[string]interface{}
				if err := json.Unmarshal([]byte(result), &taskMap); err != nil {
					t.Errorf("Failed to parse result JSON for case '%s': %v\nResult: %s", tt.description, err, result)
					return
				}

				// Check expected fields
				for field, expectedValue := range tt.checkFields {
					actualValue, exists := taskMap[field]
					if !exists {
						t.Errorf("Expected field '%s' not found in result for case '%s'", field, tt.description)
						continue
					}

					// Convert to string for comparison
					actualStr := ""
					switch v := actualValue.(type) {
					case string:
						actualStr = v
					case nil:
						actualStr = ""
					default:
						actualStr = "unexpected type"
					}

					if actualStr != expectedValue {
						t.Errorf("Field '%s' mismatch for case '%s': expected '%s', got '%s'",
							field, tt.description, expectedValue, actualStr)
					}
				}

				// Verify required fields exist
				requiredFields := []string{"priority", "status"}
				for _, field := range requiredFields {
					if _, exists := taskMap[field]; !exists {
						t.Errorf("Required field '%s' missing for case '%s'", field, tt.description)
					}
				}
			} else {
				if result != "" {
					t.Errorf("Expected empty string for invalid input case '%s', got: %s", tt.description, result)
				}
			}
		})
	}
}

// TestRepairStructuralIssues_EdgeCases tests edge cases for structural repair
func TestEnhancedJSONRecovery_RepairStructuralIssues_EdgeCases(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name          string
		malformedJSON string
		shouldRecover bool
		expectedTasks int
		description   string
	}{
		{
			name:          "multiple fixes needed simultaneously",
			malformedJSON: `[{"description": "Task 1"} {"description": "Task 2"}{"description": "Task 3"}]`,
			shouldRecover: false, // Current implementation doesn't handle missing commas between objects
			expectedTasks: 0,
			description:   "Complex case with multiple missing commas",
		},
		{
			name:          "deeply nested malformed structure",
			malformedJSON: `[{"description": "Task", "meta": {"tags": ["bug" "fix"]}}]`,
			shouldRecover: false,
			expectedTasks: 0,
			description:   "Cannot fix nested array issues",
		},
		{
			name:          "mixed quotes",
			malformedJSON: `[{"description': "Mixed quotes task"}]`,
			shouldRecover: false,
			expectedTasks: 0,
			description:   "Cannot fix mismatched quote types",
		},
		{
			name:          "extreme whitespace",
			malformedJSON: `[   {   "description"   :   "Whitespace task"   ,   "priority"   :   "high"   }   ]`,
			shouldRecover: false, // Valid JSON, repairStructuralIssues won't parse it
			expectedTasks: 0,
			description:   "Valid JSON with excessive whitespace",
		},
		{
			name:          "incomplete nested structure",
			malformedJSON: `[{"description": "Task", "subtasks": [{"desc`,
			shouldRecover: false,
			expectedTasks: 0,
			description:   "Cannot recover incomplete nested structures",
		},
		{
			name:          "repeated commas",
			malformedJSON: `[{"description": "Task 1",,,"priority": "high"}]`,
			shouldRecover: false,
			expectedTasks: 0,
			description:   "Cannot fix multiple consecutive commas",
		},
		{
			name:          "unicode boundary truncation",
			malformedJSON: `[{"description": "测试任务` + "\xe6", // incomplete UTF-8 sequence
			shouldRecover: false,
			expectedTasks: 0,
			description:   "Cannot recover from invalid UTF-8",
		},
		{
			name:          "control characters",
			malformedJSON: `[{"description": "Task` + "\x00\x01\x02" + `"}]`,
			shouldRecover: false,
			expectedTasks: 0,
			description:   "Should reject control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalError := errors.New("structural issue")
			result := recoverer.repairStructuralIssues(tt.malformedJSON, &JSONRecoveryResult{
				ErrorType: "malformed",
				Message:   originalError.Error(),
			})

			if tt.shouldRecover != result.IsRecovered {
				t.Errorf("Expected recovery=%v for case '%s', got %v", tt.shouldRecover, tt.description, result.IsRecovered)
			}

			if tt.shouldRecover && len(result.Tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks for case '%s', got %d", tt.expectedTasks, tt.description, len(result.Tasks))
			}
		})
	}
}

// TestIntelligentFieldCompletion_EdgeCases tests edge cases for field completion
func TestEnhancedJSONRecovery_IntelligentFieldCompletion_EdgeCases(t *testing.T) {
	recoverer := NewEnhancedJSONRecovery(true, false)

	tests := []struct {
		name           string
		incompleteJSON string
		shouldRecover  bool
		expectedTasks  int
		description    string
	}{
		{
			name:           "multiple incomplete objects",
			incompleteJSON: `{"description": "Task 1"} {"description": "Task 2"} {"description": "Task 3"}`,
			shouldRecover:  true,
			expectedTasks:  3,
			description:    "Should complete multiple objects",
		},
		{
			name:           "object with extra fields",
			incompleteJSON: `{"description": "Task", "extra": "field", "another": 123}`,
			shouldRecover:  true,
			expectedTasks:  1,
			description:    "Should preserve extra fields while adding defaults",
		},
		{
			name:           "empty description",
			incompleteJSON: `{"description": ""}`,
			shouldRecover:  true,
			expectedTasks:  1,
			description:    "Should handle empty description",
		},
		{
			name:           "description with special regex characters",
			incompleteJSON: `{"description": "Fix [abc] and (xyz) with $100"}`,
			shouldRecover:  true,
			expectedTasks:  1,
			description:    "Should handle regex special characters",
		},
		{
			name:           "very large number of objects",
			incompleteJSON: strings.Repeat(`{"description": "Task"} `, 100),
			shouldRecover:  true,
			expectedTasks:  100,
			description:    "Should handle many objects",
		},
		{
			name:           "malformed regex pattern",
			incompleteJSON: `{"description": "Task with [unclosed bracket"}`,
			shouldRecover:  true,
			expectedTasks:  1,
			description:    "Should handle malformed regex patterns in content",
		},
		{
			name:           "object without description field",
			incompleteJSON: `{"priority": "high", "status": "doing"}`,
			shouldRecover:  false,
			expectedTasks:  0,
			description:    "Should not match objects without description",
		},
		{
			name:           "description field not as string",
			incompleteJSON: `{"description": 12345}`,
			shouldRecover:  false,
			expectedTasks:  0,
			description:    "Should not match non-string descriptions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalError := errors.New("incomplete fields")
			result := recoverer.intelligentFieldCompletion(tt.incompleteJSON, &JSONRecoveryResult{
				ErrorType: "malformed",
				Message:   originalError.Error(),
			})

			if tt.shouldRecover != result.IsRecovered {
				t.Errorf("Expected recovery=%v for case '%s', got %v", tt.shouldRecover, tt.description, result.IsRecovered)
			}

			if tt.shouldRecover && len(result.Tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks for case '%s', got %d", tt.expectedTasks, tt.description, len(result.Tasks))
				t.Logf("Result: %+v", result)
			}
		})
	}
}

// TestRepairAndRecover_ComplexScenarios tests complex recovery scenarios
func TestEnhancedJSONRecovery_RepairAndRecover_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name           string
		verboseMode    bool
		enabled        bool
		rawResponse    string
		originalError  error
		expectRecovery bool
		minTasks       int
		description    string
	}{
		{
			name:           "disabled recovery returns immediately",
			verboseMode:    false,
			enabled:        false,
			rawResponse:    `[{"description": "Valid task"`,
			originalError:  errors.New("truncated"),
			expectRecovery: false,
			minTasks:       0,
			description:    "Should not attempt recovery when disabled",
		},
		{
			name:           "verbose mode with successful recovery",
			verboseMode:    true,
			enabled:        true,
			rawResponse:    `[{"description": "Task 1"} {"description": "Task 2"}]`,
			originalError:  errors.New("malformed"),
			expectRecovery: true,
			minTasks:       2,
			description:    "Should show verbose output and recover",
		},
		{
			name:           "all strategies fail",
			verboseMode:    false,
			enabled:        true,
			rawResponse:    `completely invalid non-json garbage @#$%^&*`,
			originalError:  errors.New("invalid"),
			expectRecovery: false,
			minTasks:       0,
			description:    "Should fail gracefully when no strategy works",
		},
		{
			name:           "recovery with size tracking",
			verboseMode:    false,
			enabled:        true,
			rawResponse:    strings.Repeat(`{"description": "Task `+strings.Repeat("x", 100)+`"} `, 10),
			originalError:  errors.New("malformed"),
			expectRecovery: true,
			minTasks:       10,
			description:    "Should track original and recovered sizes",
		},
		{
			name:           "empty response",
			verboseMode:    false,
			enabled:        true,
			rawResponse:    "",
			originalError:  errors.New("empty response"),
			expectRecovery: true, // RepairAndRecover returns a result even for empty input
			minTasks:       0,
			description:    "Should handle empty responses",
		},
		{
			name:           "response with only whitespace",
			verboseMode:    false,
			enabled:        true,
			rawResponse:    "   \n\t\r\n   ",
			originalError:  errors.New("whitespace only"),
			expectRecovery: true, // RepairAndRecover returns a result even for whitespace
			minTasks:       0,
			description:    "Should handle whitespace-only responses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recoverer := NewEnhancedJSONRecovery(tt.enabled, tt.verboseMode)
			result := recoverer.RepairAndRecover(tt.rawResponse, tt.originalError)

			if tt.expectRecovery != result.IsRecovered {
				t.Errorf("Expected recovery=%v for case '%s', got %v", tt.expectRecovery, tt.description, result.IsRecovered)
			}

			if tt.expectRecovery && len(result.Tasks) < tt.minTasks {
				t.Errorf("Expected at least %d tasks for case '%s', got %d", tt.minTasks, tt.description, len(result.Tasks))
			}

			if !tt.enabled && result.Message != "Enhanced JSON recovery disabled" {
				t.Errorf("Expected disabled message for case '%s', got: %s", tt.description, result.Message)
			}

			// Verify size tracking
			if result.OriginalSize != len(tt.rawResponse) {
				t.Errorf("Original size mismatch for case '%s': expected %d, got %d",
					tt.description, len(tt.rawResponse), result.OriginalSize)
			}

			if tt.expectRecovery && result.RecoveredSize == 0 {
				t.Errorf("Expected non-zero recovered size for successful recovery in case '%s'", tt.description)
			}
		})
	}
}

// Benchmark tests
func BenchmarkEnhancedJSONRecovery_RepairAndRecover(b *testing.B) {
	recoverer := NewEnhancedJSONRecovery(true, false)
	malformedJSON := `[{"description": "Task 1"} {"description": "Task 2"}]`
	err := errors.New("malformed JSON")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recoverer.RepairAndRecover(malformedJSON, err)
	}
}

func BenchmarkEnhancedJSONRecovery_CompleteTaskObject(b *testing.B) {
	recoverer := NewEnhancedJSONRecovery(true, false)
	partialObj := `{"description": "Test task"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recoverer.completeTaskObject(partialObj)
	}
}

func BenchmarkEnhancedJSONRecovery_LargeResponse(b *testing.B) {
	recoverer := NewEnhancedJSONRecovery(true, false)
	// Create a large malformed JSON response
	largeResponse := "[" + strings.Repeat(`{"description": "Task"} `, 1000) + "]"
	err := errors.New("malformed JSON")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recoverer.RepairAndRecover(largeResponse, err)
	}
}
