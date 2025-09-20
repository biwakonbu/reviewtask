package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reviewtask/internal/config"
	"runtime"
	"strings"
)

// RealCursorClient implements ClaudeClient using Cursor CLI (cursor-agent)
type RealCursorClient struct {
	cursorPath string
	model      string
}

// NewRealCursorClient creates a new real Cursor client
func NewRealCursorClient() (*RealCursorClient, error) {
	return NewRealCursorClientWithConfig(nil)
}

// NewRealCursorClientWithConfig creates a new real Cursor client with optional config
func NewRealCursorClientWithConfig(cfg *config.Config) (*RealCursorClient, error) {
	cursorPath, err := findCursorCLI()
	if err != nil {
		return nil, fmt.Errorf("cursor-agent command not found: %w\n\nTo resolve this issue:\n1. Install Cursor CLI: npm install -g cursor-agent\n2. Or ensure cursor-agent is in your PATH\n3. Or create an alias 'cursor-agent' pointing to your Cursor CLI installation", err)
	}

	// Check if cursor-agent is already in PATH
	if _, pathErr := exec.LookPath("cursor-agent"); pathErr != nil {
		// Cursor found but not in PATH, create symlink
		log.Printf("ℹ️  Cursor CLI found at %s but not in PATH", cursorPath)

		if err := ensureCursorAvailable(cursorPath); err != nil {
			return nil, fmt.Errorf("failed to ensure cursor-agent availability: %w", err)
		}

		log.Printf("✓ Created symlink at ~/.local/bin/cursor-agent for reviewtask compatibility")
	}

	// Get model from config, default to auto for cursor-agent
	model := "auto"
	if cfg != nil && cfg.AISettings.Model != "" {
		model = cfg.AISettings.Model
	}

	client := &RealCursorClient{
		cursorPath: cursorPath,
		model:      model,
	}

	// Skip authentication check if configured or environment variable is set
	skipAuthCheck := os.Getenv("SKIP_CURSOR_AUTH_CHECK") == "true"
	if cfg != nil && cfg.AISettings.SkipClaudeAuthCheck {
		skipAuthCheck = true
	}

	if !skipAuthCheck {
		// Check if Cursor CLI is authenticated
		if err := client.CheckAuthentication(); err != nil {
			return nil, fmt.Errorf("cursor-agent authentication check failed: %w\n\nTo authenticate:\n1. Run: cursor-agent login\n2. Follow the authentication prompts\n\nOr skip this check by setting:\n- Environment variable: SKIP_CURSOR_AUTH_CHECK=true\n- Config file: \"skip_claude_auth_check\": true", err)
		}
	}

	return client, nil
}

// CheckAuthentication verifies that Cursor CLI is properly authenticated
func (c *RealCursorClient) CheckAuthentication() error {
	// Skip authentication check if SKIP_CURSOR_AUTH_CHECK is set
	if os.Getenv("SKIP_CURSOR_AUTH_CHECK") == "true" {
		return nil
	}

	// Run cursor-agent status to check authentication
	ctx := context.Background()

	var cmd *exec.Cmd
	// Check if cursorPath contains interpreter command
	if strings.Contains(c.cursorPath, " ") {
		parts := strings.Fields(c.cursorPath)
		if len(parts) >= 2 {
			interpreter := parts[0]
			scriptAndArgs := append(parts[1:], "status")
			cmd = exec.CommandContext(ctx, interpreter, scriptAndArgs...)
		} else {
			cmd = exec.CommandContext(ctx, c.cursorPath, "status")
		}
	} else {
		cmd = exec.CommandContext(ctx, c.cursorPath, "status")
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	output := strings.ToLower(stdout.String())

	// Check if logged in
	if err != nil || !strings.Contains(output, "logged in") {
		return fmt.Errorf("cursor-agent is not authenticated. Please run: cursor-agent login")
	}

	return nil
}

// findCursorCLI implements comprehensive Cursor CLI detection strategy
func findCursorCLI() (string, error) {
	// Try standard PATH first
	if path, err := exec.LookPath("cursor-agent"); err == nil {
		return path, nil
	}

	// Get home directory in a cross-platform way
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to HOME env var if UserHomeDir fails
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE") // Windows fallback
		}
	}

	// Try common installation locations
	var commonPaths []string

	if runtime.GOOS == "windows" {
		// Windows-specific paths
		commonPaths = []string{
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "cursor-agent.cmd"),
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "cursor-agent.exe"),
			filepath.Join(homeDir, ".volta", "bin", "cursor-agent.exe"),
			filepath.Join(homeDir, ".volta", "bin", "cursor-agent.cmd"),
		}
	} else {
		// Unix-like paths
		commonPaths = []string{
			filepath.Join(homeDir, ".npm-global/bin/cursor-agent"),
			filepath.Join(homeDir, ".volta/bin/cursor-agent"),
			"/usr/local/bin/cursor-agent",
			"/opt/homebrew/bin/cursor-agent",
			filepath.Join(homeDir, ".local/bin/cursor-agent"),
		}
	}

	// Add npm global prefix bin directory (for nvm and other npm managers)
	if npmPrefix := getNpmPrefix(); npmPrefix != "" {
		if runtime.GOOS == "windows" {
			// On Windows, npm installs .cmd files
			commonPaths = append(commonPaths,
				filepath.Join(npmPrefix, "cursor-agent.cmd"),
				filepath.Join(npmPrefix, "cursor-agent.exe"))
		} else {
			npmCursorPath := filepath.Join(npmPrefix, "bin", "cursor-agent")
			commonPaths = append(commonPaths, npmCursorPath)
		}
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			// Verify it's executable by running version check
			if isValidCursorCLI(path) {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("cursor-agent CLI not found in PATH or common installation locations")
}

// isValidCursorCLI verifies that the found executable is actually Cursor CLI
func isValidCursorCLI(path string) bool {
	// Handle interpreter-based commands
	parts := strings.Fields(path)
	var cmd *exec.Cmd

	if len(parts) > 1 {
		// Command with interpreter
		interpreter := parts[0]
		scriptAndArgs := append(parts[1:], "--version")
		cmd = exec.Command(interpreter, scriptAndArgs...)
	} else {
		// Direct command
		cmd = exec.Command(path, "--version")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// Check if output contains expected Cursor CLI version pattern
	outputStr := strings.ToLower(string(output))
	// cursor-agent returns version in format like "2025.09.12-4852336"
	return strings.Contains(outputStr, "20") || strings.Contains(outputStr, "cursor")
}

// ensureCursorAvailable creates symlink if Cursor is not in PATH
func ensureCursorAvailable(cursorPath string) error {
	// Check if cursor-agent is already available in PATH
	if _, err := exec.LookPath("cursor-agent"); err == nil {
		return nil // Already available in PATH
	}

	// Skip symlink creation on Windows as it requires admin privileges
	if runtime.GOOS == "windows" {
		// On Windows, we rely on cursor-agent being in PATH or using full path
		return nil
	}

	// Create symlink in ~/.local/bin (which is commonly in PATH)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	localBin := filepath.Join(homeDir, ".local/bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return fmt.Errorf("failed to create ~/.local/bin directory: %w", err)
	}

	symlinkPath := filepath.Join(localBin, "cursor-agent")

	// Remove existing symlink if it exists
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	if err := os.Symlink(cursorPath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// Execute runs Cursor with the given input
func (c *RealCursorClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	args := []string{}

	// Add model parameter if specified
	if c.model != "" {
		args = append(args, "--model", c.model)
	}

	// Add print flag and output format for JSON mode
	if outputFormat == "json" {
		args = append(args, "--print", "--output-format", outputFormat)
	} else if outputFormat != "" {
		args = append(args, "--output-format", outputFormat)
	}

	var cmd *exec.Cmd

	// Check if cursorPath contains interpreter command
	if strings.Contains(c.cursorPath, " ") {
		// Parse command with interpreter
		parts := strings.Fields(c.cursorPath)
		if len(parts) >= 2 {
			// First part is interpreter, rest are script and its args
			interpreter := parts[0]
			scriptAndArgs := append(parts[1:], args...)
			cmd = exec.CommandContext(ctx, interpreter, scriptAndArgs...)
		} else {
			// Fallback to direct execution
			cmd = exec.CommandContext(ctx, c.cursorPath, args...)
		}
	} else {
		// Direct command execution
		cmd = exec.CommandContext(ctx, c.cursorPath, args...)
	}

	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Debug: log the command being executed (only in verbose mode)
	if os.Getenv("REVIEWTASK_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: Executing cursor-agent command: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cursor-agent execution failed: %w, stderr: %s, stdout: %.200s", err, stderr.String(), stdout.String())
	}

	output := stdout.String()

	// If output format is JSON, extract the result field
	if outputFormat == "json" {
		var response struct {
			Type    string `json:"type"`
			IsError bool   `json:"is_error"`
			Result  string `json:"result"`
			Error   string `json:"error,omitempty"`
		}

		if err := json.Unmarshal([]byte(output), &response); err != nil {
			// If unmarshaling fails, return the raw output (backward compatibility)
			return output, nil
		}

		if response.IsError {
			return "", fmt.Errorf("cursor-agent API error: %s", response.Error)
		}

		return response.Result, nil
	}

	return output, nil
}
