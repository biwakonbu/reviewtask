package github

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/go-github/v58/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
	owner  string
	repo   string
}

type PRInfo struct {
	Number     int    `json:"pr_number"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	State      string `json:"state"`
	Repository string `json:"repository"`
	Branch     string `json:"branch"` // Head branch name for this PR
}

type Review struct {
	ID          int64     `json:"id"`
	Reviewer    string    `json:"reviewer"`
	State       string    `json:"state"`
	Body        string    `json:"body"`
	SubmittedAt string    `json:"submitted_at"`
	Comments    []Comment `json:"comments"`
}

type Comment struct {
	ID        int64   `json:"id"`
	File      string  `json:"file"`
	Line      int     `json:"line"`
	Body      string  `json:"body"`
	Author    string  `json:"author"`
	CreatedAt string  `json:"created_at"`
	Replies   []Reply `json:"replies"`
}

type Reply struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

func NewClient() (*Client, error) {
	token, err := GetGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Get repository info from git
	owner, repo, err := getRepoInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Create GitHub client with authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &Client{
		client: client,
		owner:  owner,
		repo:   repo,
	}, nil
}

func (c *Client) GetCurrentBranchPR(ctx context.Context) (int, error) {
	// Get current branch
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" || branch == "main" || branch == "master" {
		return 0, fmt.Errorf("no feature branch detected (current: %s)", branch)
	}

	// Search for PR with this branch
	opts := &github.PullRequestListOptions{
		Head:  fmt.Sprintf("%s:%s", c.owner, branch),
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	}

	prs, _, err := c.client.PullRequests.List(ctx, c.owner, c.repo, opts)
	if err != nil {
		return 0, fmt.Errorf("failed to search for PR: %w", err)
	}

	if len(prs) == 0 {
		return 0, fmt.Errorf("no open PR found for branch '%s'", branch)
	}

	return prs[0].GetNumber(), nil
}

func (c *Client) GetPRInfo(ctx context.Context, prNumber int) (*PRInfo, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR #%d: %w", prNumber, err)
	}

	return &PRInfo{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		Author:     pr.GetUser().GetLogin(),
		CreatedAt:  pr.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  pr.GetUpdatedAt().Format("2006-01-02T15:04:05Z"),
		State:      pr.GetState(),
		Repository: fmt.Sprintf("%s/%s", c.owner, c.repo),
		Branch:     pr.GetHead().GetRef(),
	}, nil
}

func (c *Client) GetPRReviews(ctx context.Context, prNumber int) ([]Review, error) {
	// Get reviews
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, c.owner, c.repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}

	var result []Review
	for _, review := range reviews {
		r := Review{
			ID:          review.GetID(),
			Reviewer:    review.GetUser().GetLogin(),
			State:       review.GetState(),
			Body:        review.GetBody(),
			SubmittedAt: review.GetSubmittedAt().Format("2006-01-02T15:04:05Z"),
		}

		// Get review comments
		comments, err := c.getReviewComments(ctx, prNumber, review.GetID())
		if err != nil {
			return nil, fmt.Errorf("failed to get comments for review %d: %w", review.GetID(), err)
		}
		r.Comments = comments

		result = append(result, r)
	}

	return result, nil
}

func (c *Client) getReviewComments(ctx context.Context, prNumber int, reviewID int64) ([]Comment, error) {
	// Get all PR review comments
	allComments, _, err := c.client.PullRequests.ListComments(ctx, c.owner, c.repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Filter comments for this review and build nested structure
	commentMap := make(map[int64]*Comment)
	var rootComments []Comment

	for _, comment := range allComments {
		// Skip comments not part of this review (if we can determine that)
		// Note: GitHub API doesn't directly link comments to reviews, so we'll include all for now

		c := Comment{
			ID:        comment.GetID(),
			File:      comment.GetPath(),
			Line:      comment.GetLine(),
			Body:      comment.GetBody(),
			Author:    comment.GetUser().GetLogin(),
			CreatedAt: comment.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
			Replies:   []Reply{},
		}

		commentMap[comment.GetID()] = &c

		// For now, treat all comments as root comments since GitHub API comment nesting is complex
		// In a production version, you would implement proper comment thread detection
		rootComments = append(rootComments, c)
	}

	// Convert map back to slice for root comments
	var result []Comment
	for _, comment := range rootComments {
		if c, exists := commentMap[comment.ID]; exists {
			result = append(result, *c)
		}
	}

	return result, nil
}

func getRepoInfo() (string, string, error) {
	// Get remote URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git remote URL: %w", err)
	}

	url := strings.TrimSpace(string(output))

	// Parse GitHub URL (both SSH and HTTPS formats)
	// SSH: git@github.com:owner/repo.git
	// HTTPS: https://github.com/owner/repo.git

	var owner, repo string

	if strings.HasPrefix(url, "git@github.com:") {
		// SSH format
		parts := strings.TrimPrefix(url, "git@github.com:")
		parts = strings.TrimSuffix(parts, ".git")
		repoParts := strings.Split(parts, "/")
		if len(repoParts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL format: %s", url)
		}
		owner, repo = repoParts[0], repoParts[1]
	} else if strings.HasPrefix(url, "https://github.com/") {
		// HTTPS format
		parts := strings.TrimPrefix(url, "https://github.com/")
		parts = strings.TrimSuffix(parts, ".git")
		repoParts := strings.Split(parts, "/")
		if len(repoParts) != 2 {
			return "", "", fmt.Errorf("invalid HTTPS URL format: %s", url)
		}
		owner, repo = repoParts[0], repoParts[1]
	} else {
		return "", "", fmt.Errorf("unsupported git remote URL format: %s", url)
	}

	return owner, repo, nil
}

// NewClientWithToken creates a client with a specific token
func NewClientWithToken(token string) (*Client, error) {
	// Get repository info from git
	owner, repo, err := getRepoInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Create GitHub client with authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &Client{
		client: client,
		owner:  owner,
		repo:   repo,
	}, nil
}

// GetCurrentUser returns the authenticated user's login name
func (c *Client) GetCurrentUser() (string, error) {
	user, _, err := c.client.Users.Get(context.Background(), "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}

// GetRepoInfo gets repository information for permission testing
func (c *Client) GetRepoInfo() (*github.Repository, error) {
	repo, _, err := c.client.Repositories.Get(context.Background(), c.owner, c.repo)
	return repo, err
}

// GetPRList gets pull request list for permission testing
func (c *Client) GetPRList() ([]*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 1, // Just need one to test permissions
		},
	}

	prs, _, err := c.client.PullRequests.List(context.Background(), c.owner, c.repo, opts)
	return prs, err
}

// GetTokenScopes returns the scopes of the current token
func (c *Client) GetTokenScopes() ([]string, error) {
	// GitHub API doesn't directly provide scope information in responses
	// We need to check the response headers from any API call
	// For now, we'll make a simple API call and check what we can access

	// This is a workaround - GitHub doesn't provide a direct way to get token scopes
	// We'll return what we can determine from successful API calls
	var scopes []string

	// Test basic user access
	_, _, err := c.client.Users.Get(context.Background(), "")
	if err == nil {
		scopes = append(scopes, "user")
	}

	// Test repo access
	_, _, err = c.client.Repositories.Get(context.Background(), c.owner, c.repo)
	if err == nil {
		scopes = append(scopes, "repo")
	}

	// If we can't determine scopes, return a generic message
	if len(scopes) == 0 {
		return []string{"unable to determine scopes"}, nil
	}

	return scopes, nil
}
