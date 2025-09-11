package github

import (
	"context"
	"errors"
	"testing"
)

// This file demonstrates how to use the mock system for testing

func TestExample_UsingMockClient_BasicScenario(t *testing.T) {
	// Example 1: Using the mock client directly
	t.Run("Direct mock usage", func(t *testing.T) {
		mockClient := NewMockGitHubClient()
		
		// Setup test data
		prInfo := &PRInfo{
			Number: 123,
			Title:  "Test PR",
			Author: "developer",
			State:  "open",
		}
		mockClient.SetPRInfo(123, prInfo)
		mockClient.SetCurrentBranchPR(123)
		
		// Test the functionality
		prNumber, err := mockClient.GetCurrentBranchPR(context.Background())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if prNumber != 123 {
			t.Errorf("Expected PR number 123, got %d", prNumber)
		}
		
		retrievedInfo, err := mockClient.GetPRInfo(context.Background(), prNumber)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if retrievedInfo.Title != "Test PR" {
			t.Errorf("Expected title 'Test PR', got %s", retrievedInfo.Title)
		}
	})
	
	// Example 2: Using the test helper with predefined scenarios
	t.Run("Test helper with scenarios", func(t *testing.T) {
		err := WithMockClient(TestScenarioBasicPR, func(client GitHubClientInterface) error {
			prNumber, err := client.GetCurrentBranchPR(context.Background())
			if err != nil {
				return err
			}
			
			prInfo, err := client.GetPRInfo(context.Background(), prNumber)
			if err != nil {
				return err
			}
			
			if prInfo.Number != 123 {
				t.Errorf("Expected PR number 123, got %d", prInfo.Number)
			}
			
			return nil
		})
		
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
	})
	
	// Example 3: Testing error scenarios
	t.Run("Error scenario testing", func(t *testing.T) {
		err := WithMockClient(TestScenarioNoPR, func(client GitHubClientInterface) error {
			_, err := client.GetCurrentBranchPR(context.Background())
			
			// We expect this specific error
			if !errors.Is(err, ErrNoPRFound) {
				t.Errorf("Expected ErrNoPRFound, got: %v", err)
			}
			
			return nil // Test passed - we got the expected error
		})
		
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
	})
	
	// Example 4: Testing with call tracking
	t.Run("Call tracking", func(t *testing.T) {
		mockClient := NewMockGitHubClient()
		mockClient.SetCurrentBranchPR(123)
		
		prInfo := &PRInfo{Number: 123, Title: "Test"}
		mockClient.SetPRInfo(123, prInfo)
		
		// Make multiple calls
		mockClient.GetCurrentBranchPR(context.Background())
		mockClient.GetPRInfo(context.Background(), 123)
		mockClient.GetPRInfo(context.Background(), 456)
		
		// Verify call tracking
		if mockClient.GetCurrentBranchPRCalls != 1 {
			t.Errorf("Expected 1 GetCurrentBranchPR call, got %d", mockClient.GetCurrentBranchPRCalls)
		}
		
		if len(mockClient.GetPRInfoCalls) != 2 {
			t.Errorf("Expected 2 GetPRInfo calls, got %d", len(mockClient.GetPRInfoCalls))
		}
		
		expectedCalls := []int{123, 456}
		for i, expected := range expectedCalls {
			if mockClient.GetPRInfoCalls[i] != expected {
				t.Errorf("Expected call %d to be %d, got %d", i, expected, mockClient.GetPRInfoCalls[i])
			}
		}
	})
}

func TestExample_UsingMockAuthProvider(t *testing.T) {
	t.Run("Mock auth provider", func(t *testing.T) {
		authProvider := NewMockAuthTokenProvider()
		authProvider.SetToken("test-token")
		authProvider.SetSource("test-source")
		
		repoProvider := NewMockRepoInfoProvider()
		repoProvider.SetRepoInfo("test-owner", "test-repo")
		
		// Create a real GitHub client with mocked dependencies
		client, err := NewClientWithProviders(authProvider, repoProvider)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		
		// Verify the client was created with correct settings
		if client.owner != "test-owner" {
			t.Errorf("Expected owner 'test-owner', got %s", client.owner)
		}
		
		if client.repo != "test-repo" {
			t.Errorf("Expected repo 'test-repo', got %s", client.repo)
		}
	})
	
	t.Run("Auth provider error", func(t *testing.T) {
		authProvider := NewMockAuthTokenProvider()
		authProvider.SetError(errors.New("auth failed"))
		
		repoProvider := NewMockRepoInfoProvider()
		
		_, err := NewClientWithProviders(authProvider, repoProvider)
		if err == nil {
			t.Error("Expected error from auth provider")
		}
		
		if !contains(err.Error(), "failed to get GitHub token") {
			t.Errorf("Expected auth error, got: %v", err)
		}
	})
}

func TestExample_ComplexWorkflow(t *testing.T) {
	// This demonstrates a more complex workflow test
	t.Run("Complete PR workflow", func(t *testing.T) {
		helper := NewTestHelper()
		
		// Setup a comprehensive scenario
		prInfo := &PRInfo{
			Number:     123,
			Title:      "Add new feature",
			Author:     "developer",
			State:      "open",
			Repository: "owner/repo",
			Branch:     "feature/new-feature",
		}
		
		reviews := []Review{
			{
				ID:       1,
				Reviewer: "reviewer1",
				State:    "CHANGES_REQUESTED",
				Body:     "Please address these issues",
				Comments: []Comment{
					{
						ID:     100,
						File:   "main.go",
						Line:   10,
						Body:   "Add error handling here",
						Author: "reviewer1",
					},
					{
						ID:     101,
						File:   "test.go",
						Line:   5,
						Body:   "Missing test case",
						Author: "reviewer1",
					},
				},
			},
			{
				ID:       2,
				Reviewer: "reviewer2",
				State:    "APPROVED",
				Body:     "Looks good overall",
				Comments: []Comment{
					{
						ID:     102,
						File:   "main.go",
						Line:   20,
						Body:   "Nice implementation",
						Author: "reviewer2",
					},
				},
			},
		}
		
		selfReviews := []Review{
			{
				ID:       -1,
				Reviewer: "developer",
				State:    "COMMENTED",
				Comments: []Comment{
					{
						ID:     200,
						Body:   "Updated based on feedback",
						Author: "developer",
					},
				},
			},
		}
		
		helper.MockClient.SetPRInfo(123, prInfo)
		helper.MockClient.SetCurrentBranchPR(123)
		helper.MockClient.SetReviews(reviews)
		helper.MockClient.SetSelfReviews(selfReviews)
		
		client := helper.MockClient
		
		// Test the complete workflow
		prNumber, err := client.GetCurrentBranchPR(context.Background())
		if err != nil {
			t.Fatalf("Failed to get current branch PR: %v", err)
		}
		
		retrievedPRInfo, err := client.GetPRInfo(context.Background(), prNumber)
		if err != nil {
			t.Fatalf("Failed to get PR info: %v", err)
		}
		
		retrievedReviews, err := client.GetPRReviews(context.Background(), prNumber)
		if err != nil {
			t.Fatalf("Failed to get PR reviews: %v", err)
		}
		
		retrievedSelfReviews, err := client.GetSelfReviews(context.Background(), prNumber, retrievedPRInfo.Author)
		if err != nil {
			t.Fatalf("Failed to get self reviews: %v", err)
		}
		
		isOpen, err := client.IsPROpen(context.Background(), prNumber)
		if err != nil {
			t.Fatalf("Failed to check if PR is open: %v", err)
		}
		
		// Verify the results
		if prNumber != 123 {
			t.Errorf("Expected PR number 123, got %d", prNumber)
		}
		
		if retrievedPRInfo.Title != "Add new feature" {
			t.Errorf("Expected title 'Add new feature', got %s", retrievedPRInfo.Title)
		}
		
		if len(retrievedReviews) != 2 {
			t.Errorf("Expected 2 reviews, got %d", len(retrievedReviews))
		}
		
		if len(retrievedSelfReviews) != 1 {
			t.Errorf("Expected 1 self-review, got %d", len(retrievedSelfReviews))
		}
		
		if !isOpen {
			t.Error("Expected PR to be open")
		}
		
		// Count total comments across all reviews
		totalComments := 0
		for _, review := range retrievedReviews {
			totalComments += len(review.Comments)
		}
		
		if totalComments != 3 { // 2 from reviewer1 + 1 from reviewer2
			t.Errorf("Expected 3 total comments, got %d", totalComments)
		}
	})
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		   (len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}