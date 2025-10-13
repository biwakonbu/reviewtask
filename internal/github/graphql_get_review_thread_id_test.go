package github

import (
	"context"
	"testing"
)

// TestGetReviewThreadID_Unit_SimpleCase tests finding a comment in a single thread with no pagination
func TestGetReviewThreadID_Unit_SimpleCase(t *testing.T) {
	ctx := context.Background()

	// Create mock server
	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		threads := []MockThread{
			{
				ID: "thread1",
				Comments: []MockComment{
					{ID: "comment1", DatabaseID: 101},
					{ID: "comment2", DatabaseID: 102},
				},
				CommentsHasNextPage: false,
			},
		}
		return BuildReviewThreadIDResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	// Create client
	client := NewMockGraphQLClient(mockServer, "test-token")

	// Find comment 102
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 102)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread1" {
		t.Errorf("Expected thread1, got %s", threadID)
	}
}

// TestGetReviewThreadID_Unit_CommentNotFound tests handling when comment doesn't exist
func TestGetReviewThreadID_Unit_CommentNotFound(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		threads := []MockThread{
			{
				ID: "thread1",
				Comments: []MockComment{
					{ID: "comment1", DatabaseID: 101},
				},
				CommentsHasNextPage: false,
			},
		}
		return BuildReviewThreadIDResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Try to find non-existent comment
	_, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 999)
	if err == nil {
		t.Error("Expected error for non-existent comment, got nil")
	}
}

// TestGetReviewThreadID_Unit_MultipleThreads tests finding comment across multiple threads
func TestGetReviewThreadID_Unit_MultipleThreads(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		threads := []MockThread{
			{
				ID: "thread1",
				Comments: []MockComment{
					{ID: "comment1", DatabaseID: 101},
				},
				CommentsHasNextPage: false,
			},
			{
				ID: "thread2",
				Comments: []MockComment{
					{ID: "comment2", DatabaseID: 201},
					{ID: "comment3", DatabaseID: 202},
				},
				CommentsHasNextPage: false,
			},
			{
				ID: "thread3",
				Comments: []MockComment{
					{ID: "comment4", DatabaseID: 301},
				},
				CommentsHasNextPage: false,
			},
		}
		return BuildReviewThreadIDResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Find comment in second thread
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 202)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread2" {
		t.Errorf("Expected thread2, got %s", threadID)
	}
}

// TestGetReviewThreadID_Unit_ThreadPagination tests finding comment when threads are paginated
func TestGetReviewThreadID_Unit_ThreadPagination(t *testing.T) {
	ctx := context.Background()

	pageNumber := 0

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		_, hasCursor := variables["threadCursor"]

		if !hasCursor {
			// First page
			pageNumber = 0
			threads := []MockThread{
				{
					ID: "thread1",
					Comments: []MockComment{
						{ID: "comment1", DatabaseID: 101},
					},
					CommentsHasNextPage: false,
				},
			}
			return BuildReviewThreadIDResponse(threads, true, "cursor1"), nil
		}

		// Second page - contains our target comment
		pageNumber++
		threads := []MockThread{
			{
				ID: "thread2",
				Comments: []MockComment{
					{ID: "comment2", DatabaseID: 201},
				},
				CommentsHasNextPage: false,
			},
		}
		return BuildReviewThreadIDResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Find comment in second page
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 201)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread2" {
		t.Errorf("Expected thread2, got %s", threadID)
	}

	if pageNumber == 0 {
		t.Error("Expected pagination to occur, but only one page was fetched")
	}
}

// TestGetReviewThreadID_Unit_CommentPagination tests finding comment when comments within a thread are paginated
// This is the key test for Issue #226 - ensuring per-thread cursor isolation
func TestGetReviewThreadID_Unit_CommentPagination(t *testing.T) {
	ctx := context.Background()

	requestCount := 0

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		requestCount++

		// Check if this is a thread-scoped comment pagination query
		if threadID, hasThreadID := variables["threadId"]; hasThreadID {
			// This is a comment pagination request for a specific thread
			if threadID == "thread1" {
				// Second page of comments for thread1 (101-150)
				comments := []MockComment{}
				for i := 101; i <= 150; i++ {
					comments = append(comments, MockComment{
						ID:         "comment" + string(rune(i)),
						DatabaseID: int64(i),
					})
				}
				return BuildThreadCommentsIDResponse(comments, false, ""), nil
			}
			t.Errorf("Unexpected threadId in comment pagination: %v", threadID)
			return nil, []GraphQLError{{Message: "Unexpected thread ID"}}
		}

		// Initial request - return thread with first page of comments (1-100)
		thread1Comments := []MockComment{}
		for i := 1; i <= 100; i++ {
			thread1Comments = append(thread1Comments, MockComment{
				ID:         "comment" + string(rune(i)),
				DatabaseID: int64(i),
			})
		}

		threads := []MockThread{
			{
				ID:                  "thread1",
				Comments:            thread1Comments,
				CommentsHasNextPage: true, // Has more comments
				CommentsEndCursor:   "cursor100",
			},
		}

		return BuildReviewThreadIDResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Find comment in second page of comments (comment 125)
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 125)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread1" {
		t.Errorf("Expected thread1, got %s", threadID)
	}

	// Verify that pagination happened (initial request + comment pagination)
	if requestCount < 2 {
		t.Errorf("Expected at least 2 requests for paginated comments, got %d", requestCount)
	}
}

// TestGetReviewThreadID_Unit_PerThreadCursorIsolation tests the critical bug fix from Issue #226
// Ensures that when one thread needs comment pagination, it doesn't affect other threads
func TestGetReviewThreadID_Unit_PerThreadCursorIsolation(t *testing.T) {
	ctx := context.Background()

	requestLog := []string{}

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Check if this is a thread-scoped comment pagination query
		if threadID, hasThreadID := variables["threadId"]; hasThreadID {
			requestLog = append(requestLog, "thread-scoped:"+threadID.(string))

			// Return second page of comments for the specific thread
			if threadID == "thread1" {
				// Second page: comments 101-150
				comments := []MockComment{}
				for i := 101; i <= 150; i++ {
					comments = append(comments, MockComment{
						ID:         "comment" + string(rune(i)),
						DatabaseID: int64(i),
					})
				}
				return BuildThreadCommentsIDResponse(comments, false, ""), nil
			} else if threadID == "thread2" {
				// Second page: comments 201-250
				comments := []MockComment{}
				for i := 201; i <= 250; i++ {
					comments = append(comments, MockComment{
						ID:         "comment" + string(rune(i)),
						DatabaseID: int64(i),
					})
				}
				return BuildThreadCommentsIDResponse(comments, false, ""), nil
			}

			t.Errorf("Unexpected threadId: %v", threadID)
			return nil, []GraphQLError{{Message: "Unexpected thread ID"}}
		}

		requestLog = append(requestLog, "initial")

		// Initial request - return multiple threads, each with paginated comments
		thread1Comments := []MockComment{}
		for i := 1; i <= 100; i++ {
			thread1Comments = append(thread1Comments, MockComment{
				ID:         "comment" + string(rune(i)),
				DatabaseID: int64(i),
			})
		}

		thread2Comments := []MockComment{}
		for i := 1; i <= 100; i++ {
			thread2Comments = append(thread2Comments, MockComment{
				ID:         "comment" + string(rune(i+100)),
				DatabaseID: int64(i + 100),
			})
		}

		threads := []MockThread{
			{
				ID:                  "thread1",
				Comments:            thread1Comments,
				CommentsHasNextPage: true, // Thread1 has more comments
				CommentsEndCursor:   "cursor100",
			},
			{
				ID:                  "thread2",
				Comments:            thread2Comments,
				CommentsHasNextPage: true, // Thread2 also has more comments
				CommentsEndCursor:   "cursor200",
			},
		}

		return BuildReviewThreadIDResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Find comment in second page of thread2's comments
	// This tests that thread1's pagination doesn't interfere with thread2
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 225)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread2" {
		t.Errorf("Expected thread2, got %s", threadID)
	}

	// Verify request pattern:
	// 1. Initial request (fetches both threads)
	// 2. Thread-scoped request for thread1 (paginate its comments - won't find target)
	// 3. Thread-scoped request for thread2 (paginate its comments - finds target)
	expectedPattern := []string{"initial", "thread-scoped:thread1", "thread-scoped:thread2"}
	if len(requestLog) != len(expectedPattern) {
		t.Errorf("Expected %d requests, got %d. Log: %v", len(expectedPattern), len(requestLog), requestLog)
	}

	for i, expected := range expectedPattern {
		if i >= len(requestLog) {
			t.Errorf("Missing request %d: expected %s", i, expected)
			break
		}
		if requestLog[i] != expected {
			t.Errorf("Request %d: expected %s, got %s", i, expected, requestLog[i])
		}
	}
}

// TestGetReviewThreadID_Unit_FirstCommentInPaginatedThread tests finding first comment when thread has pagination
func TestGetReviewThreadID_Unit_FirstCommentInPaginatedThread(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// We should find the comment in the first page, so no thread-scoped query needed
		if _, hasThreadID := variables["threadId"]; hasThreadID {
			t.Error("Should not require thread-scoped pagination for first page comment")
			return nil, []GraphQLError{{Message: "Unexpected pagination"}}
		}

		// Return thread with many comments, but target is in first page
		comments := []MockComment{}
		for i := 1; i <= 100; i++ {
			comments = append(comments, MockComment{
				ID:         "comment" + string(rune(i)),
				DatabaseID: int64(i),
			})
		}

		threads := []MockThread{
			{
				ID:                  "thread1",
				Comments:            comments,
				CommentsHasNextPage: true, // Has more, but we won't need them
				CommentsEndCursor:   "cursor100",
			},
		}

		return BuildReviewThreadIDResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	// Find comment in first page (should not trigger pagination)
	threadID, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 50)
	if err != nil {
		t.Fatalf("GetReviewThreadID failed: %v", err)
	}

	if threadID != "thread1" {
		t.Errorf("Expected thread1, got %s", threadID)
	}
}

// TestGetReviewThreadID_Unit_EmptyThreads tests handling when PR has no review threads
func TestGetReviewThreadID_Unit_EmptyThreads(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		return BuildReviewThreadIDResponse([]MockThread{}, false, ""), nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	_, err := client.GetReviewThreadID(ctx, "owner", "repo", 123, 101)
	if err == nil {
		t.Error("Expected error for comment in PR with no threads")
	}
}
