package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RealClaudeClient implements ClaudeClient using actual Claude Code CLI
type RealClaudeClient struct {
	claudePath string
}

// NewRealClaudeClient creates a new real Claude client
func NewRealClaudeClient() (*RealClaudeClient, error) {
	claudePath, err := findClaudeCLI()
	if err != nil {
		return nil, fmt.Errorf("claude command not found: %w", err)
	}

	// Ensure Claude is available in PATH via symlink if needed
	if err := ensureClaudeAvailable(claudePath); err != nil {
		return nil, fmt.Errorf("failed to ensure claude availability: %w", err)
	}

	return &RealClaudeClient{claudePath: claudePath}, nil
}

// findClaudeCLI implements comprehensive Claude CLI detection strategy
func findClaudeCLI() (string, error) {
	// Try standard PATH first
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	// Try common installation locations
	commonPaths := []string{
		filepath.Join(os.Getenv("HOME"), ".claude/local/claude"),
		filepath.Join(os.Getenv("HOME"), ".npm-global/bin/claude"),
		filepath.Join(os.Getenv("HOME"), ".volta/bin/claude"),
		"/usr/local/bin/claude",
		"/opt/homebrew/bin/claude",
		filepath.Join(os.Getenv("HOME"), ".local/bin/claude"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			// Verify it's executable by running version check
			if isValidClaudeCLI(path) {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("claude CLI not found in PATH or common installation locations")
}

// isValidClaudeCLI verifies that the found executable is actually Claude CLI
func isValidClaudeCLI(path string) bool {
	cmd := exec.Command(path, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// Check if output contains expected Claude CLI version pattern
	outputStr := strings.ToLower(string(output))
	return strings.Contains(outputStr, "claude") || strings.Contains(outputStr, "anthropic")
}

// ensureClaudeAvailable creates symlink if Claude is not in PATH
func ensureClaudeAvailable(claudePath string) error {
	// Check if claude is already available in PATH
	if _, err := exec.LookPath("claude"); err == nil {
		return nil // Already available in PATH
	}

	// Create symlink in ~/.local/bin (which is commonly in PATH)
	localBin := filepath.Join(os.Getenv("HOME"), ".local/bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return fmt.Errorf("failed to create ~/.local/bin directory: %w", err)
	}

	symlinkPath := filepath.Join(localBin, "claude")

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	if err := os.Symlink(claudePath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// CleanupClaudeSymlink removes symlinks created by reviewtask
func CleanupClaudeSymlink() error {
	symlinkPath := filepath.Join(os.Getenv("HOME"), ".local/bin", "claude")

	// Check if it's our symlink (not a real installation)
	if info, err := os.Lstat(symlinkPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(symlinkPath)
		if err == nil && isReviewtaskManagedSymlink(target) {
			return os.Remove(symlinkPath)
		}
	}

	return nil
}

// isReviewtaskManagedSymlink checks if symlink target is managed by reviewtask
func isReviewtaskManagedSymlink(target string) bool {
	// Check if target points to common npm/Node.js installation paths
	reviewtaskManagedPaths := []string{
		".claude/local/claude",
		".npm-global/bin/claude",
		".volta/bin/claude",
	}

	for _, managedPath := range reviewtaskManagedPaths {
		if strings.Contains(target, managedPath) {
			return true
		}
	}

	return false
}

// Execute runs Claude with the given input
func (c *RealClaudeClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	args := []string{}
	if outputFormat != "" {
		args = append(args, "--output-format", outputFormat)
	}

	cmd := exec.CommandContext(ctx, c.claudePath, args...)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude execution failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// RealCommandExecutor implements CommandExecutor using os/exec
type RealCommandExecutor struct{}

// NewRealCommandExecutor creates a new real command executor
func NewRealCommandExecutor() *RealCommandExecutor {
	return &RealCommandExecutor{}
}

// Execute runs a system command
func (e *RealCommandExecutor) Execute(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}

	return cmd.CombinedOutput()
}
