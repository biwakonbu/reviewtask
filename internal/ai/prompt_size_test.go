package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/github"
)

func TestPromptSizeTracker(t *testing.T) {
	t.Run("Basic tracking", func(t *testing.T) {
		tracker := NewPromptSizeTracker()

		// Track different components
		tracker.TrackSystemPrompt("System prompt text")
		tracker.TrackLanguagePrompt("Language instruction")
		tracker.TrackPriorityPrompt("Priority rules")
		tracker.TrackNitpickPrompt("Nitpick handling")

		assert.Equal(t, len("System prompt text"), tracker.SystemPrompt)
		assert.Equal(t, len("Language instruction"), tracker.LanguagePrompt)
		assert.Equal(t, len("Priority rules"), tracker.PriorityPrompt)
		assert.Equal(t, len("Nitpick handling"), tracker.NitpickPrompt)

		expectedTotal := len("System prompt text") + len("Language instruction") +
			len("Priority rules") + len("Nitpick handling")
		assert.Equal(t, expectedTotal, tracker.TotalSize)
	})

	t.Run("Size exceeded detection", func(t *testing.T) {
		tracker := NewPromptSizeTracker()
		tracker.Limit = 100 // Set small limit for testing

		// Add content that exceeds limit
		largeContent := strings.Repeat("a", 101)
		tracker.TrackSystemPrompt(largeContent)

		assert.True(t, tracker.IsExceeded())
		assert.Equal(t, 101, tracker.TotalSize)
	})

	t.Run("Largest component identification", func(t *testing.T) {
		tracker := NewPromptSizeTracker()

		tracker.TrackSystemPrompt("Small")
		tracker.TrackLanguagePrompt("Medium content")
		tracker.TrackPriorityPrompt("The largest content of all components")
		tracker.TrackNitpickPrompt("Tiny")

		name, size := tracker.GetLargestComponent()
		assert.Equal(t, "Priority rules", name)
		assert.Equal(t, len("The largest content of all components"), size)
	})

	t.Run("Review data tracking", func(t *testing.T) {
		tracker := NewPromptSizeTracker()

		reviews := []github.Review{
			{
				ID:       123,
				Reviewer: "reviewer1",
				Body:     "This is a review body",
				Comments: []github.Comment{
					{
						ID:     456,
						Author: "commenter1",
						Body:   "This is a comment",
						File:   "main.go",
						Line:   10,
						Replies: []github.Reply{
							{Author: "replier1", Body: "This is a reply"},
						},
					},
				},
			},
		}

		reviewData := "Mock review data representation"
		tracker.TrackReviewsData(reviewData, reviews)

		assert.Equal(t, len(reviewData), tracker.ReviewsData)
		assert.Greater(t, tracker.ReviewBodies, 0)
		assert.Greater(t, tracker.ReviewComments, 0)
		assert.Equal(t, 1, len(tracker.ReviewBreakdown))
		assert.Equal(t, 1, len(tracker.CommentBreakdown))
	})

	t.Run("Generate report", func(t *testing.T) {
		tracker := NewPromptSizeTracker()
		tracker.Limit = 1000

		tracker.TrackSystemPrompt("System")
		tracker.TrackLanguagePrompt("Language")
		tracker.TrackPriorityPrompt("Priority")
		tracker.TrackNitpickPrompt("Nitpick")
		tracker.TrackReviewsData("Review data content", []github.Review{})

		report := tracker.GenerateReport()

		// Check report contains key elements
		assert.Contains(t, report, "Prompt size breakdown:")
		assert.Contains(t, report, "System prompt:")
		assert.Contains(t, report, "Language settings:")
		assert.Contains(t, report, "Priority rules:")
		assert.Contains(t, report, "Nitpick rules:")
		assert.Contains(t, report, "Review data:")
		assert.Contains(t, report, "Total:")
		assert.Contains(t, report, "(limit: 1000 bytes)")
	})

	t.Run("Generate suggestions for review data", func(t *testing.T) {
		tracker := NewPromptSizeTracker()
		tracker.Limit = 100 // Small limit to trigger suggestions

		// Make review data the largest component
		tracker.TrackSystemPrompt("Small")
		tracker.TrackReviewsData(strings.Repeat("x", 200), []github.Review{
			{ID: 1, Comments: make([]github.Comment, 10)},
		})

		suggestions := tracker.GenerateSuggestions()

		assert.Contains(t, suggestions, "Suggestions:")
		assert.Contains(t, suggestions, "review data is too large")
		assert.Contains(t, suggestions, "Processing reviews in smaller batches")
		assert.Contains(t, suggestions, "This PR has 10 comments")
	})

	t.Run("Generate error message", func(t *testing.T) {
		tracker := NewPromptSizeTracker()
		tracker.Limit = 50

		tracker.TrackSystemPrompt(strings.Repeat("a", 60))

		errorMsg := tracker.GenerateErrorMessage()

		assert.Contains(t, errorMsg, "❌ Prompt size limit exceeded!")
		assert.Contains(t, errorMsg, "Prompt size breakdown:")
		assert.Contains(t, errorMsg, "Suggestions:")
	})

	t.Run("Largest comment tracking", func(t *testing.T) {
		tracker := NewPromptSizeTracker()

		reviews := []github.Review{
			{
				ID: 1,
				Comments: []github.Comment{
					{ID: 100, Body: "Small comment"},
					{ID: 200, Body: strings.Repeat("Large ", 100)},
					{ID: 300, Body: "Another small"},
				},
			},
		}

		tracker.TrackReviewsData("data", reviews)

		assert.Equal(t, int64(200), tracker.LargestComment.CommentID)
		assert.Greater(t, tracker.LargestComment.Size, 100)
	})
}

func TestPromptSizeTrackerEdgeCases(t *testing.T) {
	t.Run("Empty reviews", func(t *testing.T) {
		tracker := NewPromptSizeTracker()
		tracker.TrackReviewsData("", []github.Review{})

		assert.Equal(t, 0, tracker.ReviewsData)
		assert.Equal(t, 0, tracker.ReviewBodies)
		assert.Equal(t, 0, tracker.ReviewComments)
		assert.Equal(t, 0, len(tracker.ReviewBreakdown))
	})

	t.Run("No components tracked", func(t *testing.T) {
		tracker := NewPromptSizeTracker()

		assert.False(t, tracker.IsExceeded())
		assert.Equal(t, 0, tracker.TotalSize)

		name, size := tracker.GetLargestComponent()
		assert.NotEmpty(t, name) // Should return something even with all zeros
		assert.Equal(t, 0, size)
	})

	t.Run("Review with replies", func(t *testing.T) {
		tracker := NewPromptSizeTracker()

		reviews := []github.Review{
			{
				ID: 1,
				Comments: []github.Comment{
					{
						ID:   100,
						Body: "Comment",
						Replies: []github.Reply{
							{Author: "user1", Body: "Reply 1"},
							{Author: "user2", Body: "Reply 2"},
							{Author: "user3", Body: "Reply 3"},
						},
					},
				},
			},
		}

		tracker.TrackReviewsData("data", reviews)

		// Verify replies are included in comment size
		assert.Greater(t, tracker.CommentBreakdown[0].Size, len("Comment"))
	})
}
