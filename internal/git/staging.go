package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// StagingChecker handles checking and managing git staging area
type StagingChecker struct{}

// NewStagingChecker creates a new staging checker instance
func NewStagingChecker() *StagingChecker {
	return &StagingChecker{}
}

// HasStagedChanges checks if there are any staged changes
func (s *StagingChecker) HasStagedChanges() (bool, error) {
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

// HasUnstagedChanges checks if there are any unstaged changes
func (s *StagingChecker) HasUnstagedChanges() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "--quiet")
	err := cmd.Run()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means there are changes
			if exitErr.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, fmt.Errorf("failed to check unstaged changes: %w", err)
	}

	// Exit code 0 means no changes
	return false, nil
}

// GetStagingStatus returns detailed staging status
func (s *StagingChecker) GetStagingStatus() (*StagingStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staging status: %w", err)
	}

	status := &StagingStatus{
		StagedFiles:    make([]string, 0),
		UnstagedFiles:  make([]string, 0),
		UntrackedFiles: make([]string, 0),
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		statusCode := line[0:2]
		fileName := strings.TrimSpace(line[3:])

		// Check for minimum required length
		if len(statusCode) < 2 {
			continue
		}

		// Handle untracked files separately
		if statusCode == "??" {
			status.UntrackedFiles = append(status.UntrackedFiles, fileName)
			continue
		}

		// Check staged changes (first character)
		if statusCode[0] != ' ' && statusCode[0] != '?' {
			status.StagedFiles = append(status.StagedFiles, fileName)
		}

		// Check unstaged changes (second character)
		if statusCode[1] != ' ' && statusCode[1] != '?' {
			status.UnstagedFiles = append(status.UnstagedFiles, fileName)
		}
	}

	return status, nil
}

// StagingStatus contains detailed information about git staging area
type StagingStatus struct {
	StagedFiles    []string
	UnstagedFiles  []string
	UntrackedFiles []string
}

// IsClean returns true if there are no changes at all
func (s *StagingStatus) IsClean() bool {
	return len(s.StagedFiles) == 0 &&
		len(s.UnstagedFiles) == 0 &&
		len(s.UntrackedFiles) == 0
}

// HasStagedChanges returns true if there are staged changes
func (s *StagingStatus) HasStagedChanges() bool {
	return len(s.StagedFiles) > 0
}

// HasUnstagedChanges returns true if there are unstaged changes
func (s *StagingStatus) HasUnstagedChanges() bool {
	return len(s.UnstagedFiles) > 0
}

// HasUntrackedFiles returns true if there are untracked files
func (s *StagingStatus) HasUntrackedFiles() bool {
	return len(s.UntrackedFiles) > 0
}
