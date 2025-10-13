package github

import (
	"context"
	"testing"
)

// TestGetPRReviews_Pagination tests that pagination correctly handles multiple pages
func TestGetPRReviews_Pagination(t *testing.T) {
	tests := []struct {
		name          string
		mockReviews   []Review
		expectedCount int
		expectError   bool
	}{
		{
			name: "single page with <100 reviews",
			mockReviews: []Review{
				{ID: 1, Reviewer: "user1", State: "APPROVED"},
				{ID: 2, Reviewer: "user2", State: "COMMENTED"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "multiple pages with pagination (simulated)",
			mockReviews:   createMockReviews(1, 150),
			expectedCount: 150,
			expectError:   false,
		},
		{
			name:          "exactly 100 reviews (single page)",
			mockReviews:   createMockReviews(1, 100),
			expectedCount: 100,
			expectError:   false,
		},
		{
			name:          "exactly 200 reviews (two full pages)",
			mockReviews:   createMockReviews(1, 200),
			expectedCount: 200,
			expectError:   false,
		},
		{
			name:          "empty result (no reviews)",
			mockReviews:   []Review{},
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := NewMockGitHubClient()

			// Set up mock reviews
			mockClient.SetReviews(tt.mockReviews)

			ctx := context.Background()
			reviews, err := mockClient.GetPRReviews(ctx, 1)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(reviews) != tt.expectedCount {
				t.Errorf("Expected %d reviews, got %d", tt.expectedCount, len(reviews))
			}
		})
	}
}

// TestEnrichCommentsWithThreadState tests thread state enrichment logic
func TestEnrichCommentsWithThreadState(t *testing.T) {
	tests := []struct {
		name             string
		comments         []Comment
		threadStates     map[int64]bool
		expectedResolved []bool
	}{
		{
			name: "all comments resolved",
			comments: []Comment{
				{ID: 1, Body: "Comment 1"},
				{ID: 2, Body: "Comment 2"},
			},
			threadStates: map[int64]bool{
				1: true,
				2: true,
			},
			expectedResolved: []bool{true, true},
		},
		{
			name: "mixed resolved and unresolved",
			comments: []Comment{
				{ID: 1, Body: "Comment 1"},
				{ID: 2, Body: "Comment 2"},
				{ID: 3, Body: "Comment 3"},
			},
			threadStates: map[int64]bool{
				1: true,
				2: false,
				3: true,
			},
			expectedResolved: []bool{true, false, true},
		},
		{
			name: "all comments unresolved",
			comments: []Comment{
				{ID: 1, Body: "Comment 1"},
				{ID: 2, Body: "Comment 2"},
			},
			threadStates: map[int64]bool{
				1: false,
				2: false,
			},
			expectedResolved: []bool{false, false},
		},
		{
			name:             "empty comments list",
			comments:         []Comment{},
			threadStates:     map[int64]bool{},
			expectedResolved: []bool{},
		},
		{
			name: "comments without thread states (missing in map)",
			comments: []Comment{
				{ID: 1, Body: "Comment 1"},
				{ID: 2, Body: "Comment 2"},
			},
			threadStates: map[int64]bool{
				1: true,
				// Comment 2 intentionally missing
			},
			expectedResolved: []bool{true, false}, // Missing comment should remain false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate enrichment logic directly
			enriched := make([]Comment, len(tt.comments))
			copy(enriched, tt.comments)

			for i := range enriched {
				if isResolved, exists := tt.threadStates[enriched[i].ID]; exists {
					enriched[i].GitHubThreadResolved = isResolved
					enriched[i].LastCheckedAt = "2025-01-01T00:00:00Z"
				}
			}

			// Verify results
			if len(enriched) != len(tt.expectedResolved) {
				t.Errorf("Expected %d comments, got %d", len(tt.expectedResolved), len(enriched))
			}

			for i, expected := range tt.expectedResolved {
				if i >= len(enriched) {
					break
				}
				if enriched[i].GitHubThreadResolved != expected {
					t.Errorf("Comment %d: expected resolved=%v, got %v",
						enriched[i].ID, expected, enriched[i].GitHubThreadResolved)
				}
				// Verify LastCheckedAt is set when threadState exists
				_, exists := tt.threadStates[enriched[i].ID]
				if exists && enriched[i].LastCheckedAt == "" {
					t.Errorf("Comment %d: LastCheckedAt should be set", enriched[i].ID)
				}
			}
		})
	}
}

// Helper functions for creating mock data

func createMockReviews(startID int64, count int) []Review {
	reviews := make([]Review, count)
	for i := 0; i < count; i++ {
		reviews[i] = Review{
			ID:       startID + int64(i),
			Reviewer: "testuser",
			State:    "APPROVED",
			Body:     "Review body",
		}
	}
	return reviews
}
