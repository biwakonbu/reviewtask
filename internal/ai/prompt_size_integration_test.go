package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
)

func TestPromptSizeDebugOutput(t *testing.T) {
	t.Run("Large PR with debug mode", func(t *testing.T) {
		// Create config with debug mode enabled
		cfg := &config.Config{
			AISettings: config.AISettings{
				UserLanguage: "English",
				DebugMode:    true,
			},
		}

		// Create a large review to trigger size limit
		largeBody := strings.Repeat("This is a very long review comment. ", 1000)
		reviews := []github.Review{
			{
				ID:       123,
				Reviewer: "reviewer1",
				State:    "CHANGES_REQUESTED",
				Body:     largeBody,
				Comments: []github.Comment{
					{
						ID:     456,
						Author: "commenter1",
						Body:   strings.Repeat("Large comment content. ", 500),
						File:   "main.go",
						Line:   10,
					},
				},
			},
		}

		// Create mock client that doesn't matter for this test
		mockClient := &MockClaudeClient{}
		analyzer := NewAnalyzerWithClient(cfg, mockClient)

		// Build prompt to trigger size tracking
		prompt := analyzer.buildAnalysisPrompt(reviews)

		// Verify prompt size tracking was initialized
		assert.NotNil(t, analyzer.promptSizeTracker)
		assert.Greater(t, analyzer.promptSizeTracker.TotalSize, 0)
		assert.Greater(t, analyzer.promptSizeTracker.ReviewsData, 0)

		// Call callClaudeCode to test error message generation
		_, err := analyzer.callClaudeCode(prompt)

		// Should get an error with detailed breakdown in debug mode
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Prompt size limit exceeded!")
		assert.Contains(t, err.Error(), "Prompt size breakdown:")
		assert.Contains(t, err.Error(), "Suggestions:")
	})

	t.Run("Large PR without debug mode", func(t *testing.T) {
		// Create config with debug mode disabled
		cfg := &config.Config{
			AISettings: config.AISettings{
				UserLanguage: "English",
				DebugMode:    false,
			},
		}

		// Create a large review
		largeBody := strings.Repeat("This is a very long review comment. ", 1000)
		reviews := []github.Review{
			{
				ID:       123,
				Reviewer: "reviewer1",
				Body:     largeBody,
			},
		}

		mockClient := &MockClaudeClient{}
		analyzer := NewAnalyzerWithClient(cfg, mockClient)

		// Build prompt
		prompt := analyzer.buildAnalysisPrompt(reviews)

		// Call callClaudeCode
		_, err := analyzer.callClaudeCode(prompt)

		// Should get simplified error in non-debug mode
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum limit")
		assert.Contains(t, err.Error(), "Use --debug for detailed breakdown")
		assert.NotContains(t, err.Error(), "Prompt size breakdown:")
	})
}

func TestPromptSizeOptimizationSuggestions(t *testing.T) {
	t.Run("Suggestions for large review data", func(t *testing.T) {
		tracker := NewPromptSizeTracker()
		tracker.Limit = 1000 // Small limit

		// Make review data the largest
		tracker.TrackSystemPrompt("Small")
		tracker.TrackReviewsData(strings.Repeat("x", 2000), []github.Review{
			{
				ID: 1,
				Comments: []github.Comment{
					{ID: 100, Body: strings.Repeat("Large comment ", 500)},
					{ID: 200, Body: "Small"},
					{ID: 300, Body: "Small"},
				},
			},
		})

		suggestions := tracker.GenerateSuggestions()

		// Verify suggestions are appropriate
		assert.Contains(t, suggestions, "review data is too large")
		assert.Contains(t, suggestions, "Processing reviews in smaller batches")
		assert.Contains(t, suggestions, "This PR has 3 comments")
		assert.Contains(t, suggestions, "Use parallel processing")
	})

	t.Run("Identify largest comment", func(t *testing.T) {
		tracker := NewPromptSizeTracker()

		reviews := []github.Review{
			{
				ID: 1,
				Comments: []github.Comment{
					{ID: 100, Body: "Small comment"},
					{ID: 200, Body: strings.Repeat("Very large comment content ", 200), Author: "user1", File: "large.go", Line: 42},
					{ID: 300, Body: "Another small comment"},
				},
			},
		}

		tracker.TrackReviewsData("data", reviews)

		// Verify largest comment is tracked
		assert.Equal(t, int64(200), tracker.LargestComment.CommentID)
		assert.Equal(t, "user1", tracker.LargestComment.Author)
		assert.Equal(t, "large.go:42", tracker.LargestComment.FileInfo)
		assert.Greater(t, tracker.LargestComment.Size, 1000)

		// Check it appears in report
		report := tracker.GenerateReport()
		assert.Contains(t, report, "Largest comment: ID 200")
	})
}
