package github

import (
	"context"
	"testing"
)

// TestMockGraphQLServer_BasicRequest tests that the mock server handles basic requests correctly
func TestMockGraphQLServer_BasicRequest(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		// Verify query and variables are passed correctly
		if query == "" {
			t.Error("Expected query to be non-empty")
		}

		return map[string]interface{}{
			"test": "data",
		}, nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	var result map[string]interface{}
	err := client.Execute(ctx, "test query", map[string]interface{}{"key": "value"}, &result)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result["test"] != "data" {
		t.Errorf("Expected result.test='data', got %v", result["test"])
	}
}

// TestMockGraphQLServer_ErrorResponse tests that the mock server can return GraphQL errors
func TestMockGraphQLServer_ErrorResponse(t *testing.T) {
	ctx := context.Background()

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		return nil, []GraphQLError{
			{Message: "Test error", Type: "TEST_ERROR"},
		}
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	var result map[string]interface{}
	err := client.Execute(ctx, "test query", nil, &result)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
}

// TestBuildThreadStatesResponse tests the helper function for building thread states
func TestBuildThreadStatesResponse(t *testing.T) {
	threads := []MockThread{
		{
			ID:         "thread1",
			IsResolved: true,
			Comments: []MockComment{
				{DatabaseID: 1},
				{DatabaseID: 2},
			},
			CommentsHasNextPage: false,
			CommentsEndCursor:   "",
		},
	}

	response := BuildThreadStatesResponse(threads, false, "")

	// Verify response structure
	repo, ok := response["repository"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected repository in response")
	}

	pr, ok := repo["pullRequest"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected pullRequest in response")
	}

	reviewThreads, ok := pr["reviewThreads"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected reviewThreads in response")
	}

	nodes, ok := reviewThreads["nodes"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected nodes array in reviewThreads")
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(nodes))
	}

	thread := nodes[0]
	if thread["id"] != "thread1" {
		t.Errorf("Expected thread id='thread1', got %v", thread["id"])
	}

	if thread["isResolved"] != true {
		t.Errorf("Expected isResolved=true, got %v", thread["isResolved"])
	}
}

// TestBuildResolveThreadResponse tests the helper function for building resolve responses
func TestBuildResolveThreadResponse(t *testing.T) {
	response := BuildResolveThreadResponse("thread1", true)

	resolveThread, ok := response["resolveReviewThread"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected resolveReviewThread in response")
	}

	thread, ok := resolveThread["thread"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected thread in response")
	}

	if thread["id"] != "thread1" {
		t.Errorf("Expected id='thread1', got %v", thread["id"])
	}

	if thread["isResolved"] != true {
		t.Errorf("Expected isResolved=true, got %v", thread["isResolved"])
	}
}

// TestBuildThreadCommentsResponse tests the helper function for building comment pagination responses
func TestBuildThreadCommentsResponse(t *testing.T) {
	comments := []MockComment{
		{DatabaseID: 101},
		{DatabaseID: 102},
	}

	response := BuildThreadCommentsResponse(comments, true, "cursor1")

	node, ok := response["node"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected node in response")
	}

	commentsData, ok := node["comments"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected comments in response")
	}

	pageInfo, ok := commentsData["pageInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected pageInfo in response")
	}

	if pageInfo["hasNextPage"] != true {
		t.Error("Expected hasNextPage=true")
	}

	if pageInfo["endCursor"] != "cursor1" {
		t.Errorf("Expected endCursor='cursor1', got %v", pageInfo["endCursor"])
	}

	nodes, ok := commentsData["nodes"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected nodes array in comments")
	}

	if len(nodes) != 2 {
		t.Fatalf("Expected 2 comments, got %d", len(nodes))
	}
}

// TestMockGraphQLServer_VariablesHandling tests that variables are correctly passed to handler
func TestMockGraphQLServer_VariablesHandling(t *testing.T) {
	ctx := context.Background()

	var capturedVariables map[string]interface{}

	mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
		capturedVariables = variables
		return map[string]interface{}{}, nil
	})
	defer mockServer.Close()

	client := NewMockGraphQLClient(mockServer, "test-token")

	testVariables := map[string]interface{}{
		"owner":    "test-owner",
		"repo":     "test-repo",
		"prNumber": 123,
	}

	var result map[string]interface{}
	err := client.Execute(ctx, "test query", testVariables, &result)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify variables were captured
	if capturedVariables["owner"] != "test-owner" {
		t.Errorf("Expected owner='test-owner', got %v", capturedVariables["owner"])
	}
	if capturedVariables["repo"] != "test-repo" {
		t.Errorf("Expected repo='test-repo', got %v", capturedVariables["repo"])
	}
	// Note: JSON unmarshaling converts numbers to float64
	if capturedVariables["prNumber"].(float64) != 123 {
		t.Errorf("Expected prNumber=123, got %v", capturedVariables["prNumber"])
	}
}
