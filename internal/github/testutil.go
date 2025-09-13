package github

import "context"

// TestHelper provides utilities for testing with GitHub client mocks
type TestHelper struct {
	MockClient   *MockGitHubClient
	AuthProvider *MockAuthTokenProvider
	RepoProvider *MockRepoInfoProvider
}

// NewTestHelper creates a new test helper with mock implementations
func NewTestHelper() *TestHelper {
	return &TestHelper{
		MockClient:   NewMockGitHubClient(),
		AuthProvider: NewMockAuthTokenProvider(),
		RepoProvider: NewMockRepoInfoProvider(),
	}
}

// CreateMockClient creates a real GitHub client with mocked dependencies
func (h *TestHelper) CreateMockClient() (*Client, error) {
	// Set default values if not configured
	if h.AuthProvider.token == "" {
		h.AuthProvider.SetToken("mock-token")
	}
	if h.RepoProvider.owner == "" || h.RepoProvider.repo == "" {
		h.RepoProvider.SetRepoInfo("mock-owner", "mock-repo")
	}

	return NewClientWithProviders(h.AuthProvider, h.RepoProvider)
}

// SetupBasicPR configures the mock with a basic PR scenario
func (h *TestHelper) SetupBasicPR() {
	prInfo := &PRInfo{
		Number:     123,
		Title:      "Test PR",
		Author:     "testuser",
		State:      "open",
		Repository: "mock-owner/mock-repo",
		Branch:     "feature/test",
	}

	reviews := []Review{
		{
			ID:       1,
			Reviewer: "reviewer1",
			State:    "APPROVED",
			Body:     "Looks good!",
			Comments: []Comment{
				{
					ID:     100,
					File:   "main.go",
					Line:   10,
					Body:   "Nice implementation",
					Author: "reviewer1",
				},
			},
		},
	}

	h.MockClient.SetPRInfo(123, prInfo)
	h.MockClient.SetReviews(reviews)
	h.MockClient.SetCurrentBranchPR(123)
}

// SetupErrorScenario configures the mock to return errors
func (h *TestHelper) SetupErrorScenario() {
	h.MockClient.SetGetCurrentBranchPRError(ErrNoPRFound)
	h.MockClient.SetGetPRInfoError(ErrNoPRFound)
}

// Reset resets all mocks to their initial state
func (h *TestHelper) Reset() {
	h.MockClient.Reset()
	h.AuthProvider = NewMockAuthTokenProvider()
	h.RepoProvider = NewMockRepoInfoProvider()
}

// WithClientFunction is a type for functions that take a GitHub client interface
type WithClientFunction func(client GitHubClientInterface) error

// WithMockClient executes a function with a mock GitHub client
func WithMockClient(setup func(*TestHelper), fn WithClientFunction) error {
	helper := NewTestHelper()
	if setup != nil {
		setup(helper)
	}
	return fn(helper.MockClient)
}

// WithRealMockClient executes a function with a real GitHub client using mocked dependencies
func WithRealMockClient(setup func(*TestHelper), fn WithClientFunction) error {
	helper := NewTestHelper()
	if setup != nil {
		setup(helper)
	}

	client, err := helper.CreateMockClient()
	if err != nil {
		return err
	}

	return fn(client)
}

// Common test scenarios

// TestScenarioNoPR sets up a scenario with no PR found
func TestScenarioNoPR(helper *TestHelper) {
	helper.MockClient.SetGetCurrentBranchPRError(ErrNoPRFound)
}

// TestScenarioBasicPR sets up a scenario with a basic PR
func TestScenarioBasicPR(helper *TestHelper) {
	helper.SetupBasicPR()
}

// TestScenarioAuthError sets up a scenario with authentication error
func TestScenarioAuthError(helper *TestHelper) {
	helper.AuthProvider.SetError(ErrNoPRFound)
}

// TestScenarioRepoError sets up a scenario with repository information error
func TestScenarioRepoError(helper *TestHelper) {
	helper.RepoProvider.SetError(ErrNoPRFound)
}

// Interface assertion to ensure MockGitHubClient implements GitHubClientInterface
var _ GitHubClientInterface = (*MockGitHubClient)(nil)

// Example usage functions for documentation

// ExampleBasicUsage demonstrates basic usage of the test helper
func ExampleBasicUsage() error {
	return WithMockClient(TestScenarioBasicPR, func(client GitHubClientInterface) error {
		prNumber, err := client.GetCurrentBranchPR(context.Background())
		if err != nil {
			return err
		}

		prInfo, err := client.GetPRInfo(context.Background(), prNumber)
		if err != nil {
			return err
		}

		_ = prInfo // Use the PR info
		return nil
	})
}

// ExampleErrorHandling demonstrates error handling with the test helper
func ExampleErrorHandling() error {
	return WithMockClient(TestScenarioNoPR, func(client GitHubClientInterface) error {
		_, err := client.GetCurrentBranchPR(context.Background())
		if err == ErrNoPRFound {
			// Expected error, handle appropriately
			return nil
		}
		return err
	})
}
