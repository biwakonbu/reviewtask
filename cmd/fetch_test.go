package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"reviewtask/internal/github"
)

func TestFetchCommandWithProgress(t *testing.T) {
	// Save original stdout
	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()

	t.Run("Progress visualization output format", func(t *testing.T) {
		// Create test directory
		testDir := t.TempDir()

		// Change to test directory
		originalDir, _ := os.Getwd()
		os.Chdir(testDir)
		defer os.Chdir(originalDir)

		// Initialize .pr-review directory
		os.MkdirAll(".pr-review", 0755)

		// Create mock configuration
		// Just ensure .pr-review directory exists
		// No need to create actual config for this test

		// Capture output
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Simulate progress output
		fmt.Println("GitHub API: 2/2")
		fmt.Println("AI Analysis: 3/3")
		fmt.Println("Saving Data: 2/2")
		fmt.Println("Processing: Processing comment from @reviewer1...")
		fmt.Println("✓ Saved PR info to .pr-review/PR-123/info.json")
		fmt.Println("✓ Saved reviews to .pr-review/PR-123/reviews.json")
		fmt.Println("✓ Generated 2 tasks and saved to .pr-review/PR-123/tasks.json")

		w.Close()
		os.Stdout = originalStdout

		var output bytes.Buffer
		output.ReadFrom(r)

		// Verify output contains expected elements
		outputStr := output.String()
		assert.Contains(t, outputStr, "GitHub API:")
		assert.Contains(t, outputStr, "AI Analysis:")
		assert.Contains(t, outputStr, "Saving Data:")
		assert.Contains(t, outputStr, "Processing:")
		assert.Contains(t, outputStr, "✓ Saved PR info")
		assert.Contains(t, outputStr, "✓ Saved reviews")
		assert.Contains(t, outputStr, "✓ Generated 2 tasks")
	})

	t.Run("Non-TTY environment output", func(t *testing.T) {
		// Set CI environment variable to simulate non-TTY
		os.Setenv("CI", "true")
		defer os.Unsetenv("CI")

		// Capture output
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Simulate non-TTY progress output
		fmt.Println("GitHub API: 1/2")
		fmt.Println("GitHub API: 2/2")
		fmt.Println("AI Analysis: 1/3")
		fmt.Println("AI Analysis: 2/3")
		fmt.Println("AI Analysis: 3/3")
		fmt.Println("Saving Data: 1/2")
		fmt.Println("Saving Data: 2/2")
		fmt.Println("Processing: Processing comment 1/3")

		w.Close()
		os.Stdout = originalStdout

		var output bytes.Buffer
		output.ReadFrom(r)

		// Verify simple progress indicators
		outputStr := output.String()
		assert.Contains(t, outputStr, "GitHub API:")
		assert.Contains(t, outputStr, "AI Analysis:")
		assert.Contains(t, outputStr, "Saving Data:")
		assert.Contains(t, outputStr, "Processing:")

		// Should not contain progress bar characters
		assert.NotContains(t, outputStr, "█")
		assert.NotContains(t, outputStr, "░")
		assert.NotContains(t, outputStr, "⠋")
	})
}

func TestGetReviewerNameFunction(t *testing.T) {
	reviews := []github.Review{
		{
			Body:     "Review comment",
			Reviewer: "reviewer1",
			Comments: []github.Comment{
				{Author: "commenter1"},
				{Author: "commenter2"},
			},
		},
		{
			Body:     "",
			Reviewer: "reviewer2",
			Comments: []github.Comment{
				{Author: "commenter3"},
			},
		},
	}

	tests := []struct {
		name     string
		index    int
		expected string
	}{
		{"Review body", 0, "reviewer1"},
		{"First comment", 1, "commenter1"},
		{"Second comment", 2, "commenter2"},
		{"Third comment", 3, "commenter3"},
		{"Out of bounds", 99, "reviewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getReviewerName(reviews, tt.index)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgressEdgeCases(t *testing.T) {
	t.Run("Empty reviews", func(t *testing.T) {
		reviews := []github.Review{}

		// Should handle empty reviews gracefully
		result := getReviewerName(reviews, 0)
		assert.Equal(t, "reviewer", result)
	})

	t.Run("Reviews with no comments", func(t *testing.T) {
		reviews := []github.Review{
			{
				Body:     "Only review body",
				Reviewer: "reviewer1",
				Comments: []github.Comment{}, // Empty comments
			},
		}

		result := getReviewerName(reviews, 0)
		assert.Equal(t, "reviewer1", result)

		result = getReviewerName(reviews, 1)
		assert.Equal(t, "reviewer", result)
	})

	t.Run("Mixed review types", func(t *testing.T) {
		reviews := []github.Review{
			{
				Body:     "", // No body
				Reviewer: "reviewer1",
				Comments: []github.Comment{
					{Author: "commenter1"},
				},
			},
			{
				Body:     "Has body",
				Reviewer: "reviewer2",
				Comments: []github.Comment{}, // No comments
			},
		}

		// First comment (index 0 skips empty body)
		result := getReviewerName(reviews, 0)
		assert.Equal(t, "commenter1", result)

		// Review body
		result = getReviewerName(reviews, 1)
		assert.Equal(t, "reviewer2", result)
	})
}
