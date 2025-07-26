package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

func TestGenerateTasksWithReviewBody(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "low",
			DefaultStatus:       "todo",
		},
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{false}[0], // Disable validation for simpler testing
		},
	}

	// Create mock client
	mockClient := NewMockClaudeClient()
	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Test with PR #122 CodeRabbit review format (review body only, no inline comments)
	reviews := []github.Review{
		{
			ID:       3055255377,
			Reviewer: "coderabbitai[bot]",
			State:    "COMMENTED",
			Body: `**Actionable comments posted: 0**

<details>
<summary>🧹 Nitpick comments (1)</summary><blockquote>

<details>
<summary>internal/ai/analyzer.go (1)</summary><blockquote>

` + "`618-657`: **Consider improving HTML parsing robustness.**" + `

While the method correctly identifies structured nitpick content for CodeRabbit comments, the HTML parsing logic could be more robust:

1. The ` + "`+20`" + ` buffer in ` + "`summaryContent := lowerBody[summaryStart : summaryStart+summaryEnd+20]`" + ` is arbitrary and could cause index out of bounds errors
2. The fallback logic for missing ` + "`</summary>`" + ` tags is fragile

Consider using a more robust approach:

` + "```diff" + `
-		summaryContent := lowerBody[summaryStart : summaryStart+summaryEnd+20] // +20 for buffer
+		endPos := summaryStart + summaryEnd
+		if summaryEnd != -1 {
+			endPos = summaryStart + summaryEnd + 10 // smaller, safer buffer
+		}
+		if endPos > len(lowerBody) {
+			endPos = len(lowerBody)
+		}
+		summaryContent := lowerBody[summaryStart:endPos]
` + "```" + `

However, the current implementation works correctly for the intended CodeRabbit comment formats and provides the required functionality.

</blockquote></details>

</blockquote></details>`,
			SubmittedAt: "2025-07-25T12:42:14Z",
			Comments:    []github.Comment{}, // No inline comments, only review body
		},
	}

	// Generate tasks
	tasks, err := analyzer.GenerateTasks(reviews)

	// Verify results
	assert.NoError(t, err, "GenerateTasks should not return an error")
	assert.Len(t, tasks, 1, "Should generate exactly 1 task from the review body")

	if len(tasks) > 0 {
		task := tasks[0]
		assert.NotEmpty(t, task.Description, "Task description should not be empty")
		assert.Equal(t, "todo", task.Status, "Task status should be 'todo'")
		assert.Equal(t, int64(3055255377), task.SourceReviewID, "Task should reference the correct review ID")
		assert.Equal(t, int64(3055255377), task.SourceCommentID, "Task should reference the review ID as comment ID (since it's from review body)")
		// Note: Mock client may extract some file info from prompt, but the key point is that we process review body
		// The important assertion is that a task was generated from the review body
		assert.NotEmpty(t, task.OriginText, "Task should have origin text from review body")
		t.Logf("Generated task: %s (Priority: %s, Status: %s)", task.Description, task.Priority, task.Status)
	}
}

func TestGenerateTasksWithBothReviewBodyAndInlineComments(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:"},
			LowPriorityStatus:   "low",
			DefaultStatus:       "todo",
		},
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{false}[0],
		},
	}

	mockClient := NewMockClaudeClient()
	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Test with both review body and inline comments
	reviews := []github.Review{
		{
			ID:          12345,
			Reviewer:    "reviewer1",
			State:       "COMMENTED",
			Body:        "nit: Overall the code looks good but has some minor issues.",
			SubmittedAt: "2025-07-25T12:00:00Z",
			Comments: []github.Comment{
				{
					ID:        67890,
					File:      "test.go",
					Line:      42,
					Body:      "This function needs error handling.",
					Author:    "reviewer1",
					CreatedAt: "2025-07-25T12:00:00Z",
				},
			},
		},
	}

	// Generate tasks
	tasks, err := analyzer.GenerateTasks(reviews)

	// Verify results - should generate 2 tasks (1 from review body + 1 from inline comment)
	assert.NoError(t, err, "GenerateTasks should not return an error")
	assert.Len(t, tasks, 2, "Should generate 2 tasks (1 from review body + 1 from inline comment)")

	// Check that we have both review body and inline comment tasks
	var reviewBodyTask, inlineCommentTask *storage.Task
	for i := range tasks {
		if tasks[i].SourceCommentID == 12345 {
			reviewBodyTask = &tasks[i]
		} else if tasks[i].SourceCommentID == 67890 {
			inlineCommentTask = &tasks[i]
		}
	}

	assert.NotNil(t, reviewBodyTask, "Should have a task from review body")
	assert.NotNil(t, inlineCommentTask, "Should have a task from inline comment")

	// Note: Mock client behavior may vary, but the key point is that we process both review body and inline comments
	// The important assertion is that we have tasks from both sources
	if reviewBodyTask != nil {
		t.Logf("Review body task: %s", reviewBodyTask.Description)
	}

	if inlineCommentTask != nil {
		t.Logf("Inline comment task: %s", inlineCommentTask.Description)
	}
}

func TestGenerateTasksWithEmptyReviewBody(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{false}[0],
		},
	}

	mockClient := NewMockClaudeClient()
	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Test with empty review body
	reviews := []github.Review{
		{
			ID:          12345,
			Reviewer:    "reviewer1",
			State:       "APPROVED",
			Body:        "", // Empty review body
			SubmittedAt: "2025-07-25T12:00:00Z",
			Comments: []github.Comment{
				{
					ID:        67890,
					File:      "test.go",
					Line:      42,
					Body:      "Good work!",
					Author:    "reviewer1",
					CreatedAt: "2025-07-25T12:00:00Z",
				},
			},
		},
	}

	// Generate tasks
	tasks, err := analyzer.GenerateTasks(reviews)

	// Verify results - should generate only 1 task from inline comment
	assert.NoError(t, err, "GenerateTasks should not return an error")
	assert.Len(t, tasks, 1, "Should generate only 1 task from inline comment (empty review body should be ignored)")

	if len(tasks) > 0 {
		task := tasks[0]
		assert.Equal(t, int64(67890), task.SourceCommentID, "Task should be from inline comment, not empty review body")
		t.Logf("Generated task from inline comment: %s", task.Description)
	}
}
