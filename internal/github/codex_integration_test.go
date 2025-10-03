package github

import (
	"testing"
)

// TestCodexIntegration_RealWorldScenario tests the complete Codex review processing flow
// This test validates that Codex reviews are parsed, deduplicated, and converted correctly
func TestCodexIntegration_RealWorldScenario(t *testing.T) {
	// Skip in short mode as this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Real Codex review body from biwakonbu/pylay PR #26
	realCodexReviewBody := `https://github.com/biwakonbu/pylay/blob/e031f51d8a09d2ce9081897a5e2ac35fc5901453/src/core/schemas/yaml_spec.py#L1-L5
**<sub><sub>![P1 Badge](https://img.shields.io/badge/P1-orange?style=flat)</sub></sub>  analyze_issuesのmypy対象ファイルを新名称に合わせる**

このコミットで` + "`src/core/schemas/yaml_type_spec.py`" + `を` + "`yaml_spec.py`" + `へリネームしていますが、
` + "`analyze_issues.sh`" + `内でのmypy実行対象ファイルが旧名称` + "`yaml_type_spec.py`" + `のままです。
対象を新名称の` + "`yaml_spec.py`" + `に修正しましょう。

https://github.com/biwakonbu/pylay/blob/e031f51d8a09d2ce9081897a5e2ac35fc5901453/analyze_issues.sh#L25-L30
**<sub><sub>![P2 Badge](https://img.shields.io/badge/P2-yellow?style=flat)</sub></sub>  mypy実行コマンドのファイルパスを確認**

mypyの実行時に旧名称` + "`yaml_type_spec.py`" + `を指定していますが、
これも新名称` + "`yaml_spec.py`" + `に更新が必要です。`

	// Step 1: Parse embedded comments
	embeddedComments := ParseEmbeddedComments(realCodexReviewBody)

	if len(embeddedComments) != 2 {
		t.Fatalf("Expected 2 embedded comments, got %d", len(embeddedComments))
	}

	// Step 2: Convert to standard Comment format
	submittedAt := "2025-10-04T12:00:00Z"
	reviewer := "chatgpt-codex-connector"

	var convertedComments []Comment
	for _, ec := range embeddedComments {
		comment := ConvertEmbeddedCommentToComment(ec, reviewer, submittedAt)
		convertedComments = append(convertedComments, comment)
	}

	// Verify first comment (P1 priority)
	firstComment := convertedComments[0]
	if firstComment.File != "src/core/schemas/yaml_spec.py" {
		t.Errorf("Expected file 'src/core/schemas/yaml_spec.py', got %q", firstComment.File)
	}
	if firstComment.Line != 1 {
		t.Errorf("Expected line 1, got %d", firstComment.Line)
	}
	if firstComment.Author != reviewer {
		t.Errorf("Expected author %q, got %q", reviewer, firstComment.Author)
	}
	if !containsString(firstComment.Body, "analyze_issues") {
		t.Errorf("Expected body to contain 'analyze_issues'")
	}

	// Verify priority mapping
	firstPriority := MapPriorityToTaskPriority(embeddedComments[0].Priority)
	if firstPriority != "HIGH" {
		t.Errorf("Expected P1 to map to HIGH, got %q", firstPriority)
	}

	// Verify second comment (P2 priority)
	secondComment := convertedComments[1]
	if secondComment.File != "analyze_issues.sh" {
		t.Errorf("Expected file 'analyze_issues.sh', got %q", secondComment.File)
	}
	if secondComment.Line != 25 {
		t.Errorf("Expected line 25, got %d", secondComment.Line)
	}

	secondPriority := MapPriorityToTaskPriority(embeddedComments[1].Priority)
	if secondPriority != "MEDIUM" {
		t.Errorf("Expected P2 to map to MEDIUM, got %q", secondPriority)
	}

	// Step 3: Test deduplication with duplicate reviews
	review1 := Review{
		ID:          1,
		Reviewer:    reviewer,
		Body:        realCodexReviewBody,
		State:       "COMMENTED",
		SubmittedAt: "2025-10-04T10:00:00Z",
		Comments:    convertedComments,
	}

	// Simulate duplicate review (same content, submitted later)
	review2 := Review{
		ID:          2,
		Reviewer:    reviewer,
		Body:        realCodexReviewBody,
		State:       "COMMENTED",
		SubmittedAt: "2025-10-04T10:05:00Z",
		Comments:    convertedComments,
	}

	reviews := []Review{review1, review2}
	deduplicated := DeduplicateReviews(reviews)

	if len(deduplicated) != 1 {
		t.Errorf("Expected 1 deduplicated review, got %d", len(deduplicated))
	}

	// Step 4: Verify the complete flow
	// Simulate how GetPRReviews would process this
	processedReview := review1
	if IsCodexReview(processedReview.Reviewer) {
		embeddedComments := ParseEmbeddedComments(processedReview.Body)
		for _, ec := range embeddedComments {
			comment := ConvertEmbeddedCommentToComment(ec, processedReview.Reviewer, processedReview.SubmittedAt)
			// In the real implementation, this would be appended to processedReview.Comments
			_ = comment
		}
	}

	t.Logf("✓ Successfully processed real Codex review with %d comments", len(convertedComments))
}

// TestCodexIntegration_EmptyReviewBody tests handling of empty review bodies
func TestCodexIntegration_EmptyReviewBody(t *testing.T) {
	// Empty review body should produce no embedded comments
	embeddedComments := ParseEmbeddedComments("")

	if len(embeddedComments) != 0 {
		t.Errorf("Expected 0 embedded comments for empty body, got %d", len(embeddedComments))
	}
}

// TestCodexIntegration_MixedReviewers tests that only Codex reviews are processed
func TestCodexIntegration_MixedReviewers(t *testing.T) {
	codexReview := Review{
		ID:          1,
		Reviewer:    "chatgpt-codex-connector",
		Body:        "Some review content",
		State:       "COMMENTED",
		SubmittedAt: "2025-10-04T10:00:00Z",
	}

	regularReview := Review{
		ID:          2,
		Reviewer:    "john-doe",
		Body:        "Regular review",
		State:       "COMMENTED",
		SubmittedAt: "2025-10-04T10:05:00Z",
	}

	// Only Codex reviews should be detected
	if !IsCodexReview(codexReview.Reviewer) {
		t.Error("Expected Codex review to be detected")
	}

	if IsCodexReview(regularReview.Reviewer) {
		t.Error("Expected regular review to not be detected as Codex")
	}
}

// TestCodexIntegration_ThreadIDTracking tests that embedded comments have no thread IDs
func TestCodexIntegration_ThreadIDTracking(t *testing.T) {
	ec := EmbeddedComment{
		FilePath:    "main.go",
		StartLine:   10,
		EndLine:     20,
		Priority:    "P1",
		Title:       "Fix issue",
		Description: "Description",
		Permalink:   "https://github.com/owner/repo/blob/hash/main.go#L10-L20",
	}

	comment := ConvertEmbeddedCommentToComment(ec, "codex-bot", "2025-10-04T12:00:00Z")

	// Embedded comments should have ID = 0 (no GitHub comment ID)
	if comment.ID != 0 {
		t.Errorf("Expected embedded comment to have ID=0, got %d", comment.ID)
	}

	// This means they won't be auto-resolved (no thread ID to resolve)
	// This is expected behavior documented in the implementation
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findStringSubstring(s, substr)))
}

func findStringSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGraphQLClientIntegration tests GraphQL client functionality
// Note: This test requires valid GitHub credentials and is skipped by default
func TestGraphQLClientIntegration(t *testing.T) {
	t.Skip("Skipping GraphQL integration test - requires valid GitHub credentials")

	// This test would validate:
	// 1. Creating a GraphQL client
	// 2. Fetching review thread IDs
	// 3. Resolving threads
	//
	// Example implementation:
	// client := NewGraphQLClient("test-token")
	// threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 456)
	// err = client.ResolveReviewThread(ctx, threadID)
}
