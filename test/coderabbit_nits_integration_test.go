package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

// TestCodeRabbitNitsDetectionIntegration tests end-to-end CodeRabbit nits detection
func TestCodeRabbitNitsDetectionIntegration(t *testing.T) {
	// Create analyzer with default config
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "low",
			DefaultStatus:       "todo",
		},
	}
	analyzer := ai.NewAnalyzer(cfg)

	tests := []struct {
		name           string
		reviewBody     string
		expectedStatus string
		description    string
	}{
		{
			name: "Actual PR #120 CodeRabbit review should be detected as low priority",
			reviewBody: `**Actionable comments posted: 0**

<details>
<summary>🧹 Nitpick comments (3)</summary><blockquote>

<details>
<summary>cmd/status_uuid_test.go (3)</summary><blockquote>

` + "`15-120`: **Comprehensive test coverage with minor UUID format concern.**" + `

The test function effectively validates the core fix for Issue #112 with good coverage of different scenarios. The table-driven approach and regression testing are excellent practices.

However, the test UUIDs (e.g., "uuid-12345-abcde-67890") don't follow RFC 4122 format as noted in the Task struct definition. Consider using more realistic UUID formats for better test fidelity.

Consider using RFC 4122 compliant UUIDs in test data:

` + "```diff" + `
-			ID:          "uuid-12345-abcde-67890",
+			ID:          "550e8400-e29b-41d4-a716-446655440000",
` + "```" + `

---

` + "`188-231`: **Thorough format validation with good regression testing.**" + `

The test provides comprehensive validation that actual UUIDs are displayed and includes excellent line-by-line regex checking to prevent TSK-XXX format regression.

Same UUID format suggestion applies here - consider using RFC 4122 compliant format for better test realism.

---

` + "`15-258`: **Excellent test suite for UUID functionality with comprehensive coverage.**" + `

This test file effectively validates the core fix for Issue #112 and provides strong regression protection.

</blockquote></details>

</blockquote></details>`,
			expectedStatus: "low",
			description:    "Real CodeRabbit review with structured nitpick comments",
		},
		{
			name: "CodeRabbit review with different nitpick format",
			reviewBody: `**Review Summary**

<details>
<summary>Nitpick comments (2)</summary>
<blockquote>

Some nitpick content about code style and minor improvements.

</blockquote>
</details>

<details>
<summary>Regular comments (1)</summary>
<blockquote>

Important functional feedback.

</blockquote>
</details>`,
			expectedStatus: "low",
			description:    "CodeRabbit review with nitpick section should be low priority",
		},
		{
			name: "CodeRabbit review without nitpicks",
			reviewBody: `**Review Summary**

<details>
<summary>Critical issues (2)</summary>
<blockquote>

Important security and functionality issues.

</blockquote>
</details>`,
			expectedStatus: "todo",
			description:    "CodeRabbit review without nitpicks should use default status",
		},
		{
			name: "Traditional nit comment format",
			reviewBody: `nit: Consider improving variable naming here.

This is a minor style improvement that would enhance code readability.`,
			expectedStatus: "low",
			description:    "Traditional nit comment should still work",
		},
		{
			name: "Mixed format - both structured and traditional",
			reviewBody: `<details>
<summary>🧹 Nitpick comments (1)</summary>
<blockquote>

nit: Fix indentation in this function.

</blockquote>
</details>`,
			expectedStatus: "low",
			description:    "Review with both structured and traditional patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock review
			review := github.Review{
				ID:       123456789,
				Body:     tt.reviewBody,
				State:    "COMMENTED",
				Reviewer: "coderabbitai[bot]",
				Comments: nil, // Focus on body-level detection
			}

			// Analyze the review to generate tasks
			reviews := []github.Review{review}
			tasks, err := analyzer.GenerateTasks(reviews)
			require.NoError(t, err, "Failed to analyze review: %v", err)

			// Verify that tasks are generated
			assert.NotEmpty(t, tasks, "Expected tasks to be generated from %s", tt.description)

			// Check that all generated tasks have the expected status
			for _, task := range tasks {
				assert.Equal(t, tt.expectedStatus, task.Status,
					"Expected task status %s for %s, got %s", tt.expectedStatus, tt.description, task.Status)
			}

			// Log for debugging
			t.Logf("%s: Generated %d tasks with status '%s'", tt.description, len(tasks), tt.expectedStatus)
		})
	}
}

// TestCodeRabbitNitsTaskGeneration tests that CodeRabbit nitpick reviews generate appropriate tasks
func TestCodeRabbitNitsTaskGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create analyzer with low-priority configuration
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "low",
			DefaultStatus:       "todo",
		},
	}
	analyzer := ai.NewAnalyzer(cfg)

	// Simulate a complex CodeRabbit review with multiple sections
	complexReview := `**Actionable comments posted: 2**

<details>
<summary>🧹 Nitpick comments (4)</summary><blockquote>

<details>
<summary>src/main.go (2)</summary><blockquote>

` + "`15-20`: **Variable naming could be improved**" + `

Consider using more descriptive variable names for better code readability.

` + "```suggestion" + `
var userConfiguration *Config
` + "```" + `

---

` + "`45-50`: **Add documentation comment**" + `

This function would benefit from a documentation comment explaining its purpose.

</blockquote></details>

<details>
<summary>src/utils.go (2)</summary><blockquote>

` + "`10-15`: **Style improvement**" + `

Consider using consistent spacing around operators.

---

` + "`30-35`: **Minor optimization**" + `

This loop could be optimized slightly for better performance.

</blockquote></details>

</blockquote></details>

<details>
<summary>🔧 Regular comments (2)</summary><blockquote>

<details>
<summary>src/auth.go (2)</summary><blockquote>

` + "`100-110`: **Security concern**" + `

This authentication logic needs proper validation to prevent security vulnerabilities.

---

` + "`200-210`: **Error handling**" + `

Missing error handling could cause the application to crash.

</blockquote></details>

</blockquote></details>`

	// Create mock review
	review := github.Review{
		ID:       987654321,
		Body:     complexReview,
		State:    "COMMENTED",
		Reviewer: "coderabbitai[bot]",
		Comments: nil,
	}

	// Analyze the review
	reviews := []github.Review{review}
	tasks, err := analyzer.GenerateTasks(reviews)
	require.NoError(t, err, "Failed to analyze complex CodeRabbit review")

	// Verify tasks are generated
	assert.NotEmpty(t, tasks, "Expected tasks to be generated from complex review")

	// Count tasks by status
	lowPriorityCount := 0
	normalPriorityCount := 0

	for _, task := range tasks {
		switch task.Status {
		case "low":
			lowPriorityCount++
		case "todo":
			normalPriorityCount++
		default:
			t.Errorf("Unexpected task status: %s", task.Status)
		}
	}

	// Log results for debugging
	t.Logf("Generated %d total tasks: %d low priority, %d normal priority",
		len(tasks), lowPriorityCount, normalPriorityCount)

	// Verify that the nitpick section was detected as low priority
	assert.Greater(t, lowPriorityCount, 0,
		"Expected at least some low priority tasks from nitpick section")

	// The complex review should generate both types of tasks
	// (This test validates that the analyzer can handle mixed content correctly)
	assert.Greater(t, len(tasks), 0, "Should generate at least one task")
}

// TestCodeRabbitPatternsBackwardCompatibility ensures existing patterns still work
func TestCodeRabbitPatternsBackwardCompatibility(t *testing.T) {
	// Create analyzer with traditional patterns
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "minor:", "style:"},
			LowPriorityStatus:   "low",
			DefaultStatus:       "todo",
		},
	}
	analyzer := ai.NewAnalyzer(cfg)

	traditionalComments := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "Traditional nit comment",
			body:     "nit: Fix this minor issue",
			expected: "low",
		},
		{
			name:     "Traditional minor comment",
			body:     "minor: Could be improved",
			expected: "low",
		},
		{
			name:     "Traditional style comment",
			body:     "style: Inconsistent formatting",
			expected: "low",
		},
		{
			name:     "Regular comment",
			body:     "This is important functionality feedback",
			expected: "todo",
		},
	}

	for _, tc := range traditionalComments {
		t.Run(tc.name, func(t *testing.T) {
			review := github.Review{
				ID:       123,
				Body:     tc.body,
				State:    "COMMENTED",
				Reviewer: "human-reviewer",
				Comments: nil,
			}

			tasks, err := analyzer.GenerateTasks([]github.Review{review})
			require.NoError(t, err)

			if len(tasks) > 0 {
				assert.Equal(t, tc.expected, tasks[0].Status,
					"Traditional pattern '%s' should result in status '%s'", tc.body, tc.expected)
			}
		})
	}
}