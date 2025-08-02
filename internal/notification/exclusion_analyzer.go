package notification

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// ExclusionAnalyzer analyzes why review comments were not converted to tasks
type ExclusionAnalyzer struct {
	projectRoot string
}

// NewExclusionAnalyzer creates a new exclusion analyzer
func NewExclusionAnalyzer() *ExclusionAnalyzer {
	return &ExclusionAnalyzer{
		projectRoot: ".", // Current directory
	}
}

// AnalyzeExclusions determines which review comments were not converted to tasks and why
func (ea *ExclusionAnalyzer) AnalyzeExclusions(reviews []github.Review, tasks []storage.Task) []ExcludedComment {
	// Create map of comment IDs that have tasks
	tasksByCommentID := make(map[int64]bool)
	for _, task := range tasks {
		if task.SourceCommentID != 0 {
			tasksByCommentID[task.SourceCommentID] = true
		}
		// Also check CommentID field for compatibility
		if task.CommentID != 0 {
			tasksByCommentID[task.CommentID] = true
		}
	}

	var excluded []ExcludedComment

	// Check each review and comment
	for _, review := range reviews {
		// Check review body
		if review.Body != "" && !tasksByCommentID[review.ID] {
			reason := ea.analyzeExclusionReason(review.Body, review)
			excluded = append(excluded, ExcludedComment{
				Review:          review,
				IsReviewBody:    true,
				ExclusionReason: reason,
			})
		}

		// Check individual comments
		for _, comment := range review.Comments {
			if !tasksByCommentID[comment.ID] {
				reason := ea.analyzeExclusionReason(comment.Body, review)
				excluded = append(excluded, ExcludedComment{
					Review:          review,
					Comment:         comment,
					IsReviewBody:    false,
					ExclusionReason: reason,
				})
			}
		}
	}

	return excluded
}

// ExcludedComment represents a comment that was not converted to a task
type ExcludedComment struct {
	Review          github.Review
	Comment         github.Comment
	IsReviewBody    bool
	ExclusionReason *ExclusionReason
}

// analyzeExclusionReason determines why a comment was excluded
func (ea *ExclusionAnalyzer) analyzeExclusionReason(commentBody string, review github.Review) *ExclusionReason {
	lowerBody := strings.ToLower(commentBody)

	// Check for resolution markers
	if ea.isResolved(lowerBody) {
		return &ExclusionReason{
			Type:        ExclusionTypeAlreadyImplemented,
			Explanation: "This comment has been marked as resolved or addressed",
			Confidence:  0.9,
		}
	}

	// Check for non-actionable comments
	if ea.isNonActionable(lowerBody) {
		return &ExclusionReason{
			Type:        ExclusionTypeInvalid,
			Explanation: "This comment doesn't contain actionable feedback",
			Confidence:  0.8,
		}
	}

	// Check for low priority patterns
	if ea.isLowPriority(lowerBody) {
		return &ExclusionReason{
			Type:        ExclusionTypeLowPriority,
			Explanation: "This comment contains only low-priority suggestions or nitpicks",
			Confidence:  0.7,
		}
	}

	// Check for policy violations
	if violations := ea.checkPolicyViolations(commentBody); len(violations) > 0 {
		return &ExclusionReason{
			Type:        ExclusionTypePolicy,
			Explanation: "This suggestion violates project policies",
			References:  violations,
			Confidence:  0.85,
		}
	}

	// Default: unclear why it was excluded
	return &ExclusionReason{
		Type:        ExclusionTypeInvalid,
		Explanation: "The AI determined this comment doesn't require action",
		Confidence:  0.5,
	}
}

// isResolved checks if a comment has been marked as resolved
func (ea *ExclusionAnalyzer) isResolved(lowerBody string) bool {
	resolvedMarkers := []string{
		"addressed in commit",
		"fixed in commit",
		"resolved in commit",
		"✅ addressed",
		"✅ fixed",
		"✅ resolved",
		"already fixed",
		"already addressed",
		"already implemented",
	}

	for _, marker := range resolvedMarkers {
		if strings.Contains(lowerBody, marker) {
			return true
		}
	}
	return false
}

// isNonActionable checks if a comment is not actionable
func (ea *ExclusionAnalyzer) isNonActionable(lowerBody string) bool {
	nonActionablePatterns := []string{
		"lgtm",
		"looks good to me",
		"nice work",
		"good job",
		"thank you",
		"thanks",
		"acknowledged",
		"noted",
		"understood",
		"+1",
		"👍",
		"no action required",
		"no changes needed",
		"for your information",
		"fyi",
		"great work",
		"well done",
	}

	// Check if the entire comment is just one of these patterns
	trimmed := strings.TrimSpace(lowerBody)
	for _, pattern := range nonActionablePatterns {
		if trimmed == pattern || trimmed == pattern+"." || trimmed == pattern+"!" {
			return true
		}
		// Also check if comment starts with pattern followed by punctuation
		if strings.HasPrefix(trimmed, pattern+"!") || strings.HasPrefix(trimmed, pattern+".") {
			// Check if the rest is also non-actionable
			remainder := strings.TrimSpace(strings.TrimPrefix(trimmed, pattern))
			remainder = strings.TrimPrefix(remainder, "!")
			remainder = strings.TrimPrefix(remainder, ".")
			remainder = strings.TrimSpace(remainder)
			if remainder == "" {
				return true
			}
			// Recursively check if remainder is also non-actionable
			if ea.isNonActionable(remainder) {
				return true
			}
		}
	}

	// Check if it's a question without actionable content
	if strings.HasSuffix(trimmed, "?") && len(trimmed) < 50 {
		return true
	}

	return false
}

// isLowPriority checks if a comment is low priority
func (ea *ExclusionAnalyzer) isLowPriority(lowerBody string) bool {
	lowPriorityPatterns := []string{
		"nit:",
		"nitpick:",
		"minor:",
		"suggestion:",
		"consider:",
		"optional:",
		"style:",
		"🧹 nitpick",
		"nitpick comments",
		"actionable comments posted: 0",
	}

	for _, pattern := range lowPriorityPatterns {
		if strings.Contains(lowerBody, pattern) {
			return true
		}
	}
	return false
}

// checkPolicyViolations checks if a comment violates project policies
func (ea *ExclusionAnalyzer) checkPolicyViolations(commentBody string) []string {
	var violations []string

	// Check for common policy documents
	policyFiles := []string{
		"CONTRIBUTING.md",
		"CODE_OF_CONDUCT.md",
		"ARCHITECTURE.md",
		".github/PULL_REQUEST_TEMPLATE.md",
	}

	for _, file := range policyFiles {
		fullPath := filepath.Join(ea.projectRoot, file)
		if _, err := os.Stat(fullPath); err == nil {
			// File exists, check if comment might violate its guidelines
			// This is a simplified check - in reality, you'd parse the file
			if strings.Contains(strings.ToLower(commentBody), "rewrite") &&
				file == "CONTRIBUTING.md" {
				violations = append(violations, fmt.Sprintf("%s - Major refactoring guidelines", file))
			}
		}
	}

	// Check for architecture violations
	if strings.Contains(strings.ToLower(commentBody), "change architecture") ||
		strings.Contains(strings.ToLower(commentBody), "restructure") {
		violations = append(violations, "ARCHITECTURE.md - Architectural changes require RFC")
	}

	return violations
}
