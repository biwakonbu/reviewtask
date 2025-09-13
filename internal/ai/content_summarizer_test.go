package ai

import (
	"strings"
	"testing"

	"reviewtask/internal/github"
)

func TestNewContentSummarizer(t *testing.T) {
	tests := []struct {
		name            string
		maxSize         int
		verboseMode     bool
		expectedMaxSize int
	}{
		{
			name:            "default size when zero",
			maxSize:         0,
			verboseMode:     false,
			expectedMaxSize: 20000,
		},
		{
			name:            "default size when negative",
			maxSize:         -1,
			verboseMode:     true,
			expectedMaxSize: 20000,
		},
		{
			name:            "custom size",
			maxSize:         15000,
			verboseMode:     false,
			expectedMaxSize: 15000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewContentSummarizer(tt.maxSize, tt.verboseMode)

			if cs.maxSize != tt.expectedMaxSize {
				t.Errorf("Expected maxSize=%d, got %d", tt.expectedMaxSize, cs.maxSize)
			}

			if cs.verboseMode != tt.verboseMode {
				t.Errorf("Expected verboseMode=%v, got %v", tt.verboseMode, cs.verboseMode)
			}
		})
	}
}

func TestContentSummarizer_ShouldSummarize(t *testing.T) {
	cs := NewContentSummarizer(1000, false)

	tests := []struct {
		name           string
		commentBody    string
		shouldSummarize bool
	}{
		{
			name:           "small comment",
			commentBody:    "This is a short comment",
			shouldSummarize: false,
		},
		{
			name:           "large comment",
			commentBody:    strings.Repeat("This is a very long comment. ", 50), // ~1500 chars
			shouldSummarize: true,
		},
		{
			name:           "exactly at threshold",
			commentBody:    strings.Repeat("a", 1000),
			shouldSummarize: false,
		},
		{
			name:           "just over threshold",
			commentBody:    strings.Repeat("a", 1001),
			shouldSummarize: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := github.Comment{
				ID:   12345,
				Body: tt.commentBody,
			}

			result := cs.ShouldSummarize(comment)
			if result != tt.shouldSummarize {
				t.Errorf("Expected ShouldSummarize=%v, got %v for comment length %d",
					tt.shouldSummarize, result, len(tt.commentBody))
			}
		})
	}
}

func TestContentSummarizer_SummarizeComment(t *testing.T) {
	cs := NewContentSummarizer(500, false) // Small threshold for testing

	tests := []struct {
		name         string
		commentBody  string
		expectSummary bool
	}{
		{
			name:         "small comment not summarized",
			commentBody:  "Short comment that should not be summarized",
			expectSummary: false,
		},
		{
			name:         "large comment gets summarized",
			commentBody:  strings.Repeat("This is a long comment that needs summarization. ", 20),
			expectSummary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalComment := github.Comment{
				ID:     12345,
				Body:   tt.commentBody,
				Author: "testuser",
				File:   "test.go",
				Line:   10,
			}

			result := cs.SummarizeComment(originalComment)

			if tt.expectSummary {
				// Should be summarized - body should be different and shorter
				if result.Body == originalComment.Body {
					t.Error("Expected comment to be summarized but body is unchanged")
				}

				if len(result.Body) >= len(originalComment.Body) {
					t.Errorf("Expected summarized comment to be shorter, original=%d, summarized=%d",
						len(originalComment.Body), len(result.Body))
				}

				// Should contain summary indicator
				if !strings.Contains(result.Body, "[SUMMARIZED:") {
					t.Error("Expected summarized comment to contain summary indicator")
				}
			} else {
				// Should not be summarized - body should be unchanged
				if result.Body != originalComment.Body {
					t.Error("Expected small comment to remain unchanged")
				}
			}

			// Other fields should be preserved
			if result.ID != originalComment.ID {
				t.Errorf("Expected ID to be preserved, got %d, want %d", result.ID, originalComment.ID)
			}

			if result.Author != originalComment.Author {
				t.Errorf("Expected Author to be preserved, got %s, want %s", result.Author, originalComment.Author)
			}
		})
	}
}

func TestContentSummarizer_ExtractKeyInformation(t *testing.T) {
	cs := NewContentSummarizer(20000, false)

	tests := []struct {
		name            string
		content         string
		expectSections  []string
		unexpectSections []string
	}{
		{
			name: "security and error content",
			content: `This is a review comment.

SECURITY: There's a vulnerability in the authentication system.
BUG: The error handling is incorrect.

Some regular content here.

SUGGESTION: Consider using a different approach.
			`,
			expectSections: []string{"**Key Points:**", "**Suggestions:**"},
			unexpectSections: []string{"**Code Examples:**"},
		},
		{
			name: "code blocks",
			content: "Here's a code example:\n\n```go\nfunc main() {\n    fmt.Println(\"Hello\")\n    // More code here\n    for i := 0; i < 10; i++ {\n        doSomething(i)\n    }\n}\n```\n\nAnd some explanation.",
			expectSections: []string{"**Code Examples:**"},
			unexpectSections: []string{},
		},
		{
			name: "mixed content",
			content: "# Important Review\n\nThis is a critical issue that needs attention.\n\n```python\ndef vulnerable_function():\n    # This has a security problem\n    return user_input\n```\n\nFIXME: This needs to be fixed immediately.\nTODO: Add proper validation.",
			expectSections: []string{"**Key Points:**", "**Code Examples:**"},
			unexpectSections: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.content, "\n")
			result := cs.extractKeyInformation(lines)

			for _, section := range tt.expectSections {
				if !strings.Contains(result, section) {
					t.Errorf("Expected result to contain section '%s', but it didn't. Result: %s", section, result)
				}
			}

			for _, section := range tt.unexpectSections {
				if strings.Contains(result, section) {
					t.Errorf("Did not expect result to contain section '%s', but it did. Result: %s", section, result)
				}
			}
		})
	}
}

func TestContentSummarizer_IsHighPriorityLine(t *testing.T) {
	cs := NewContentSummarizer(20000, false)

	tests := []struct {
		name           string
		line           string
		isHighPriority bool
	}{
		{
			name:           "security keyword",
			line:           "this is a security issue",
			isHighPriority: true,
		},
		{
			name:           "critical keyword",
			line:           "critical bug found here",
			isHighPriority: true,
		},
		{
			name:           "todo marker",
			line:           "TODO: fix this later",
			isHighPriority: true,
		},
		{
			name:           "warning marker",
			line:           "WARNING: potential memory leak",
			isHighPriority: true,
		},
		{
			name:           "regular content",
			line:           "this is just regular content",
			isHighPriority: false,
		},
		{
			name:           "performance keyword",
			line:           "performance issues detected",
			isHighPriority: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cs.isHighPriorityLine(strings.ToLower(tt.line))
			if result != tt.isHighPriority {
				t.Errorf("Expected isHighPriorityLine('%s')=%v, got %v", tt.line, tt.isHighPriority, result)
			}
		})
	}
}

func TestContentSummarizer_IsIssueLine(t *testing.T) {
	cs := NewContentSummarizer(20000, false)

	tests := []struct {
		name    string
		line    string
		isIssue bool
	}{
		{
			name:    "issue marker",
			line:    "issue: the function doesn't work correctly",
			isIssue: true,
		},
		{
			name:    "problem statement",
			line:    "problem: missing error handling",
			isIssue: true,
		},
		{
			name:    "incorrect statement",
			line:    "this is incorrect implementation",
			isIssue: true,
		},
		{
			name:    "failure description",
			line:    "the test fails consistently",
			isIssue: true,
		},
		{
			name:    "regular content",
			line:    "this looks good to me",
			isIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cs.isIssueLine(strings.ToLower(tt.line))
			if result != tt.isIssue {
				t.Errorf("Expected isIssueLine('%s')=%v, got %v", tt.line, tt.isIssue, result)
			}
		})
	}
}

func TestContentSummarizer_IsSuggestionLine(t *testing.T) {
	cs := NewContentSummarizer(20000, false)

	tests := []struct {
		name         string
		line         string
		isSuggestion bool
	}{
		{
			name:         "suggest keyword",
			line:         "I suggest using a different approach",
			isSuggestion: true,
		},
		{
			name:         "recommend keyword",
			line:         "I recommend refactoring this function",
			isSuggestion: true,
		},
		{
			name:         "should keyword",
			line:         "You should add error handling here",
			isSuggestion: true,
		},
		{
			name:         "consider keyword",
			line:         "Consider using a more efficient algorithm",
			isSuggestion: true,
		},
		{
			name:         "improvement keyword",
			line:         "This is an improvement over the previous version",
			isSuggestion: true,
		},
		{
			name:         "regular content",
			line:         "this is working correctly",
			isSuggestion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cs.isSuggestionLine(strings.ToLower(tt.line))
			if result != tt.isSuggestion {
				t.Errorf("Expected isSuggestionLine('%s')=%v, got %v", tt.line, tt.isSuggestion, result)
			}
		})
	}
}

func TestContentSummarizer_IsImportantLine(t *testing.T) {
	cs := NewContentSummarizer(20000, false)

	tests := []struct {
		name        string
		line        string
		isImportant bool
	}{
		{
			name:        "important keyword",
			line:        "This is important to understand",
			isImportant: true,
		},
		{
			name:        "bullet point",
			line:        "* This is a bullet point",
			isImportant: true,
		},
		{
			name:        "numbered list",
			line:        "1. This is a numbered item",
			isImportant: true,
		},
		{
			name:        "question",
			line:        "How does this work?",
			isImportant: true,
		},
		{
			name:        "too short",
			line:        "ok",
			isImportant: false,
		},
		{
			name:        "too long",
			line:        strings.Repeat("This is a very long line that exceeds the maximum length threshold and should not be considered important due to its excessive verbosity. ", 10),
			isImportant: false,
		},
		{
			name:        "note keyword",
			line:        "Note that this behavior changed in version 2.0",
			isImportant: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cs.isImportantLine(strings.ToLower(tt.line))
			if result != tt.isImportant {
				t.Errorf("Expected isImportantLine('%s')=%v, got %v", tt.line, tt.isImportant, result)
			}
		})
	}
}

func TestContentSummarizer_SummarizeCodeBlock(t *testing.T) {
	cs := NewContentSummarizer(20000, false)

	tests := []struct {
		name         string
		lines        []string
		expectShort  bool
		expectSummary bool
	}{
		{
			name:         "short code block",
			lines:        []string{"func main() {", "  fmt.Println(\"Hello\")", "}"},
			expectShort:  true,
			expectSummary: false,
		},
		{
			name: "long code block",
			lines: []string{
				"func complexFunction() {",
				"  // Line 1",
				"  // Line 2",
				"  // Line 3",
				"  // Line 4",
				"  // Line 5",
				"  // Line 6",
				"}",
			},
			expectShort:  false,
			expectSummary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cs.summarizeCodeBlock(tt.lines)

			if tt.expectShort {
				// Should include all lines
				for _, line := range tt.lines {
					if !strings.Contains(result, line) {
						t.Errorf("Expected short code block to contain line '%s', but it didn't", line)
					}
				}
			}

			if tt.expectSummary {
				// Should contain summary indicator
				if !strings.Contains(result, "Code block with") {
					t.Error("Expected long code block to contain summary indicator")
				}

				if !strings.Contains(result, "showing first 3") {
					t.Error("Expected long code block to show truncation message")
				}
			}
		})
	}
}

func TestContentSummarizer_AggressiveSummarization(t *testing.T) {
	cs := NewContentSummarizer(200, false) // Very small threshold

	// Create content that will need aggressive summarization
	content := strings.Repeat("This is some content that should be reduced significantly. ", 50)

	result := cs.applyAggressiveSummarization(content)

	// Should be significantly smaller
	if len(result) >= len(content) {
		t.Errorf("Expected aggressive summarization to reduce size, original=%d, result=%d", len(content), len(result))
	}

	// Should respect the size limit
	if len(result) > cs.maxSize {
		t.Errorf("Expected result to be within size limit %d, got %d", cs.maxSize, len(result))
	}

	// Test with content that has no critical lines
	nonCriticalContent := strings.Repeat("Regular content without keywords. ", 100)
	result2 := cs.applyAggressiveSummarization(nonCriticalContent)

	// Should still produce some result
	if len(result2) == 0 {
		t.Error("Expected aggressive summarization to produce some result even for non-critical content")
	}

	// Should contain truncation indicator if size was exceeded
	if len(nonCriticalContent) > cs.maxSize && !strings.Contains(result2, "[TRUNCATED...]") {
		t.Error("Expected truncation indicator for content exceeding size limit")
	}
}

func TestContentSummarizer_CreateSummary(t *testing.T) {
	cs := NewContentSummarizer(500, false)

	content := "# Security Review\n\nSECURITY: There's a potential XSS vulnerability in the user input handling.\n\n```javascript\nfunction processInput(input) {\n    document.innerHTML = input; // Dangerous!\n}\n```\n\nSUGGESTION: Use proper input sanitization.\nBUG: Missing error handling in the authentication flow.\n\nSome additional regular content that provides context.\nTODO: Add comprehensive unit tests."

	result := cs.createSummary(content)

	// Should contain summary header
	if !strings.Contains(result, "[SUMMARIZED:") {
		t.Error("Expected summary to contain summarization header")
	}

	// Should preserve high-priority content
	if !strings.Contains(result, "SECURITY") {
		t.Error("Expected summary to preserve security content")
	}

	// Should be shorter than original
	if len(result) >= len(content) {
		t.Errorf("Expected summary to be shorter than original, original=%d, summary=%d", len(content), len(result))
	}

	// Should contain size information
	originalSize := len(content)
	summarySize := len(result)
	if !strings.Contains(result, "Original") || !strings.Contains(result, "Summary") {
		t.Error("Expected summary header to contain size information")
	}

	t.Logf("Original size: %d, Summary size: %d, Reduction: %.1f%%",
		originalSize, summarySize, 100.0*(1.0-float64(summarySize)/float64(originalSize)))
}