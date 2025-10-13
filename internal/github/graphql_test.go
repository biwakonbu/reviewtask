package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetAllThreadStates_SinglePage tests batch fetching with single page using mock server
func TestGetAllThreadStates_SinglePage(t *testing.T) {
	ctx := context.Background()

	// Create mock server
	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Build response with two threads
		threads := []MockThread{
			{
				ID:         "thread1",
				IsResolved: true,
				Comments: []MockComment{
					{DatabaseID: 101},
					{DatabaseID: 102},
				},
				CommentsHasNextPage: false,
				CommentsEndCursor:   "",
			},
			{
				ID:         "thread2",
				IsResolved: false,
				Comments: []MockComment{
					{DatabaseID: 201},
				},
				CommentsHasNextPage: false,
				CommentsEndCursor:   "",
			},
		}

		return BuildThreadStatesResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	// Create client using mock server
	client := NewMockGraphQLClient(mockServer, "test-token")

	// Execute the actual GetAllThreadStates method
	states, err := client.GetAllThreadStates(ctx, "owner", "repo", 123)
	if err != nil {
		t.Fatalf("GetAllThreadStates failed: %v", err)
	}

	// Verify results
	expectedStates := map[int64]bool{
		101: true,  // thread1 is resolved
		102: true,  // thread1 is resolved
		201: false, // thread2 is unresolved
	}

	if len(states) != len(expectedStates) {
		t.Errorf("Expected %d comment states, got %d", len(expectedStates), len(states))
	}

	for commentID, expectedResolved := range expectedStates {
		if actualResolved, exists := states[commentID]; !exists {
			t.Errorf("Comment %d not found in results", commentID)
		} else if actualResolved != expectedResolved {
			t.Errorf("Comment %d: expected resolved=%v, got %v", commentID, expectedResolved, actualResolved)
		}
	}
}

// TestGetAllThreadStates_Pagination tests batch fetching with thread pagination using mock server
func TestGetAllThreadStates_Pagination(t *testing.T) {
	ctx := context.Background()

	// Track which page we're on
	pageNumber := 0

	// Create mock server that simulates pagination
	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Check if this is a paginated request
		_, hasCursor := variables["threadCursor"]

		if !hasCursor {
			// First page: threads 1-2
			pageNumber = 0
			threads := []MockThread{
				{
					ID:         "thread1",
					IsResolved: true,
					Comments: []MockComment{
						{DatabaseID: 1},
					},
					CommentsHasNextPage: false,
				},
				{
					ID:         "thread2",
					IsResolved: false,
					Comments: []MockComment{
						{DatabaseID: 2},
					},
					CommentsHasNextPage: false,
				},
			}
			return BuildThreadStatesResponse(threads, true, "cursor1"), nil
		} else {
			// Second page: threads 3-4
			pageNumber++
			threads := []MockThread{
				{
					ID:         "thread3",
					IsResolved: true,
					Comments: []MockComment{
						{DatabaseID: 3},
					},
					CommentsHasNextPage: false,
				},
				{
					ID:         "thread4",
					IsResolved: false,
					Comments: []MockComment{
						{DatabaseID: 4},
					},
					CommentsHasNextPage: false,
				},
			}
			return BuildThreadStatesResponse(threads, false, ""), nil
		}
	})
	defer mockServer.Close()

	// Create client using mock server
	client := NewMockGraphQLClient(mockServer, "test-token")

	// Execute the actual GetAllThreadStates method
	states, err := client.GetAllThreadStates(ctx, "owner", "repo", 123)
	if err != nil {
		t.Fatalf("GetAllThreadStates failed: %v", err)
	}

	// Verify results from both pages
	expectedStates := map[int64]bool{
		1: true,  // page 1
		2: false, // page 1
		3: true,  // page 2
		4: false, // page 2
	}

	if len(states) != len(expectedStates) {
		t.Errorf("Expected %d comment states, got %d", len(expectedStates), len(states))
	}

	for commentID, expectedResolved := range expectedStates {
		if actualResolved, exists := states[commentID]; !exists {
			t.Errorf("Comment %d not found in results", commentID)
		} else if actualResolved != expectedResolved {
			t.Errorf("Comment %d: expected resolved=%v, got %v", commentID, expectedResolved, actualResolved)
		}
	}

	// Verify that pagination actually happened
	if pageNumber < 1 {
		t.Error("Expected pagination to occur, but only one page was fetched")
	}
}

// TestGetAllThreadStates_EmptyResult tests handling of PRs with no review threads using mock server
func TestGetAllThreadStates_EmptyResult(t *testing.T) {
	ctx := context.Background()

	// Create mock server with empty response
	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Return empty threads
		return BuildThreadStatesResponse([]MockThread{}, false, ""), nil
	})
	defer mockServer.Close()

	// Create client using mock server
	client := NewMockGraphQLClient(mockServer, "test-token")

	// Execute the actual GetAllThreadStates method
	states, err := client.GetAllThreadStates(ctx, "owner", "repo", 123)
	if err != nil {
		t.Fatalf("GetAllThreadStates failed: %v", err)
	}

	// Verify empty result
	if len(states) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(states))
	}
}

// TestGetAllThreadStates_MixedResolutionStates tests mixed resolved/unresolved threads using mock server
func TestGetAllThreadStates_MixedResolutionStates(t *testing.T) {
	ctx := context.Background()

	// Create mock server with mixed resolved/unresolved threads
	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		threads := []MockThread{
			{
				ID:                  "thread1",
				IsResolved:          true,
				Comments:            []MockComment{{DatabaseID: 1}},
				CommentsHasNextPage: false,
			},
			{
				ID:                  "thread2",
				IsResolved:          false,
				Comments:            []MockComment{{DatabaseID: 2}},
				CommentsHasNextPage: false,
			},
			{
				ID:                  "thread3",
				IsResolved:          true,
				Comments:            []MockComment{{DatabaseID: 3}},
				CommentsHasNextPage: false,
			},
			{
				ID:                  "thread4",
				IsResolved:          false,
				Comments:            []MockComment{{DatabaseID: 4}},
				CommentsHasNextPage: false,
			},
			{
				ID:                  "thread5",
				IsResolved:          true,
				Comments:            []MockComment{{DatabaseID: 5}},
				CommentsHasNextPage: false,
			},
		}
		return BuildThreadStatesResponse(threads, false, ""), nil
	})
	defer mockServer.Close()

	// Create client using mock server
	client := NewMockGraphQLClient(mockServer, "test-token")

	// Execute the actual GetAllThreadStates method
	states, err := client.GetAllThreadStates(ctx, "owner", "repo", 123)
	if err != nil {
		t.Fatalf("GetAllThreadStates failed: %v", err)
	}

	// Count resolved and unresolved
	resolvedCount := 0
	unresolvedCount := 0

	for _, isResolved := range states {
		if isResolved {
			resolvedCount++
		} else {
			unresolvedCount++
		}
	}

	if resolvedCount != 3 {
		t.Errorf("Expected 3 resolved threads, got %d", resolvedCount)
	}
	if unresolvedCount != 2 {
		t.Errorf("Expected 2 unresolved threads, got %d", unresolvedCount)
	}
}

// TestGetAllThreadStates_CommentPagination tests that comments within threads are paginated correctly
func TestGetAllThreadStates_CommentPagination(t *testing.T) {
	// This test verifies the fix for Issue #222 - ensuring that threads with >100 comments
	// are fully processed by paginating through all comment pages using the mock server

	ctx := context.Background()

	tests := []struct {
		name                  string
		commentsPerThread     int
		threadsCount          int
		expectedTotalComments int
	}{
		{
			name:                  "single thread with 150 comments (2 pages)",
			commentsPerThread:     150,
			threadsCount:          1,
			expectedTotalComments: 150,
		},
		{
			name:                  "thread with exactly 100 comments (1 page)",
			commentsPerThread:     100,
			threadsCount:          1,
			expectedTotalComments: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track pagination requests
			requestCount := 0

			// Create mock server that simulates comment pagination
			mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
				requestCount++

				// Check if this is the thread-scoped comment pagination query
				if _, hasThreadID := variables["threadId"]; hasThreadID {
					// This is a comment pagination request
					// Return second page of comments (101-150)
					comments := []MockComment{}
					for i := 101; i <= tt.commentsPerThread; i++ {
						comments = append(comments, MockComment{DatabaseID: int64(i)})
					}
					return BuildThreadCommentsResponse(comments, false, ""), nil
				}

				// Initial request - return thread with first page of comments
				threads := []MockThread{}
				for threadIdx := 0; threadIdx < tt.threadsCount; threadIdx++ {
					comments := []MockComment{}
					// First page: comments 1-100 (or all if <= 100)
					maxFirstPage := tt.commentsPerThread
					if maxFirstPage > 100 {
						maxFirstPage = 100
					}
					for i := 1; i <= maxFirstPage; i++ {
						comments = append(comments, MockComment{DatabaseID: int64(threadIdx*1000 + i)})
					}

					thread := MockThread{
						ID:                  "thread1",
						IsResolved:          true,
						Comments:            comments,
						CommentsHasNextPage: tt.commentsPerThread > 100,
						CommentsEndCursor:   "cursor1",
					}
					threads = append(threads, thread)
				}

				return BuildThreadStatesResponse(threads, false, ""), nil
			})
			defer mockServer.Close()

			// Create client using mock server
			client := NewMockGraphQLClient(mockServer, "test-token")

			// Execute the actual GetAllThreadStates method
			states, err := client.GetAllThreadStates(ctx, "owner", "repo", 123)
			if err != nil {
				t.Fatalf("GetAllThreadStates failed: %v", err)
			}

			// Verify all comments were included
			if len(states) != tt.expectedTotalComments {
				t.Errorf("Expected %d comments in state map, got %d",
					tt.expectedTotalComments, len(states))
			}

			// Verify all comments are marked as resolved (matching thread state)
			for commentID, isResolved := range states {
				if !isResolved {
					t.Errorf("Comment %d: expected resolved=true, got false", commentID)
				}
			}

			// Verify pagination happened when expected
			if tt.commentsPerThread > 100 && requestCount < 2 {
				t.Errorf("Expected at least 2 requests for %d comments, got %d",
					tt.commentsPerThread, requestCount)
			}
		})
	}
}

// Helper functions

func createThreadStates(startID int64, count int, resolved bool) map[int64]bool {
	states := make(map[int64]bool)
	for i := 0; i < count; i++ {
		states[startID+int64(i)] = resolved
	}
	return states
}

// TestGraphQLClient_Execute tests basic GraphQL execution with mock server
func TestGraphQLClient_Execute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		handler     func(string, map[string]interface{}) (interface{}, []GraphQLError)
		expectError bool
	}{
		{
			name: "successful query",
			handler: func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
				return map[string]interface{}{
					"repository": map[string]interface{}{
						"name": "test",
					},
				}, nil
			},
			expectError: false,
		},
		{
			name: "GraphQL error response",
			handler: func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
				return nil, []GraphQLError{
					{Message: "Field not found", Type: "NOT_FOUND"},
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			mockServer := NewMockGraphQLServer(t, tt.handler)
			defer mockServer.Close()

			// Create client using mock server
			client := NewMockGraphQLClient(mockServer, "test-token")

			// Execute a simple query
			var result map[string]interface{}
			err := client.Execute(ctx, "query { repository { name } }", nil, &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				// Verify the result structure
				if repo, ok := result["repository"].(map[string]interface{}); !ok {
					t.Error("Expected repository field in result")
				} else if name, ok := repo["name"].(string); !ok || name != "test" {
					t.Errorf("Expected repository.name='test', got %v", name)
				}
			}
		})
	}
}

// TestGraphQLClient_Authentication tests that authentication header is set correctly with mock server
func TestGraphQLClient_Authentication(t *testing.T) {
	ctx := context.Background()
	expectedToken := "test-token-12345"

	// Variable to capture the auth header from the request
	var capturedAuthHeader string

	// Create a custom mock server to capture headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")

		// Verify it's a POST request with correct headers
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Send success response
		response := GraphQLResponse{
			Data: map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with custom token and endpoint
	client := NewGraphQLClient(expectedToken)
	client.endpoint = server.URL
	client.httpClient = server.Client()

	// Execute a query
	var result map[string]interface{}
	err := client.Execute(ctx, "query {}", nil, &result)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify the Authorization header
	expectedHeader := "Bearer " + expectedToken
	if capturedAuthHeader != expectedHeader {
		t.Errorf("Expected Authorization header '%s', got '%s'", expectedHeader, capturedAuthHeader)
	}
}
