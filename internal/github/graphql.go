package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GraphQLClient provides GraphQL API operations
type GraphQLClient struct {
	token      string
	httpClient *http.Client
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{}    `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// NewGraphQLClient creates a new GraphQL client
func NewGraphQLClient(token string) *GraphQLClient {
	return &GraphQLClient{
		token:      token,
		httpClient: &http.Client{},
	}
}

// Execute executes a GraphQL query
func (c *GraphQLClient) Execute(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	request := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL response: %w", err)
	}

	if len(graphqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %+v", graphqlResp.Errors)
	}

	// Marshal the data field and unmarshal into result
	dataJSON, err := json.Marshal(graphqlResp.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL data: %w", err)
	}

	if err := json.Unmarshal(dataJSON, result); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL data into result: %w", err)
	}

	return nil
}

// ResolveReviewThread resolves a review thread by its thread ID
func (c *GraphQLClient) ResolveReviewThread(ctx context.Context, threadID string) error {
	query := `
		mutation($threadId: ID!) {
			resolveReviewThread(input: {threadId: $threadId}) {
				thread {
					id
					isResolved
				}
			}
		}
	`

	variables := map[string]interface{}{
		"threadId": threadID,
	}

	var result struct {
		ResolveReviewThread struct {
			Thread struct {
				ID         string `json:"id"`
				IsResolved bool   `json:"isResolved"`
			} `json:"thread"`
		} `json:"resolveReviewThread"`
	}

	if err := c.Execute(ctx, query, variables, &result); err != nil {
		return fmt.Errorf("failed to resolve review thread: %w", err)
	}

	if !result.ResolveReviewThread.Thread.IsResolved {
		return fmt.Errorf("thread %s was not resolved", threadID)
	}

	return nil
}

// GetReviewThreadID gets the review thread ID for a review comment
// This maps a review comment ID to its corresponding thread ID
// Supports pagination for large PRs with >100 threads or >100 comments per thread
func (c *GraphQLClient) GetReviewThreadID(ctx context.Context, owner, repo string, prNumber int, commentID int64) (string, error) {
	query := `
		query($owner: String!, $repo: String!, $prNumber: Int!, $threadCursor: String, $commentCursor: String) {
			repository(owner: $owner, name: $repo) {
				pullRequest(number: $prNumber) {
					reviewThreads(first: 100, after: $threadCursor) {
						pageInfo {
							hasNextPage
							endCursor
						}
						nodes {
							id
							comments(first: 100, after: $commentCursor) {
								pageInfo {
									hasNextPage
									endCursor
								}
								nodes {
									id
									databaseId
								}
							}
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":    owner,
		"repo":     repo,
		"prNumber": prNumber,
	}

	type CommentNode struct {
		ID         string `json:"id"`
		DatabaseID int64  `json:"databaseId"`
	}

	type PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	}

	type Comments struct {
		PageInfo PageInfo      `json:"pageInfo"`
		Nodes    []CommentNode `json:"nodes"`
	}

	type ThreadNode struct {
		ID       string   `json:"id"`
		Comments Comments `json:"comments"`
	}

	type ReviewThreads struct {
		PageInfo PageInfo     `json:"pageInfo"`
		Nodes    []ThreadNode `json:"nodes"`
	}

	var result struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads ReviewThreads `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	// Paginate through review threads
	var threadCursor *string
	for {
		// Set thread cursor if we have one
		if threadCursor != nil {
			variables["threadCursor"] = *threadCursor
		}

		// Execute query for current page of threads
		if err := c.Execute(ctx, query, variables, &result); err != nil {
			return "", fmt.Errorf("failed to get review threads: %w", err)
		}

		// Search through threads in current page
		for _, thread := range result.Repository.PullRequest.ReviewThreads.Nodes {
			// For each thread, paginate through comments
			var commentCursor *string
			threadID := thread.ID
			currentComments := thread.Comments

			for {
				// Search through comments in current page
				for _, comment := range currentComments.Nodes {
					if comment.DatabaseID == commentID {
						return threadID, nil
					}
				}

				// Check if there are more comments in this thread
				if !currentComments.PageInfo.HasNextPage {
					break
				}

				// Fetch next page of comments for this thread
				commentCursor = &currentComments.PageInfo.EndCursor
				variables["commentCursor"] = *commentCursor

				if err := c.Execute(ctx, query, variables, &result); err != nil {
					return "", fmt.Errorf("failed to get review thread comments: %w", err)
				}

				// Find the same thread in the new result (threads are re-fetched but we only care about comments)
				found := false
				for _, t := range result.Repository.PullRequest.ReviewThreads.Nodes {
					if t.ID == threadID {
						currentComments = t.Comments
						found = true
						break
					}
				}

				if !found {
					return "", fmt.Errorf("thread %s not found in paginated results", threadID)
				}
			}

			// Reset comment cursor for next thread
			delete(variables, "commentCursor")
		}

		// Check if there are more threads to fetch
		if !result.Repository.PullRequest.ReviewThreads.PageInfo.HasNextPage {
			break
		}

		// Move to next page of threads
		cursor := result.Repository.PullRequest.ReviewThreads.PageInfo.EndCursor
		threadCursor = &cursor
	}

	return "", fmt.Errorf("no thread found for comment ID %d", commentID)
}

// AddGraphQLClientToGitHubClient adds GraphQL client to existing GitHub client
func (c *Client) NewGraphQLClient() (*GraphQLClient, error) {
	token, err := GetGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	return NewGraphQLClient(token), nil
}

// ResolveCommentThread resolves a review thread for a specific comment
func (c *Client) ResolveCommentThread(ctx context.Context, prNumber int, commentID int64) error {
	// Create GraphQL client
	graphqlClient, err := c.NewGraphQLClient()
	if err != nil {
		return fmt.Errorf("failed to create GraphQL client: %w", err)
	}

	// Get thread ID for this comment
	threadID, err := graphqlClient.GetReviewThreadID(ctx, c.owner, c.repo, prNumber, commentID)
	if err != nil {
		return fmt.Errorf("failed to get thread ID: %w", err)
	}

	// Resolve the thread
	if err := graphqlClient.ResolveReviewThread(ctx, threadID); err != nil {
		return fmt.Errorf("failed to resolve thread: %w", err)
	}

	return nil
}
