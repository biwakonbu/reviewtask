package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-github/v58/github"
)

// MockGitHubServer creates a mock GitHub API server for testing
func MockGitHubServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Mock PR endpoint
	mux.HandleFunc("/repos/test/repo/pulls/123", func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"number": 123,
			"title": "Test PR",
			"user": {"login": "testuser"},
			"created_at": "2023-01-01T00:00:00Z",
			"updated_at": "2023-01-01T01:00:00Z",
			"state": "open",
			"head": {"ref": "feature/test-branch"}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	// Mock PR search endpoint
	mux.HandleFunc("/repos/test/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		head := r.URL.Query().Get("head")
		if head == "test:feature/test-branch" {
			response := `[{
				"number": 123,
				"title": "Test PR",
				"user": {"login": "testuser"},
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T01:00:00Z",
				"state": "open",
				"head": {"ref": "feature/test-branch"}
			}]`
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(response))
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		}
	})

	// Mock reviews endpoint
	mux.HandleFunc("/repos/test/repo/pulls/123/reviews", func(w http.ResponseWriter, r *http.Request) {
		response := `[{
			"id": 1,
			"user": {"login": "reviewer1"},
			"state": "APPROVED",
			"body": "Looks good!",
			"submitted_at": "2023-01-01T02:00:00Z"
		}]`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	// Mock comments endpoint
	mux.HandleFunc("/repos/test/repo/pulls/123/comments", func(w http.ResponseWriter, r *http.Request) {
		response := `[{
			"id": 1,
			"path": "test.go",
			"line": 10,
			"body": "Fix this issue",
			"user": {"login": "reviewer1"},
			"created_at": "2023-01-01T02:00:00Z"
		}, {
			"id": 2,
			"path": "main.go",
			"line": 20,
			"body": "TODO: Add error handling here",
			"user": {"login": "testuser"},
			"created_at": "2023-01-01T03:00:00Z"
		}]`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	// Mock issue comments endpoint
	mux.HandleFunc("/repos/test/repo/issues/123/comments", func(w http.ResponseWriter, r *http.Request) {
		response := `[{
			"id": 3,
			"body": "I noticed a potential issue with the error handling",
			"user": {"login": "testuser"},
			"created_at": "2023-01-01T04:00:00Z"
		}, {
			"id": 4,
			"body": "Great implementation!",
			"user": {"login": "reviewer2"},
			"created_at": "2023-01-01T05:00:00Z"
		}]`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	return httptest.NewServer(mux)
}

// TestClient_GetPRInfo tests PR information retrieval
func TestClient_GetPRInfo(t *testing.T) {
	server := MockGitHubServer(t)
	defer server.Close()

	// Create client with mock server
	client := github.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")

	ghClient := &Client{
		client: client,
		owner:  "test",
		repo:   "repo",
		cache:  NewAPICache(5 * time.Minute),
	}

	// Test getting PR info
	prInfo, err := ghClient.GetPRInfo(context.Background(), 123)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify PR info
	if prInfo.Number != 123 {
		t.Errorf("Expected PR number 123, got: %d", prInfo.Number)
	}

	if prInfo.Title != "Test PR" {
		t.Errorf("Expected title 'Test PR', got: %s", prInfo.Title)
	}

	if prInfo.Author != "testuser" {
		t.Errorf("Expected author 'testuser', got: %s", prInfo.Author)
	}

	if prInfo.Branch != "feature/test-branch" {
		t.Errorf("Expected branch 'feature/test-branch', got: %s", prInfo.Branch)
	}

	if prInfo.State != "open" {
		t.Errorf("Expected state 'open', got: %s", prInfo.State)
	}

	if prInfo.Repository != "test/repo" {
		t.Errorf("Expected repository 'test/repo', got: %s", prInfo.Repository)
	}
}

// TestClient_GetPRReviews tests PR reviews retrieval
func TestClient_GetPRReviews(t *testing.T) {
	server := MockGitHubServer(t)
	defer server.Close()

	// Create client with mock server
	client := github.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")

	ghClient := &Client{
		client: client,
		owner:  "test",
		repo:   "repo",
		cache:  NewAPICache(5 * time.Minute),
	}

	// Test getting PR reviews
	reviews, err := ghClient.GetPRReviews(context.Background(), 123)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify reviews
	if len(reviews) != 1 {
		t.Errorf("Expected 1 review, got: %d", len(reviews))
	}

	review := reviews[0]
	if review.ID != 1 {
		t.Errorf("Expected review ID 1, got: %d", review.ID)
	}

	if review.Reviewer != "reviewer1" {
		t.Errorf("Expected reviewer 'reviewer1', got: %s", review.Reviewer)
	}

	if review.State != "APPROVED" {
		t.Errorf("Expected state 'APPROVED', got: %s", review.State)
	}

	if review.Body != "Looks good!" {
		t.Errorf("Expected body 'Looks good!', got: %s", review.Body)
	}

	// Verify comments - now includes all PR comments (including self-review comment)
	if len(review.Comments) != 2 {
		t.Errorf("Expected 2 comments, got: %d", len(review.Comments))
	}

	// Find the reviewer1 comment
	var reviewerComment *Comment
	for i, c := range review.Comments {
		if c.Author == "reviewer1" {
			reviewerComment = &review.Comments[i]
			break
		}
	}

	if reviewerComment == nil {
		t.Fatal("Expected to find comment from reviewer1")
	}

	if reviewerComment.ID != 1 {
		t.Errorf("Expected comment ID 1, got: %d", reviewerComment.ID)
	}

	if reviewerComment.File != "test.go" {
		t.Errorf("Expected file 'test.go', got: %s", reviewerComment.File)
	}

	if reviewerComment.Line != 10 {
		t.Errorf("Expected line 10, got: %d", reviewerComment.Line)
	}

	if reviewerComment.Body != "Fix this issue" {
		t.Errorf("Expected body 'Fix this issue', got: %s", reviewerComment.Body)
	}
}

// TestClient_GetSelfReviews tests self-review retrieval
func TestClient_GetSelfReviews(t *testing.T) {
	server := MockGitHubServer(t)
	defer server.Close()

	// Create client with mock server
	client := github.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")

	ghClient := &Client{
		client: client,
		owner:  "test",
		repo:   "repo",
		cache:  NewAPICache(5 * time.Minute),
	}

	// Test getting self-reviews for PR author "testuser"
	selfReviews, err := ghClient.GetSelfReviews(context.Background(), 123, "testuser")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify we got a self-review
	if len(selfReviews) != 1 {
		t.Fatalf("Expected 1 self-review, got: %d", len(selfReviews))
	}

	selfReview := selfReviews[0]
	
	// Verify self-review properties
	if selfReview.ID != -1 {
		t.Errorf("Expected self-review ID -1, got: %d", selfReview.ID)
	}

	if selfReview.Reviewer != "testuser" {
		t.Errorf("Expected reviewer 'testuser', got: %s", selfReview.Reviewer)
	}

	if selfReview.State != "COMMENTED" {
		t.Errorf("Expected state 'COMMENTED', got: %s", selfReview.State)
	}

	// Verify we have 2 comments (1 issue comment + 1 PR review comment)
	if len(selfReview.Comments) != 2 {
		t.Fatalf("Expected 2 comments, got: %d", len(selfReview.Comments))
	}

	// Verify issue comment
	issueComment := selfReview.Comments[0]
	if issueComment.ID != 3 {
		t.Errorf("Expected issue comment ID 3, got: %d", issueComment.ID)
	}
	if issueComment.Body != "I noticed a potential issue with the error handling" {
		t.Errorf("Expected specific body, got: %s", issueComment.Body)
	}
	if issueComment.Author != "testuser" {
		t.Errorf("Expected author 'testuser', got: %s", issueComment.Author)
	}
	if issueComment.File != "" {
		t.Errorf("Expected empty file for issue comment, got: %s", issueComment.File)
	}
	if issueComment.Line != 0 {
		t.Errorf("Expected line 0 for issue comment, got: %d", issueComment.Line)
	}

	// Verify PR review comment
	prComment := selfReview.Comments[1]
	if prComment.ID != 2 {
		t.Errorf("Expected PR comment ID 2, got: %d", prComment.ID)
	}
	if prComment.Body != "TODO: Add error handling here" {
		t.Errorf("Expected specific body, got: %s", prComment.Body)
	}
	if prComment.Author != "testuser" {
		t.Errorf("Expected author 'testuser', got: %s", prComment.Author)
	}
	if prComment.File != "main.go" {
		t.Errorf("Expected file 'main.go', got: %s", prComment.File)
	}
	if prComment.Line != 20 {
		t.Errorf("Expected line 20, got: %d", prComment.Line)
	}
}

// TestClient_GetSelfReviews_NoComments tests self-review with no comments
func TestClient_GetSelfReviews_NoComments(t *testing.T) {
	server := MockGitHubServer(t)
	defer server.Close()

	// Create client with mock server
	client := github.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")

	ghClient := &Client{
		client: client,
		owner:  "test",
		repo:   "repo",
		cache:  NewAPICache(5 * time.Minute),
	}

	// Test getting self-reviews for a different user (no self-comments expected)
	selfReviews, err := ghClient.GetSelfReviews(context.Background(), 123, "otheruser")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify we got no self-reviews
	if len(selfReviews) != 0 {
		t.Errorf("Expected 0 self-reviews, got: %d", len(selfReviews))
	}
}

// MockClient creates a mock GitHub client for testing
type MockClient struct {
	prInfo    *PRInfo
	prInfoMap map[int]*PRInfo // Support multiple PRs
	reviews   []Review
	prNumber  int
	err       error
}

func NewMockClient() *MockClient {
	return &MockClient{
		prInfoMap: make(map[int]*PRInfo),
	}
}

func (m *MockClient) SetPRInfo(prNumber int, prInfo *PRInfo) {
	m.prInfoMap[prNumber] = prInfo
	// Keep backward compatibility
	m.prInfo = prInfo
}

func (m *MockClient) SetReviews(reviews []Review) {
	m.reviews = reviews
}

func (m *MockClient) SetCurrentBranchPR(prNumber int) {
	m.prNumber = prNumber
}

func (m *MockClient) SetError(err error) {
	m.err = err
}

func (m *MockClient) GetPRInfo(ctx context.Context, prNumber int) (*PRInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Check map first
	if pr, ok := m.prInfoMap[prNumber]; ok {
		return pr, nil
	}
	// Fallback to single PR for backward compatibility
	return m.prInfo, nil
}

func (m *MockClient) GetPRReviews(ctx context.Context, prNumber int) ([]Review, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.reviews, nil
}

func (m *MockClient) GetSelfReviews(ctx context.Context, prNumber int, prAuthor string) ([]Review, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Return empty array by default for MockClient
	return []Review{}, nil
}

func (m *MockClient) GetCurrentBranchPR(ctx context.Context) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.prNumber, nil
}

func (m *MockClient) IsPROpen(ctx context.Context, prNumber int) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	// Check map first
	if pr, ok := m.prInfoMap[prNumber]; ok {
		return pr.State == "open", nil
	}
	// If not found, return error
	return false, fmt.Errorf("PR #%d not found", prNumber)
}

// TestMockClient tests the mock client functionality
func TestMockClient(t *testing.T) {
	mockClient := NewMockClient()

	// Test PR info
	expectedPRInfo := &PRInfo{
		Number: 123,
		Title:  "Test PR",
		Author: "testuser",
		Branch: "feature/test",
	}

	mockClient.SetPRInfo(123, expectedPRInfo)

	prInfo, err := mockClient.GetPRInfo(context.Background(), 123)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if prInfo.Number != expectedPRInfo.Number {
		t.Errorf("Expected PR number %d, got: %d", expectedPRInfo.Number, prInfo.Number)
	}

	if prInfo.Branch != expectedPRInfo.Branch {
		t.Errorf("Expected branch %s, got: %s", expectedPRInfo.Branch, prInfo.Branch)
	}

	// Test reviews
	expectedReviews := []Review{
		{
			ID:       1,
			Reviewer: "reviewer1",
			State:    "APPROVED",
			Comments: []Comment{
				{
					ID:   1,
					File: "test.go",
					Line: 10,
					Body: "Fix this",
				},
			},
		},
	}

	mockClient.SetReviews(expectedReviews)

	reviews, err := mockClient.GetPRReviews(context.Background(), 123)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(reviews) != 1 {
		t.Errorf("Expected 1 review, got: %d", len(reviews))
	}

	if reviews[0].ID != expectedReviews[0].ID {
		t.Errorf("Expected review ID %d, got: %d", expectedReviews[0].ID, reviews[0].ID)
	}
}

// Integration test showing how mock client can be used
func TestIntegrationWithMockClient(t *testing.T) {
	mockClient := NewMockClient()

	// Set up test data
	prInfo := &PRInfo{
		Number: 123,
		Title:  "Feature: Add branch support",
		Author: "developer",
		Branch: "feature/branch-support",
		State:  "open",
	}

	reviews := []Review{
		{
			ID:       1,
			Reviewer: "reviewer1",
			State:    "CHANGES_REQUESTED",
			Body:     "Please add tests",
			Comments: []Comment{
				{
					ID:   100,
					File: "main.go",
					Line: 25,
					Body: "Add error handling here",
				},
				{
					ID:   101,
					File: "main.go",
					Line: 30,
					Body: "Consider using a constant",
				},
			},
		},
	}

	mockClient.SetPRInfo(123, prInfo)
	mockClient.SetReviews(reviews)
	mockClient.SetCurrentBranchPR(123)

	// Test the workflow
	ctx := context.Background()

	// 1. Get current branch PR
	prNumber, err := mockClient.GetCurrentBranchPR(ctx)
	if err != nil {
		t.Fatalf("Failed to get current branch PR: %v", err)
	}

	if prNumber != 123 {
		t.Errorf("Expected PR number 123, got: %d", prNumber)
	}

	// 2. Get PR info
	retrievedPRInfo, err := mockClient.GetPRInfo(ctx, prNumber)
	if err != nil {
		t.Fatalf("Failed to get PR info: %v", err)
	}

	if retrievedPRInfo.Branch != "feature/branch-support" {
		t.Errorf("Expected branch 'feature/branch-support', got: %s", retrievedPRInfo.Branch)
	}

	// 3. Get reviews
	retrievedReviews, err := mockClient.GetPRReviews(ctx, prNumber)
	if err != nil {
		t.Fatalf("Failed to get PR reviews: %v", err)
	}

	if len(retrievedReviews) != 1 {
		t.Errorf("Expected 1 review, got: %d", len(retrievedReviews))
	}

	if len(retrievedReviews[0].Comments) != 2 {
		t.Errorf("Expected 2 comments, got: %d", len(retrievedReviews[0].Comments))
	}
}

// TestMockClient_NoPRFound tests the ErrNoPRFound error scenario
func TestMockClient_NoPRFound(t *testing.T) {
	mockClient := NewMockClient()

	// Set up the error
	mockClient.SetError(ErrNoPRFound)

	// Test GetCurrentBranchPR returns ErrNoPRFound
	_, err := mockClient.GetCurrentBranchPR(context.Background())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check if it's the correct error type
	if !errors.Is(err, ErrNoPRFound) {
		t.Errorf("Expected ErrNoPRFound, got: %v", err)
	}
}

// TestMockClient_IsPROpen tests the IsPROpen method
func TestMockClient_IsPROpen(t *testing.T) {
	mockClient := NewMockClient()
	ctx := context.Background()

	// Set up test data
	openPR := &PRInfo{
		Number: 1,
		Title:  "Open PR",
		State:  "open",
	}
	closedPR := &PRInfo{
		Number: 2,
		Title:  "Closed PR",
		State:  "closed",
	}
	mergedPR := &PRInfo{
		Number: 3,
		Title:  "Merged PR",
		State:  "closed", // GitHub uses "closed" for merged PRs too
	}

	mockClient.SetPRInfo(1, openPR)
	mockClient.SetPRInfo(2, closedPR)
	mockClient.SetPRInfo(3, mergedPR)

	// Test open PR
	isOpen, err := mockClient.IsPROpen(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to check if PR is open: %v", err)
	}
	if !isOpen {
		t.Error("Expected PR 1 to be open")
	}

	// Test closed PR
	isOpen, err = mockClient.IsPROpen(ctx, 2)
	if err != nil {
		t.Fatalf("Failed to check if PR is open: %v", err)
	}
	if isOpen {
		t.Error("Expected PR 2 to be closed")
	}

	// Test merged PR
	isOpen, err = mockClient.IsPROpen(ctx, 3)
	if err != nil {
		t.Fatalf("Failed to check if PR is open: %v", err)
	}
	if isOpen {
		t.Error("Expected PR 3 to be closed")
	}

	// Test non-existent PR
	_, err = mockClient.IsPROpen(ctx, 999)
	if err == nil {
		t.Error("Expected error for non-existent PR")
	}
}
