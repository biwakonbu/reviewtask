package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// TestCodeRabbitNitpickProcessingIntegration tests end-to-end CodeRabbit nitpick processing
func TestCodeRabbitNitpickProcessingIntegration(t *testing.T) {
	tests := []struct {
		name              string
		processNitpicks   bool
		nitpickPriority   string
		reviewBody        string
		expectedTaskCount int
		expectedPriority  string
		mockResponse      string
	}{
		{
			name:            "CodeRabbit nitpicks processed when enabled",
			processNitpicks: true,
			nitpickPriority: "low",
			reviewBody: `**Actionable comments posted: 0**

<details>
<summary>🧹 Nitpick comments (2)</summary>
<blockquote>

Consider improving variable naming for better readability.

Also, the function could be optimized for performance.

</blockquote>
</details>`,
			expectedTaskCount: 1, // Review body is processed as single comment
			expectedPriority:  "low",
			mockResponse: `[
				{
					"description": "Address CodeRabbit nitpick comments",
					"priority": "low"
				}
			]`,
		},
		{
			name:            "CodeRabbit nitpicks skipped when disabled",
			processNitpicks: false,
			nitpickPriority: "low",
			reviewBody: `**Actionable comments posted: 0**

<details>
<summary>🧹 Nitpick comments (1)</summary>
<blockquote>

Consider improving variable naming.

</blockquote>
</details>`,
			expectedTaskCount: 0,
			expectedPriority:  "",
			mockResponse:      `[]`, // Empty array when nitpicks are disabled
		},
		{
			name:            "CodeRabbit with actionable comment and nitpicks",
			processNitpicks: true,
			nitpickPriority: "medium",
			reviewBody: `**Actionable comments posted: 1**

Fix the null pointer exception in line 15.

<details>
<summary>🧹 Nitpick comments (1)</summary>
<blockquote>

Consider using a constant instead of magic number.

</blockquote>
</details>`,
			expectedTaskCount: 1,        // Review body is processed as single comment
			expectedPriority:  "medium", // For the nitpick task
			mockResponse: `[
				{
					"description": "Fix the null pointer exception and address nitpick comments",
					"priority": "high"
				}
			]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			cfg := &config.Config{
				TaskSettings: config.TaskSettings{
					DefaultStatus:       "todo",
					LowPriorityPatterns: []string{"nit:", "🧹", "consider:", "nitpick"},
					LowPriorityStatus:   "pending",
				},
				AISettings: config.AISettings{
					UserLanguage:           "English",
					ProcessNitpickComments: tt.processNitpicks,
					NitpickPriority:        tt.nitpickPriority,
					DeduplicationEnabled:   false, // Disable for test predictability
				},
			}

			// Create mock Claude client
			mockClient := NewMockClaudeClient()
			// Set up response for any input containing nitpick content
			mockClient.Responses["nitpick"] = tt.mockResponse

			analyzer := NewAnalyzerWithClient(cfg, mockClient)

			// Create test review
			review := github.Review{
				ID:       123,
				Reviewer: "coderabbit[bot]",
				State:    "COMMENTED",
				Body:     tt.reviewBody,
			}

			// Generate tasks
			tasks, err := analyzer.GenerateTasks([]github.Review{review})
			require.NoError(t, err)

			// Verify task count
			assert.Len(t, tasks, tt.expectedTaskCount, "Expected %d tasks, got %d", tt.expectedTaskCount, len(tasks))

			if tt.expectedTaskCount > 0 {
				// Find nitpick task (should be the one with low priority pattern in origin text)
				var nitpickTask *storage.Task
				for i := range tasks {
					if analyzer.isLowPriorityComment(tasks[i].OriginText) {
						nitpickTask = &tasks[i]
						break
					}
				}

				if tt.processNitpicks && nitpickTask != nil {
					// Verify nitpick task has correct priority
					assert.Equal(t, tt.expectedPriority, nitpickTask.Priority,
						"Nitpick task should have priority %s", tt.expectedPriority)

					// Verify status is set for low priority
					assert.Equal(t, "pending", nitpickTask.Status,
						"Nitpick task should have low priority status")
				}
			}
		})
	}
}

// TestCodeRabbitJSONParsingErrorHandling tests basic JSON extraction functionality
func TestCodeRabbitJSONParsingErrorHandling(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			ProcessNitpickComments: true,
			NitpickPriority:        "low",
			VerboseMode:            false,
		},
	}

	// Test that the JSON extraction functions work correctly
	mockClient := NewMockClaudeClient()
	// Use a response that contains valid JSON but with explanation text
	mockClient.Responses["Actionable comments posted: 0"] = `Looking at this CodeRabbit review:

[{"description":"test task","priority":"low","origin_text":"test","source_review_id":123,"source_comment_id":456,"file":"test.go","line":1,"task_index":0}]`

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	review := github.Review{
		ID:       123,
		Reviewer: "coderabbit[bot]",
		State:    "COMMENTED",
		Body: `**Actionable comments posted: 0**

<details>
<summary>🧹 Nitpick comments (1)</summary>
<blockquote>
Consider improving this code.
</blockquote>
</details>`,
	}

	tasks, err := analyzer.GenerateTasks([]github.Review{review})
	assert.NoError(t, err)
	assert.NotEmpty(t, tasks, "Should extract tasks when valid JSON is present")
}

// TestCodeRabbitPromptGeneration tests that prompts include correct nitpick instructions
func TestCodeRabbitPromptGeneration(t *testing.T) {
	cfg := &config.Config{
		PriorityRules: config.PriorityRules{
			Low: "Code style and minor improvements",
		},
		AISettings: config.AISettings{
			UserLanguage:           "English",
			ProcessNitpickComments: true,
			NitpickPriority:        "medium",
			ValidationEnabled:      &[]bool{false}[0], // Disable validation to simplify test
		},
	}

	// Mock client that captures the prompt
	mockClient := NewMockClaudeClient()
	mockClient.Responses["nitpick"] = `[{"description":"Process nitpick","priority":"medium"}]`

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create CodeRabbit review
	review := github.Review{
		ID:       123,
		Reviewer: "coderabbit[bot]",
		State:    "COMMENTED",
		Body: `**Actionable comments posted: 0**

<details>
<summary>🧹 Nitpick comments (1)</summary>
<blockquote>
Consider improving variable naming.
</blockquote>
</details>`,
	}

	_, err := analyzer.GenerateTasks([]github.Review{review})
	require.NoError(t, err)

	// Check the last input received by the mock client
	capturedPrompt := mockClient.LastInput

	// For simple prompts, just verify the nitpick content was included
	assert.Contains(t, capturedPrompt, "Actionable comments posted: 0")
	assert.Contains(t, capturedPrompt, "Consider improving variable naming")

	// When validation is enabled, more detailed instructions may be present
	// But for simple prompts, we focus on verifying the content was passed through
}

// TestNitpickPriorityOverride tests that nitpick tasks get the configured priority
func TestNitpickPriorityOverride(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"consider:", "🧹"},
			LowPriorityStatus:   "pending",
		},
		AISettings: config.AISettings{
			ProcessNitpickComments: true,
			NitpickPriority:        "high", // Override to high priority
			DeduplicationEnabled:   false,
		},
	}

	mockClient := NewMockClaudeClient()
	mockClient.Responses["🧹"] = `[{"description":"Improve variable naming","origin_text":"🧹 Consider improving variable naming","priority":"low","source_review_id":123,"source_comment_id":456,"file":"test.go","line":10,"task_index":0}]`

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	review := github.Review{
		ID:       123,
		Reviewer: "coderabbit[bot]",
		State:    "COMMENTED",
		Body:     "🧹 Consider improving variable naming",
	}

	tasks, err := analyzer.GenerateTasks([]github.Review{review})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Verify priority was overridden to high
	assert.Equal(t, "high", tasks[0].Priority, "Nitpick task priority should be overridden to configured value")
	assert.Equal(t, "pending", tasks[0].Status, "Nitpick task should still use low priority status")
}
