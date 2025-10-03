package github

import (
	"testing"
)

func TestGenerateReviewFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		review   Review
		expected ReviewFingerprint
	}{
		{
			name: "Review with body and comments",
			review: Review{
				Reviewer: "test-user",
				Body:     "This is a review",
				Comments: []Comment{
					{
						ID:   1,
						File: "main.go",
						Line: 10,
						Body: "Fix this",
					},
					{
						ID:   2,
						File: "test.go",
						Line: 20,
						Body: "Add test",
					},
				},
			},
			expected: ReviewFingerprint{
				Reviewer:   "test-user",
				CommentIDs: []int64{1, 2},
				// ContentHash will be deterministic based on content
			},
		},
		{
			name: "Review with only body",
			review: Review{
				Reviewer: "test-user",
				Body:     "General feedback",
				Comments: []Comment{},
			},
			expected: ReviewFingerprint{
				Reviewer:   "test-user",
				CommentIDs: []int64{},
			},
		},
		{
			name: "Review with embedded comments (no IDs)",
			review: Review{
				Reviewer: "codex-bot",
				Body:     "",
				Comments: []Comment{
					{
						ID:   0, // Embedded comments don't have IDs
						File: "main.go",
						Line: 5,
						Body: "Codex suggestion",
					},
				},
			},
			expected: ReviewFingerprint{
				Reviewer:   "codex-bot",
				CommentIDs: []int64{}, // No IDs for embedded comments
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := GenerateReviewFingerprint(tt.review)

			if fp.Reviewer != tt.expected.Reviewer {
				t.Errorf("Expected Reviewer %q, got %q", tt.expected.Reviewer, fp.Reviewer)
			}

			if len(fp.CommentIDs) != len(tt.expected.CommentIDs) {
				t.Errorf("Expected %d comment IDs, got %d", len(tt.expected.CommentIDs), len(fp.CommentIDs))
			}

			// Verify comment IDs match (order matters after sorting)
			for i, expectedID := range tt.expected.CommentIDs {
				if i < len(fp.CommentIDs) && fp.CommentIDs[i] != expectedID {
					t.Errorf("Expected comment ID %d at position %d, got %d", expectedID, i, fp.CommentIDs[i])
				}
			}

			// ContentHash should be non-empty
			if fp.ContentHash == "" {
				t.Error("Expected non-empty ContentHash")
			}
		})
	}
}

func TestGenerateReviewFingerprint_Consistency(t *testing.T) {
	// Same review content should produce the same fingerprint
	review := Review{
		Reviewer: "test-user",
		Body:     "Review body",
		Comments: []Comment{
			{
				ID:   1,
				File: "main.go",
				Line: 10,
				Body: "Comment 1",
			},
			{
				ID:   2,
				File: "test.go",
				Line: 20,
				Body: "Comment 2",
			},
		},
	}

	fp1 := GenerateReviewFingerprint(review)
	fp2 := GenerateReviewFingerprint(review)

	if fp1.ContentHash != fp2.ContentHash {
		t.Errorf("Expected consistent content hash, got %q and %q", fp1.ContentHash, fp2.ContentHash)
	}
}

func TestDeduplicateReviews(t *testing.T) {
	tests := []struct {
		name          string
		reviews       []Review
		expectedCount int
		description   string
	}{
		{
			name:          "No reviews",
			reviews:       []Review{},
			expectedCount: 0,
			description:   "Empty input should return empty output",
		},
		{
			name: "Single review",
			reviews: []Review{
				{
					ID:       1,
					Reviewer: "user1",
					Body:     "Review 1",
				},
			},
			expectedCount: 1,
			description:   "Single review should be returned as-is",
		},
		{
			name: "Two different reviews from different reviewers",
			reviews: []Review{
				{
					ID:       1,
					Reviewer: "user1",
					Body:     "Review 1",
				},
				{
					ID:       2,
					Reviewer: "user2",
					Body:     "Review 2",
				},
			},
			expectedCount: 2,
			description:   "Different reviewers should not be deduplicated",
		},
		{
			name: "Duplicate reviews from same reviewer",
			reviews: []Review{
				{
					ID:          1,
					Reviewer:    "codex-bot",
					Body:        "Same review content",
					SubmittedAt: "2025-10-04T10:00:00Z",
					Comments: []Comment{
						{
							ID:   10,
							File: "main.go",
							Line: 5,
							Body: "Fix this",
						},
					},
				},
				{
					ID:          2,
					Reviewer:    "codex-bot",
					Body:        "Same review content",
					SubmittedAt: "2025-10-04T10:05:00Z", // Submitted later
					Comments: []Comment{
						{
							ID:   10,
							File: "main.go",
							Line: 5,
							Body: "Fix this",
						},
					},
				},
			},
			expectedCount: 1,
			description:   "Duplicate content from same reviewer should be deduplicated",
		},
		{
			name: "Different reviews from same reviewer",
			reviews: []Review{
				{
					ID:          1,
					Reviewer:    "user1",
					Body:        "First review",
					SubmittedAt: "2025-10-04T10:00:00Z",
				},
				{
					ID:          2,
					Reviewer:    "user1",
					Body:        "Second review",
					SubmittedAt: "2025-10-04T11:00:00Z",
				},
			},
			expectedCount: 2,
			description:   "Different content from same reviewer should not be deduplicated",
		},
		{
			name: "Multiple reviewers with some duplicates",
			reviews: []Review{
				{
					ID:          1,
					Reviewer:    "user1",
					Body:        "Review A",
					SubmittedAt: "2025-10-04T10:00:00Z",
				},
				{
					ID:          2,
					Reviewer:    "user2",
					Body:        "Review B",
					SubmittedAt: "2025-10-04T10:05:00Z",
				},
				{
					ID:          3,
					Reviewer:    "user1",
					Body:        "Review A",
					SubmittedAt: "2025-10-04T10:10:00Z", // Duplicate
				},
				{
					ID:          4,
					Reviewer:    "user3",
					Body:        "Review C",
					SubmittedAt: "2025-10-04T10:15:00Z",
				},
			},
			expectedCount: 3,
			description:   "Should deduplicate user1's duplicate but keep others",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeduplicateReviews(tt.reviews)

			if len(result) != tt.expectedCount {
				t.Errorf("%s: Expected %d reviews, got %d", tt.description, tt.expectedCount, len(result))
			}

			// Verify no duplicate content hashes from the same reviewer
			seen := make(map[string]map[string]bool) // reviewer -> contentHash -> bool
			for _, review := range result {
				fp := GenerateReviewFingerprint(review)
				if seen[review.Reviewer] == nil {
					seen[review.Reviewer] = make(map[string]bool)
				}
				if seen[review.Reviewer][fp.ContentHash] {
					t.Errorf("Found duplicate content hash %q for reviewer %q", fp.ContentHash, review.Reviewer)
				}
				seen[review.Reviewer][fp.ContentHash] = true
			}
		})
	}
}

func TestDeduplicateReviews_PreservesOrder(t *testing.T) {
	reviews := []Review{
		{
			ID:          1,
			Reviewer:    "user1",
			Body:        "Review A",
			SubmittedAt: "2025-10-04T10:00:00Z",
		},
		{
			ID:          2,
			Reviewer:    "user2",
			Body:        "Review B",
			SubmittedAt: "2025-10-04T11:00:00Z",
		},
		{
			ID:          3,
			Reviewer:    "user3",
			Body:        "Review C",
			SubmittedAt: "2025-10-04T12:00:00Z",
		},
	}

	result := DeduplicateReviews(reviews)

	// Should preserve chronological order
	for i := 1; i < len(result); i++ {
		if result[i-1].SubmittedAt > result[i].SubmittedAt {
			t.Errorf("Reviews not in chronological order: %s > %s", result[i-1].SubmittedAt, result[i].SubmittedAt)
		}
	}
}

func TestIsSimilarContent(t *testing.T) {
	tests := []struct {
		name     string
		content1 string
		content2 string
		expected bool
	}{
		{
			name:     "Exact match",
			content1: "This is a review comment",
			content2: "This is a review comment",
			expected: true,
		},
		{
			name:     "Same content with different whitespace",
			content1: "This   is  a   review\ncomment",
			content2: "This is a review comment",
			expected: true,
		},
		{
			name:     "Completely different content",
			content1: "First review",
			content2: "Second review",
			expected: false,
		},
		{
			name:     "One is substring of the other (>80% match)",
			content1: "This is a very detailed review comment with lots of information about the code quality and suggested improvements",
			content2: "This is a very detailed review comment with lots of information about the code quality and suggested improvements and even more details",
			expected: true,
		},
		{
			name:     "Short strings that are different",
			content1: "Fix this",
			content2: "Change that",
			expected: false,
		},
		{
			name:     "Short strings that are same",
			content1: "LGTM",
			content2: "LGTM",
			expected: true,
		},
		{
			name:     "Similar but not quite matching (< 80%)",
			content1: "This is the first part of the review",
			content2: "This is the first part of the review and this is a completely different second part with lots of new information that makes it less than 80% similar",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSimilarContent(tt.content1, tt.content2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for content1=%q content2=%q", tt.expected, result, tt.content1, tt.content2)
			}
		})
	}
}
