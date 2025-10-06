package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"reviewtask/internal/github"
)

// ExtendedReviewsFile represents reviews.json with extended comment tracking fields
type ExtendedReviewsFile struct {
	GeneratedAt string          `json:"generated_at"`
	Reviews     []github.Review `json:"reviews"`
	LastSyncAt  string          `json:"last_sync_at,omitempty"`
	SyncVersion string          `json:"sync_version,omitempty"`
}

// ExtendedComment extends github.Comment with additional tracking fields
type ExtendedComment struct {
	github.Comment
	// Additional tracking fields for unresolved comment detection
	LocalTasksGenerated    bool      `json:"local_tasks_generated"`
	LocalAllTasksCompleted bool      `json:"local_all_tasks_completed"`
	LastLocalCheckAt       time.Time `json:"last_local_check_at"`
	// GitHub thread resolution tracking
	ThreadResolutionStatus *ThreadResolutionStatus `json:"thread_resolution_status,omitempty"`
}

// ThreadResolutionStatus tracks GitHub thread resolution state
type ThreadResolutionStatus struct {
	Resolved            bool      `json:"resolved"`
	LastCheckedAt       time.Time `json:"last_checked_at"`
	InReplyToID         int64     `json:"in_reply_to_id,omitempty"`
	ResolutionCommentID int64     `json:"resolution_comment_id,omitempty"`
}

// SaveExtendedReviews saves reviews with extended comment tracking
func (m *Manager) SaveExtendedReviews(prNumber int, reviews []github.Review) error {
	prDir := m.getPRDir(prNumber)
	if err := m.ensureDir(prDir); err != nil {
		return err
	}

	// Ensure reviews is never nil to avoid null in JSON
	if reviews == nil {
		reviews = []github.Review{}
	}

	// Ensure each review's Comments slice is never nil
	for i := range reviews {
		if reviews[i].Comments == nil {
			reviews[i].Comments = []github.Comment{}
		}
	}

	extendedFile := ExtendedReviewsFile{
		GeneratedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		Reviews:     reviews,
		LastSyncAt:  time.Now().Format("2006-01-02T15:04:05Z"),
		SyncVersion: "1.0",
	}

	filePath := filepath.Join(prDir, "reviews.json")
	data, err := json.MarshalIndent(extendedFile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// LoadExtendedReviews loads reviews with extended comment tracking
func (m *Manager) LoadExtendedReviews(prNumber int) (*ExtendedReviewsFile, error) {
	prDir := m.getPRDir(prNumber)
	filePath := filepath.Join(prDir, "reviews.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var extendedFile ExtendedReviewsFile
	if err := json.Unmarshal(data, &extendedFile); err != nil {
		return nil, err
	}

	return &extendedFile, nil
}

// UpdateCommentThreadStatus updates the thread resolution status for a comment
func (m *Manager) UpdateCommentThreadStatus(prNumber int, commentID int64, status *ThreadResolutionStatus) error {
	extendedFile, err := m.LoadExtendedReviews(prNumber)
	if err != nil {
		return fmt.Errorf("failed to load extended reviews: %w", err)
	}

	// Find and update the comment
	commentUpdated := false
	for i := range extendedFile.Reviews {
		for j := range extendedFile.Reviews[i].Comments {
			if extendedFile.Reviews[i].Comments[j].ID == commentID {
				// Convert to ExtendedComment for additional fields
				extendedComment := ExtendedComment{
					Comment: extendedFile.Reviews[i].Comments[j],
				}

				// Update thread resolution status
				extendedComment.ThreadResolutionStatus = status
				extendedComment.Comment.LastCheckedAt = status.LastCheckedAt.Format("2006-01-02T15:04:05Z")

				// Convert back to github.Comment for storage
				extendedFile.Reviews[i].Comments[j] = extendedComment.Comment
				commentUpdated = true
				break
			}
		}
		if commentUpdated {
			break
		}
	}

	if !commentUpdated {
		return fmt.Errorf("comment %d not found in PR %d", commentID, prNumber)
	}

	// Save updated file
	extendedFile.LastSyncAt = time.Now().Format("2006-01-02T15:04:05Z")
	return m.SaveExtendedReviews(prNumber, extendedFile.Reviews)
}

// GetCommentsWithThreadStatus returns all comments with their thread resolution status
func (m *Manager) GetCommentsWithThreadStatus(prNumber int) ([]ExtendedComment, error) {
	extendedFile, err := m.LoadExtendedReviews(prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to load extended reviews: %w", err)
	}

	var extendedComments []ExtendedComment
	for _, review := range extendedFile.Reviews {
		for _, comment := range review.Comments {
			extendedComment := ExtendedComment{
				Comment: comment,
			}

			// Add thread resolution status if available
			if comment.LastCheckedAt != "" {
				lastChecked, _ := time.Parse("2006-01-02T15:04:05Z", comment.LastCheckedAt)
				extendedComment.LastLocalCheckAt = lastChecked
				extendedComment.LocalTasksGenerated = comment.TasksGenerated
				extendedComment.LocalAllTasksCompleted = comment.AllTasksCompleted
			}

			extendedComments = append(extendedComments, extendedComment)
		}
	}

	return extendedComments, nil
}

// MarkCommentAsAnalyzed marks a comment as having had tasks generated
func (m *Manager) MarkCommentAsAnalyzed(prNumber int, commentID int64) error {
	extendedFile, err := m.LoadExtendedReviews(prNumber)
	if err != nil {
		return fmt.Errorf("failed to load extended reviews: %w", err)
	}

	// Find and update the comment
	commentUpdated := false
	for i := range extendedFile.Reviews {
		for j := range extendedFile.Reviews[i].Comments {
			if extendedFile.Reviews[i].Comments[j].ID == commentID {
				extendedFile.Reviews[i].Comments[j].TasksGenerated = true
				extendedFile.Reviews[i].Comments[j].LastCheckedAt = time.Now().Format("2006-01-02T15:04:05Z")
				commentUpdated = true
				break
			}
		}
		if commentUpdated {
			break
		}
	}

	if !commentUpdated {
		return fmt.Errorf("comment %d not found in PR %d", commentID, prNumber)
	}

	// Save updated file
	extendedFile.LastSyncAt = time.Now().Format("2006-01-02T15:04:05Z")
	return m.SaveExtendedReviews(prNumber, extendedFile.Reviews)
}

// MarkCommentTasksAsCompleted marks all tasks for a comment as completed
func (m *Manager) MarkCommentTasksAsCompleted(prNumber int, commentID int64) error {
	extendedFile, err := m.LoadExtendedReviews(prNumber)
	if err != nil {
		return fmt.Errorf("failed to load extended reviews: %w", err)
	}

	// Find and update the comment
	commentUpdated := false
	for i := range extendedFile.Reviews {
		for j := range extendedFile.Reviews[i].Comments {
			if extendedFile.Reviews[i].Comments[j].ID == commentID {
				extendedFile.Reviews[i].Comments[j].AllTasksCompleted = true
				extendedFile.Reviews[i].Comments[j].LastCheckedAt = time.Now().Format("2006-01-02T15:04:05Z")
				commentUpdated = true
				break
			}
		}
		if commentUpdated {
			break
		}
	}

	if !commentUpdated {
		return fmt.Errorf("comment %d not found in PR %d", commentID, prNumber)
	}

	// Save updated file
	extendedFile.LastSyncAt = time.Now().Format("2006-01-02T15:04:05Z")
	return m.SaveExtendedReviews(prNumber, extendedFile.Reviews)
}

// GetUnresolvedCommentsReport generates a report of unresolved comments for a PR
func (m *Manager) GetUnresolvedCommentsReport(prNumber int) (*github.UnresolvedCommentsReport, error) {
	comments, err := m.GetCommentsWithThreadStatus(prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments with thread status: %w", err)
	}

	// Convert to simple Comment slice for the report
	var simpleComments []github.Comment
	var threadStatuses []github.ReviewThreadStatus

	for _, extendedComment := range comments {
		simpleComments = append(simpleComments, extendedComment.Comment)

		// Create thread status from extended comment
		var resolved bool
		if extendedComment.ThreadResolutionStatus != nil {
			resolved = extendedComment.ThreadResolutionStatus.Resolved
		} else {
			resolved = extendedComment.Comment.GitHubThreadResolved
		}

		threadStatus := github.ReviewThreadStatus{
			CommentID:            extendedComment.ID,
			GitHubThreadResolved: resolved,
			LastCheckedAt:        extendedComment.LastLocalCheckAt,
		}

		threadStatuses = append(threadStatuses, threadStatus)
	}

	// Create tracker to generate report
	tracker := github.NewThreadResolutionTracker(nil) // We'll need to pass a real client later
	report := tracker.DetectUnresolvedComments(simpleComments, threadStatuses)

	return report, nil
}
