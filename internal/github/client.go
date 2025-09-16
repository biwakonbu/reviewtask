package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v58/github"
	htmlparser "golang.org/x/net/html"
	"golang.org/x/oauth2"
)

// ErrNoPRFound is returned when no PR is found for the current branch
var ErrNoPRFound = errors.New("no PR found for current branch")

type Client struct {
	client *github.Client
	owner  string
	repo   string
	cache  *APICache
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
	URL       string  `json:"url"` // GitHub comment URL for reference
	Replies   []Reply `json:"replies"`
}

type Reply struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
	URL       string `json:"url"` // GitHub comment URL for reference
}

// Regular expressions for code block removal and GitHub PR features
var (
	// Matches fenced code blocks (```...```) including language specifiers
	fencedCodeBlockRegex = regexp.MustCompile(`(?s)` + "`" + `{3}[^` + "`" + `\n]*\n.*?\n` + "`" + `{3}`)
	// Matches inline code (`...`)
	inlineCodeRegex = regexp.MustCompile("`[^`\n]+`")
	// Matches indented code blocks (consecutive lines with 4+ spaces/tabs at start)
	indentedCodeBlockRegex = regexp.MustCompile(`(?m)^( {4,}|\t+).*(\n( {4,}|\t+).*)*`)

	// GitHub PR suggestion blocks (with HTML escaping)
	suggestionBlockRegex = regexp.MustCompile(`(?s)(?:\\u003c!--|<!)-- suggestion_start --(?:\\u003e|>).*?(?:\\u003c!--|<!)-- suggestion_end --(?:\\u003e|>)`)
	// GitHub committable suggestions (detailed collapsible sections)
	committableSuggestionRegex = regexp.MustCompile(`(?s)(?:\\u003c|<)details(?:\\u003e|>)\s*(?:\\u003c|<)summary(?:\\u003e|>)📝 Committable suggestion(?:\\u003c|<)/summary(?:\\u003e|>).*?(?:\\u003c|<)/details(?:\\u003e|>)`)
	// GitHub prompt sections for AI agents
	promptSectionRegex = regexp.MustCompile(`(?s)(?:\\u003c|<)details(?:\\u003e|>)\s*(?:\\u003c|<)summary(?:\\u003e|>)🤖 Prompt for AI Agents(?:\\u003c|<)/summary(?:\\u003e|>).*?(?:\\u003c|<)/details(?:\\u003e|>)`)
	// GitHub fingerprinting comments
	fingerprintRegex = regexp.MustCompile(`(?s)(?:\\u003c!--|<!)-- fingerprinting:.*? --(?:\\u003e|>)`)

	// CodeRabbit detailed review sections (aggressive cleanup)
	cautionSectionRegex = regexp.MustCompile(`(?s)(?:\\u003e|>) \[!CAUTION\].*`)
	nitpickSectionRegex = regexp.MustCompile(`(?s)(?:\\u003c|<)details(?:\\u003e|>)\s*(?:\\u003c|<)summary(?:\\u003e|>)🧹 Nitpick comments.*`)
	reviewDetailsRegex = regexp.MustCompile(`(?s)(?:\\u003c|<)details(?:\\u003e|>)\s*(?:\\u003c|<)summary(?:\\u003e|>)📜 Review details.*`)
	codeGraphRegex = regexp.MustCompile(`(?s)(?:\\u003c|<)details(?:\\u003e|>)\s*(?:\\u003c|<)summary(?:\\u003e|>)🧬 Code graph analysis.*`)
	learningsRegex = regexp.MustCompile(`(?s)(?:\\u003c|<)details(?:\\u003e|>)\s*(?:\\u003c|<)summary(?:\\u003e|>)🧠 Learnings.*`)
	additionalContextRegex = regexp.MustCompile(`(?s)(?:\\u003c|<)details(?:\\u003e|>)\s*(?:\\u003c|<)summary(?:\\u003e|>)🧰 Additional context used.*`)
)

// removeHTMLElements removes specified HTML elements from text using proper HTML parsing
func removeHTMLElements(text string) string {
	if text == "" {
		return text
	}

	// First unescape HTML entities (\u003c -> <, etc.)
	unescaped := html.UnescapeString(text)

	// Parse HTML
	doc, err := htmlparser.Parse(strings.NewReader("<div>" + unescaped + "</div>"))
	if err != nil {
		// Fallback to original text if parsing fails
		return text
	}

	// Remove unwanted elements
	removeElementsRecursive(doc, shouldRemoveElement)

	// Convert back to text
	var result strings.Builder
	renderHTMLText(doc, &result)

	cleaned := strings.TrimSpace(result.String())

	// Clean up multiple consecutive newlines
	cleaned = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleaned, "\n\n")

	return cleaned
}

// shouldRemoveElement determines if an HTML element should be removed
func shouldRemoveElement(n *htmlparser.Node) (bool, string) {
	if n.Type != htmlparser.ElementNode {
		return false, ""
	}

	switch n.Data {
	case "details":
		// Check if it's a CodeRabbit section
		if summary := findChildElement(n, "summary"); summary != nil {
			summaryText := getTextContent(summary)
			if strings.Contains(summaryText, "🧹 Nitpick comments") {
				return true, "[nitpick comments removed]"
			}
			if strings.Contains(summaryText, "📜 Review details") {
				return true, "[review metadata removed]"
			}
			if strings.Contains(summaryText, "🧬 Code graph analysis") {
				return true, "[code graph analysis removed]"
			}
			if strings.Contains(summaryText, "🧠 Learnings") {
				return true, "[AI learnings removed]"
			}
			if strings.Contains(summaryText, "🧰 Additional context used") {
				return true, "[additional context removed]"
			}
			if strings.Contains(summaryText, "📝 Committable suggestion") {
				return true, "[committable suggestion removed]"
			}
			if strings.Contains(summaryText, "🤖 Prompt for AI Agents") {
				// Extract AI prompt content instead of removing it
				promptContent := extractAIPromptContent(n)
				if promptContent != "" {
					return true, "AI Task: " + promptContent
				}
				return true, "[AI prompt removed]"
			}
			// Remove outside diff range comments sections
			if strings.Contains(summaryText, "Outside diff range comments") {
				return true, "[outside diff comments removed]"
			}
		}
	case "blockquote":
		// Remove large blockquotes (likely review content)
		textContent := getTextContent(n)
		if len(textContent) > 2000 {
			return true, "[detailed review sections removed]"
		}
	}

	return false, ""
}

// extractAIPromptContent extracts meaningful content from AI prompt sections
func extractAIPromptContent(n *htmlparser.Node) string {
	content := getTextContent(n)

	// Clean up the content
	content = strings.ReplaceAll(content, "🤖 Prompt for AI Agents", "")
	content = strings.TrimSpace(content)

	// Limit length to avoid bloat
	if len(content) > 500 {
		content = content[:497] + "..."
	}

	return content
}

// removeElementsRecursive removes elements based on the shouldRemove function
func removeElementsRecursive(n *htmlparser.Node, shouldRemove func(*htmlparser.Node) (bool, string)) {
	var next *htmlparser.Node
	for child := n.FirstChild; child != nil; child = next {
		next = child.NextSibling

		if remove, replacement := shouldRemove(child); remove {
			// Replace with text node
			if replacement != "" {
				textNode := &htmlparser.Node{
					Type: htmlparser.TextNode,
					Data: replacement,
				}
				n.InsertBefore(textNode, child)
			}
			n.RemoveChild(child)
		} else {
			removeElementsRecursive(child, shouldRemove)
		}
	}
}

// findChildElement finds the first child element with the given tag name
func findChildElement(n *htmlparser.Node, tagName string) *htmlparser.Node {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == htmlparser.ElementNode && child.Data == tagName {
			return child
		}
	}
	return nil
}

// getTextContent extracts all text content from an HTML node
func getTextContent(n *htmlparser.Node) string {
	var result strings.Builder
	var extract func(*htmlparser.Node)
	extract = func(node *htmlparser.Node) {
		if node.Type == htmlparser.TextNode {
			result.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extract(child)
		}
	}
	extract(n)
	return result.String()
}

// renderHTMLText converts HTML node back to text, preserving structure
func renderHTMLText(n *htmlparser.Node, result *strings.Builder) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case htmlparser.TextNode:
			result.WriteString(child.Data)
		case htmlparser.ElementNode:
			// Add newlines for block elements
			switch child.Data {
			case "p", "div", "details", "blockquote":
				result.WriteString("\n")
				renderHTMLText(child, result)
				result.WriteString("\n")
			default:
				renderHTMLText(child, result)
			}
		default:
			renderHTMLText(child, result)
		}
	}
}

// removeCodeBlocks removes all code blocks, inline code, and GitHub PR features from markdown text
// while preserving other formatting and content structure
func removeCodeBlocks(text string) string {
	if text == "" {
		return text
	}

	// Remove CodeRabbit actionable comments header but keep the actual content
	actionableHeaderRegex := regexp.MustCompile(`^\*\*Actionable comments posted: \d+\*\*\s*\n\n?`)
	text = actionableHeaderRegex.ReplaceAllString(text, "")

	// Extract AI Prompt content and remove other suggestion blocks (including HTML processing and fingerprints)
	text = processAIPromptAndSuggestions(text)

	return strings.TrimSpace(text)
}

// processAIPromptAndSuggestions extracts AI Prompt content and removes other suggestion blocks
func processAIPromptAndSuggestions(text string) string {

	// First, unescape Unicode HTML entities
	text = strings.ReplaceAll(text, `\u003c`, `<`)
	text = strings.ReplaceAll(text, `\u003e`, `>`)

	// Then, unescape standard HTML entities
	text = html.UnescapeString(text)


	// Remove verbose suggestion blocks but keep AI Prompt blocks intact
	text = regexp.MustCompile(`(?s)<!-- suggestion_start -->.*?<!-- suggestion_end -->`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?s)<details>\s*<summary>📝 Committable suggestion</summary>.*?</details>`).ReplaceAllString(text, "")

	// Remove GitHub fingerprinting comments (these are truly not useful)
	text = fingerprintRegex.ReplaceAllString(text, "")

	// Clean up multiple consecutive newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}


// Injectable function variables for easier testing/mocking
var (
	getRepoInfoFn = getRepoInfo
)

func NewClient() (*Client, error) {
	return NewClientWithProviders(&DefaultAuthTokenProvider{}, &DefaultRepoInfoProvider{})
}

// NewClientWithProviders creates a client with dependency injection for testing
func NewClientWithProviders(authProvider AuthTokenProvider, repoProvider RepoInfoProvider) (*Client, error) {
	token, err := authProvider.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Get repository info
	owner, repo, err := repoProvider.GetRepoInfo()
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
		cache:  NewAPICache(5 * time.Minute),
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
		return 0, fmt.Errorf("%w: branch '%s'", ErrNoPRFound, branch)
	}

	return prs[0].GetNumber(), nil
}

func (c *Client) GetPRInfo(ctx context.Context, prNumber int) (*PRInfo, error) {
	// Check cache first
	if cached, ok := c.cache.Get("GetPRInfo", c.owner, c.repo, prNumber); ok {
		// Handle both direct *PRInfo and map[string]interface{} from JSON
		switch v := cached.(type) {
		case *PRInfo:
			return v, nil
		default:
			// Re-marshal and unmarshal to handle JSON-decoded cache entries
			jsonData, err := json.Marshal(cached)
			if err == nil {
				var prInfo PRInfo
				if err := json.Unmarshal(jsonData, &prInfo); err == nil {
					return &prInfo, nil
				}
			}
		}
	}

	pr, _, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR #%d: %w", prNumber, err)
	}

	prInfo := &PRInfo{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		Author:     pr.GetUser().GetLogin(),
		CreatedAt:  pr.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  pr.GetUpdatedAt().Format("2006-01-02T15:04:05Z"),
		State:      pr.GetState(),
		Repository: fmt.Sprintf("%s/%s", c.owner, c.repo),
		Branch:     pr.GetHead().GetRef(),
	}

	// Cache the result (ignore cache error)
	_ = c.cache.Set("GetPRInfo", c.owner, c.repo, prInfo, prNumber)

	return prInfo, nil
}

func (c *Client) GetPRReviews(ctx context.Context, prNumber int) ([]Review, error) {
	// Check cache first
	if cached, ok := c.cache.Get("GetPRReviews", c.owner, c.repo, prNumber); ok {
		// JSON-marshal the generic interface{} and unmarshal into []Review
		raw, err := json.Marshal(cached)
		if err == nil {
			reviews := []Review{}
			if err := json.Unmarshal(raw, &reviews); err == nil {
				// Ensure we never return nil slice
				if reviews == nil {
					reviews = []Review{}
				}
				return reviews, nil
			}
		}
	}

	// Get reviews
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, c.owner, c.repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}

	result := []Review{}
	for _, review := range reviews {
		reviewBody := review.GetBody()

		// For CodeRabbit actionable comments, remove the summary body but keep individual comments
		if review.GetUser().GetLogin() == "coderabbitai[bot]" &&
		   strings.HasPrefix(reviewBody, "**Actionable comments posted:") {
			reviewBody = "" // Clear the body but keep the review for its comments
		} else {
			reviewBody = removeCodeBlocks(reviewBody)
		}

		r := Review{
			ID:          review.GetID(),
			Reviewer:    review.GetUser().GetLogin(),
			State:       review.GetState(),
			Body:        reviewBody,
			SubmittedAt: review.GetSubmittedAt().Format("2006-01-02T15:04:05Z"),
			Comments:    []Comment{}, // Initialize with empty slice to avoid null
		}

		// Get review comments (these are the important individual comments)
		comments, err := c.getReviewComments(ctx, prNumber, review.GetID())
		if err != nil {
			return nil, fmt.Errorf("failed to get comments for review %d: %w", review.GetID(), err)
		}
		if comments != nil {
			r.Comments = comments
		}

		result = append(result, r)
	}

	// Cache the result (ignore cache error)
	_ = c.cache.Set("GetPRReviews", c.owner, c.repo, result, prNumber)

	return result, nil
}

// GetSelfReviews fetches review comments made by the PR author (self-reviews)
func (c *Client) GetSelfReviews(ctx context.Context, prNumber int, prAuthor string) ([]Review, error) {
	// Create a synthetic review for self-review comments
	selfReview := Review{
		ID:          -1, // Special ID for self-review
		Reviewer:    prAuthor,
		State:       "COMMENTED", // Self-reviews are always comments
		Body:        "",          // Will be populated with aggregated comments
		SubmittedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		Comments:    []Comment{},
	}

	// Get issue comments from the PR author
	issueComments, _, err := c.client.Issues.ListComments(ctx, c.owner, c.repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue comments: %w", err)
	}

	// Filter and add issue comments from the author
	for _, comment := range issueComments {
		if comment.GetUser().GetLogin() == prAuthor {
			selfReview.Comments = append(selfReview.Comments, Comment{
				ID:        comment.GetID(),
				Body:      removeCodeBlocks(comment.GetBody()),
				Author:    prAuthor,
				CreatedAt: comment.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
				URL:       comment.GetHTMLURL(),
				// Issue comments don't have file/line info
				File: "",
				Line: 0,
			})
		}
	}

	// Get PR review comments from the author
	allPRComments, _, err := c.client.PullRequests.ListComments(ctx, c.owner, c.repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Filter and add PR comments from the author
	for _, comment := range allPRComments {
		if comment.GetUser().GetLogin() == prAuthor {
			selfReview.Comments = append(selfReview.Comments, Comment{
				ID:        comment.GetID(),
				Body:      removeCodeBlocks(comment.GetBody()),
				Author:    prAuthor,
				CreatedAt: comment.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
				URL:       comment.GetHTMLURL(),
				File:      comment.GetPath(),
				Line:      comment.GetLine(),
			})
		}
	}

	// Only return the self-review if there are any comments
	if len(selfReview.Comments) > 0 {
		return []Review{selfReview}, nil
	}

	return []Review{}, nil
}

func (c *Client) getReviewComments(ctx context.Context, prNumber int, reviewID int64) ([]Comment, error) {
	// Check cache for PR comments
	cacheKey := fmt.Sprintf("prcomments-%d", prNumber)
	var allComments []*github.PullRequestComment

	if cached, ok := c.cache.Get("ListComments", c.owner, c.repo, cacheKey); ok {
		// Handle both direct []*github.PullRequestComment and []interface{} from JSON
		switch v := cached.(type) {
		case []*github.PullRequestComment:
			allComments = v
		default:
			// Re-marshal and unmarshal to handle JSON-decoded cache entries
			jsonData, err := json.Marshal(cached)
			if err == nil {
				if err := json.Unmarshal(jsonData, &allComments); err == nil {
					// Successfully decoded from JSON cache
				}
			}
		}
	}

	// If we don't have comments from cache, fetch them
	if allComments == nil {
		// Get all PR review comments
		var err error
		allComments, _, err = c.client.PullRequests.ListComments(ctx, c.owner, c.repo, prNumber, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get PR comments: %w", err)
		}
		// Cache the raw comments (ignore cache error)
		_ = c.cache.Set("ListComments", c.owner, c.repo, allComments, cacheKey)
	}

	// Filter comments for this review and build flat list
	commentMap := make(map[int64]*Comment)
	var rootComments []Comment

	for _, comment := range allComments {
		// Only include comments that belong to this specific review
		// GitHub PR comments include PullRequestReviewID when they are part of a review
		if rid := comment.GetPullRequestReviewID(); rid != 0 && rid != reviewID {
			continue
		}

		c := Comment{
			ID:        comment.GetID(),
			File:      comment.GetPath(),
			Line:      comment.GetLine(),
			Body:      removeCodeBlocks(comment.GetBody()),
			Author:    comment.GetUser().GetLogin(),
			CreatedAt: comment.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
			URL:       comment.GetHTMLURL(),
			Replies:   []Reply{},
		}

		commentMap[comment.GetID()] = &c

		// For now, treat all comments as root comments
		rootComments = append(rootComments, c)
	}

	// Convert map back to slice for root comments
	result := make([]Comment, 0, len(rootComments)) // Initialize with capacity to avoid null
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
	owner, repo, err := getRepoInfoFn()
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
		cache:  NewAPICache(5 * time.Minute),
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

// IsPROpen checks if a PR is open (not closed or merged)
func (c *Client) IsPROpen(ctx context.Context, prNumber int) (bool, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, c.owner, c.repo, prNumber)
	if err != nil {
		return false, fmt.Errorf("failed to get PR #%d: %w", prNumber, err)
	}

	// PR is open if state is "open"
	return pr.GetState() == "open", nil
}
