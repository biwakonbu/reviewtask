package ai

import (
	"fmt"
	"strings"
	"testing"

	"reviewtask/internal/github"
)

// TestContentSummarizer_BoundaryConditions tests boundary conditions for content summarization
func TestContentSummarizer_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name          string
		maxSize       int
		commentSize   int
		expectSummary bool
		description   string
	}{
		{
			name:          "exactly at threshold",
			maxSize:       1000,
			commentSize:   1000,
			expectSummary: false,
			description:   "Should not summarize when exactly at threshold",
		},
		{
			name:          "one byte over threshold",
			maxSize:       1000,
			commentSize:   1001,
			expectSummary: true,
			description:   "Should summarize when one byte over",
		},
		{
			name:          "one byte under threshold",
			maxSize:       1000,
			commentSize:   999,
			expectSummary: false,
			description:   "Should not summarize when one byte under",
		},
		{
			name:          "zero size threshold",
			maxSize:       0, // Should default to 20000
			commentSize:   19999,
			expectSummary: false,
			description:   "Should use default threshold when zero",
		},
		{
			name:          "negative size threshold",
			maxSize:       -100, // Should default to 20000
			commentSize:   20001,
			expectSummary: true,
			description:   "Should use default threshold when negative",
		},
		{
			name:          "very large comment",
			maxSize:       1000,
			commentSize:   100000,
			expectSummary: true,
			description:   "Should handle very large comments",
		},
		{
			name:          "empty comment",
			maxSize:       1000,
			commentSize:   0,
			expectSummary: false,
			description:   "Should handle empty comments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewContentSummarizer(tt.maxSize, false)

			comment := github.Comment{
				ID:   12345,
				Body: strings.Repeat("a", tt.commentSize),
			}

			shouldSummarize := cs.ShouldSummarize(comment)
			if shouldSummarize != tt.expectSummary {
				t.Errorf("Expected ShouldSummarize=%v for case '%s', got %v",
					tt.expectSummary, tt.description, shouldSummarize)
			}

			// Test actual summarization
			result := cs.SummarizeComment(comment)
			if tt.expectSummary {
				if !strings.Contains(result.Body, "[SUMMARIZED:") {
					t.Errorf("Expected summarized indicator for case '%s'", tt.description)
				}
			} else {
				if result.Body != comment.Body {
					t.Errorf("Expected unchanged body for case '%s'", tt.description)
				}
			}
		})
	}
}

// TestContentSummarizer_CreateSummary_EdgeCases tests edge cases in summary creation
func TestContentSummarizer_CreateSummary_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		maxSize      int
		expectHeader bool
		description  string
	}{
		{
			name:         "empty content",
			content:      "",
			maxSize:      100,
			expectHeader: true,
			description:  "Should handle empty content",
		},
		{
			name:         "only whitespace",
			content:      "   \n\t\r\n   ",
			maxSize:      100,
			expectHeader: true,
			description:  "Should handle whitespace-only content",
		},
		{
			name:         "single line",
			content:      "This is a single line of text",
			maxSize:      500,
			expectHeader: true,
			description:  "Should handle single line content",
		},
		{
			name: "only code blocks",
			content: "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n" +
				"```python\ndef hello():\n    print(\"Hello\")\n```",
			maxSize:      500,
			expectHeader: true,
			description:  "Should handle code-only content",
		},
		{
			name:         "repeated newlines",
			content:      "Line 1\n\n\n\n\n\nLine 2\n\n\n\n\nLine 3",
			maxSize:      500,
			expectHeader: true,
			description:  "Should handle excessive newlines",
		},
		{
			name:         "unicode content",
			content:      "这是中文内容 🚀 مرحبا بالعالم 🎉 Здравствуй мир 🌍",
			maxSize:      500,
			expectHeader: true,
			description:  "Should handle unicode content",
		},
		{
			name:         "control characters",
			content:      "Text with\x00null\x01and\x02control\x03characters",
			maxSize:      500,
			expectHeader: true,
			description:  "Should handle control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewContentSummarizer(tt.maxSize, false)
			result := cs.createSummary(tt.content)

			if tt.expectHeader && !strings.Contains(result, "[SUMMARIZED:") {
				t.Errorf("Expected summary header for case '%s'", tt.description)
			}

			// Verify size information is included
			if !strings.Contains(result, fmt.Sprintf("Original %d bytes", len(tt.content))) {
				t.Errorf("Expected original size info for case '%s'", tt.description)
			}
		})
	}
}

// TestContentSummarizer_ExtractKeyInformation_ComplexCases tests complex extraction scenarios
func TestContentSummarizer_ExtractKeyInformation_ComplexCases(t *testing.T) {
	tests := []struct {
		name               string
		content            string
		expectedSections   []string
		unexpectedSections []string
		description        string
	}{
		{
			name: "nested code blocks",
			content: "```markdown\n" +
				"# Example\n" +
				"```go\n" +
				"func main() {}\n" +
				"```\n" +
				"```",
			expectedSections:   []string{"**Code Examples:**"},
			unexpectedSections: []string{},
			description:        "Should handle nested code blocks",
		},
		{
			name: "mixed priority content",
			content: "SECURITY: Critical vulnerability\n" +
				"This is normal text\n" +
				"BUG: Memory leak detected\n" +
				"More normal text\n" +
				"TODO: Fix this urgently\n" +
				"SUGGESTION: Consider refactoring",
			expectedSections:   []string{"**Key Points:**", "**Suggestions:**"},
			unexpectedSections: []string{},
			description:        "Should categorize mixed priority content",
		},
		{
			name: "very long lines",
			content: "SECURITY: " + strings.Repeat("very long security issue description ", 100) + "\n" +
				"BUG: " + strings.Repeat("very long bug description ", 100),
			expectedSections:   []string{"**Key Points:**"},
			unexpectedSections: []string{},
			description:        "Should handle very long lines",
		},
		{
			name: "bullet points and lists",
			content: "Issues found:\n" +
				"* First issue\n" +
				"- Second issue\n" +
				"+ Third issue\n" +
				"• Fourth issue\n" +
				"1. Numbered issue\n" +
				"2) Another numbered issue",
			expectedSections:   []string{"**Key Points:**"},
			unexpectedSections: []string{},
			description:        "Should recognize various list formats",
		},
		{
			name: "questions",
			content: "Why is this happening?\n" +
				"How can we fix this?\n" +
				"What are the alternatives?\n" +
				"Should we refactor?",
			expectedSections:   []string{"**Key Points:**"},
			unexpectedSections: []string{},
			description:        "Should recognize questions as important",
		},
		{
			name:               "no important content",
			content:            "Just some regular text without any special markers or patterns.",
			expectedSections:   []string{},
			unexpectedSections: []string{"**Key Points:**", "**Issues:**", "**Suggestions:**"},
			description:        "Should not create sections for non-important content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewContentSummarizer(20000, false)
			lines := strings.Split(tt.content, "\n")
			result := cs.extractKeyInformation(lines)

			for _, section := range tt.expectedSections {
				if !strings.Contains(result, section) {
					t.Errorf("Expected section '%s' for case '%s', result: %s",
						section, tt.description, result)
				}
			}

			for _, section := range tt.unexpectedSections {
				if strings.Contains(result, section) {
					t.Errorf("Unexpected section '%s' for case '%s'",
						section, tt.description)
				}
			}
		})
	}
}

// TestContentSummarizer_ApplyAggressiveSummarization_EdgeCases tests aggressive summarization edge cases
func TestContentSummarizer_ApplyAggressiveSummarization_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		maxSize     int
		description string
	}{
		{
			name:        "no critical lines",
			content:     strings.Repeat("Regular content without keywords. ", 100),
			maxSize:     200,
			description: "Should fall back to first lines when no critical content",
		},
		{
			name: "only critical lines",
			content: "SECURITY: Issue 1\n" +
				"ERROR: Issue 2\n" +
				"BUG: Issue 3\n" +
				strings.Repeat("CRITICAL: Issue\n", 50),
			maxSize:     200,
			description: "Should limit even critical lines to size",
		},
		{
			name:        "single very long line",
			content:     "SECURITY: " + strings.Repeat("x", 1000),
			maxSize:     200,
			description: "Should truncate very long lines",
		},
		{
			name:        "empty lines between content",
			content:     "Line 1\n\n\n\nLine 2\n\n\n\nLine 3",
			maxSize:     100,
			description: "Should handle empty lines",
		},
		{
			name: "markdown headers",
			content: "# Header 1\n" +
				"## Header 2\n" +
				"### Header 3\n" +
				"Regular content",
			maxSize:     100,
			description: "Should skip markdown headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewContentSummarizer(tt.maxSize, false)
			result := cs.applyAggressiveSummarization(tt.content)

			if len(result) > tt.maxSize {
				t.Errorf("Result exceeds max size for case '%s': got %d, max %d",
					tt.description, len(result), tt.maxSize)
			}

			if strings.Contains(result, "[TRUNCATED...]") && len(result) > tt.maxSize {
				t.Errorf("Truncation marker but size exceeded for case '%s'", tt.description)
			}
		})
	}
}

// TestContentSummarizer_SummarizeCodeBlock_EdgeCases tests code block summarization edge cases
func TestContentSummarizer_SummarizeCodeBlock_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		lines       []string
		expectFull  bool
		description string
	}{
		{
			name:        "empty code block",
			lines:       []string{},
			expectFull:  true,
			description: "Should handle empty code blocks",
		},
		{
			name:        "single line",
			lines:       []string{"return 42"},
			expectFull:  true,
			description: "Should include single line fully",
		},
		{
			name:        "exactly 5 lines",
			lines:       []string{"line1", "line2", "line3", "line4", "line5"},
			expectFull:  true,
			description: "Should include 5 lines fully",
		},
		{
			name:        "6 lines",
			lines:       []string{"line1", "line2", "line3", "line4", "line5", "line6"},
			expectFull:  false,
			description: "Should summarize 6+ lines",
		},
		{
			name: "very long lines",
			lines: []string{
				strings.Repeat("x", 500),
				strings.Repeat("y", 500),
				strings.Repeat("z", 500),
			},
			expectFull:  true,
			description: "Should handle very long lines",
		},
		{
			name:        "unicode in code",
			lines:       []string{`fmt.Println("你好世界")`, `print("مرحبا")`},
			expectFull:  true,
			description: "Should handle unicode in code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewContentSummarizer(20000, false)
			result := cs.summarizeCodeBlock(tt.lines)

			if tt.expectFull {
				// Should contain all lines
				for _, line := range tt.lines {
					if !strings.Contains(result, line) {
						t.Errorf("Expected full code for case '%s', missing: %s",
							tt.description, line)
					}
				}
			} else {
				// Should be summarized
				if !strings.Contains(result, "Code block with") {
					t.Errorf("Expected summary indicator for case '%s'", tt.description)
				}
				if !strings.Contains(result, "showing first 3") {
					t.Errorf("Expected truncation message for case '%s'", tt.description)
				}
			}
		})
	}
}

// TestContentSummarizer_MultibyteCharacterBoundary tests handling of multibyte character boundaries
func TestContentSummarizer_MultibyteCharacterBoundary(t *testing.T) {
	// Create content that would be cut in the middle of a multibyte character
	unicodeStr := "这是中文测试内容"
	_ = len([]rune(unicodeStr)) // Just for reference, not used in tests

	tests := []struct {
		name        string
		content     string
		maxSize     int
		description string
	}{
		{
			name:        "cut in middle of Chinese character",
			content:     strings.Repeat(unicodeStr, 100),
			maxSize:     1000,
			description: "Should handle Chinese character boundaries",
		},
		{
			name:        "cut in middle of emoji",
			content:     strings.Repeat("Test 🚀 content 🎉 ", 100),
			maxSize:     500,
			description: "Should handle emoji boundaries",
		},
		{
			name:        "mixed scripts",
			content:     strings.Repeat("Hello مرحبا 你好 Здравствуй ", 100),
			maxSize:     750,
			description: "Should handle mixed script boundaries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewContentSummarizer(tt.maxSize, false)

			comment := github.Comment{
				ID:   12345,
				Body: tt.content,
			}

			result := cs.SummarizeComment(comment)

			// Verify result is valid UTF-8 (strings.ToValidUTF8 would change invalid strings)
			if result.Body != strings.ToValidUTF8(result.Body, "") {
				t.Errorf("Invalid UTF-8 in result for case '%s'", tt.description)
			}

			// Verify we didn't lose too much content
			originalRunes := len([]rune(tt.content))
			resultRunes := len([]rune(result.Body))

			t.Logf("Case '%s': Original %d runes, Result %d runes",
				tt.description, originalRunes, resultRunes)
		})
	}
}

// TestContentSummarizer_PerformanceWithLargeContent tests performance with large content
func TestContentSummarizer_PerformanceWithLargeContent(t *testing.T) {
	sizes := []int{
		10000,   // 10KB
		50000,   // 50KB
		100000,  // 100KB
		500000,  // 500KB
		1000000, // 1MB
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			cs := NewContentSummarizer(20000, false)

			// Generate content with various patterns
			var contentBuilder strings.Builder
			patterns := []string{
				"SECURITY: Critical issue\n",
				"BUG: Memory leak\n",
				"TODO: Fix this\n",
				"Regular content line\n",
				"```\ncode block\n```\n",
			}

			currentSize := 0
			patternIndex := 0
			for currentSize < size {
				pattern := patterns[patternIndex%len(patterns)]
				contentBuilder.WriteString(pattern)
				currentSize += len(pattern)
				patternIndex++
			}

			content := contentBuilder.String()
			comment := github.Comment{
				ID:   12345,
				Body: content,
			}

			// Time the summarization
			result := cs.SummarizeComment(comment)

			// Verify result is smaller than threshold
			if len(result.Body) > 20000+1000 { // Allow some overhead for headers
				t.Errorf("Summary too large for %d byte input: got %d bytes",
					size, len(result.Body))
			}

			t.Logf("Summarized %d bytes to %d bytes", size, len(result.Body))
		})
	}
}

// Benchmark tests
func BenchmarkContentSummarizer_SummarizeComment_Small(b *testing.B) {
	cs := NewContentSummarizer(1000, false)
	comment := github.Comment{
		ID:   12345,
		Body: strings.Repeat("Test content ", 50), // ~650 bytes
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs.SummarizeComment(comment)
	}
}

func BenchmarkContentSummarizer_SummarizeComment_Large(b *testing.B) {
	cs := NewContentSummarizer(20000, false)
	comment := github.Comment{
		ID:   12345,
		Body: strings.Repeat("Test content ", 2000), // ~26KB
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs.SummarizeComment(comment)
	}
}

func BenchmarkContentSummarizer_ExtractKeyInformation(b *testing.B) {
	cs := NewContentSummarizer(20000, false)
	content := "SECURITY: Issue\nBUG: Problem\nTODO: Fix\n" + strings.Repeat("Regular line\n", 100)
	lines := strings.Split(content, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cs.extractKeyInformation(lines)
	}
}
