package notification

import (
	"testing"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

func TestAnalyzeExclusions(t *testing.T) {
	analyzer := NewExclusionAnalyzer()

	// Create test reviews
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "reviewer1",
			Body:     "LGTM! Great work.",
			Comments: []github.Comment{
				{
					ID:     100,
					Body:   "Please fix the memory leak in the parser",
					Author: "reviewer1",
				},
				{
					ID:     101,
					Body:   "nit: Add a comment here",
					Author: "reviewer1",
				},
				{
					ID:     102,
					Body:   "✅ Fixed in commit abc123",
					Author: "author",
				},
			},
		},
	}

	// Create tasks for only one comment
	tasks := []storage.Task{
		{
			ID:              "task-1",
			SourceCommentID: 100,
			Description:     "Fix memory leak in parser",
		},
	}

	// Analyze exclusions
	excluded := analyzer.AnalyzeExclusions(reviews, tasks)

	// Should have 3 exclusions (review body + 2 comments)
	if len(excluded) != 3 {
		t.Errorf("Expected 3 excluded comments, got %d", len(excluded))
	}

	// Check exclusion reasons
	for _, exc := range excluded {
		if exc.IsReviewBody && exc.Review.Body == "LGTM! Great work." {
			if exc.ExclusionReason.Type != ExclusionTypeInvalid {
				t.Errorf("Expected LGTM to be excluded as non-actionable, got %s", exc.ExclusionReason.Type)
			}
		}
		
		if exc.Comment.ID == 101 {
			if exc.ExclusionReason.Type != ExclusionTypeLowPriority {
				t.Errorf("Expected nit comment to be excluded as low priority, got %s", exc.ExclusionReason.Type)
			}
		}
		
		if exc.Comment.ID == 102 {
			if exc.ExclusionReason.Type != ExclusionTypeAlreadyImplemented {
				t.Errorf("Expected resolved comment to be excluded as already implemented, got %s", exc.ExclusionReason.Type)
			}
		}
	}
}

func TestIsResolved(t *testing.T) {
	analyzer := NewExclusionAnalyzer()

	tests := []struct {
		comment  string
		expected bool
	}{
		{"✅ Fixed in commit abc123", true},
		{"Already addressed in the latest commit", true},
		{"Resolved in commit def456", true},
		{"This needs to be fixed", false},
		{"Please address this issue", false},
	}

	for _, test := range tests {
		result := analyzer.isResolved(test.comment)
		if result != test.expected {
			t.Errorf("isResolved(%q) = %v, want %v", test.comment, result, test.expected)
		}
	}
}

func TestIsNonActionable(t *testing.T) {
	analyzer := NewExclusionAnalyzer()

	tests := []struct {
		comment  string
		expected bool
	}{
		{"LGTM", true},
		{"lgtm", true},
		{"Looks good to me!", true},
		{"Nice work", true},
		{"Thanks!", true},
		{"+1", true},
		{"Why did you do this?", true}, // Short question
		{"Please fix the memory leak in the parser", false},
		{"This approach is wrong. Consider using a different algorithm", false},
	}

	for _, test := range tests {
		result := analyzer.isNonActionable(test.comment)
		if result != test.expected {
			t.Errorf("isNonActionable(%q) = %v, want %v", test.comment, result, test.expected)
		}
	}
}

func TestIsLowPriority(t *testing.T) {
	analyzer := NewExclusionAnalyzer()

	tests := []struct {
		comment  string
		expected bool
	}{
		{"nit: Add a comment here", true},
		{"nitpick: Variable naming", true},
		{"minor: Formatting issue", true},
		{"🧹 nitpick", true},
		{"Actionable comments posted: 0", true},
		{"Fix the critical security vulnerability", false},
		{"Memory leak detected", false},
	}

	for _, test := range tests {
		result := analyzer.isLowPriority(test.comment)
		if result != test.expected {
			t.Errorf("isLowPriority(%q) = %v, want %v", test.comment, result, test.expected)
		}
	}
}