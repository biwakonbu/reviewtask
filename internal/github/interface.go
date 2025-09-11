package github

import "context"

// GitHubClientInterface defines the interface for GitHub operations
// This allows for easy mocking and dependency injection in tests
type GitHubClientInterface interface {
	GetCurrentBranchPR(ctx context.Context) (int, error)
	GetPRInfo(ctx context.Context, prNumber int) (*PRInfo, error)
	GetPRReviews(ctx context.Context, prNumber int) ([]Review, error)
	GetSelfReviews(ctx context.Context, prNumber int, prAuthor string) ([]Review, error)
	IsPROpen(ctx context.Context, prNumber int) (bool, error)
}

// AuthTokenProvider defines the interface for authentication token retrieval
type AuthTokenProvider interface {
	GetToken() (string, error)
	GetTokenWithSource() (source, token string, err error)
}

// RepoInfoProvider defines the interface for repository information retrieval
type RepoInfoProvider interface {
	GetRepoInfo() (owner, repo string, err error)
}

// DefaultAuthTokenProvider implements AuthTokenProvider using the existing auth functions
type DefaultAuthTokenProvider struct{}

func (p *DefaultAuthTokenProvider) GetToken() (string, error) {
	return GetGitHubToken()
}

func (p *DefaultAuthTokenProvider) GetTokenWithSource() (string, string, error) {
	return GetTokenWithSource()
}

// DefaultRepoInfoProvider implements RepoInfoProvider using git commands
type DefaultRepoInfoProvider struct{}

func (p *DefaultRepoInfoProvider) GetRepoInfo() (string, string, error) {
	return getRepoInfo()
}

// Ensure Client implements GitHubClientInterface
var _ GitHubClientInterface = (*Client)(nil)