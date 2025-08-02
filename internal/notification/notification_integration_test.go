package notification

import (
	"context"
	"strings"
	"testing"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// MockGitHubClientIntegration for integration tests
type MockGitHubClientIntegration struct {
	PostedComments []PostedCommentIntegration
}

type PostedCommentIntegration struct {
	PR       int
	Body     string
	PostedAt time.Time
}

func (m *MockGitHubClientIntegration) CreateIssueComment(ctx context.Context, prNumber int, body string) error {
	m.PostedComments = append(m.PostedComments, PostedCommentIntegration{
		PR:       prNumber,
		Body:     body,
		PostedAt: time.Now(),
	})
	return nil
}

func TestNotificationWorkflowIntegration(t *testing.T) {
	// Setup
	cfg := &config.Config{
		CommentSettings: config.CommentSettings{
			Enabled: true,
			AutoCommentOn: config.AutoCommentSettings{
				TaskCompletion:   true,
				TaskCancellation: true,
				TaskPending:      true,
				TaskExclusion:    true,
			},
			Templates: config.CommentTemplates{
				Completion:   "", // Use default templates
				Cancellation: "",
				Pending:      "",
				Exclusion:    "",
			},
			Throttling: config.ThrottlingSettings{
				Enabled:              false, // Disable throttling for tests
				MaxCommentsPerHour:   100,
				BatchWindowMinutes:   30,
				BatchSimilarComments: false,
			},
		},
	}

	mockGitHub := &MockGitHubClientIntegration{}
	notifier := New(mockGitHub, cfg)

	ctx := context.Background()

	t.Run("TaskCompletion", func(t *testing.T) {
		task := &storage.Task{
			ID:            "test-task-1",
			Description:   "Fix memory leak in parser",
			Status:        "done",
			PR:            123,
			ReviewerLogin: "reviewer1",
		}

		err := notifier.NotifyTaskCompletion(ctx, task)
		if err != nil {
			t.Fatalf("NotifyTaskCompletion failed: %v", err)
		}

		if len(mockGitHub.PostedComments) != 1 {
			t.Fatalf("Expected 1 comment, got %d", len(mockGitHub.PostedComments))
		}

		comment := mockGitHub.PostedComments[0]
		if comment.PR != 123 {
			t.Errorf("Expected PR 123, got %d", comment.PR)
		}
		if !containsString(comment.Body, "✅") || !containsString(comment.Body, "Task Completed") {
			t.Errorf("Comment body doesn't match expected: %s", comment.Body)
		}
	})

	t.Run("TaskCancellation", func(t *testing.T) {
		mockGitHub.PostedComments = nil // Reset

		task := &storage.Task{
			ID:            "test-task-2",
			Description:   "Refactor authentication system",
			Status:        "cancelled",
			PR:            124,
			ReviewerLogin: "reviewer2",
		}

		err := notifier.NotifyTaskCancellation(ctx, task, "Not applicable to current architecture")
		if err != nil {
			t.Fatalf("NotifyTaskCancellation failed: %v", err)
		}

		if len(mockGitHub.PostedComments) != 1 {
			t.Fatalf("Expected 1 comment, got %d", len(mockGitHub.PostedComments))
		}

		comment := mockGitHub.PostedComments[0]
		if !containsString(comment.Body, "🚫") || !containsString(comment.Body, "Task Cancelled") {
			t.Errorf("Comment body doesn't match expected: %s", comment.Body)
		}
	})

	t.Run("TaskPending", func(t *testing.T) {
		mockGitHub.PostedComments = nil // Reset

		task := &storage.Task{
			ID:            "test-task-3",
			Description:   "Update documentation",
			Status:        "pending",
			PR:            125,
			ReviewerLogin: "reviewer3",
		}

		err := notifier.NotifyTaskPending(ctx, task, "Waiting for architecture decision")
		if err != nil {
			t.Fatalf("NotifyTaskPending failed: %v", err)
		}

		if len(mockGitHub.PostedComments) != 1 {
			t.Fatalf("Expected 1 comment, got %d", len(mockGitHub.PostedComments))
		}

		comment := mockGitHub.PostedComments[0]
		if !containsString(comment.Body, "⏳") || !containsString(comment.Body, "Task Pending") {
			t.Errorf("Comment body doesn't match expected: %s", comment.Body)
		}
	})

	t.Run("TaskExclusion", func(t *testing.T) {
		mockGitHub.PostedComments = nil // Reset

		review := github.Review{
			ID:       1,
			Reviewer: "reviewer4",
			Body:     "LGTM! Great work on this feature.",
			Comments: []github.Comment{
				{
					ID:     100,
					Body:   "LGTM! Great work on this feature.",
					Author: "reviewer4",
				},
			},
		}

		exclusionReason := &ExclusionReason{
			Type:        ExclusionTypeInvalid,
			Explanation: "This comment contains only praise and doesn't require any action",
			Confidence:  0.9,
		}

		err := notifier.NotifyTaskExclusion(ctx, review, exclusionReason)
		if err != nil {
			t.Fatalf("NotifyTaskExclusion failed: %v", err)
		}

		if len(mockGitHub.PostedComments) != 1 {
			t.Fatalf("Expected 1 comment, got %d", len(mockGitHub.PostedComments))
		}

		comment := mockGitHub.PostedComments[0]
		if !containsString(comment.Body, "ℹ️") || !containsString(comment.Body, "not converted to a task") {
			t.Errorf("Comment body doesn't match expected: %s", comment.Body)
		}
	})
}

func TestThrottlingIntegration(t *testing.T) {
	cfg := &config.Config{
		CommentSettings: config.CommentSettings{
			Enabled: true,
			AutoCommentOn: config.AutoCommentSettings{
				TaskCompletion: true,
			},
			Throttling: config.ThrottlingSettings{
				Enabled:              true, // Enable throttling for this test
				MaxCommentsPerHour:   2,    // Very low for testing
				BatchWindowMinutes:   1,
				BatchSimilarComments: true,
			},
		},
	}

	mockGitHub := &MockGitHubClientIntegration{}
	notifier := New(mockGitHub, cfg)

	ctx := context.Background()

	// Post multiple comments rapidly
	for i := 0; i < 5; i++ {
		task := &storage.Task{
			ID:            "test-task-" + string(rune('1'+i)),
			Description:   "Test task " + string(rune('1'+i)),
			Status:        "done",
			PR:            200 + i,
			ReviewerLogin: "reviewer1",
		}

		err := notifier.NotifyTaskCompletion(ctx, task)
		if err != nil {
			t.Fatalf("NotifyTaskCompletion %d failed: %v", i, err)
		}
	}

	// Check that throttling kicked in
	if len(mockGitHub.PostedComments) > 2 {
		t.Errorf("Expected throttling to limit comments to 2, got %d", len(mockGitHub.PostedComments))
	}
}

func TestExclusionAnalysisIntegration(t *testing.T) {
	analyzer := NewExclusionAnalyzer()

	// Create comprehensive test data
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "reviewer1",
			Body:     "Overall looks good, just a few minor suggestions",
			Comments: []github.Comment{
				{
					ID:     101,
					Body:   "Fix the memory leak on line 42",
					Author: "reviewer1",
				},
				{
					ID:     102,
					Body:   "nit: consider using camelCase for variable names",
					Author: "reviewer1",
				},
				{
					ID:     103,
					Body:   "LGTM! Great implementation.",
					Author: "reviewer1",
				},
				{
					ID:     104,
					Body:   "✅ Fixed in commit abc123",
					Author: "author",
				},
			},
		},
	}

	// Create tasks for only some comments
	tasks := []storage.Task{
		{
			ID:              "task-1",
			SourceCommentID: 101,
			Description:     "Fix memory leak on line 42",
		},
	}

	// Analyze exclusions
	excluded := analyzer.AnalyzeExclusions(reviews, tasks)

	// Should exclude: review body + 3 comments (102, 103, 104)
	expectedExclusions := 4
	if len(excluded) != expectedExclusions {
		t.Errorf("Expected %d exclusions, got %d", expectedExclusions, len(excluded))
	}

	// Verify exclusion reasons
	exclusionTypes := make(map[string]int)
	for _, exc := range excluded {
		exclusionTypes[exc.ExclusionReason.Type]++
	}

	// Should have examples of different exclusion types
	if exclusionTypes[ExclusionTypeLowPriority] == 0 {
		t.Error("Expected at least one low priority exclusion")
	}
	if exclusionTypes[ExclusionTypeInvalid] == 0 {
		t.Error("Expected at least one non-actionable exclusion")
	}
	if exclusionTypes[ExclusionTypeAlreadyImplemented] == 0 {
		t.Error("Expected at least one already implemented exclusion")
	}
}

// Helper function for integration tests
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
