package github

import (
	"context"
	"testing"
	stdtime "time"

	gh "github.com/google/go-github/v58/github"
)

// helper constructors for pointers
func strp(s string) *string { return &s }
func int64p(i int64) *int64 { return &i }
func intp(i int) *int       { return &i }

func TestGetReviewCommentsFiltersByReviewID(t *testing.T) {
	// Prepare a client with a fresh cache namespace
	c := &Client{
		owner: "owner-test-filter",
		repo:  "repo-test-filter",
		cache: NewAPICache(5 * stdtime.Minute),
	}

	prNumber := 154
	reviewID1 := int64(111)
	reviewID2 := int64(222)

	now := gh.Timestamp{Time: stdtime.Now()}

	// Build three comments, two for reviewID1 and one for reviewID2
	comments := []*gh.PullRequestComment{
		{
			ID:                  int64p(1),
			PullRequestReviewID: int64p(reviewID1),
			Path:                strp("file1.go"),
			Line:                intp(10),
			Body:                strp("c1"),
			User:                &gh.User{Login: strp("alice")},
			CreatedAt:           &now,
		},
		{
			ID:                  int64p(2),
			PullRequestReviewID: int64p(reviewID1),
			Path:                strp("file2.go"),
			Line:                intp(20),
			Body:                strp("c2"),
			User:                &gh.User{Login: strp("bob")},
			CreatedAt:           &now,
		},
		{
			ID:                  int64p(3),
			PullRequestReviewID: int64p(reviewID2),
			Path:                strp("file3.go"),
			Line:                intp(30),
			Body:                strp("c3"),
			User:                &gh.User{Login: strp("carol")},
			CreatedAt:           &now,
		},
	}

	// Seed cache so getReviewComments doesn't call the API client
	cacheKey := "prcomments-154"
	if err := c.cache.Set("ListComments", c.owner, c.repo, comments, cacheKey); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	// Should only return the two comments for reviewID1
	got, err := c.getReviewComments(context.Background(), prNumber, reviewID1)
	if err != nil {
		t.Fatalf("getReviewComments error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 comments for review %d, got %d", reviewID1, len(got))
	}
	ids := map[int64]bool{}
	for _, cmt := range got {
		ids[cmt.ID] = true
		if cmt.File == "" {
			t.Fatalf("expected File to be set, got empty")
		}
	}
	if !ids[1] || !ids[2] || ids[3] {
		t.Fatalf("unexpected IDs in result: %+v", ids)
	}
}
