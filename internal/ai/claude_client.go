package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// RealClaudeClient implements ClaudeClient using actual Claude Code CLI
type RealClaudeClient struct {
	claudePath string
}

// NewRealClaudeClient creates a new real Claude client
func NewRealClaudeClient() (*RealClaudeClient, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude command not found: %w", err)
	}
	return &RealClaudeClient{claudePath: claudePath}, nil
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
