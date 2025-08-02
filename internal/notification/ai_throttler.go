package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

// AIThrottler provides intelligent throttling decisions using AI
type AIThrottler struct {
	config       *config.Config
	claudeClient ai.ClaudeClient
}

// NewAIThrottler creates a new AI-powered throttler
func NewAIThrottler(cfg *config.Config) (*AIThrottler, error) {
	client, err := ai.NewRealClaudeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude client: %w", err)
	}

	return &AIThrottler{
		config:       cfg,
		claudeClient: client,
	}, nil
}

// BatchingDecision represents AI's decision on how to batch comments
type BatchingDecision struct {
	ShouldBatch   bool         `json:"should_batch"`
	BatchGroups   []BatchGroup `json:"batch_groups"`
	ImmediateSend []string     `json:"immediate_send"` // Task IDs to send immediately
	DelayedSend   []string     `json:"delayed_send"`   // Task IDs to delay
	Reason        string       `json:"reason"`
	OptimalDelay  int          `json:"optimal_delay_minutes"`
}

// BatchGroup represents a group of related comments to batch together
type BatchGroup struct {
	GroupID     string   `json:"group_id"`
	TaskIDs     []string `json:"task_ids"`
	GroupReason string   `json:"group_reason"`
	Priority    string   `json:"priority"` // high, medium, low
}

// AnalyzeBatchingStrategy uses AI to determine optimal batching strategy
func (at *AIThrottler) AnalyzeBatchingStrategy(ctx context.Context, pendingComments []PendingComment, recentActivity []CommentRecord) (*BatchingDecision, error) {
	prompt := at.buildBatchingPrompt(pendingComments, recentActivity)

	response, err := at.claudeClient.Execute(ctx, prompt, "json")
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	return at.parseBatchingDecision(response)
}

// buildBatchingPrompt creates the prompt for batching analysis
func (at *AIThrottler) buildBatchingPrompt(pendingComments []PendingComment, recentActivity []CommentRecord) string {
	// Build pending comments summary
	var pendingStr strings.Builder
	for _, pc := range pendingComments {
		pendingStr.WriteString(fmt.Sprintf("- Task %s: %s (PR #%d)\n",
			pc.Task.ID, pc.NotificationType, pc.Task.PR))
	}

	// Build recent activity summary
	var activityStr strings.Builder
	now := time.Now()
	for _, rec := range recentActivity {
		age := now.Sub(rec.Timestamp)
		activityStr.WriteString(fmt.Sprintf("- %s ago: %s notification to @%s (PR #%d)\n",
			age.Round(time.Minute), rec.Type, rec.ReviewerLogin, rec.PR))
	}

	prompt := fmt.Sprintf(`You are optimizing GitHub PR notification batching to reduce noise while maintaining effective communication.

Current Configuration:
- Max comments per hour: %d
- Batch window: %d minutes
- Batching enabled: %v

Pending Notifications:
%s

Recent Activity (last hour):
%s

Analyze the pending notifications and recent activity to determine:
1. Which notifications should be sent immediately (urgent/important)
2. Which should be batched together (related/similar)
3. Which should be delayed to avoid overwhelming reviewers
4. Optimal grouping strategy

Consider:
- Reviewer fatigue (too many notifications)
- PR context (group by PR when sensible)
- Notification type (completions vs cancellations)
- Time since last notification to same reviewer
- Priority of the changes

Respond with JSON in this format:
{
  "should_batch": true/false,
  "batch_groups": [
    {
      "group_id": "string",
      "task_ids": ["task-id1", "task-id2"],
      "group_reason": "string",
      "priority": "high/medium/low"
    }
  ],
  "immediate_send": ["task-id3", "task-id4"],
  "delayed_send": ["task-id5"],
  "reason": "Overall strategy explanation",
  "optimal_delay_minutes": 30
}`,
		at.config.CommentSettings.Throttling.MaxCommentsPerHour,
		at.config.CommentSettings.Throttling.BatchWindowMinutes,
		at.config.CommentSettings.Throttling.BatchSimilarComments,
		pendingStr.String(),
		activityStr.String())

	return prompt
}

// parseBatchingDecision parses the AI response into a BatchingDecision
func (at *AIThrottler) parseBatchingDecision(response string) (*BatchingDecision, error) {
	// Extract JSON from response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	var decision BatchingDecision
	if err := json.Unmarshal([]byte(jsonStr), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &decision, nil
}

// PendingComment represents a comment waiting to be sent
type PendingComment struct {
	Task             *storage.Task
	Comment          string
	NotificationType string
	CreatedAt        time.Time
}

// OptimizeCommentTiming determines the best time to send a comment
func (at *AIThrottler) OptimizeCommentTiming(ctx context.Context, task *storage.Task, notificationType string, reviewerActivity []CommentRecord) (bool, time.Duration, error) {
	prompt := at.buildTimingPrompt(task, notificationType, reviewerActivity)

	response, err := at.claudeClient.Execute(ctx, prompt, "json")
	if err != nil {
		return false, 0, fmt.Errorf("AI analysis failed: %w", err)
	}

	return at.parseTimingDecision(response)
}

// buildTimingPrompt creates the prompt for timing optimization
func (at *AIThrottler) buildTimingPrompt(task *storage.Task, notificationType string, reviewerActivity []CommentRecord) string {
	// Build activity summary for this reviewer
	var activityStr strings.Builder
	now := time.Now()
	notificationCount := 0

	for _, rec := range reviewerActivity {
		if rec.ReviewerLogin == task.ReviewerLogin {
			age := now.Sub(rec.Timestamp)
			activityStr.WriteString(fmt.Sprintf("- %s ago: %s\n", age.Round(time.Minute), rec.Type))
			notificationCount++
		}
	}

	prompt := fmt.Sprintf(`Determine optimal timing for sending a GitHub PR notification.

Task Details:
- Type: %s
- Priority: %s
- Reviewer: @%s
- PR: #%d

Recent notifications to this reviewer:
%s
Total in last hour: %d

Current time: %s (consider timezone/working hours)

Should this notification be:
1. Sent immediately (urgent/important)
2. Delayed (specify optimal delay in minutes)
3. Batched with other pending notifications

Respond with JSON:
{
  "send_now": true/false,
  "delay_minutes": 0,
  "reason": "Explanation"
}`,
		notificationType,
		task.Priority,
		task.ReviewerLogin,
		task.PR,
		activityStr.String(),
		notificationCount,
		now.Format("15:04 MST"))

	return prompt
}

// parseTimingDecision parses the AI timing response
func (at *AIThrottler) parseTimingDecision(response string) (bool, time.Duration, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return false, 0, fmt.Errorf("no valid JSON found in response")
	}

	var result struct {
		SendNow      bool   `json:"send_now"`
		DelayMinutes int    `json:"delay_minutes"`
		Reason       string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return false, 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	delay := time.Duration(result.DelayMinutes) * time.Minute
	return result.SendNow, delay, nil
}
