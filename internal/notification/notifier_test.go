package notification

import (
	"context"
	"strings"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// MockGitHubClient is a mock implementation of the GitHub client for testing
type MockGitHubClient struct {
	CreateIssueCommentFunc func(ctx context.Context, prNumber int, body string) error
	Comments               []string // Store posted comments for verification
}

func (m *MockGitHubClient) CreateIssueComment(ctx context.Context, prNumber int, body string) error {
	m.Comments = append(m.Comments, body)
	if m.CreateIssueCommentFunc != nil {
		return m.CreateIssueCommentFunc(ctx, prNumber, body)
	}
	return nil
}

func TestNotifyTaskCompletion(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		CommentSettings: config.CommentSettings{
			Enabled: true,
			AutoCommentOn: config.AutoCommentSettings{
				TaskCompletion: true,
			},
			Templates: config.CommentTemplates{
				Completion: "default",
			},
		},
	}

	// Create mock GitHub client
	mockClient := &MockGitHubClient{}

	// Create notifier with mock
	notifier := &Notifier{
		githubClient: mockClient,
		config:       cfg,
		throttler:    NewThrottler(cfg.CommentSettings.Throttling),
	}

	// Create test task
	task := &storage.Task{
		ID:            "test-task-1",
		Title:         "Fix memory leak",
		Description:   "Fix memory leak in parser",
		PR:            42,
		ReviewerLogin: "reviewer123",
	}

	// Test notification
	ctx := context.Background()
	err := notifier.NotifyTaskCompletion(ctx, task)
	if err != nil {
		t.Fatalf("NotifyTaskCompletion failed: %v", err)
	}

	// Verify comment was posted
	if len(mockClient.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(mockClient.Comments))
	}

	// Verify comment content
	comment := mockClient.Comments[0]
	if !contains(comment, "Task Completed") {
		t.Errorf("Comment missing 'Task Completed' header")
	}
	if !contains(comment, "@reviewer123") {
		t.Errorf("Comment missing reviewer mention")
	}
	if !contains(comment, "Fix memory leak") {
		t.Errorf("Comment missing task title")
	}
}

func TestNotifyTaskCancellation(t *testing.T) {
	cfg := &config.Config{
		CommentSettings: config.CommentSettings{
			Enabled: true,
			AutoCommentOn: config.AutoCommentSettings{
				TaskCancellation: true,
			},
			Templates: config.CommentTemplates{
				Cancellation: "default",
			},
		},
	}

	mockClient := &MockGitHubClient{}
	notifier := &Notifier{
		githubClient: mockClient,
		config:       cfg,
		throttler:    NewThrottler(cfg.CommentSettings.Throttling),
	}

	task := &storage.Task{
		ID:    "test-task-2",
		Title: "Add new feature",
		PR:    42,
	}

	ctx := context.Background()
	reason := "Already implemented in PR #123"
	err := notifier.NotifyTaskCancellation(ctx, task, reason)
	if err != nil {
		t.Fatalf("NotifyTaskCancellation failed: %v", err)
	}

	if len(mockClient.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(mockClient.Comments))
	}

	comment := mockClient.Comments[0]
	if !contains(comment, "Task Cancelled") {
		t.Errorf("Comment missing 'Task Cancelled' header")
	}
	if !contains(comment, reason) {
		t.Errorf("Comment missing cancellation reason")
	}
}

func TestNotifyWhenDisabled(t *testing.T) {
	cfg := &config.Config{
		CommentSettings: config.CommentSettings{
			Enabled: false, // Notifications disabled
		},
	}

	mockClient := &MockGitHubClient{}
	notifier := &Notifier{
		githubClient: mockClient,
		config:       cfg,
		throttler:    NewThrottler(cfg.CommentSettings.Throttling),
	}

	task := &storage.Task{
		ID: "test-task-3",
		PR: 42,
	}

	ctx := context.Background()
	err := notifier.NotifyTaskCompletion(ctx, task)
	if err != nil {
		t.Fatalf("NotifyTaskCompletion failed: %v", err)
	}

	// Should not post any comments when disabled
	if len(mockClient.Comments) != 0 {
		t.Errorf("Expected no comments when disabled, got %d", len(mockClient.Comments))
	}
}

func TestExclusionNotification(t *testing.T) {
	cfg := &config.Config{
		CommentSettings: config.CommentSettings{
			Enabled: true,
			AutoCommentOn: config.AutoCommentSettings{
				TaskExclusion: true,
			},
			Templates: config.CommentTemplates{
				Exclusion: "default",
			},
		},
	}

	mockClient := &MockGitHubClient{}
	notifier := &Notifier{
		githubClient: mockClient,
		config:       cfg,
		throttler:    NewThrottler(cfg.CommentSettings.Throttling),
	}

	review := github.Review{
		PR:        42,
		CommentID: 12345,
		User: struct {
			Login string `json:"login"`
		}{
			Login: "reviewer456",
		},
	}

	exclusionReason := &ExclusionReason{
		Type:        ExclusionTypePolicy,
		Explanation: "This violates our coding standards",
		References:  []string{"CONTRIBUTING.md", "PR #45"},
	}

	ctx := context.Background()
	err := notifier.NotifyTaskExclusion(ctx, review, exclusionReason)
	if err != nil {
		t.Fatalf("NotifyTaskExclusion failed: %v", err)
	}

	if len(mockClient.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(mockClient.Comments))
	}

	comment := mockClient.Comments[0]
	if !contains(comment, "not converted to a task") {
		t.Errorf("Comment missing exclusion header")
	}
	if !contains(comment, ExclusionTypePolicy) {
		t.Errorf("Comment missing exclusion type")
	}
	if !contains(comment, "CONTRIBUTING.md") {
		t.Errorf("Comment missing references")
	}
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}