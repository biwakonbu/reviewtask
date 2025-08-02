package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// Notifier handles GitHub comment notifications for task status changes
type Notifier struct {
	githubClient GitHubClient
	config       *config.Config
	throttler    *Throttler
}

// New creates a new Notifier instance
func New(githubClient GitHubClient, cfg *config.Config) *Notifier {
	throttler := NewThrottler(cfg.CommentSettings.Throttling)

	// Enable AI throttling if verbose mode is on
	if cfg.AISettings.VerboseMode {
		aiThrottler, err := NewAIThrottler(cfg)
		if err == nil {
			throttler.SetAIThrottler(aiThrottler)
		}
	}

	return &Notifier{
		githubClient: githubClient,
		config:       cfg,
		throttler:    throttler,
	}
}

// NotifyTaskCompletion posts a comment when a task is marked as done
func (n *Notifier) NotifyTaskCompletion(ctx context.Context, task *storage.Task) error {
	if !n.shouldNotify(task, "completion") {
		return nil
	}

	comment := n.formatCompletionComment(task)
	return n.postComment(ctx, task, comment, "completion")
}

// NotifyTaskCancellation posts a comment when a task is cancelled
func (n *Notifier) NotifyTaskCancellation(ctx context.Context, task *storage.Task, reason string) error {
	if !n.shouldNotify(task, "cancellation") {
		return nil
	}

	comment := n.formatCancellationComment(task, reason)
	return n.postComment(ctx, task, comment, "cancellation")
}

// NotifyTaskPending posts a comment when a task is marked as pending
func (n *Notifier) NotifyTaskPending(ctx context.Context, task *storage.Task, reason string) error {
	if !n.shouldNotify(task, "pending") {
		return nil
	}

	comment := n.formatPendingComment(task, reason)
	return n.postComment(ctx, task, comment, "pending")
}

// NotifyTaskExclusion posts a comment explaining why a review comment wasn't converted to a task
func (n *Notifier) NotifyTaskExclusion(ctx context.Context, review github.Review, exclusionReason *ExclusionReason) error {
	if !n.config.CommentSettings.Enabled || !n.config.CommentSettings.AutoCommentOn.TaskExclusion {
		return nil
	}

	comment := n.formatExclusionComment(exclusionReason)

	// Create a pseudo-task for throttling purposes
	pseudoTask := &storage.Task{
		PR:            review.PR,
		CommentID:     review.CommentID,
		ReviewerLogin: review.User.Login,
	}

	return n.postComment(ctx, pseudoTask, comment, "exclusion")
}

// shouldNotify checks if a notification should be sent based on configuration
func (n *Notifier) shouldNotify(task *storage.Task, notificationType string) bool {
	if !n.config.CommentSettings.Enabled {
		return false
	}

	switch notificationType {
	case "completion":
		return n.config.CommentSettings.AutoCommentOn.TaskCompletion
	case "cancellation":
		return n.config.CommentSettings.AutoCommentOn.TaskCancellation
	case "pending":
		return n.config.CommentSettings.AutoCommentOn.TaskPending
	case "exclusion":
		return n.config.CommentSettings.AutoCommentOn.TaskExclusion
	default:
		return false
	}
}

// postComment handles the actual posting of comments with throttling
func (n *Notifier) postComment(ctx context.Context, task *storage.Task, comment string, notificationType string) error {
	// Check throttling
	shouldPost, batchSuggestion := n.throttler.ShouldPostNow(task, notificationType)

	if !shouldPost {
		if batchSuggestion != nil {
			// Add to batch queue
			return n.throttler.AddToBatch(task, comment, notificationType)
		}
		// Skip due to rate limiting
		return fmt.Errorf("comment throttled due to rate limiting")
	}

	// Post the comment
	err := n.githubClient.CreateIssueComment(ctx, task.PR, comment)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	// Record the comment for throttling
	n.throttler.RecordComment(task, notificationType)

	return nil
}

// formatCompletionComment formats a completion notification comment
func (n *Notifier) formatCompletionComment(task *storage.Task) string {
	template := n.getTemplate("completion")
	if template != "default" {
		// TODO: Load and parse custom template
		return template
	}

	// Default template
	var sb strings.Builder
	sb.WriteString("✅ **Task Completed**\n\n")

	if task.ReviewerLogin != "" {
		sb.WriteString(fmt.Sprintf("@%s ", task.ReviewerLogin))
	}
	sb.WriteString("This feedback has been addressed.\n\n")

	sb.WriteString(fmt.Sprintf("**Task**: %s\n", task.Title))

	if task.Implementation != "" {
		sb.WriteString(fmt.Sprintf("**Implementation**: %s\n", task.Implementation))
	}

	return sb.String()
}

// formatCancellationComment formats a cancellation notification comment
func (n *Notifier) formatCancellationComment(task *storage.Task, reason string) string {
	template := n.getTemplate("cancellation")
	if template != "default" {
		// TODO: Load and parse custom template
		return template
	}

	// Default template
	var sb strings.Builder
	sb.WriteString("🚫 **Task Cancelled**\n\n")

	if reason != "" {
		sb.WriteString(fmt.Sprintf("**Reason**: %s\n", reason))
	}

	sb.WriteString(fmt.Sprintf("**Task**: %s\n", task.Title))

	return sb.String()
}

// formatPendingComment formats a pending notification comment
func (n *Notifier) formatPendingComment(task *storage.Task, reason string) string {
	template := n.getTemplate("pending")
	if template != "default" {
		// TODO: Load and parse custom template
		return template
	}

	// Default template
	var sb strings.Builder
	sb.WriteString("⏳ **Task Pending**\n\n")

	if task.ReviewerLogin != "" {
		sb.WriteString(fmt.Sprintf("@%s ", task.ReviewerLogin))
	}

	if reason != "" {
		sb.WriteString(fmt.Sprintf("**Reason**: %s\n\n", reason))
	}

	sb.WriteString(fmt.Sprintf("**Task**: %s\n", task.Title))

	return sb.String()
}

// formatExclusionComment formats an exclusion notification comment
func (n *Notifier) formatExclusionComment(reason *ExclusionReason) string {
	template := n.getTemplate("exclusion")
	if template != "default" {
		// TODO: Load and parse custom template
		return template
	}

	// Default template
	var sb strings.Builder
	sb.WriteString("ℹ️ **This comment was not converted to a task**\n\n")

	sb.WriteString(fmt.Sprintf("**Reason**: %s\n", reason.Type))

	if reason.Explanation != "" {
		sb.WriteString(fmt.Sprintf("%s\n", reason.Explanation))
	}

	if len(reason.References) > 0 {
		sb.WriteString("\n**References**:\n")
		for _, ref := range reason.References {
			sb.WriteString(fmt.Sprintf("- %s\n", ref))
		}
	}

	return sb.String()
}

// getTemplate retrieves the template configuration
func (n *Notifier) getTemplate(templateType string) string {
	var template string
	switch templateType {
	case "completion":
		template = n.config.CommentSettings.Templates.Completion
	case "cancellation":
		template = n.config.CommentSettings.Templates.Cancellation
	case "pending":
		template = n.config.CommentSettings.Templates.Pending
	case "exclusion":
		template = n.config.CommentSettings.Templates.Exclusion
	default:
		return "default"
	}

	// Return default if template is empty
	if template == "" {
		return "default"
	}
	return template
}

// ProcessBatchedComments processes any batched comments that are ready to be sent
func (n *Notifier) ProcessBatchedComments(ctx context.Context) error {
	if !n.config.CommentSettings.Throttling.BatchSimilarComments {
		return nil
	}

	batches := n.throttler.GetReadyBatches()
	for _, batch := range batches {
		comment := n.formatBatchComment(batch)
		err := n.githubClient.CreateIssueComment(ctx, batch.PR, comment)
		if err != nil {
			return fmt.Errorf("failed to post batched comment: %w", err)
		}
		n.throttler.ClearBatch(batch.ID)
	}

	return nil
}

// formatBatchComment formats multiple notifications into a single comment
func (n *Notifier) formatBatchComment(batch *CommentBatch) string {
	var sb strings.Builder
	sb.WriteString("📋 **Batched Notifications**\n\n")
	sb.WriteString(fmt.Sprintf("The following %d updates have been batched to reduce notification noise:\n\n", len(batch.Comments)))

	for i, comment := range batch.Comments {
		sb.WriteString(fmt.Sprintf("---\n\n%s\n", comment.Content))
		if i < len(batch.Comments)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString(fmt.Sprintf("\n---\n*Batched at %s*", time.Now().Format(time.RFC3339)))

	return sb.String()
}
