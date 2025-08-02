package notification

import "context"

// GitHubClient interface for GitHub operations needed by the notifier
type GitHubClient interface {
	CreateIssueComment(ctx context.Context, prNumber int, body string) error
}