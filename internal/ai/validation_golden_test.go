package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	cfgpkg "reviewtask/internal/config"
	gh "reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// MockAIProviderForValidation simulates AI responses for validation testing.
// It tracks calls to distinguish between analysis and validation requests by
// inspecting the prompt content.
type MockAIProviderForValidation struct {
	AnalysisResponse   string   // Response for analysis/task generation calls
	ValidationResponse string   // Response for validation calls
	CallHistory        []string // Track sequence of calls (analysis/validation)
}

func (m *MockAIProviderForValidation) Execute(ctx context.Context, prompt string, outputType string) (string, error) {
	// Detect if this is a validation call or analysis call based on prompt content
	isValidation := strings.Contains(prompt, "VALIDATION CRITERIA") ||
		strings.Contains(prompt, "GENERATED TASKS TO VALIDATE")

	if isValidation {
		m.CallHistory = append(m.CallHistory, "validation")
		return m.ValidationResponse, nil
	}

	m.CallHistory = append(m.CallHistory, "analysis")
	return m.AnalysisResponse, nil
}

func (m *MockAIProviderForValidation) Name() string {
	return "mock-validation"
}

func configWithValidation(enabled bool) *cfgpkg.Config {
	// Helper to create bool pointer
	boolPtr := func(b bool) *bool { return &b }

	return &cfgpkg.Config{
		PriorityRules: cfgpkg.PriorityRules{
			Critical: "Security vulnerabilities, authentication bypasses, data exposure risks",
			High:     "Performance bottlenecks, memory leaks, database optimization issues",
			Medium:   "Functional bugs, logic improvements, error handling",
			Low:      "Code style, naming conventions, comment improvements",
		},
		TaskSettings: cfgpkg.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: cfgpkg.AISettings{
			UserLanguage:            "English",
			PromptProfile:           "v2",
			ValidationEnabled:       boolPtr(enabled),
			MaxRetries:              3,
			QualityThreshold:        0.8,
			DeduplicationEnabled:    false,
			StreamProcessingEnabled: false, // Disable stream processing for simpler test flow
		},
	}
}

// TestValidationFlow_Golden verifies the validation-enabled flow produces correct output.
// It tests three scenarios:
// 1. validation_disabled: Uses processCommentSimple (1 AI call)
// 2. validation_enabled_pass: Uses processCommentWithValidation (2 AI calls: analysis + validation)
// 3. validation_enabled_with_issues: Validation finds issues but still returns tasks
func TestValidationFlow_Golden(t *testing.T) {
	cases := []struct {
		name               string
		validationEnabled  bool
		analysisResponse   string
		validationResponse string
		goldenPath         string
	}{
		{
			name:              "validation_disabled",
			validationEnabled: false,
			// When validation is disabled, processCommentSimple expects SimpleTaskRequest format (array)
			analysisResponse: `[
  {
    "description": "Add input validation to prevent security vulnerabilities",
    "priority": "critical"
  }
]`,
			validationResponse: "", // Not used when validation disabled
			goldenPath:         "testdata/validation/disabled.golden",
		},
		{
			name:              "validation_enabled_pass",
			validationEnabled: true,
			// When validation is enabled, processCommentWithValidation expects TaskRequest format (object with tasks array)
			analysisResponse: `{
  "tasks": [
    {
      "description": "Add input validation to prevent security vulnerabilities",
      "priority": "critical",
      "origin_text": "Input validation is missing. This could lead to security issues.",
      "source_review_id": 1,
      "source_comment_id": 101,
      "file": "internal/auth.go",
      "line": 42,
      "status": "todo",
      "task_index": 0
    }
  ]
}`,
			validationResponse: `{
  "validation": true,
  "score": 0.95,
  "issues": []
}`,
			goldenPath: "testdata/validation/enabled_pass.golden",
		},
		{
			name:              "validation_enabled_with_issues",
			validationEnabled: true,
			// Validation detects issues (low score) but still returns tasks
			// Note: In real scenarios with MaxRetries>1, this would trigger a retry,
			// but this test verifies that tasks are still returned even with validation issues
			analysisResponse: `{
  "tasks": [
    {
      "description": "Fix the bug",
      "priority": "medium",
      "origin_text": "Input validation is missing. This could lead to security issues.",
      "source_review_id": 1,
      "source_comment_id": 101,
      "file": "internal/auth.go",
      "line": 42,
      "status": "todo",
      "task_index": 0
    }
  ]
}`,
			validationResponse: `{
  "validation": false,
  "score": 0.65,
  "issues": [
    {
      "type": "content",
      "task_index": 0,
      "description": "Task description is too vague",
      "severity": "major",
      "suggestion": "Specify which bug and how to fix it"
    }
  ]
}`,
			goldenPath: "testdata/validation/enabled_with_issues.golden",
		},
	}

	// Common test comment context
	testCommentContext := CommentContext{
		Comment: gh.Comment{
			ID:        101,
			File:      "internal/auth.go",
			Line:      42,
			Body:      "Input validation is missing. This could lead to security issues.",
			Author:    "alice",
			CreatedAt: "2025-01-01T00:00:00Z",
		},
		SourceReview: gh.Review{
			ID:          1,
			Reviewer:    "alice",
			State:       "CHANGES_REQUESTED",
			Body:        "Please address the following issues.",
			SubmittedAt: "2025-01-01T00:00:00Z",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configWithValidation(tc.validationEnabled)

			// Create mock AI provider
			mockProvider := &MockAIProviderForValidation{
				AnalysisResponse:   tc.analysisResponse,
				ValidationResponse: tc.validationResponse,
			}

			// Create analyzer with mock provider
			analyzer := NewAnalyzerWithClient(cfg, mockProvider)

			// Test processCommentWithValidation vs processCommentSimple based on config
			var tasks []TaskRequest
			var err error

			if tc.validationEnabled {
				// Call validation-enabled processing
				tasks, err = analyzer.processCommentWithValidation(testCommentContext)
			} else {
				// Call simple processing (no validation)
				tasks, err = analyzer.processCommentSimple(testCommentContext)
			}

			if err != nil {
				t.Fatalf("Processing failed: %v", err)
			}

			// Build output representation
			got := buildValidationTestOutput(analyzer.convertToStorageTasks(tasks), mockProvider.CallHistory)

			goldenPath := filepath.Clean(tc.goldenPath)
			if updateGoldenEnabled() {
				writeGolden(t, goldenPath, got)
			}

			want := loadGolden(t, goldenPath)
			if got != want {
				t.Fatalf("validation output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}
		})
	}
}

// buildValidationTestOutput creates a normalized test output for golden comparison
func buildValidationTestOutput(tasks []storage.Task, callHistory []string) string {
	var b strings.Builder

	b.WriteString("=== Validation Test Output ===\n")
	b.WriteString(fmt.Sprintf("AI Call Sequence: %v\n", callHistory))
	b.WriteString(fmt.Sprintf("Total AI Calls: %d\n\n", len(callHistory)))

	b.WriteString("Generated Tasks:\n")
	for i, task := range tasks {
		b.WriteString("---\n")
		b.WriteString(fmt.Sprintf("Task %d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Description: %s\n", task.Description))
		b.WriteString(fmt.Sprintf("  Priority: %s\n", task.Priority))
		b.WriteString(fmt.Sprintf("  Status: %s\n", task.Status))
		b.WriteString(fmt.Sprintf("  Source Comment ID: %d\n", task.SourceCommentID))
		b.WriteString(fmt.Sprintf("  File: %s\n", task.File))
		b.WriteString(fmt.Sprintf("  Line: %d\n", task.Line))
		// Omit ID, timestamps, and URLs as they vary per run
	}

	if len(tasks) == 0 {
		b.WriteString("  (No tasks generated)\n")
	}

	return b.String()
}
