package github

import (
	"context"
	"fmt"
	"time"
)

// CommentManager handles comment operations and state tracking
type CommentManager struct {
	client  *Client
	tracker *ThreadResolutionTracker
}

// NewCommentManager creates a new comment manager
func NewCommentManager(client *Client) *CommentManager {
	return &CommentManager{
		client:  client,
		tracker: NewThreadResolutionTracker(client),
	}
}

// FetchAndCompareComments fetches comments from GitHub and compares with local state
func (cm *CommentManager) FetchAndCompareComments(ctx context.Context, prNumber int, localComments []Comment) (*CommentComparisonResult, error) {
	// Fetch current comments from GitHub
	currentReviews, err := cm.client.GetPRReviews(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR reviews: %w", err)
	}

	// Flatten all comments from reviews
	var githubComments []Comment
	for _, review := range currentReviews {
		githubComments = append(githubComments, review.Comments...)
	}

	// Update thread resolution status
	threadStatuses, err := cm.tracker.UpdateThreadResolutionStatus(ctx, prNumber, githubComments)
	if err != nil {
		return nil, fmt.Errorf("failed to update thread resolution status: %w", err)
	}

	// Compare local vs GitHub state
	comparison := cm.compareCommentStates(localComments, githubComments, threadStatuses)

	return comparison, nil
}

// CommentComparisonResult contains the results of comparing local vs GitHub comment state
type CommentComparisonResult struct {
	LocalComments    []Comment                 `json:"local_comments"`
	GitHubComments   []Comment                 `json:"github_comments"`
	ThreadStatuses   []ReviewThreadStatus      `json:"thread_statuses"`
	NewComments      []Comment                 `json:"new_comments"`
	ModifiedComments []Comment                 `json:"modified_comments"`
	DeletedComments  []Comment                 `json:"deleted_comments"`
	UnresolvedReport *UnresolvedCommentsReport `json:"unresolved_report"`
}

// compareCommentStates compares local comment state with GitHub state
func (cm *CommentManager) compareCommentStates(local, github []Comment, threadStatuses []ReviewThreadStatus) *CommentComparisonResult {
	result := &CommentComparisonResult{
		LocalComments:  local,
		GitHubComments: github,
		ThreadStatuses: threadStatuses,
	}

	// Create maps for quick lookup
	localMap := make(map[int64]*Comment)
	githubMap := make(map[int64]*Comment)
	threadStatusMap := make(map[int64]*ReviewThreadStatus)

	for i := range local {
		localMap[local[i].ID] = &local[i]
	}
	for i := range github {
		githubMap[github[i].ID] = &github[i]
	}
	for i := range threadStatuses {
		threadStatusMap[threadStatuses[i].CommentID] = &threadStatuses[i]
	}

	// Find new and modified comments
	for _, ghComment := range github {
		if localComment, exists := localMap[ghComment.ID]; exists {
			// Comment exists locally, check if modified
			if cm.isCommentModified(*localComment, ghComment) {
				result.ModifiedComments = append(result.ModifiedComments, ghComment)
			}
		} else {
			// New comment found on GitHub
			result.NewComments = append(result.NewComments, ghComment)
		}
	}

	// Find deleted comments (exist locally but not on GitHub)
	for _, localComment := range local {
		if _, exists := githubMap[localComment.ID]; !exists {
			result.DeletedComments = append(result.DeletedComments, localComment)
		}
	}

	// Generate unresolved comments report
	result.UnresolvedReport = cm.tracker.DetectUnresolvedComments(github, threadStatuses)

	return result
}

// isCommentModified checks if a comment has been modified
func (cm *CommentManager) isCommentModified(local, github Comment) bool {
	// Simple comparison - in practice, you might want more sophisticated comparison
	if local.Body != github.Body {
		return true
	}
	if local.GitHubThreadResolved != github.GitHubThreadResolved {
		return true
	}
	return false
}

// UpdateCommentStates updates local comment states based on GitHub comparison
func (cm *CommentManager) UpdateCommentStates(ctx context.Context, prNumber int, localComments []Comment) ([]Comment, error) {
	comparison, err := cm.FetchAndCompareComments(ctx, prNumber, localComments)
	if err != nil {
		return nil, err
	}

	// Merge GitHub state into local comments
	updatedComments := cm.mergeCommentStates(localComments, comparison.GitHubComments, comparison.ThreadStatuses)

	return updatedComments, nil
}

// mergeCommentStates merges GitHub state into local comments
func (cm *CommentManager) mergeCommentStates(local, github []Comment, threadStatuses []ReviewThreadStatus) []Comment {
	// Create maps for quick lookup
	githubMap := make(map[int64]*Comment)
	threadStatusMap := make(map[int64]*ReviewThreadStatus)

	for i := range github {
		githubMap[github[i].ID] = &github[i]
	}
	for i := range threadStatuses {
		threadStatusMap[threadStatuses[i].CommentID] = &threadStatuses[i]
	}

	// Update local comments with GitHub state
	var updatedComments []Comment
	for _, localComment := range local {
		if githubComment, exists := githubMap[localComment.ID]; exists {
			// Update with GitHub state
			updatedComment := *githubComment

			// Preserve local tracking fields if they exist
			if threadStatus, hasStatus := threadStatusMap[localComment.ID]; hasStatus {
				updatedComment.GitHubThreadResolved = threadStatus.GitHubThreadResolved
				updatedComment.LastCheckedAt = threadStatus.LastCheckedAt.Format("2006-01-02T15:04:05Z")
			}

			// Preserve local task generation and completion status
			updatedComment.TasksGenerated = localComment.TasksGenerated
			updatedComment.AllTasksCompleted = localComment.AllTasksCompleted

			updatedComments = append(updatedComments, updatedComment)
		} else {
			// Comment was deleted from GitHub, mark for cleanup
			localComment.LastCheckedAt = time.Now().Format("2006-01-02T15:04:05Z")
			updatedComments = append(updatedComments, localComment)
		}
	}

	// Add any new comments from GitHub
	for _, githubComment := range github {
		if _, exists := githubMap[githubComment.ID]; !exists {
			// This is a new comment
			newComment := githubComment
			newComment.LastCheckedAt = time.Now().Format("2006-01-02T15:04:05Z")
			newComment.TasksGenerated = false
			newComment.AllTasksCompleted = false
			updatedComments = append(updatedComments, newComment)
		}
	}

	return updatedComments
}

// GetUnresolvedCommentsReport generates a report of unresolved comments for a PR
func (cm *CommentManager) GetUnresolvedCommentsReport(ctx context.Context, prNumber int) (*UnresolvedCommentsReport, error) {
	// Fetch current comments from GitHub
	currentReviews, err := cm.client.GetPRReviews(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR reviews: %w", err)
	}

	// Flatten all comments
	var allComments []Comment
	for _, review := range currentReviews {
		allComments = append(allComments, review.Comments...)
	}

	// Update thread resolution status
	threadStatuses, err := cm.tracker.UpdateThreadResolutionStatus(ctx, prNumber, allComments)
	if err != nil {
		return nil, fmt.Errorf("failed to update thread resolution status: %w", err)
	}

	// Generate report
	report := cm.tracker.DetectUnresolvedComments(allComments, threadStatuses)

	return report, nil
}
