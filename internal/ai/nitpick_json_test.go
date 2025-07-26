package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// TestExtractJSON tests the improved JSON extraction logic
func TestExtractJSON(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			DebugMode: false,
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name:     "valid JSON array",
			response: `[{"description": "test task", "priority": "low"}]`,
			expected: `[{"description": "test task", "priority": "low"}]`,
		},
		{
			name:     "JSON array with markdown",
			response: "```json\n[{\"description\": \"test task\", \"priority\": \"low\"}]\n```",
			expected: `[{"description": "test task", "priority": "low"}]`,
		},
		{
			name:     "single JSON object",
			response: `{"description": "test task", "priority": "low"}`,
			expected: `[{"description": "test task", "priority": "low"}]`,
		},
		{
			name:     "text before JSON array",
			response: "I need to analyze the comments.\n\n[{\"description\": \"test task\", \"priority\": \"low\"}]",
			expected: `[{"description": "test task", "priority": "low"}]`,
		},
		{
			name:     "no actionable tasks response",
			response: "After analyzing the comment, I found no actionable tasks that require implementation.",
			expected: "[]",
		},
		{
			name:     "empty array literal",
			response: "[]",
			expected: "[]",
		},
		{
			name:     "already resolved response",
			response: "This comment appears to be already resolved in the discussion thread.",
			expected: "[]",
		},
		{
			name:     "no JSON content",
			response: "This is just a plain text response without any JSON.",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractJSON(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsCodeRabbitNitpickResponse tests CodeRabbit nitpick detection
func TestIsCodeRabbitNitpickResponse(t *testing.T) {
	cfg := &config.Config{}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		response string
		expected bool
	}{
		{
			name:     "actionable comments zero",
			response: "I see this comment has 'Actionable comments posted: 0' but contains nitpick suggestions.",
			expected: true,
		},
		{
			name:     "nitpick comments mentioned",
			response: "This appears to be from a CodeRabbit review with nitpick comments that don't require immediate action.",
			expected: true,
		},
		{
			name:     "analyze actionable tasks",
			response: "I need to analyze if it contains any actionable tasks despite the zero count.",
			expected: true,
		},
		{
			name:     "no actionable tasks in nitpick",
			response: "After review, I found no actionable tasks in the nitpick suggestions.",
			expected: true,
		},
		{
			name:     "regular comment",
			response: "This is a regular code review comment requiring implementation changes.",
			expected: false,
		},
		{
			name:     "empty response",
			response: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isCodeRabbitNitpickResponse(tt.response)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildAnalysisPromptWithNitpickInstructions tests prompt generation with nitpick handling
func TestBuildAnalysisPromptWithNitpickInstructions(t *testing.T) {
	tests := []struct {
		name                      string
		processNitpicks           bool
		nitpickPriority           string
		expectNitpickInstructions bool
		expectPriorityMention     bool
	}{
		{
			name:                      "nitpick processing enabled",
			processNitpicks:           true,
			nitpickPriority:           "low",
			expectNitpickInstructions: true,
			expectPriorityMention:     true,
		},
		{
			name:                      "nitpick processing disabled",
			processNitpicks:           false,
			nitpickPriority:           "low",
			expectNitpickInstructions: true, // Still includes instructions but to skip
			expectPriorityMention:     false,
		},
		{
			name:                      "custom nitpick priority",
			processNitpicks:           true,
			nitpickPriority:           "medium",
			expectNitpickInstructions: true,
			expectPriorityMention:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				PriorityRules: config.PriorityRules{
					Critical: "Security issues",
					High:     "Performance issues",
					Medium:   "Functional bugs",
					Low:      "Code style",
				},
				AISettings: config.AISettings{
					UserLanguage:           "English",
					ProcessNitpickComments: tt.processNitpicks,
					NitpickPriority:        tt.nitpickPriority,
				},
			}
			analyzer := NewAnalyzer(cfg)

			// Create empty reviews for test
			reviews := []github.Review{}
			prompt := analyzer.buildAnalysisPrompt(reviews)

			if tt.expectNitpickInstructions {
				assert.Contains(t, prompt, "Nitpick Comment Processing")
			}

			if tt.expectPriorityMention && tt.processNitpicks {
				assert.Contains(t, prompt, tt.nitpickPriority)
			}

			if tt.processNitpicks {
				assert.Contains(t, prompt, "Process nitpick comments")
				assert.Contains(t, prompt, "Actionable comments posted: 0")
			} else {
				assert.Contains(t, prompt, "Skip nitpick comments")
			}
		})
	}
}

// TestBuildCommentPromptWithNitpickInstructions tests single comment prompt generation
func TestBuildCommentPromptWithNitpickInstructions(t *testing.T) {
	cfg := &config.Config{
		PriorityRules: config.PriorityRules{
			Low: "Code style issues",
		},
		AISettings: config.AISettings{
			UserLanguage:           "English",
			ProcessNitpickComments: true,
			NitpickPriority:        "low",
		},
	}
	analyzer := NewAnalyzer(cfg)

	ctx := CommentContext{
		Comment: github.Comment{
			ID:     123,
			File:   "test.go",
			Line:   42,
			Body:   "**Actionable comments posted: 0**\n\n<details>\n<summary>🧹 Nitpick comments (1)</summary>\nConsider improving variable naming.\n</details>",
			Author: "coderabbit[bot]",
		},
		SourceReview: github.Review{
			ID:       456,
			Reviewer: "coderabbit[bot]",
			State:    "COMMENTED",
		},
	}

	prompt := analyzer.buildCommentPrompt(ctx)

	// Verify nitpick instructions are included
	assert.Contains(t, prompt, "Nitpick Comment Processing Instructions")
	assert.Contains(t, prompt, "Process nitpick comments")
	assert.Contains(t, prompt, "Actionable comments posted: 0")
	assert.Contains(t, prompt, "low") // Priority setting

	// Verify comment details are included
	assert.Contains(t, prompt, "123")     // Comment ID
	assert.Contains(t, prompt, "test.go") // File
	assert.Contains(t, prompt, "42")      // Line number
}

// TestConvertToStorageTasksWithNitpickPriority tests priority override for nitpick comments
func TestConvertToStorageTasksWithNitpickPriority(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "style:", "🧹"},
			LowPriorityStatus:   "pending",
		},
		AISettings: config.AISettings{
			ProcessNitpickComments: true,
			NitpickPriority:        "medium",
		},
	}
	analyzer := NewAnalyzer(cfg)

	tasks := []TaskRequest{
		{
			Description:     "Regular task",
			OriginText:      "This needs to be implemented",
			Priority:        "high",
			SourceReviewID:  1,
			SourceCommentID: 1,
			File:            "test.go",
			Line:            10,
			TaskIndex:       0,
		},
		{
			Description:     "Nitpick task",
			OriginText:      "nit: Consider improving variable names",
			Priority:        "low", // Original priority from AI
			SourceReviewID:  1,
			SourceCommentID: 2,
			File:            "test.go",
			Line:            20,
			TaskIndex:       0,
		},
		{
			Description:     "CodeRabbit structured nitpick",
			OriginText:      "🧹 Nitpick: Code style improvement",
			Priority:        "low", // Original priority from AI
			SourceReviewID:  1,
			SourceCommentID: 3,
			File:            "test.go",
			Line:            30,
			TaskIndex:       0,
		},
	}

	result := analyzer.convertToStorageTasks(tasks)

	// Verify regular task keeps original priority
	assert.Equal(t, "high", result[0].Priority)
	assert.Equal(t, "todo", result[0].Status) // Default status

	// Verify nitpick tasks get overridden priority
	assert.Equal(t, "medium", result[1].Priority) // Overridden to nitpick priority
	assert.Equal(t, "pending", result[1].Status)  // Low priority status

	assert.Equal(t, "medium", result[2].Priority) // Overridden to nitpick priority
	assert.Equal(t, "pending", result[2].Status)  // Low priority status

	// Verify all tasks have UUIDs
	for _, task := range result {
		assert.NotEmpty(t, task.ID)
		assert.Len(t, task.ID, 36) // UUID length
	}
}

// TestNitpickProcessingDisabled tests behavior when nitpick processing is disabled
func TestNitpickProcessingDisabled(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "style:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: config.AISettings{
			ProcessNitpickComments: false, // Disabled
			NitpickPriority:        "medium",
		},
	}
	analyzer := NewAnalyzer(cfg)

	tasks := []TaskRequest{
		{
			Description:     "Nitpick task",
			OriginText:      "nit: Consider improving variable names",
			Priority:        "low",
			SourceReviewID:  1,
			SourceCommentID: 1,
			File:            "test.go",
			Line:            10,
			TaskIndex:       0,
		},
	}

	result := analyzer.convertToStorageTasks(tasks)

	// When nitpick processing is disabled, priority should NOT be overridden
	assert.Equal(t, "low", result[0].Priority)   // Original priority preserved
	assert.Equal(t, "pending", result[0].Status) // Still gets low priority status
}
