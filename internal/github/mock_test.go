package github

import (
	"context"
	"errors"
	"testing"
)

func TestMockGitHubClient(t *testing.T) {
	mockClient := NewMockGitHubClient()

	t.Run("Default behavior", func(t *testing.T) {
		// Test no PR found
		_, err := mockClient.GetCurrentBranchPR(context.Background())
		if !errors.Is(err, ErrNoPRFound) {
			t.Errorf("Expected ErrNoPRFound, got: %v", err)
		}

		// Test no PR info
		_, err = mockClient.GetPRInfo(context.Background(), 123)
		if err == nil {
			t.Error("Expected error for non-existent PR")
		}

		// Test empty reviews
		reviews, err := mockClient.GetPRReviews(context.Background(), 123)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(reviews) != 0 {
			t.Errorf("Expected empty reviews, got %d", len(reviews))
		}
	})

	t.Run("Configuration and retrieval", func(t *testing.T) {
		// Configure mock data
		prInfo := &PRInfo{
			Number: 123,
			Title:  "Test PR",
			Author: "testuser",
			State:  "open",
		}
		mockClient.SetPRInfo(123, prInfo)
		mockClient.SetCurrentBranchPR(123)

		reviews := []Review{
			{
				ID:       1,
				Reviewer: "reviewer1",
				State:    "APPROVED",
				Body:     "Looks good!",
			},
		}
		mockClient.SetReviews(reviews)

		selfReviews := []Review{
			{
				ID:       -1,
				Reviewer: "testuser",
				State:    "COMMENTED",
			},
		}
		mockClient.SetSelfReviews(selfReviews)

		// Test configured data
		prNumber, err := mockClient.GetCurrentBranchPR(context.Background())
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if prNumber != 123 {
			t.Errorf("Expected PR number 123, got %d", prNumber)
		}

		retrievedPRInfo, err := mockClient.GetPRInfo(context.Background(), 123)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if retrievedPRInfo.Number != 123 {
			t.Errorf("Expected PR number 123, got %d", retrievedPRInfo.Number)
		}

		retrievedReviews, err := mockClient.GetPRReviews(context.Background(), 123)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(retrievedReviews) != 1 {
			t.Errorf("Expected 1 review, got %d", len(retrievedReviews))
		}

		retrievedSelfReviews, err := mockClient.GetSelfReviews(context.Background(), 123, "testuser")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(retrievedSelfReviews) != 1 {
			t.Errorf("Expected 1 self-review, got %d", len(retrievedSelfReviews))
		}

		isOpen, err := mockClient.IsPROpen(context.Background(), 123)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !isOpen {
			t.Error("Expected PR to be open")
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		mockClient.Reset()

		// Configure errors
		expectedError := errors.New("test error")
		mockClient.SetGetPRInfoError(expectedError)
		mockClient.SetGetPRReviewsError(expectedError)
		mockClient.SetGetSelfReviewsError(expectedError)
		mockClient.SetGetCurrentBranchPRError(expectedError)
		mockClient.SetIsPROpenError(expectedError)

		// Test errors are returned
		_, err := mockClient.GetPRInfo(context.Background(), 123)
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}

		_, err = mockClient.GetPRReviews(context.Background(), 123)
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}

		_, err = mockClient.GetSelfReviews(context.Background(), 123, "user")
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}

		_, err = mockClient.GetCurrentBranchPR(context.Background())
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}

		_, err = mockClient.IsPROpen(context.Background(), 123)
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}
	})

	t.Run("Call tracking", func(t *testing.T) {
		mockClient.Reset()

		// Make some calls
		mockClient.GetPRInfo(context.Background(), 123)
		mockClient.GetPRInfo(context.Background(), 456)
		mockClient.GetPRReviews(context.Background(), 123)
		mockClient.GetSelfReviews(context.Background(), 123, "user1")
		mockClient.GetSelfReviews(context.Background(), 456, "user2")
		mockClient.GetCurrentBranchPR(context.Background())
		mockClient.IsPROpen(context.Background(), 123)

		// Verify call tracking
		if len(mockClient.GetPRInfoCalls) != 2 {
			t.Errorf("Expected 2 GetPRInfo calls, got %d", len(mockClient.GetPRInfoCalls))
		}
		if mockClient.GetPRInfoCalls[0] != 123 || mockClient.GetPRInfoCalls[1] != 456 {
			t.Errorf("Unexpected GetPRInfo call parameters: %v", mockClient.GetPRInfoCalls)
		}

		if len(mockClient.GetPRReviewsCalls) != 1 {
			t.Errorf("Expected 1 GetPRReviews call, got %d", len(mockClient.GetPRReviewsCalls))
		}

		if len(mockClient.GetSelfReviewsCalls) != 2 {
			t.Errorf("Expected 2 GetSelfReviews calls, got %d", len(mockClient.GetSelfReviewsCalls))
		}

		if mockClient.GetCurrentBranchPRCalls != 1 {
			t.Errorf("Expected 1 GetCurrentBranchPR call, got %d", mockClient.GetCurrentBranchPRCalls)
		}

		if len(mockClient.IsPROpenCalls) != 1 {
			t.Errorf("Expected 1 IsPROpen call, got %d", len(mockClient.IsPROpenCalls))
		}
	})
}

func TestMockAuthTokenProvider(t *testing.T) {
	provider := NewMockAuthTokenProvider()

	t.Run("Default behavior", func(t *testing.T) {
		token, err := provider.GetToken()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if token != "" {
			t.Errorf("Expected empty token, got: %s", token)
		}

		source, token, err := provider.GetTokenWithSource()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if source != "" || token != "" {
			t.Errorf("Expected empty source and token, got: %s, %s", source, token)
		}
	})

	t.Run("Configured values", func(t *testing.T) {
		provider.SetToken("test-token")
		provider.SetSource("test-source")

		token, err := provider.GetToken()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if token != "test-token" {
			t.Errorf("Expected 'test-token', got: %s", token)
		}

		source, token, err := provider.GetTokenWithSource()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if source != "test-source" {
			t.Errorf("Expected 'test-source', got: %s", source)
		}
		if token != "test-token" {
			t.Errorf("Expected 'test-token', got: %s", token)
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		expectedError := errors.New("test error")
		provider.SetError(expectedError)
		provider.SetSourceError(expectedError)

		_, err := provider.GetToken()
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}

		_, _, err = provider.GetTokenWithSource()
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}
	})
}

func TestMockRepoInfoProvider(t *testing.T) {
	provider := NewMockRepoInfoProvider()

	t.Run("Default behavior", func(t *testing.T) {
		owner, repo, err := provider.GetRepoInfo()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if owner != "" || repo != "" {
			t.Errorf("Expected empty owner and repo, got: %s, %s", owner, repo)
		}
	})

	t.Run("Configured values", func(t *testing.T) {
		provider.SetRepoInfo("test-owner", "test-repo")

		owner, repo, err := provider.GetRepoInfo()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if owner != "test-owner" {
			t.Errorf("Expected 'test-owner', got: %s", owner)
		}
		if repo != "test-repo" {
			t.Errorf("Expected 'test-repo', got: %s", repo)
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		expectedError := errors.New("test error")
		provider.SetError(expectedError)

		_, _, err := provider.GetRepoInfo()
		if !errors.Is(err, expectedError) {
			t.Errorf("Expected configured error, got: %v", err)
		}
	})
}

func TestNewClientWithProviders(t *testing.T) {
	t.Run("Successful client creation", func(t *testing.T) {
		authProvider := NewMockAuthTokenProvider()
		authProvider.SetToken("test-token")

		repoProvider := NewMockRepoInfoProvider()
		repoProvider.SetRepoInfo("test-owner", "test-repo")

		client, err := NewClientWithProviders(authProvider, repoProvider)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if client == nil {
			t.Error("Expected client to be created")
		}

		if client.owner != "test-owner" {
			t.Errorf("Expected owner 'test-owner', got: %s", client.owner)
		}

		if client.repo != "test-repo" {
			t.Errorf("Expected repo 'test-repo', got: %s", client.repo)
		}
	})

	t.Run("Auth provider error", func(t *testing.T) {
		authProvider := NewMockAuthTokenProvider()
		authProvider.SetError(errors.New("auth error"))

		repoProvider := NewMockRepoInfoProvider()
		repoProvider.SetRepoInfo("test-owner", "test-repo")

		_, err := NewClientWithProviders(authProvider, repoProvider)
		if err == nil {
			t.Error("Expected error from auth provider")
		}
	})

	t.Run("Repo provider error", func(t *testing.T) {
		authProvider := NewMockAuthTokenProvider()
		authProvider.SetToken("test-token")

		repoProvider := NewMockRepoInfoProvider()
		repoProvider.SetError(errors.New("repo error"))

		_, err := NewClientWithProviders(authProvider, repoProvider)
		if err == nil {
			t.Error("Expected error from repo provider")
		}
	})
}