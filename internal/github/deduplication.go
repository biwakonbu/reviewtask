package github

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// ReviewFingerprint represents a unique identifier for a review's content
type ReviewFingerprint struct {
	Reviewer    string
	ContentHash string
	CommentIDs  []int64
}

// GenerateReviewFingerprint creates a fingerprint for a review based on its content
// This is used to detect duplicate reviews from the same author
func GenerateReviewFingerprint(review Review) ReviewFingerprint {
	// Create a content signature from review body and comments
	var contentParts []string

	// Include review body (trimmed)
	if body := strings.TrimSpace(review.Body); body != "" {
		contentParts = append(contentParts, body)
	}

	// Include all comment bodies and file locations
	var commentIDs []int64
	for _, comment := range review.Comments {
		// Build comment signature: file:line:body
		sig := fmt.Sprintf("%s:%d:%s",
			comment.File,
			comment.Line,
			strings.TrimSpace(comment.Body))
		contentParts = append(contentParts, sig)

		// Track comment IDs for identifying which reviews to keep
		if comment.ID != 0 {
			commentIDs = append(commentIDs, comment.ID)
		}
	}

	// Sort comment IDs for consistent fingerprinting
	sort.Slice(commentIDs, func(i, j int) bool {
		return commentIDs[i] < commentIDs[j]
	})

	// Generate SHA256 hash of the content
	contentStr := strings.Join(contentParts, "|")
	hash := sha256.Sum256([]byte(contentStr))
	contentHash := fmt.Sprintf("%x", hash)

	return ReviewFingerprint{
		Reviewer:    review.Reviewer,
		ContentHash: contentHash,
		CommentIDs:  commentIDs,
	}
}

// DeduplicateReviews removes duplicate reviews from the same reviewer
// It keeps the most recent review when duplicates are detected
func DeduplicateReviews(reviews []Review) []Review {
	if len(reviews) <= 1 {
		return reviews
	}

	// Group reviews by reviewer
	reviewsByReviewer := make(map[string][]Review)
	for _, review := range reviews {
		reviewsByReviewer[review.Reviewer] = append(reviewsByReviewer[review.Reviewer], review)
	}

	var deduplicated []Review

	// Process each reviewer's reviews
	for _, reviewerReviews := range reviewsByReviewer {
		// If only one review from this reviewer, keep it
		if len(reviewerReviews) == 1 {
			deduplicated = append(deduplicated, reviewerReviews[0])
			continue
		}

		// Create fingerprints for all reviews from this reviewer
		fingerprints := make(map[string][]int) // contentHash -> review indices
		for i, review := range reviewerReviews {
			fp := GenerateReviewFingerprint(review)
			fingerprints[fp.ContentHash] = append(fingerprints[fp.ContentHash], i)
		}

		// Keep only the most recent review for each unique content hash
		// For each content hash, find the most recent review by comparing SubmittedAt
		hashToLatest := make(map[string]Review)
		for _, review := range reviewerReviews {
			fp := GenerateReviewFingerprint(review)

			// If we haven't seen this hash, or this review is more recent, keep it
			if existing, exists := hashToLatest[fp.ContentHash]; !exists || review.SubmittedAt > existing.SubmittedAt {
				hashToLatest[fp.ContentHash] = review
			}
		}

		// Add the latest reviews to deduplicated list
		for _, review := range hashToLatest {
			deduplicated = append(deduplicated, review)
		}
	}

	// Sort by submission time to maintain chronological order
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].SubmittedAt < deduplicated[j].SubmittedAt
	})

	return deduplicated
}

// IsSimilarContent checks if two review contents are similar enough to be considered duplicates
// This is more lenient than exact matching and useful for detecting near-duplicates
func IsSimilarContent(content1, content2 string) bool {
	// Normalize whitespace
	normalize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.Join(strings.Fields(s), " ")
		return s
	}

	norm1 := normalize(content1)
	norm2 := normalize(content2)

	// Exact match after normalization
	if norm1 == norm2 {
		return true
	}

	// Check if one is a substring of the other (allowing for additions/edits)
	if len(norm1) > 50 && len(norm2) > 50 {
		shorter := norm1
		longer := norm2
		if len(norm1) > len(norm2) {
			shorter = norm2
			longer = norm1
		}

		// If the shorter is >80% contained in the longer, consider them similar
		if strings.Contains(longer, shorter) {
			ratio := float64(len(shorter)) / float64(len(longer))
			return ratio > 0.8
		}
	}

	return false
}
