package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockGraphQLServer provides a mock GraphQL server for testing
type MockGraphQLServer struct {
	Server  *httptest.Server
	Handler func(query string, variables map[string]interface{}) (interface{}, []GraphQLError)
}

// NewMockGraphQLServer creates a new mock GraphQL server
func NewMockGraphQLServer(t *testing.T, handler func(query string, variables map[string]interface{}) (interface{}, []GraphQLError)) *MockGraphQLServer {
	mock := &MockGraphQLServer{
		Handler: handler,
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Parse the GraphQL request
		var req GraphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Call the handler to get response data and errors
		data, errors := mock.Handler(req.Query, req.Variables)

		// Build the response
		response := GraphQLResponse{
			Data:   data,
			Errors: errors,
		}

		// Send the response
		w.Header().Set("Content-Type", "application/json")
		if len(errors) > 0 {
			// GraphQL errors still return 200 status code
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(response)
	}))

	return mock
}

// Close closes the mock server
func (m *MockGraphQLServer) Close() {
	m.Server.Close()
}

// NewMockGraphQLClient creates a GraphQL client that uses the mock server
func NewMockGraphQLClient(mockServer *MockGraphQLServer, token string) *GraphQLClient {
	client := NewGraphQLClient(token)
	// Override the httpClient and endpoint to use the mock server
	client.httpClient = mockServer.Server.Client()
	client.endpoint = mockServer.Server.URL
	return client
}

// Helper functions for building common test responses

// BuildThreadStatesResponse builds a response for GetAllThreadStates query
func BuildThreadStatesResponse(threads []MockThread, hasNextPage bool, endCursor string) map[string]interface{} {
	threadNodes := make([]map[string]interface{}, len(threads))
	for i, thread := range threads {
		commentNodes := make([]map[string]interface{}, len(thread.Comments))
		for j, comment := range thread.Comments {
			commentNodes[j] = map[string]interface{}{
				"databaseId": comment.DatabaseID,
			}
		}

		threadNodes[i] = map[string]interface{}{
			"id":         thread.ID,
			"isResolved": thread.IsResolved,
			"comments": map[string]interface{}{
				"pageInfo": map[string]interface{}{
					"hasNextPage": thread.CommentsHasNextPage,
					"endCursor":   thread.CommentsEndCursor,
				},
				"nodes": commentNodes,
			},
		}
	}

	return map[string]interface{}{
		"repository": map[string]interface{}{
			"pullRequest": map[string]interface{}{
				"reviewThreads": map[string]interface{}{
					"pageInfo": map[string]interface{}{
						"hasNextPage": hasNextPage,
						"endCursor":   endCursor,
					},
					"nodes": threadNodes,
				},
			},
		},
	}
}

// BuildResolveThreadResponse builds a response for ResolveReviewThread mutation
func BuildResolveThreadResponse(threadID string, isResolved bool) map[string]interface{} {
	return map[string]interface{}{
		"resolveReviewThread": map[string]interface{}{
			"thread": map[string]interface{}{
				"id":         threadID,
				"isResolved": isResolved,
			},
		},
	}
}

// BuildThreadCommentsResponse builds a response for paginated thread comments query
func BuildThreadCommentsResponse(comments []MockComment, hasNextPage bool, endCursor string) map[string]interface{} {
	commentNodes := make([]map[string]interface{}, len(comments))
	for i, comment := range comments {
		commentNodes[i] = map[string]interface{}{
			"databaseId": comment.DatabaseID,
		}
	}

	return map[string]interface{}{
		"node": map[string]interface{}{
			"comments": map[string]interface{}{
				"pageInfo": map[string]interface{}{
					"hasNextPage": hasNextPage,
					"endCursor":   endCursor,
				},
				"nodes": commentNodes,
			},
		},
	}
}

// BuildReviewThreadIDResponse builds a response for GetReviewThreadID query
func BuildReviewThreadIDResponse(threads []MockThread, hasNextPage bool, endCursor string) map[string]interface{} {
	threadNodes := make([]map[string]interface{}, len(threads))
	for i, thread := range threads {
		commentNodes := make([]map[string]interface{}, len(thread.Comments))
		for j, comment := range thread.Comments {
			commentNodes[j] = map[string]interface{}{
				"id":         comment.ID,
				"databaseId": comment.DatabaseID,
			}
		}

		threadNodes[i] = map[string]interface{}{
			"id": thread.ID,
			"comments": map[string]interface{}{
				"pageInfo": map[string]interface{}{
					"hasNextPage": thread.CommentsHasNextPage,
					"endCursor":   thread.CommentsEndCursor,
				},
				"nodes": commentNodes,
			},
		}
	}

	return map[string]interface{}{
		"repository": map[string]interface{}{
			"pullRequest": map[string]interface{}{
				"reviewThreads": map[string]interface{}{
					"pageInfo": map[string]interface{}{
						"hasNextPage": hasNextPage,
						"endCursor":   endCursor,
					},
					"nodes": threadNodes,
				},
			},
		},
	}
}

// BuildThreadCommentsIDResponse builds a response for paginated thread comments query (GetReviewThreadID)
func BuildThreadCommentsIDResponse(comments []MockComment, hasNextPage bool, endCursor string) map[string]interface{} {
	commentNodes := make([]map[string]interface{}, len(comments))
	for i, comment := range comments {
		commentNodes[i] = map[string]interface{}{
			"id":         comment.ID,
			"databaseId": comment.DatabaseID,
		}
	}

	return map[string]interface{}{
		"node": map[string]interface{}{
			"comments": map[string]interface{}{
				"pageInfo": map[string]interface{}{
					"hasNextPage": hasNextPage,
					"endCursor":   endCursor,
				},
				"nodes": commentNodes,
			},
		},
	}
}

// MockThread represents a mock review thread
type MockThread struct {
	ID                  string
	IsResolved          bool
	Comments            []MockComment
	CommentsHasNextPage bool
	CommentsEndCursor   string
}

// MockComment represents a mock review comment
type MockComment struct {
	ID         string
	DatabaseID int64
}
