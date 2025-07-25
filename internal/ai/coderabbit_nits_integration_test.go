package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/config"
)

// TestCodeRabbitNitsDetectionIntegration tests CodeRabbit nits detection logic directly
func TestCodeRabbitNitsDetectionIntegration(t *testing.T) {
	// Create analyzer with default config
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "low",
			DefaultStatus:       "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name        string
		reviewBody  string
		expected    bool
		description string
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
			expected:    true,
			description: "Real CodeRabbit review with structured nitpick comments",
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
			expected:    true,
			description: "CodeRabbit review with nitpick section should be low priority",
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
			expected:    false,
			description: "CodeRabbit review without nitpicks should not be detected",
		},
		{
			name: "Traditional nit comment format",
			reviewBody: `nit: Consider improving variable naming here.

This is a minor style improvement that would enhance code readability.`,
			expected:    true,
			description: "Traditional nit comment should still work",
		},
		{
			name: "Mixed format - both structured and traditional",
			reviewBody: `<details>
<summary>🧹 Nitpick comments (1)</summary>
<blockquote>

nit: Fix indentation in this function.

</blockquote>
</details>`,
			expected:    true,
			description: "Review with both structured and traditional patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the low priority detection logic directly
			result := analyzer.isLowPriorityComment(tt.reviewBody)
			assert.Equal(t, tt.expected, result,
				"Expected %v for %s, got %v", tt.expected, tt.description, result)

			t.Logf("%s: Detection result = %v", tt.description, result)
		})
	}
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
	analyzer := NewAnalyzer(cfg)

	traditionalComments := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "Traditional nit comment",
			body:     "nit: Fix this minor issue",
			expected: true,
		},
		{
			name:     "Traditional minor comment",
			body:     "minor: Could be improved",
			expected: true,
		},
		{
			name:     "Traditional style comment",
			body:     "style: Inconsistent formatting",
			expected: true,
		},
		{
			name:     "Regular comment",
			body:     "This is important functionality feedback",
			expected: false,
		},
		{
			name: "CodeRabbit structured nitpick with traditional patterns",
			body: `<details>
<summary>🧹 Nitpick comments (1)</summary>
<blockquote>
Some structured nitpick content
</blockquote>
</details>`,
			expected: true,
		},
	}

	for _, tc := range traditionalComments {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.isLowPriorityComment(tc.body)
			assert.Equal(t, tc.expected, result,
				"Traditional pattern '%s' should result in %v", tc.body, tc.expected)
		})
	}
}

// TestActualPR120CodeRabbitReview tests with the exact review content from PR #120
func TestActualPR120CodeRabbitReview(t *testing.T) {
	// Create analyzer with realistic config
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "low",
			DefaultStatus:       "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Exact content from PR #120 CodeRabbit review
	actualReviewBody := `**Actionable comments posted: 0**

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

This test file effectively validates the core fix for Issue #112 and provides strong regression protection. The tests cover:

- Single and multiple task scenarios
- Priority ordering validation
- Status filtering (doing/todo vs done)
- Command compatibility between status and show
- Format validation and regression testing
- Empty state edge cases

The table-driven tests and comprehensive assertions align well with Go testing best practices and the PR objectives.



For consistency with the Task struct's RFC 4122 UUID requirement, consider creating a test helper for generating realistic UUIDs:

` + "```go" + `
func generateTestUUID(suffix string) string {
    return fmt.Sprintf("550e8400-e29b-41d4-a716-44665544%04s", suffix)
}
` + "```" + `

</blockquote></details>

</blockquote></details>`

	// Test that this exact review is detected as low priority
	result := analyzer.isLowPriorityComment(actualReviewBody)
	assert.True(t, result, "PR #120 CodeRabbit review should be detected as low priority")

	t.Logf("PR #120 CodeRabbit review detection result: %v", result)
}
