package github

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

func TestUpdateThreadResolutionStatus_UseBatchAPI(t *testing.T) {
	tests := []struct {
		name                string
		comments            []Comment
		mockThreads         []MockThread
		expectedStatuses    int
		expectedResolved    map[int64]bool
		expectError         bool
		expectedGraphQLCall int // Number of expected GraphQL calls (should be minimal)
	}{
		{
			name: "Basic batch fetch - 3 comments in 2 threads",
			comments: []Comment{
				{ID: 101, Body: "Comment 1"},
				{ID: 102, Body: "Comment 2"},
				{ID: 103, Body: "Comment 3"},
			},
			mockThreads: []MockThread{
				{
					ID:         "thread1",
					IsResolved: true,
					Comments: []MockComment{
						{DatabaseID: 101},
						{DatabaseID: 102},
					},
					CommentsHasNextPage: false,
				},
				{
					ID:         "thread2",
					IsResolved: false,
					Comments: []MockComment{
						{DatabaseID: 103},
					},
					CommentsHasNextPage: false,
				},
			},
			expectedStatuses: 3,
			expectedResolved: map[int64]bool{
				101: true,
				102: true,
				103: false,
			},
			expectError:         false,
			expectedGraphQLCall: 1, // Only 1 batch call expected
		},
		{
			name: "Large number of comments - 100 comments (N+1 problem test)",
			comments: func() []Comment {
				comments := make([]Comment, 100)
				for i := 0; i < 100; i++ {
					comments[i] = Comment{ID: int64(i + 1), Body: "Comment"}
				}
				return comments
			}(),
			mockThreads: func() []MockThread {
				// Create 50 threads with 2 comments each
				threads := make([]MockThread, 50)
				for i := 0; i < 50; i++ {
					threads[i] = MockThread{
						ID:         "thread_" + strconv.Itoa(i),
						IsResolved: i%2 == 0, // Alternate resolved status
						Comments: []MockComment{
							{DatabaseID: int64(i*2 + 1)},
							{DatabaseID: int64(i*2 + 2)},
						},
						CommentsHasNextPage: false,
					}
				}
				return threads
			}(),
			expectedStatuses: 100,
			expectedResolved: func() map[int64]bool {
				resolved := make(map[int64]bool)
				for i := 0; i < 50; i++ {
					isResolved := i%2 == 0
					resolved[int64(i*2+1)] = isResolved
					resolved[int64(i*2+2)] = isResolved
				}
				return resolved
			}(),
			expectError:         false,
			expectedGraphQLCall: 1, // Still only 1 batch call for 100 comments!
		},
		{
			name: "Comment not found in threads - should default to unresolved",
			comments: []Comment{
				{ID: 101, Body: "Comment 1"},
				{ID: 999, Body: "Missing comment"},
			},
			mockThreads: []MockThread{
				{
					ID:         "thread1",
					IsResolved: true,
					Comments: []MockComment{
						{DatabaseID: 101},
					},
					CommentsHasNextPage: false,
				},
			},
			expectedStatuses: 2,
			expectedResolved: map[int64]bool{
				101: true,
				999: false, // Not found, should default to false
			},
			expectError:         false,
			expectedGraphQLCall: 1,
		},
		{
			name:                "Empty comments list",
			comments:            []Comment{},
			mockThreads:         []MockThread{},
			expectedStatuses:    0,
			expectedResolved:    map[int64]bool{},
			expectError:         false,
			expectedGraphQLCall: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graphQLCallCount := 0

			// Create mock GraphQL server with call counting
			mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
				graphQLCallCount++
				return BuildThreadStatesResponse(tt.mockThreads, false, ""), nil
			})
			defer mockServer.Close()

			// Create mock GraphQL client
			graphQLClient := NewMockGraphQLClient(mockServer, "test-token")

			// Create mock GitHub client with injected GraphQL client
			mockClient := &Client{
				owner:         "test-owner",
				repo:          "test-repo",
				client:        nil, // Not needed for this test
				graphqlClient: graphQLClient,
			}

			// Create tracker with mock client
			tracker := NewThreadResolutionTracker(mockClient)

			// Call the actual production method
			ctx := context.Background()
			statuses, err := tracker.UpdateThreadResolutionStatus(ctx, 123, tt.comments)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify GraphQL call count (should be minimal for batch API)
			if graphQLCallCount != tt.expectedGraphQLCall {
				t.Errorf("Expected %d GraphQL calls, got %d (N+1 problem detected!)", tt.expectedGraphQLCall, graphQLCallCount)
			}

			// Verify results
			if len(statuses) != tt.expectedStatuses {
				t.Errorf("Expected %d statuses, got %d", tt.expectedStatuses, len(statuses))
			}

			for _, status := range statuses {
				expected, ok := tt.expectedResolved[status.CommentID]
				if !ok {
					t.Errorf("Unexpected comment ID %d in results", status.CommentID)
					continue
				}
				if status.GitHubThreadResolved != expected {
					t.Errorf("Comment %d: expected resolved=%v, got %v",
						status.CommentID, expected, status.GitHubThreadResolved)
				}
			}
		})
	}
}

func TestUpdateThreadResolutionStatus_Integration(t *testing.T) {
	// Integration test with real Client structure
	t.Run("Real client integration", func(t *testing.T) {
		mockServer := NewMockGraphQLServer(t, func(query string, variables map[string]interface{}) (interface{}, []GraphQLError) {
			threads := []MockThread{
				{
					ID:         "thread1",
					IsResolved: true,
					Comments: []MockComment{
						{DatabaseID: 100},
						{DatabaseID: 101},
					},
					CommentsHasNextPage: false,
				},
			}
			return BuildThreadStatesResponse(threads, false, ""), nil
		})
		defer mockServer.Close()

		graphQLClient := NewMockGraphQLClient(mockServer, "test-token")

		// Create a mock client with injected GraphQL client
		mockClient := &Client{
			owner:         "test-owner",
			repo:          "test-repo",
			client:        nil, // Not needed for this test
			graphqlClient: graphQLClient,
		}

		// Create tracker with the mock client
		tracker := NewThreadResolutionTracker(mockClient)

		// Define test comments
		comments := []Comment{
			{ID: 100, Body: "Comment 1"},
			{ID: 101, Body: "Comment 2"},
		}

		// Call the actual production method
		ctx := context.Background()
		statuses, err := tracker.UpdateThreadResolutionStatus(ctx, 123, comments)
		if err != nil {
			t.Fatalf("UpdateThreadResolutionStatus failed: %v", err)
		}

		// Verify results
		if len(statuses) != 2 {
			t.Errorf("Expected 2 statuses, got %d", len(statuses))
		}

		for _, status := range statuses {
			if !status.GitHubThreadResolved {
				t.Errorf("Comment %d: Expected resolved=true, got false", status.CommentID)
			}
		}

		// Verify tracker was created correctly
		if tracker.client != mockClient {
			t.Errorf("Tracker client not set correctly")
		}
	})
}

func TestDetectUnresolvedComments(t *testing.T) {
	tests := []struct {
		name             string
		localComments    []Comment
		githubStatuses   []ReviewThreadStatus
		expectUnanalyzed int
		expectInProgress int
		expectResolved   int
		expectIsComplete bool
	}{
		{
			name: "All comments resolved",
			localComments: []Comment{
				{ID: 1, TasksGenerated: true, AllTasksCompleted: true},
				{ID: 2, TasksGenerated: true, AllTasksCompleted: true},
			},
			githubStatuses: []ReviewThreadStatus{
				{CommentID: 1, GitHubThreadResolved: true},
				{CommentID: 2, GitHubThreadResolved: true},
			},
			expectUnanalyzed: 0,
			expectInProgress: 0,
			expectResolved:   2,
			expectIsComplete: true,
		},
		{
			name: "Mixed status comments",
			localComments: []Comment{
				{ID: 1, TasksGenerated: false},                          // Unanalyzed
				{ID: 2, TasksGenerated: true, AllTasksCompleted: false}, // In progress
				{ID: 3, TasksGenerated: true, AllTasksCompleted: true},  // Should be resolved
			},
			githubStatuses: []ReviewThreadStatus{
				{CommentID: 1, GitHubThreadResolved: false},
				{CommentID: 2, GitHubThreadResolved: false},
				{CommentID: 3, GitHubThreadResolved: true},
			},
			expectUnanalyzed: 1,
			expectInProgress: 1,
			expectResolved:   1,
			expectIsComplete: false,
		},
		{
			name: "Comment exists locally but not in GitHub status",
			localComments: []Comment{
				{ID: 1, TasksGenerated: false},
				{ID: 999, TasksGenerated: false}, // Not in GitHub statuses
			},
			githubStatuses: []ReviewThreadStatus{
				{CommentID: 1, GitHubThreadResolved: false},
			},
			expectUnanalyzed: 2, // Both should be unanalyzed
			expectInProgress: 0,
			expectResolved:   0,
			expectIsComplete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &ThreadResolutionTracker{
				client: &Client{},
			}

			report := tracker.DetectUnresolvedComments(tt.localComments, tt.githubStatuses)

			if len(report.UnanalyzedComments) != tt.expectUnanalyzed {
				t.Errorf("Expected %d unanalyzed comments, got %d",
					tt.expectUnanalyzed, len(report.UnanalyzedComments))
			}

			if len(report.InProgressComments) != tt.expectInProgress {
				t.Errorf("Expected %d in-progress comments, got %d",
					tt.expectInProgress, len(report.InProgressComments))
			}

			if len(report.ResolvedComments) != tt.expectResolved {
				t.Errorf("Expected %d resolved comments, got %d",
					tt.expectResolved, len(report.ResolvedComments))
			}

			if report.IsComplete() != tt.expectIsComplete {
				t.Errorf("Expected IsComplete=%v, got %v",
					tt.expectIsComplete, report.IsComplete())
			}
		})
	}
}

func TestUnresolvedCommentsReport_GetSummary(t *testing.T) {
	tests := []struct {
		name     string
		report   *UnresolvedCommentsReport
		contains []string
	}{
		{
			name: "Complete report",
			report: &UnresolvedCommentsReport{
				UnanalyzedComments: []Comment{},
				InProgressComments: []Comment{},
				ResolvedComments:   []Comment{{ID: 1}, {ID: 2}},
			},
			contains: []string{"All comments analyzed and resolved"},
		},
		{
			name: "Mixed status report",
			report: &UnresolvedCommentsReport{
				UnanalyzedComments: []Comment{{ID: 1}},
				InProgressComments: []Comment{{ID: 2}, {ID: 3}},
				ResolvedComments:   []Comment{{ID: 4}},
			},
			contains: []string{
				"Unresolved Comments: 3",
				"1 comments not yet analyzed",
				"2 comments with pending tasks",
				"1 comments resolved",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.report.GetSummary()

			for _, expected := range tt.contains {
				if !strings.Contains(summary, expected) {
					t.Errorf("Summary does not contain expected string: %q\nGot: %s",
						expected, summary)
				}
			}
		})
	}
}
