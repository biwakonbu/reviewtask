package github

import (
	"context"
	"strings"
	"testing"
)

// TestResolveReviewThread_Integration tests the complete ResolveReviewThread flow
func TestResolveReviewThread_Integration(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Verify it's the resolve mutation
		if !strings.Contains(query, "resolveReviewThread") {
			t.Errorf("Expected resolveReviewThread mutation, got query: %s", query)
		}

		// Verify variables
		threadID, ok := variables["threadId"].(string)
		if !ok || threadID == "" {
			t.Error("Expected threadId variable")
		}

		return BuildResolveThreadResponse(threadID, true), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Test resolving a thread
	err := client.ResolveReviewThread(ctx, "test-thread-id")
	if err != nil {
		t.Fatalf("ResolveReviewThread failed: %v", err)
	}
}

// TestResolveReviewThread_FailedResolution tests handling of failed thread resolution
func TestResolveReviewThread_FailedResolution(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		threadID := variables["threadId"].(string)
		// Return false for isResolved to simulate failure
		return BuildResolveThreadResponse(threadID, false), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Test should fail when thread is not resolved
	err := client.ResolveReviewThread(ctx, "test-thread-id")
	if err == nil {
		t.Fatal("Expected error when thread resolution fails")
	}

	if !strings.Contains(err.Error(), "was not resolved") {
		t.Errorf("Expected 'was not resolved' error, got: %v", err)
	}
}

// TestGetReviewThreadID_Integration tests finding a thread ID for a comment
func TestGetReviewThreadID_Integration(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Build response with threads containing the target comment
		threads := []MockThread{
			{
				ID:         "thread1",
				IsResolved: false,
				Comments: []MockComment{
					{DatabaseID: 100},
					{DatabaseID: 101},
				},
				CommentsHasNextPage: false,
			},
			{
				ID:         "thread2",
				IsResolved: true,
				Comments: []MockComment{
					{DatabaseID: 200},
					{DatabaseID: 201}, // Target comment
				},
				CommentsHasNextPage: false,
			},
		}

		return BuildThreadStatesResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Test finding thread ID for comment 201
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 201)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread2" {
		t.Errorf("Expected threadID='thread2', got %s", threadID)
	}
}

// TestGetReviewThreadID_CommentNotFound tests handling when comment is not found
func TestGetReviewThreadID_CommentNotFound(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		threads := []MockThread{
			{
				ID:         "thread1",
				IsResolved: false,
				Comments: []MockComment{
					{DatabaseID: 100},
				},
				CommentsHasNextPage: false,
			},
		}

		return BuildThreadStatesResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Test finding a non-existent comment
	_, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 999)
	if err == nil {
		t.Fatal("Expected error when comment is not found")
	}

	if !strings.Contains(err.Error(), "no thread found") {
		t.Errorf("Expected 'no thread found' error, got: %v", err)
	}
}

// TestGetReviewThreadID_WithPagination tests finding a comment across multiple pages
func TestGetReviewThreadID_WithPagination(t *testing.T) {
	ctx := context.Background()

	requestCount := 0

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		requestCount++

		// Check if this is a paginated request
		_, hasCursor := variables["threadCursor"]

		if !hasCursor {
			// First page - comment not here
			threads := []MockThread{
				{
					ID:         "thread1",
					IsResolved: false,
					Comments: []MockComment{
						{DatabaseID: 100},
					},
					CommentsHasNextPage: false,
				},
			}
			return BuildThreadStatesResponse(threads, true, "cursor1"), nil
		} else {
			// Second page - comment is here
			threads := []MockThread{
				{
					ID:         "thread2",
					IsResolved: true,
					Comments: []MockComment{
						{DatabaseID: 200}, // Target comment
					},
					CommentsHasNextPage: false,
				},
			}
			return BuildThreadStatesResponse(threads, false, ""), nil
		}
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Test finding comment across pages
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 200)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread2" {
		t.Errorf("Expected threadID='thread2', got %s", threadID)
	}

	// Verify pagination happened
	if requestCount < 2 {
		t.Errorf("Expected at least 2 requests for pagination, got %d", requestCount)
	}
}

// TestGetAllThreadStates_ErrorHandling tests error handling in GetAllThreadStates
func TestGetAllThreadStates_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		return nil, []GraphQLError{
			{Message: "API rate limit exceeded", Type: "RATE_LIMITED"},
		}
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Test should handle GraphQL errors
	_, err := client.GetAllThreadStates(ctx, "owner", "repo", 123)
	if err == nil {
		t.Fatal("Expected error when GraphQL returns errors")
	}

	if !strings.Contains(err.Error(), "API rate limit") {
		t.Errorf("Expected rate limit error message, got: %v", err)
	}
}

// TestGetAllThreadStates_LargeResponse tests handling of large responses with many threads
func TestGetAllThreadStates_LargeResponse(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Create 50 threads with 10 comments each = 500 total comments
		threads := []MockThread{}
		for i := 0; i < 50; i++ {
			comments := []MockComment{}
			for j := 0; j < 10; j++ {
				comments = append(comments, MockComment{
					DatabaseID: int64(i*100 + j),
				})
			}

			threads = append(threads, MockThread{
				ID:                  "thread" + string(rune(i)),
				IsResolved:          i%2 == 0, // Alternate resolved/unresolved
				Comments:            comments,
				CommentsHasNextPage: false,
			})
		}

		return BuildThreadStatesResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Test should handle large response
	states, err := client.GetAllThreadStates(ctx, "owner", "repo", 123)
	if err != nil {
		t.Fatalf("GetAllThreadStates failed: %v", err)
	}

	expectedCount := 50 * 10 // 500 comments
	if len(states) != expectedCount {
		t.Errorf("Expected %d comments, got %d", expectedCount, len(states))
	}

	// Verify resolution states alternate
	resolvedCount := 0
	unresolvedCount := 0
	for _, isResolved := range states {
		if isResolved {
			resolvedCount++
		} else {
			unresolvedCount++
		}
	}

	// 25 threads resolved * 10 comments each = 250
	expectedResolved := 25 * 10
	if resolvedCount != expectedResolved {
		t.Errorf("Expected %d resolved comments, got %d", expectedResolved, resolvedCount)
	}
}
