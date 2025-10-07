package github

import (
	"strings"
	"testing"
)

func TestParseEmbeddedComments(t *testing.T) {
	tests := []struct {
		name           string
		reviewBody     string
		expectedCount  int
		expectedFirst  *EmbeddedComment
		expectedSecond *EmbeddedComment
	}{
		{
			name: "Single P1 comment with full structure",
			reviewBody: `https://github.com/biwakonbu/pylay/blob/e031f51d8a09d2ce9081897a5e2ac35fc5901453/src/core/schemas/yaml_spec.py#L1-L5
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  analyze_issuesのmypy対象ファイルを新名称に合わせる**

このコミットで` + "`src/core/schemas/yaml_type_spec.py`" + `を` + "`yaml_spec.py`" + `へリネームしていますが、
` + "`analyze_issues.sh`" + `内でのmypy実行対象ファイルが旧名称` + "`yaml_type_spec.py`" + `のままです。`,
			expectedCount: 1,
			expectedFirst: &EmbeddedComment{
				FilePath:    "src/core/schemas/yaml_spec.py",
				StartLine:   1,
				EndLine:     5,
				Priority:    "P1",
				Title:       "analyze_issuesのmypy対象ファイルを新名称に合わせる",
				Description: "このコミットで`src/core/schemas/yaml_type_spec.py`を`yaml_spec.py`へリネームしていますが、\n`analyze_issues.sh`内でのmypy実行対象ファイルが旧名称`yaml_type_spec.py`のままです。",
				Permalink:   "https://github.com/biwakonbu/pylay/blob/e031f51d8a09d2ce9081897a5e2ac35fc5901453/src/core/schemas/yaml_spec.py#L1-L5",
			},
		},
		{
			name: "Multiple comments with different priorities",
			reviewBody: `https://github.com/owner/repo/blob/abc123/file1.py#L10-L20
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  Critical bug fix needed**

This is a critical issue that needs immediate attention.

https://github.com/owner/repo/blob/abc123/file2.py#L30
**<sub><sub>![P2 Badge](https://img.shields.io/badge/P2-yellow?style=flat)</sub></sub>  Medium priority refactoring**

Consider refactoring this section for better readability.`,
			expectedCount: 2,
			expectedFirst: &EmbeddedComment{
				FilePath:    "file1.py",
				StartLine:   10,
				EndLine:     20,
				Priority:    "P1",
				Title:       "Critical bug fix needed",
				Description: "This is a critical issue that needs immediate attention.",
				Permalink:   "https://github.com/owner/repo/blob/abc123/file1.py#L10-L20",
			},
			expectedSecond: &EmbeddedComment{
				FilePath:    "file2.py",
				StartLine:   30,
				EndLine:     30,
				Priority:    "P2",
				Title:       "Medium priority refactoring",
				Description: "Consider refactoring this section for better readability.",
				Permalink:   "https://github.com/owner/repo/blob/abc123/file2.py#L30",
			},
		},
		{
			name: "P3 priority comment",
			reviewBody: `https://github.com/owner/repo/blob/abc123/file3.py#L5
**<sub><sub>![P3 Badge](https://img.shields.io/badge/P3-green?style=flat)</sub></sub>  Minor code style improvement**

This is a minor suggestion for code style consistency.`,
			expectedCount: 1,
			expectedFirst: &EmbeddedComment{
				FilePath:    "file3.py",
				StartLine:   5,
				EndLine:     5,
				Priority:    "P3",
				Title:       "Minor code style improvement",
				Description: "This is a minor suggestion for code style consistency.",
				Permalink:   "https://github.com/owner/repo/blob/abc123/file3.py#L5",
			},
		},
		{
			name:          "Empty review body",
			reviewBody:    "",
			expectedCount: 0,
		},
		{
			name: "Review body with no embedded comments",
			reviewBody: `This is a general review comment without any specific file references.
Just some overall feedback on the PR.`,
			expectedCount: 0,
		},
		{
			name: "URL-encoded file path",
			reviewBody: `https://github.com/owner/repo/blob/abc123/src%2Fcore%2Fschemas%2Fyaml_spec.py#L1
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  Test URL encoding**

Test description.`,
			expectedCount: 1,
			expectedFirst: &EmbeddedComment{
				FilePath:    "src/core/schemas/yaml_spec.py",
				StartLine:   1,
				EndLine:     1,
				Priority:    "P1",
				Title:       "Test URL encoding",
				Description: "Test description.",
				Permalink:   "https://github.com/owner/repo/blob/abc123/src%2Fcore%2Fschemas%2Fyaml_spec.py#L1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments := ParseEmbeddedComments(tt.reviewBody)

			if len(comments) != tt.expectedCount {
				t.Errorf("Expected %d comments, got %d", tt.expectedCount, len(comments))
				return
			}

			if tt.expectedFirst != nil && len(comments) > 0 {
				compareEmbeddedComments(t, "first comment", tt.expectedFirst, &comments[0])
			}

			if tt.expectedSecond != nil && len(comments) > 1 {
				compareEmbeddedComments(t, "second comment", tt.expectedSecond, &comments[1])
			}
		})
	}
}

func compareEmbeddedComments(t *testing.T, label string, expected, actual *EmbeddedComment) {
	if actual.FilePath != expected.FilePath {
		t.Errorf("%s: Expected FilePath %q, got %q", label, expected.FilePath, actual.FilePath)
	}
	if actual.StartLine != expected.StartLine {
		t.Errorf("%s: Expected StartLine %d, got %d", label, expected.StartLine, actual.StartLine)
	}
	if actual.EndLine != expected.EndLine {
		t.Errorf("%s: Expected EndLine %d, got %d", label, expected.EndLine, actual.EndLine)
	}
	if actual.Priority != expected.Priority {
		t.Errorf("%s: Expected Priority %q, got %q", label, expected.Priority, actual.Priority)
	}
	if actual.Title != expected.Title {
		t.Errorf("%s: Expected Title %q, got %q", label, expected.Title, actual.Title)
	}
	if actual.Description != expected.Description {
		t.Errorf("%s: Expected Description %q, got %q", label, expected.Description, actual.Description)
	}
	if actual.Permalink != expected.Permalink {
		t.Errorf("%s: Expected Permalink %q, got %q", label, expected.Permalink, actual.Permalink)
	}
}

func TestIsCodexReview(t *testing.T) {
	tests := []struct {
		name     string
		reviewer string
		expected bool
	}{
		{
			name:     "Exact Codex username",
			reviewer: "chatgpt-codex-connector",
			expected: true,
		},
		{
			name:     "Contains 'codex' lowercase",
			reviewer: "some-codex-bot",
			expected: true,
		},
		{
			name:     "Contains 'CODEX' uppercase",
			reviewer: "CODEX-REVIEWER",
			expected: true,
		},
		{
			name:     "Regular user",
			reviewer: "john-doe",
			expected: false,
		},
		{
			name:     "CodeRabbit bot",
			reviewer: "coderabbitai[bot]",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCodexReview(tt.reviewer)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for reviewer %q", tt.expected, result, tt.reviewer)
			}
		})
	}
}

func TestConvertEmbeddedCommentToComment(t *testing.T) {
	ec := EmbeddedComment{
		FilePath:    "src/main.go",
		StartLine:   10,
		EndLine:     20,
		Priority:    "P1",
		Title:       "Fix bug",
		Description: "This is a detailed description\nof the bug fix needed.",
		Permalink:   "https://github.com/owner/repo/blob/abc123/src/main.go#L10-L20",
	}

	author := "chatgpt-codex-connector"
	createdAt := "2025-10-04T12:00:00Z"

	comment := ConvertEmbeddedCommentToComment(ec, author, createdAt)

	if comment.File != ec.FilePath {
		t.Errorf("Expected File %q, got %q", ec.FilePath, comment.File)
	}
	if comment.Line != ec.StartLine {
		t.Errorf("Expected Line %d, got %d", ec.StartLine, comment.Line)
	}
	if comment.Author != author {
		t.Errorf("Expected Author %q, got %q", author, comment.Author)
	}
	if comment.CreatedAt != createdAt {
		t.Errorf("Expected CreatedAt %q, got %q", createdAt, comment.CreatedAt)
	}
	if comment.URL != ec.Permalink {
		t.Errorf("Expected URL %q, got %q", ec.Permalink, comment.URL)
	}

	expectedBody := "Fix bug\n\nThis is a detailed description\nof the bug fix needed."
	if comment.Body != expectedBody {
		t.Errorf("Expected Body %q, got %q", expectedBody, comment.Body)
	}

	if len(comment.Replies) != 0 {
		t.Errorf("Expected empty Replies, got %d replies", len(comment.Replies))
	}
}

func TestMapPriorityToTaskPriority(t *testing.T) {
	tests := []struct {
		name           string
		codexPriority  string
		expectedResult string
	}{
		{
			name:           "P1 maps to high",
			codexPriority:  "P1",
			expectedResult: "high",
		},
		{
			name:           "P2 maps to medium",
			codexPriority:  "P2",
			expectedResult: "medium",
		},
		{
			name:           "P3 maps to low",
			codexPriority:  "P3",
			expectedResult: "low",
		},
		{
			name:           "Unknown priority defaults to medium",
			codexPriority:  "P4",
			expectedResult: "medium",
		},
		{
			name:           "Empty string defaults to medium",
			codexPriority:  "",
			expectedResult: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapPriorityToTaskPriority(tt.codexPriority)
			if result != tt.expectedResult {
				t.Errorf("Expected %q, got %q for priority %q", tt.expectedResult, result, tt.codexPriority)
			}
		})
	}
}

func TestExtractPriority(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "P1 badge",
			line:     "**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  Test title**",
			expected: "P1",
		},
		{
			name:     "P2 badge",
			line:     "**<sub><sub>![P2 Badge](https://img.shields.io/badge/P2-yellow?style=flat)</sub></sub>  Test title**",
			expected: "P2",
		},
		{
			name:     "P3 badge",
			line:     "**<sub><sub>![P3 Badge](https://img.shields.io/badge/P3-green?style=flat)</sub></sub>  Test title**",
			expected: "P3",
		},
		{
			name:     "No badge",
			line:     "Just a regular line",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPriority(tt.line)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "Title with P1 badge",
			line:     "**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  analyze_issuesのmypy対象ファイルを新名称に合わせる**",
			expected: "analyze_issuesのmypy対象ファイルを新名称に合わせる",
		},
		{
			name:     "Title with P2 badge",
			line:     "**<sub><sub>![P2 Badge](https://img.shields.io/badge/P2-yellow?style=flat)</sub></sub>  Refactor this code**",
			expected: "Refactor this code",
		},
		{
			name:     "Title with HTML tags",
			line:     "**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  <strong>Important</strong> fix**",
			expected: "Important fix",
		},
		{
			name:     "Title without badge",
			line:     "**Just a regular title**",
			expected: "Just a regular title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.line)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestContainsPriorityBadge(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "Contains P1 badge",
			line:     "**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  Title**",
			expected: true,
		},
		{
			name:     "Contains P2 badge",
			line:     "**<sub><sub>![P2 Badge](https://img.shields.io/badge/P2-yellow?style=flat)</sub></sub>  Title**",
			expected: true,
		},
		{
			name:     "Contains P3 badge",
			line:     "**<sub><sub>![P3 Badge](https://img.shields.io/badge/P3-green?style=flat)</sub></sub>  Title**",
			expected: true,
		},
		{
			name:     "No badge",
			line:     "Just a regular line",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsPriorityBadge(tt.line)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseEmbeddedComments_RealCodexExample(t *testing.T) {
	// Real example from biwakonbu/pylay PR #26
	realReviewBody := `https://github.com/biwakonbu/pylay/blob/e031f51d8a09d2ce9081897a5e2ac35fc5901453/src/core/schemas/yaml_spec.py#L1-L5
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  analyze_issuesのmypy対象ファイルを新名称に合わせる**

このコミットで` + "`src/core/schemas/yaml_type_spec.py`" + `を` + "`yaml_spec.py`" + `へリネームしていますが、
` + "`analyze_issues.sh`" + `内でのmypy実行対象ファイルが旧名称` + "`yaml_type_spec.py`" + `のままです。
対象を新名称の` + "`yaml_spec.py`" + `に修正しましょう。

https://github.com/biwakonbu/pylay/blob/e031f51d8a09d2ce9081897a5e2ac35fc5901453/analyze_issues.sh#L25-L30
**<sub><sub>![P2 Badge](https://img.shields.io/badge/P2-yellow?style=flat)</sub></sub>  mypy実行コマンドのファイルパスを確認**

mypyの実行時に旧名称` + "`yaml_type_spec.py`" + `を指定していますが、
これも新名称` + "`yaml_spec.py`" + `に更新が必要です。`

	comments := ParseEmbeddedComments(realReviewBody)

	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
		return
	}

	// Verify first comment
	first := comments[0]
	if first.FilePath != "src/core/schemas/yaml_spec.py" {
		t.Errorf("Expected FilePath 'src/core/schemas/yaml_spec.py', got %q", first.FilePath)
	}
	if first.StartLine != 1 {
		t.Errorf("Expected StartLine 1, got %d", first.StartLine)
	}
	if first.EndLine != 5 {
		t.Errorf("Expected EndLine 5, got %d", first.EndLine)
	}
	if first.Priority != "P1" {
		t.Errorf("Expected Priority 'P1', got %q", first.Priority)
	}
	if !strings.Contains(first.Title, "analyze_issues") {
		t.Errorf("Expected Title to contain 'analyze_issues', got %q", first.Title)
	}
	if !strings.Contains(first.Description, "yaml_type_spec.py") {
		t.Errorf("Expected Description to contain 'yaml_type_spec.py', got %q", first.Description)
	}

	// Verify second comment
	second := comments[1]
	if second.FilePath != "analyze_issues.sh" {
		t.Errorf("Expected FilePath 'analyze_issues.sh', got %q", second.FilePath)
	}
	if second.StartLine != 25 {
		t.Errorf("Expected StartLine 25, got %d", second.StartLine)
	}
	if second.EndLine != 30 {
		t.Errorf("Expected EndLine 30, got %d", second.EndLine)
	}
	if second.Priority != "P2" {
		t.Errorf("Expected Priority 'P2', got %q", second.Priority)
	}
}

func TestParseEmbeddedComments_InvalidLineNumbers(t *testing.T) {
	tests := []struct {
		name          string
		reviewBody    string
		expectedCount int
		checkFunc     func(*testing.T, []EmbeddedComment)
	}{
		{
			name: "Invalid start line number",
			reviewBody: `https://github.com/owner/repo/blob/abc123/file.py#Linvalid-L20
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  Test**

Content`,
			expectedCount: 1,
			checkFunc: func(t *testing.T, comments []EmbeddedComment) {
				if comments[0].StartLine != 0 {
					t.Errorf("Expected StartLine 0 for invalid input, got %d", comments[0].StartLine)
				}
			},
		},
		{
			name: "Invalid end line number",
			reviewBody: `https://github.com/owner/repo/blob/abc123/file.py#L10-Linvalid
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  Test**

Content`,
			expectedCount: 1,
			checkFunc: func(t *testing.T, comments []EmbeddedComment) {
				if comments[0].StartLine != 10 {
					t.Errorf("Expected StartLine 10, got %d", comments[0].StartLine)
				}
				// End line should default to start line when parsing fails
				if comments[0].EndLine != 10 {
					t.Errorf("Expected EndLine 10 (fallback to StartLine), got %d", comments[0].EndLine)
				}
			},
		},
		{
			name: "Valid line numbers should parse correctly",
			reviewBody: `https://github.com/owner/repo/blob/abc123/file.py#L15-L25
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  Test**

Content`,
			expectedCount: 1,
			checkFunc: func(t *testing.T, comments []EmbeddedComment) {
				if comments[0].StartLine != 15 {
					t.Errorf("Expected StartLine 15, got %d", comments[0].StartLine)
				}
				if comments[0].EndLine != 25 {
					t.Errorf("Expected EndLine 25, got %d", comments[0].EndLine)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments := ParseEmbeddedComments(tt.reviewBody)
			if len(comments) != tt.expectedCount {
				t.Fatalf("Expected %d comments, got %d", tt.expectedCount, len(comments))
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, comments)
			}
		})
	}
}
