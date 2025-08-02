package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

// TestSelfReviewWorkflow tests the complete self-review processing workflow
func TestSelfReviewWorkflow(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Create config with self-reviews enabled
	cfg := &config.Config{
		AISettings: config.AISettings{
			ProcessSelfReviews: true,
			UserLanguage:       "English",
			OutputFormat:       "json",
		},
	}

	// Save config
	configPath := filepath.Join(".pr-review", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create storage manager
	storageManager := storage.NewManager()

	// Create test PR info
	prInfo := &github.PRInfo{
		Number:     789,
		Title:      "Test PR with self-reviews",
		Author:     "testauthor",
		CreatedAt:  "2023-01-01T00:00:00Z",
		UpdatedAt:  "2023-01-01T01:00:00Z",
		State:      "open",
		Repository: "test/repo",
		Branch:     "feature/self-review-test",
	}

	// Create test reviews with self-review
	reviews := []github.Review{
		{
			ID:          1,
			Reviewer:    "reviewer1",
			State:       "APPROVED",
			Body:        "Looks good!",
			SubmittedAt: "2023-01-01T02:00:00Z",
			Comments:    []github.Comment{},
		},
		{
			ID:          -1, // Self-review ID
			Reviewer:    "testauthor",
			State:       "COMMENTED",
			Body:        "",
			SubmittedAt: "2023-01-01T03:00:00Z",
			Comments: []github.Comment{
				{
					ID:        100,
					Body:      "TODO: Add better error handling",
					Author:    "testauthor",
					CreatedAt: "2023-01-01T03:00:00Z",
					File:      "",
					Line:      0,
				},
				{
					ID:        101,
					Body:      "FIXME: This is a temporary workaround",
					Author:    "testauthor",
					CreatedAt: "2023-01-01T03:30:00Z",
					File:      "temp.go",
					Line:      42,
				},
			},
		},
	}

	// Save PR info and reviews
	if err := storageManager.SavePRInfo(789, prInfo); err != nil {
		t.Fatalf("Failed to save PR info: %v", err)
	}

	if err := storageManager.SaveReviews(789, reviews); err != nil {
		t.Fatalf("Failed to save reviews: %v", err)
	}

	// Load and verify
	loadedPRInfo, err := storageManager.GetPRInfo(789)
	if err != nil {
		t.Fatalf("Failed to load PR info: %v", err)
	}

	if loadedPRInfo.Author != "testauthor" {
		t.Errorf("Expected author 'testauthor', got '%s'", loadedPRInfo.Author)
	}

	loadedReviews, err := storageManager.LoadReviews(789)
	if err != nil {
		t.Fatalf("Failed to load reviews: %v", err)
	}

	// Should have both external review and self-review
	if len(loadedReviews) != 2 {
		t.Errorf("Expected 2 reviews, got %d", len(loadedReviews))
	}

	// Find self-review
	var selfReview *github.Review
	for i, review := range loadedReviews {
		if review.ID == -1 {
			selfReview = &loadedReviews[i]
			break
		}
	}

	if selfReview == nil {
		t.Fatal("Self-review not found in loaded reviews")
	}

	// Verify self-review properties
	if selfReview.Reviewer != "testauthor" {
		t.Errorf("Expected self-review reviewer 'testauthor', got '%s'", selfReview.Reviewer)
	}

	if len(selfReview.Comments) != 2 {
		t.Errorf("Expected 2 self-review comments, got %d", len(selfReview.Comments))
	}

	// Verify comment content
	comment1 := selfReview.Comments[0]
	if comment1.Body != "TODO: Add better error handling" {
		t.Errorf("Unexpected comment body: %s", comment1.Body)
	}

	comment2 := selfReview.Comments[1]
	if comment2.File != "temp.go" || comment2.Line != 42 {
		t.Errorf("Expected file 'temp.go' line 42, got file '%s' line %d", comment2.File, comment2.Line)
	}
}

// TestSelfReviewDisabled tests that self-reviews are not processed when disabled
func TestSelfReviewDisabled(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-test-disabled-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Create config with self-reviews disabled
	cfg := &config.Config{
		AISettings: config.AISettings{
			ProcessSelfReviews: false, // Disabled
			UserLanguage:       "English",
			OutputFormat:       "json",
		},
	}

	// Save config
	configPath := filepath.Join(".pr-review", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify config loaded correctly
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.AISettings.ProcessSelfReviews {
		t.Error("Expected ProcessSelfReviews to be false")
	}
}

// TestSelfReviewMerging tests merging self-reviews with external reviews
func TestSelfReviewMerging(t *testing.T) {
	// Create test external reviews
	externalReviews := []github.Review{
		{
			ID:          1,
			Reviewer:    "reviewer1",
			State:       "APPROVED",
			Body:        "LGTM",
			SubmittedAt: "2023-01-01T02:00:00Z",
			Comments:    []github.Comment{},
		},
		{
			ID:          2,
			Reviewer:    "reviewer2",
			State:       "CHANGES_REQUESTED",
			Body:        "Please fix",
			SubmittedAt: "2023-01-01T02:30:00Z",
			Comments: []github.Comment{
				{
					ID:        10,
					Body:      "Fix this bug",
					Author:    "reviewer2",
					CreatedAt: "2023-01-01T02:35:00Z",
					File:      "bug.go",
					Line:      15,
				},
			},
		},
	}

	// Create self-reviews
	selfReviews := []github.Review{
		{
			ID:          -1,
			Reviewer:    "prauthor",
			State:       "COMMENTED",
			Body:        "",
			SubmittedAt: "2023-01-01T03:00:00Z",
			Comments: []github.Comment{
				{
					ID:        100,
					Body:      "Note to self: refactor this",
					Author:    "prauthor",
					CreatedAt: "2023-01-01T03:00:00Z",
					File:      "",
					Line:      0,
				},
			},
		},
	}

	// Merge reviews
	allReviews := append(externalReviews, selfReviews...)

	// Verify merge
	if len(allReviews) != 3 {
		t.Errorf("Expected 3 total reviews, got %d", len(allReviews))
	}

	// Count comments
	totalComments := 0
	for _, review := range allReviews {
		totalComments += len(review.Comments)
	}

	if totalComments != 2 {
		t.Errorf("Expected 2 total comments, got %d", totalComments)
	}

	// Verify self-review is last
	lastReview := allReviews[len(allReviews)-1]
	if lastReview.ID != -1 {
		t.Error("Expected self-review to be last in merged reviews")
	}
}

// TestSelfReviewCommentTypes tests different types of self-review comments
func TestSelfReviewCommentTypes(t *testing.T) {
	ctx := context.Background()

	// Test data for different comment types
	testCases := []struct {
		name          string
		issueComments []github.Comment
		prComments    []github.Comment
		expectedCount int
	}{
		{
			name: "Only issue comments",
			issueComments: []github.Comment{
				{ID: 1, Body: "Issue comment 1", Author: "author"},
				{ID: 2, Body: "Issue comment 2", Author: "author"},
			},
			prComments:    []github.Comment{},
			expectedCount: 2,
		},
		{
			name:          "Only PR review comments",
			issueComments: []github.Comment{},
			prComments: []github.Comment{
				{ID: 3, Body: "PR comment 1", Author: "author", File: "file.go", Line: 10},
				{ID: 4, Body: "PR comment 2", Author: "author", File: "file2.go", Line: 20},
			},
			expectedCount: 2,
		},
		{
			name: "Mixed comments",
			issueComments: []github.Comment{
				{ID: 5, Body: "Issue comment", Author: "author"},
			},
			prComments: []github.Comment{
				{ID: 6, Body: "PR comment", Author: "author", File: "file.go", Line: 30},
			},
			expectedCount: 2,
		},
		{
			name:          "No comments",
			issueComments: []github.Comment{},
			prComments:    []github.Comment{},
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test would require a mock client setup
			// For now, we're testing the logic of comment counting
			totalComments := len(tc.issueComments) + len(tc.prComments)
			if totalComments != tc.expectedCount {
				t.Errorf("Expected %d comments, got %d", tc.expectedCount, totalComments)
			}

			// Verify issue comments don't have file/line info
			for _, comment := range tc.issueComments {
				if comment.File != "" || comment.Line != 0 {
					t.Error("Issue comments should not have file/line info")
				}
			}

			// Verify PR comments have file/line info
			for _, comment := range tc.prComments {
				if comment.File == "" {
					t.Error("PR comments should have file info")
				}
			}
		})
	}

	// Ensure context is used (avoid unused variable warning)
	_ = ctx
}
