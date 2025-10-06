package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

// CommitInfo contains information for generating commit messages
type CommitInfo struct {
	TaskSummary      string
	ReviewCommentURL string
	OriginalComment  string
	Changes          []string
	PRNumber         int
	Language         string
}

// CommitResult contains the result of a commit operation
type CommitResult struct {
	Success    bool
	CommitHash string
	Message    string
	Error      error
}

// GitCommitter handles automatic git commits for tasks
type GitCommitter struct {
	config *config.Config
}

// NewGitCommitter creates a new git committer instance
func NewGitCommitter() (*GitCommitter, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &GitCommitter{
		config: cfg,
	}, nil
}

// CreateCommitForTask creates a commit for a completed task
func (g *GitCommitter) CreateCommitForTask(task *storage.Task) (*CommitResult, error) {
	// Check if auto-commit is enabled
	if !g.config.DoneWorkflow.EnableAutoCommit {
		return &CommitResult{
			Success: false,
			Message: "auto-commit disabled",
		}, nil
	}

	// Check if there are staged changes
	hasStaged, err := g.hasStagedChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to check staged changes: %w", err)
	}

	if !hasStaged {
		return &CommitResult{
			Success: false,
			Message: "no staged changes",
		}, nil
	}

	// Build commit message
	commitInfo := g.buildCommitInfo(task)
	commitMessage := g.buildCommitMessage(commitInfo)

	// Create commit
	commitHash, err := g.createCommit(commitMessage)
	if err != nil {
		return &CommitResult{
			Success: false,
			Error:   err,
		}, nil
	}

	return &CommitResult{
		Success:    true,
		CommitHash: commitHash,
		Message:    "commit created successfully",
	}, nil
}

// hasStagedChanges checks if there are staged changes in git
func (g *GitCommitter) hasStagedChanges() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	err := cmd.Run()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means there are changes
			if exitErr.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}

	// Exit code 0 means no changes
	return false, nil
}

// buildCommitInfo extracts commit information from task
func (g *GitCommitter) buildCommitInfo(task *storage.Task) *CommitInfo {
	language := g.config.Language
	if language == "" {
		language = "en" // Default to English
	}

	return &CommitInfo{
		TaskSummary:      task.Description,
		ReviewCommentURL: task.URL,
		OriginalComment:  task.OriginText,
		Changes:          g.extractChangesFromDiff(),
		PRNumber:         task.PRNumber,
		Language:         language,
	}
}

// extractChangesFromDiff extracts change details from git diff
func (g *GitCommitter) extractChangesFromDiff() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--stat")
	output, err := cmd.Output()
	if err != nil {
		return []string{"Unable to extract changes"}
	}

	lines := strings.Split(string(output), "\n")
	changes := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "files changed") {
			changes = append(changes, line)
		}
	}

	return changes
}

// createCommit creates a git commit with the provided message
func (g *GitCommitter) createCommit(message string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create commit
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// Get commit hash
	hashCmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(string(hashOutput)), nil
}

// GetCurrentBranch returns the current git branch name
func (g *GitCommitter) GetCurrentBranch() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetUnstagedFiles returns list of unstaged files
func (g *GitCommitter) GetUnstagedFiles() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get unstaged files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]string, 0)
	for _, file := range files {
		if file != "" {
			result = append(result, file)
		}
	}

	return result, nil
}

// GetStagedFiles returns list of staged files
func (g *GitCommitter) GetStagedFiles() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make([]string, 0)
	for _, file := range files {
		if file != "" {
			result = append(result, file)
		}
	}

	return result, nil
}

// StageAllChanges stages all changes in the working directory
func (g *GitCommitter) StageAllChanges() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	return nil
}

// buildCommitMessage builds a commit message from commit info
func (g *GitCommitter) buildCommitMessage(info *CommitInfo) string {
	var buf bytes.Buffer

	// Task summary (first line)
	buf.WriteString(info.TaskSummary)
	buf.WriteString("\n\n")

	// Review comment URL
	if info.ReviewCommentURL != "" {
		buf.WriteString("Review Comment: ")
		buf.WriteString(info.ReviewCommentURL)
		buf.WriteString("\n\n")
	}

	// Original comment (quoted)
	if info.OriginalComment != "" {
		buf.WriteString("Original Comment:\n")
		lines := strings.Split(info.OriginalComment, "\n")
		for _, line := range lines {
			buf.WriteString("> ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	// Changes (if any)
	if len(info.Changes) > 0 {
		buf.WriteString("Changes:\n")
		for _, change := range info.Changes {
			buf.WriteString("- ")
			buf.WriteString(change)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	// PR number
	if info.PRNumber > 0 {
		buf.WriteString(fmt.Sprintf("PR: #%d\n", info.PRNumber))
	}

	return buf.String()
}
