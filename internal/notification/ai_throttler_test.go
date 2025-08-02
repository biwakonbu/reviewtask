package notification

import (
	"context"
	"testing"
	"time"

	"reviewtask/internal/storage"
)

// MockClaudeClient for testing AI throttler
type MockClaudeClient struct {
	SendMessageFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *MockClaudeClient) SendMessage(ctx context.Context, prompt string) (string, error) {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, prompt)
	}
	return "", nil
}

func TestAnalyzeBatchingStrategy(t *testing.T) {
	mockClient := &MockClaudeClient{
		SendMessageFunc: func(ctx context.Context, prompt string) (string, error) {
			// Return a mock AI response
			return `{
				"should_batch": true,
				"batch_groups": [
					{
						"group_id": "pr-42-completions",
						"task_ids": ["task-1", "task-2"],
						"group_reason": "Multiple completions for same PR",
						"priority": "medium"
					}
				],
				"immediate_send": ["task-3"],
				"delayed_send": ["task-4"],
				"reason": "Batch similar completion notifications to reduce noise",
				"optimal_delay_minutes": 30
			}`, nil
		},
	}

	aiThrottler := &AIThrottler{
		claudeClient: mockClient,
	}

	pendingComments := []PendingComment{
		{
			Task: &storage.Task{
				ID: "task-1",
				PR: 42,
			},
			NotificationType: "completion",
			CreatedAt:        time.Now(),
		},
		{
			Task: &storage.Task{
				ID: "task-2",
				PR: 42,
			},
			NotificationType: "completion",
			CreatedAt:        time.Now(),
		},
		{
			Task: &storage.Task{
				ID: "task-3",
				PR: 43,
			},
			NotificationType: "completion",
			CreatedAt:        time.Now(),
		},
		{
			Task: &storage.Task{
				ID: "task-4",
				PR: 44,
			},
			NotificationType: "pending",
			CreatedAt:        time.Now(),
		},
	}

	decision, err := aiThrottler.AnalyzeBatchingStrategy(context.Background(), pendingComments, nil)
	if err != nil {
		t.Fatalf("AnalyzeBatchingStrategy failed: %v", err)
	}

	// Verify the decision
	if !decision.ShouldBatch {
		t.Error("Expected ShouldBatch to be true")
	}

	if len(decision.BatchGroups) != 1 {
		t.Errorf("Expected 1 batch group, got %d", len(decision.BatchGroups))
	}

	if len(decision.ImmediateSend) != 1 || decision.ImmediateSend[0] != "task-3" {
		t.Errorf("Expected task-3 in immediate send, got %v", decision.ImmediateSend)
	}

	if len(decision.DelayedSend) != 1 || decision.DelayedSend[0] != "task-4" {
		t.Errorf("Expected task-4 in delayed send, got %v", decision.DelayedSend)
	}

	if decision.OptimalDelay != 30 {
		t.Errorf("Expected optimal delay of 30 minutes, got %d", decision.OptimalDelay)
	}
}

func TestOptimizeCommentTiming(t *testing.T) {
	mockClient := &MockClaudeClient{
		SendMessageFunc: func(ctx context.Context, prompt string) (string, error) {
			// Return different responses based on prompt content
			if contains(prompt, "high") {
				return `{"send_now": true, "delay_minutes": 0, "reason": "High priority task"}`, nil
			}
			return `{"send_now": false, "delay_minutes": 15, "reason": "Reviewer recently notified"}`, nil
		},
	}

	aiThrottler := &AIThrottler{
		claudeClient: mockClient,
	}

	// Test high priority task
	highPriorityTask := &storage.Task{
		ID:            "task-high",
		Priority:      "high",
		ReviewerLogin: "reviewer1",
		PR:            42,
	}

	sendNow, delay, err := aiThrottler.OptimizeCommentTiming(
		context.Background(),
		highPriorityTask,
		"completion",
		nil,
	)

	if err != nil {
		t.Fatalf("OptimizeCommentTiming failed: %v", err)
	}

	if !sendNow {
		t.Error("Expected high priority task to be sent immediately")
	}

	if delay != 0 {
		t.Errorf("Expected no delay for high priority task, got %v", delay)
	}

	// Test regular task with recent activity
	regularTask := &storage.Task{
		ID:            "task-regular",
		Priority:      "medium",
		ReviewerLogin: "reviewer2",
		PR:            43,
	}

	sendNow, delay, err = aiThrottler.OptimizeCommentTiming(
		context.Background(),
		regularTask,
		"completion",
		[]CommentRecord{
			{
				ReviewerLogin: "reviewer2",
				Timestamp:     time.Now().Add(-5 * time.Minute),
			},
		},
	)

	if err != nil {
		t.Fatalf("OptimizeCommentTiming failed: %v", err)
	}

	if sendNow {
		t.Error("Expected regular task to be delayed")
	}

	expectedDelay := 15 * time.Minute
	if delay != expectedDelay {
		t.Errorf("Expected delay of %v, got %v", expectedDelay, delay)
	}
}