package github

import (
	"context"
	"fmt"
)

// MockGitHubClient provides a mock implementation for testing
type MockGitHubClient struct {
	// Data to return
	prInfo      *PRInfo
	prInfoMap   map[int]*PRInfo // Support multiple PRs
	reviews     []Review
	selfReviews []Review
	prNumber    int

	// Error control
	getPRInfoError          error
	getPRReviewsError       error
	getSelfReviewsError     error
	getCurrentBranchPRError error
	isPROpenError           error

	// Call tracking
	GetPRInfoCalls      []int
	GetPRReviewsCalls   []int
	GetSelfReviewsCalls []struct {
		PRNumber int
		Author   string
	}
	GetCurrentBranchPRCalls int
	IsPROpenCalls           []int
}

// NewMockGitHubClient creates a new mock client
func NewMockGitHubClient() *MockGitHubClient {
	return &MockGitHubClient{
		prInfoMap: make(map[int]*PRInfo),
	}
}

// Configuration methods

func (m *MockGitHubClient) SetPRInfo(prNumber int, prInfo *PRInfo) {
	m.prInfoMap[prNumber] = prInfo
	// Keep backward compatibility
	m.prInfo = prInfo
}

func (m *MockGitHubClient) SetReviews(reviews []Review) {
	m.reviews = reviews
}

func (m *MockGitHubClient) SetSelfReviews(selfReviews []Review) {
	m.selfReviews = selfReviews
}

func (m *MockGitHubClient) SetCurrentBranchPR(prNumber int) {
	m.prNumber = prNumber
}

// Error configuration methods

func (m *MockGitHubClient) SetGetPRInfoError(err error) {
	m.getPRInfoError = err
}

func (m *MockGitHubClient) SetGetPRReviewsError(err error) {
	m.getPRReviewsError = err
}

func (m *MockGitHubClient) SetGetSelfReviewsError(err error) {
	m.getSelfReviewsError = err
}

func (m *MockGitHubClient) SetGetCurrentBranchPRError(err error) {
	m.getCurrentBranchPRError = err
}

func (m *MockGitHubClient) SetIsPROpenError(err error) {
	m.isPROpenError = err
}

// GitHubClientInterface implementation

func (m *MockGitHubClient) GetCurrentBranchPR(ctx context.Context) (int, error) {
	m.GetCurrentBranchPRCalls++
	if m.getCurrentBranchPRError != nil {
		return 0, m.getCurrentBranchPRError
	}
	if m.prNumber == 0 {
		return 0, ErrNoPRFound
	}
	return m.prNumber, nil
}

func (m *MockGitHubClient) GetPRInfo(ctx context.Context, prNumber int) (*PRInfo, error) {
	m.GetPRInfoCalls = append(m.GetPRInfoCalls, prNumber)
	if m.getPRInfoError != nil {
		return nil, m.getPRInfoError
	}
	// Check map first
	if pr, ok := m.prInfoMap[prNumber]; ok {
		return pr, nil
	}
	// Fallback to single PR for backward compatibility
	if m.prInfo != nil {
		return m.prInfo, nil
	}
	return nil, fmt.Errorf("PR #%d not found", prNumber)
}

func (m *MockGitHubClient) GetPRReviews(ctx context.Context, prNumber int) ([]Review, error) {
	m.GetPRReviewsCalls = append(m.GetPRReviewsCalls, prNumber)
	if m.getPRReviewsError != nil {
		return nil, m.getPRReviewsError
	}
	return m.reviews, nil
}

func (m *MockGitHubClient) GetSelfReviews(ctx context.Context, prNumber int, prAuthor string) ([]Review, error) {
	m.GetSelfReviewsCalls = append(m.GetSelfReviewsCalls, struct {
		PRNumber int
		Author   string
	}{prNumber, prAuthor})
	if m.getSelfReviewsError != nil {
		return nil, m.getSelfReviewsError
	}
	return m.selfReviews, nil
}

func (m *MockGitHubClient) IsPROpen(ctx context.Context, prNumber int) (bool, error) {
	m.IsPROpenCalls = append(m.IsPROpenCalls, prNumber)
	if m.isPROpenError != nil {
		return false, m.isPROpenError
	}
	// Check map first
	if pr, ok := m.prInfoMap[prNumber]; ok {
		return pr.State == "open", nil
	}
	// If not found, return error
	return false, fmt.Errorf("PR #%d not found", prNumber)
}

// Helper methods for testing

func (m *MockGitHubClient) Reset() {
	m.prInfo = nil
	m.prInfoMap = make(map[int]*PRInfo)
	m.reviews = nil
	m.selfReviews = nil
	m.prNumber = 0

	m.getPRInfoError = nil
	m.getPRReviewsError = nil
	m.getSelfReviewsError = nil
	m.getCurrentBranchPRError = nil
	m.isPROpenError = nil

	m.GetPRInfoCalls = nil
	m.GetPRReviewsCalls = nil
	m.GetSelfReviewsCalls = nil
	m.GetCurrentBranchPRCalls = 0
	m.IsPROpenCalls = nil
}

// MockAuthTokenProvider provides a mock implementation for authentication
type MockAuthTokenProvider struct {
	token     string
	source    string
	err       error
	sourceErr error
}

func NewMockAuthTokenProvider() *MockAuthTokenProvider {
	return &MockAuthTokenProvider{}
}

func (m *MockAuthTokenProvider) SetToken(token string) {
	m.token = token
}

func (m *MockAuthTokenProvider) SetSource(source string) {
	m.source = source
}

func (m *MockAuthTokenProvider) SetError(err error) {
	m.err = err
}

func (m *MockAuthTokenProvider) SetSourceError(err error) {
	m.sourceErr = err
}

func (m *MockAuthTokenProvider) GetToken() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func (m *MockAuthTokenProvider) GetTokenWithSource() (string, string, error) {
	if m.sourceErr != nil {
		return "", "", m.sourceErr
	}
	return m.source, m.token, nil
}

// MockRepoInfoProvider provides a mock implementation for repository info
type MockRepoInfoProvider struct {
	owner string
	repo  string
	err   error
}

func NewMockRepoInfoProvider() *MockRepoInfoProvider {
	return &MockRepoInfoProvider{}
}

func (m *MockRepoInfoProvider) SetRepoInfo(owner, repo string) {
	m.owner = owner
	m.repo = repo
}

func (m *MockRepoInfoProvider) SetError(err error) {
	m.err = err
}

func (m *MockRepoInfoProvider) GetRepoInfo() (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	return m.owner, m.repo, nil
}
