package ai

import (
	"context"
	"io"
)

// ClaudeClient defines the interface for interacting with Claude
type ClaudeClient interface {
	// Execute runs a Claude command with the given input and returns the output
	Execute(ctx context.Context, input string, outputFormat string) (string, error)
}

// CommandExecutor defines the interface for executing system commands
type CommandExecutor interface {
	// Execute runs a command with stdin and returns stdout/stderr
	Execute(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, error)
}