package github

import (
	"context"
	"fmt"
	"time"
)

// ThreadResolutionTracker handles GitHub review thread resolution state tracking
type ThreadResolutionTracker struct {
	client *Client
}

// NewThreadResolutionTracker creates a new thread resolution tracker
func NewThreadResolutionTracker(client *Client) *ThreadResolutionTracker {
	return &ThreadResolutionTracker{
		client: client,
	}
}

// ReviewThreadStatus represents the resolution status of a review thread
type ReviewThreadStatus struct {
	CommentID            int64     `json:"comment_id"`
	GitHubThreadResolved bool      `json:"github_thread_resolved"`
	LastCheckedAt        time.Time `json:"last_checked_at"`
	InReplyToID          int64     `json:"in_reply_to_id,omitempty"` // For tracking thread relationships
}

// UpdateThreadResolutionStatus fetches and updates the thread resolution status from GitHub
func (tr *ThreadResolutionTracker) UpdateThreadResolutionStatus(ctx context.Context, prNumber int, comments []Comment) ([]ReviewThreadStatus, error) {
	var threadStatuses []ReviewThreadStatus

	// Get current thread resolution status from GitHub
	for _, comment := range comments {
		status, err := tr.getThreadStatusFromGitHub(ctx, prNumber, comment.ID)
		if err != nil {
			// Log error but continue with other comments
			continue
		}

		threadStatuses = append(threadStatuses, ReviewThreadStatus{
			CommentID:            comment.ID,
			GitHubThreadResolved: status.Resolved,
			LastCheckedAt:        time.Now(),
			InReplyToID:          status.InReplyToID,
		})
	}

	return threadStatuses, nil
}

// threadStatusFromGitHub represents the GitHub API response for thread status
type threadStatusFromGitHub struct {
	Resolved    bool  `json:"resolved"`
	InReplyToID int64 `json:"in_reply_to_id,omitempty"`
}

// getThreadStatusFromGitHub fetches thread resolution status from GitHub API
func (tr *ThreadResolutionTracker) getThreadStatusFromGitHub(ctx context.Context, prNumber int, commentID int64) (*threadStatusFromGitHub, error) {
	// Use GitHub's GraphQL API to get thread resolution status
	// This is a simplified implementation - in practice, you'd need to implement
	// the actual GraphQL query to fetch thread resolution status

	// For now, return a default status indicating not resolved
	// TODO: Implement actual GitHub GraphQL query
	return &threadStatusFromGitHub{
		Resolved:    false,
		InReplyToID: 0,
	}, nil
}

// DetectUnresolvedComments compares local comment state with GitHub state
func (tr *ThreadResolutionTracker) DetectUnresolvedComments(localComments []Comment, githubStatuses []ReviewThreadStatus) *UnresolvedCommentsReport {
	report := &UnresolvedCommentsReport{
		UnanalyzedComments: []Comment{},
		InProgressComments: []Comment{},
		ResolvedComments:   []Comment{},
	}

	// Create a map of GitHub statuses for quick lookup
	githubStatusMap := make(map[int64]*ReviewThreadStatus)
	for _, status := range githubStatuses {
		githubStatusMap[status.CommentID] = &status
	}

	for _, comment := range localComments {
		githubStatus, exists := githubStatusMap[comment.ID]

		if !exists {
			// Comment exists locally but no GitHub status - likely unanalyzed
			report.UnanalyzedComments = append(report.UnanalyzedComments, comment)
			continue
		}

		if githubStatus.GitHubThreadResolved {
			// Thread is resolved on GitHub
			report.ResolvedComments = append(report.ResolvedComments, comment)
		} else {
			// Thread exists on GitHub but not resolved
			if comment.TasksGenerated && !comment.AllTasksCompleted {
				// Tasks were generated but not all completed
				report.InProgressComments = append(report.InProgressComments, comment)
			} else if !comment.TasksGenerated {
				// No tasks generated yet
				report.UnanalyzedComments = append(report.UnanalyzedComments, comment)
			} else {
				// All tasks completed but thread not resolved
				report.InProgressComments = append(report.InProgressComments, comment)
			}
		}
	}

	return report
}

// UnresolvedCommentsReport contains the categorized comments
type UnresolvedCommentsReport struct {
	UnanalyzedComments []Comment `json:"unanalyzed_comments"`
	InProgressComments []Comment `json:"in_progress_comments"`
	ResolvedComments   []Comment `json:"resolved_comments"`
}

// IsComplete checks if all comments are properly resolved
func (r *UnresolvedCommentsReport) IsComplete() bool {
	return len(r.UnanalyzedComments) == 0 && len(r.InProgressComments) == 0
}

// GetSummary returns a summary of the resolution status
func (r *UnresolvedCommentsReport) GetSummary() string {
	if r.IsComplete() {
		return "✅ All comments analyzed and resolved"
	}

	summary := fmt.Sprintf("Unresolved Comments: %d", len(r.UnanalyzedComments)+len(r.InProgressComments))
	if len(r.UnanalyzedComments) > 0 {
		summary += fmt.Sprintf("\n  ❌ %d comments not yet analyzed", len(r.UnanalyzedComments))
	}
	if len(r.InProgressComments) > 0 {
		summary += fmt.Sprintf("\n  ⏳ %d comments with pending tasks", len(r.InProgressComments))
	}
	if len(r.ResolvedComments) > 0 {
		summary += fmt.Sprintf("\n  ✅ %d comments resolved", len(r.ResolvedComments))
	}

	return summary
}
