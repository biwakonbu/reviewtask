package github

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// EmbeddedComment represents a comment extracted from a review body
// This is typically used for Codex-style reviews where comments are embedded in the review body
// instead of using GitHub's review comment API
type EmbeddedComment struct {
	FilePath    string
	StartLine   int
	EndLine     int
	Priority    string // P1, P2, P3
	Title       string
	Description string
	Permalink   string
}

var (
	// GitHub permalink pattern: https://github.com/owner/repo/blob/commit_hash/file_path#L1-L5
	githubPermalinkRegex = regexp.MustCompile(`https://github\.com/[^/]+/[^/]+/blob/[^/]+/([^#\s]+)(?:#L(\d+)(?:-L(\d+))?)?`)

	// Priority badge patterns
	p1BadgeRegex = regexp.MustCompile(`!\[P1 Badge\]\(https://img\.shields\.io/badge/P1-orange\?style=flat\)`)
	p2BadgeRegex = regexp.MustCompile(`!\[P2 Badge\]\(https://img\.shields\.io/badge/P2-yellow\?style=flat\)`)
	p3BadgeRegex = regexp.MustCompile(`!\[P3 Badge\]\(https://img\.shields\.io/badge/P3-green\?style=flat\)`)
)

// ParseEmbeddedComments extracts embedded comments from a review body
// This is used to parse Codex-style reviews where comments are formatted within the review body
func ParseEmbeddedComments(reviewBody string) []EmbeddedComment {
	if reviewBody == "" {
		return nil
	}

	var comments []EmbeddedComment

	// Split review body into sections by GitHub permalinks
	lines := strings.Split(reviewBody, "\n")
	var currentComment *EmbeddedComment
	var descriptionLines []string

	for i := 0; i < len(lines); i++ {
		// Preserve raw line with only \r removed, keep indentation and empty lines
		rawLine := strings.TrimRight(lines[i], "\r")
		// Use trimmed line for pattern matching and checks only
		line := strings.TrimSpace(rawLine)

		// Check if this line contains a GitHub permalink
		if matches := githubPermalinkRegex.FindStringSubmatch(line); matches != nil {
			// If we have a previous comment, save it
			if currentComment != nil {
				currentComment.Description = strings.TrimSpace(strings.Join(descriptionLines, "\n"))
				comments = append(comments, *currentComment)
			}

			// Start a new embedded comment
			currentComment = &EmbeddedComment{
				Permalink: matches[0],
			}
			descriptionLines = []string{}

			// Extract file path (URL-decode it)
			filePath := matches[1]
			if decoded, err := url.QueryUnescape(filePath); err == nil {
				filePath = decoded
			}
			currentComment.FilePath = filePath

			// Extract line numbers
			if len(matches) > 2 && matches[2] != "" {
				fmt.Sscanf(matches[2], "%d", &currentComment.StartLine)
				if len(matches) > 3 && matches[3] != "" {
					fmt.Sscanf(matches[3], "%d", &currentComment.EndLine)
				} else {
					currentComment.EndLine = currentComment.StartLine
				}
			}

			continue
		}

		// If we're building a comment, check for title and priority
		if currentComment != nil {
			// Check for priority badges and title in the same line (use trimmed for detection)
			if containsPriorityBadge(line) {
				priority := extractPriority(line)
				title := extractTitle(line)

				currentComment.Priority = priority
				currentComment.Title = title
				continue
			}

			// Otherwise, accumulate description lines (use raw line to preserve formatting)
			// Allow empty lines to preserve code block structure
			descriptionLines = append(descriptionLines, rawLine)
		}
	}

	// Don't forget the last comment
	if currentComment != nil {
		currentComment.Description = strings.TrimSpace(strings.Join(descriptionLines, "\n"))
		comments = append(comments, *currentComment)
	}

	return comments
}

// containsPriorityBadge checks if a line contains any priority badge
func containsPriorityBadge(line string) bool {
	return p1BadgeRegex.MatchString(line) ||
		p2BadgeRegex.MatchString(line) ||
		p3BadgeRegex.MatchString(line)
}

// extractPriority extracts the priority level from a line with a badge
func extractPriority(line string) string {
	if p1BadgeRegex.MatchString(line) {
		return "P1"
	}
	if p2BadgeRegex.MatchString(line) {
		return "P2"
	}
	if p3BadgeRegex.MatchString(line) {
		return "P3"
	}
	return ""
}

// extractTitle extracts the title from a line containing a priority badge
// The title is typically everything after the badge, trimmed of markdown formatting
func extractTitle(line string) string {
	// Remove all badge patterns
	title := p1BadgeRegex.ReplaceAllString(line, "")
	title = p2BadgeRegex.ReplaceAllString(title, "")
	title = p3BadgeRegex.ReplaceAllString(title, "")

	// Remove markdown bold formatting (**text**)
	title = strings.ReplaceAll(title, "**", "")

	// Remove HTML tags if present
	title = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(title, "")

	// Clean up whitespace
	title = strings.TrimSpace(title)

	return title
}

// IsCodexReview checks if a review is from the Codex bot
func IsCodexReview(reviewer string) bool {
	return reviewer == "chatgpt-codex-connector" ||
		strings.Contains(strings.ToLower(reviewer), "codex")
}

// ConvertEmbeddedCommentToComment converts an EmbeddedComment to a standard Comment
// This allows Codex-style reviews to be processed through the same task generation pipeline
func ConvertEmbeddedCommentToComment(ec EmbeddedComment, author string, createdAt string) Comment {
	// Build the comment body from title and description
	body := ec.Title
	if ec.Description != "" {
		body = fmt.Sprintf("%s\n\n%s", ec.Title, ec.Description)
	}

	return Comment{
		ID:        0, // Embedded comments don't have GitHub comment IDs
		File:      ec.FilePath,
		Line:      ec.StartLine,
		Body:      body,
		Author:    author,
		CreatedAt: createdAt,
		URL:       ec.Permalink,
		Replies:   []Reply{},
	}
}

// MapPriorityToTaskPriority maps Codex priority (P1/P2/P3) to task priority
func MapPriorityToTaskPriority(codexPriority string) string {
	switch codexPriority {
	case "P1":
		return "high"
	case "P2":
		return "medium"
	case "P3":
		return "low"
	default:
		return "medium" // Default to medium if unknown
	}
}
