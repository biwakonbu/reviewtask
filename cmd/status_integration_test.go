package cmd

import (
	"context"
	"os"
	"testing"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnresolvedCommentsIntegration tests the complete unresolved comments detection workflow
func TestUnresolvedCommentsIntegration(t *testing.T) {
	// Skip if no GitHub token available (integration test)
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping integration test - no GITHUB_TOKEN available")
	}

	ctx := context.Background()

	// Initialize GitHub client
	githubClient, err := github.NewClient()
	require.NoError(t, err)

	// Initialize storage manager
	storageManager := storage.NewManager()

	// Test with current branch PR
	prNumber, err := githubClient.GetCurrentBranchPR(ctx)
	if err != nil {
		t.Skip("No PR found for current branch, skipping integration test")
	}

	t.Run("FetchAndCompareComments", func(t *testing.T) {
		// Load existing reviews
		reviews, err := storageManager.LoadReviews(prNumber)
		if err != nil {
			t.Skip("No reviews found, skipping test")
		}

		// Flatten comments for testing
		var allComments []github.Comment
		for _, review := range reviews {
			allComments = append(allComments, review.Comments...)
		}

		if len(allComments) == 0 {
			t.Skip("No comments found, skipping test")
		}

		// Test comment manager
		commentManager := github.NewCommentManager(githubClient)
		comparison, err := commentManager.FetchAndCompareComments(ctx, prNumber, allComments)
		require.NoError(t, err)

		// Verify comparison result structure
		assert.NotNil(t, comparison)
		assert.NotNil(t, comparison.LocalComments)
		assert.NotNil(t, comparison.GitHubComments)
		assert.NotNil(t, comparison.ThreadStatuses)
		assert.NotNil(t, comparison.UnresolvedReport)
	})

	t.Run("GetUnresolvedCommentsReport", func(t *testing.T) {
		commentManager := github.NewCommentManager(githubClient)
		report, err := commentManager.GetUnresolvedCommentsReport(ctx, prNumber)
		require.NoError(t, err)

		// Verify report structure
		assert.NotNil(t, report)
		assert.NotNil(t, report.UnanalyzedComments)
		assert.NotNil(t, report.InProgressComments)
		assert.NotNil(t, report.ResolvedComments)

		// Test report methods
		summary := report.GetSummary()
		assert.NotEmpty(t, summary)
	})

	t.Run("StatusCommandWithUnresolvedComments", func(t *testing.T) {
		// This tests the complete status command workflow
		// We'll create a temporary scenario for testing

		// Create test tasks
		testTasks := []storage.Task{
			{ID: "test-1", Status: "done", PRNumber: prNumber},
			{ID: "test-2", Status: "todo", PRNumber: prNumber},
		}

		// Save test tasks
		err := storageManager.SaveTasks(prNumber, testTasks)
		require.NoError(t, err)

		// Get unresolved comments report
		commentManager := github.NewCommentManager(githubClient)
		unresolvedReport, err := commentManager.GetUnresolvedCommentsReport(ctx, prNumber)
		require.NoError(t, err)

		// Test completion detection
		completionResult := DetectCompletionState(testTasks, unresolvedReport, prNumber)
		assert.NotNil(t, completionResult)

		// Verify completion detection logic
		assert.Equal(t, 50.0, completionResult.CompletionPercentage) // 1/2 tasks completed
		assert.Contains(t, completionResult.CompletionSummary, "pending tasks")
	})
}

// TestThreadResolutionIntegration tests thread resolution state tracking
func TestThreadResolutionIntegration(t *testing.T) {
	// Skip if no GitHub token available
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping integration test - no GITHUB_TOKEN available")
	}

	ctx := context.Background()

	// Initialize GitHub client
	githubClient, err := github.NewClient()
	require.NoError(t, err)

	// Get current branch PR
	prNumber, err := githubClient.GetCurrentBranchPR(ctx)
	if err != nil {
		t.Skip("No PR found for current branch, skipping integration test")
	}

	t.Run("ThreadResolutionTracker", func(t *testing.T) {
		// Load existing reviews to get comments for testing
		storageManager := storage.NewManager()
		reviews, err := storageManager.LoadReviews(prNumber)
		if err != nil {
			t.Skip("No reviews found, skipping test")
		}

		// Flatten comments
		var allComments []github.Comment
		for _, review := range reviews {
			allComments = append(allComments, review.Comments...)
		}

		if len(allComments) == 0 {
			t.Skip("No comments found, skipping test")
		}

		// Test thread resolution tracker
		tracker := github.NewThreadResolutionTracker(githubClient)
		threadStatuses, err := tracker.UpdateThreadResolutionStatus(ctx, prNumber, allComments)
		require.NoError(t, err)

		// Verify thread statuses
		assert.NotNil(t, threadStatuses)
		assert.Len(t, threadStatuses, len(allComments))

		for _, status := range threadStatuses {
			assert.NotZero(t, status.CommentID)
			assert.NotZero(t, status.LastCheckedAt)
		}
	})

	t.Run("CommentStateComparison", func(t *testing.T) {
		// Load existing reviews
		storageManager := storage.NewManager()
		reviews, err := storageManager.LoadReviews(prNumber)
		if err != nil {
			t.Skip("No reviews found, skipping test")
		}

		var allComments []github.Comment
		for _, review := range reviews {
			allComments = append(allComments, review.Comments...)
		}

		if len(allComments) == 0 {
			t.Skip("No comments found, skipping test")
		}

		// Get current GitHub state
		currentReviews, err := githubClient.GetPRReviews(ctx, prNumber)
		require.NoError(t, err)

		var githubComments []github.Comment
		for _, review := range currentReviews {
			githubComments = append(githubComments, review.Comments...)
		}

		// Test comment state comparison
		tracker := github.NewThreadResolutionTracker(githubClient)
		threadStatuses, err := tracker.UpdateThreadResolutionStatus(ctx, prNumber, githubComments)
		require.NoError(t, err)

		report := tracker.DetectUnresolvedComments(githubComments, threadStatuses)
		assert.NotNil(t, report)

		// Verify report categories
		assert.NotNil(t, report.UnanalyzedComments)
		assert.NotNil(t, report.InProgressComments)
		assert.NotNil(t, report.ResolvedComments)
	})
}

// TestStatusCommandIntegrationWithGitHubAPI tests the status command with real GitHub API data
func TestStatusCommandIntegrationWithGitHubAPI(t *testing.T) {
	// Skip if no GitHub token available
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping integration test - no GITHUB_TOKEN available")
	}

	ctx := context.Background()

	// Initialize clients
	githubClient, err := github.NewClient()
	require.NoError(t, err)

	storageManager := storage.NewManager()

	// Get current branch PR
	prNumber, err := githubClient.GetCurrentBranchPR(ctx)
	if err != nil {
		t.Skip("No PR found for current branch, skipping integration test")
	}

	t.Run("CompleteStatusWorkflow", func(t *testing.T) {
		// Load existing tasks
		tasks, err := storageManager.GetTasksByPR(prNumber)
		if err != nil {
			t.Skip("No tasks found, skipping test")
		}

		// Get unresolved comments report
		commentManager := github.NewCommentManager(githubClient)
		unresolvedReport, err := commentManager.GetUnresolvedCommentsReport(ctx, prNumber)
		require.NoError(t, err)

		// Test completion detection with real data
		completionResult := DetectCompletionState(tasks, unresolvedReport, prNumber)
		assert.NotNil(t, completionResult)

		// Verify completion result structure
		assert.NotEmpty(t, completionResult.CompletionSummary)
		assert.GreaterOrEqual(t, completionResult.CompletionPercentage, 0.0)
		assert.LessOrEqual(t, completionResult.CompletionPercentage, 100.0)

		// Verify unresolved items lists
		assert.NotNil(t, completionResult.UnresolvedTasks)
		assert.NotNil(t, completionResult.UnresolvedComments)
	})
}
