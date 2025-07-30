package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		return nil, fmt.Errorf("claude command not found: %w\n\nTo resolve this issue:\n1. Install Claude CLI: npm install -g @anthropic-ai/claude-code\n2. Or ensure Claude CLI is in your PATH\n3. Or create an alias 'claude' pointing to your Claude CLI installation\n4. Or place it in one of these locations:\n   - ~/.claude/local/claude\n   - ~/.npm-global/bin/claude\n   - ~/.volta/bin/claude", err)
	}

	// Check if Claude is already in PATH
	if _, pathErr := exec.LookPath("claude"); pathErr != nil {
		// Claude found but not in PATH, create symlink
		log.Printf("ℹ️  Claude CLI found at %s but not in PATH", claudePath)

		if err := ensureClaudeAvailable(claudePath); err != nil {
			return nil, fmt.Errorf("failed to ensure claude availability: %w", err)
		}

		log.Printf("✓ Created symlink at ~/.local/bin/claude for reviewtask compatibility")
	}

	return &RealClaudeClient{claudePath: claudePath}, nil
}

// findClaudeCLI implements comprehensive Claude CLI detection strategy
func findClaudeCLI() (string, error) {
	// Try standard PATH first
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	// Try to resolve alias
	if aliasPath, err := resolveClaudeAlias(); err == nil && aliasPath != "" {
		// Verify the resolved alias path is valid
		if isValidClaudeCLI(aliasPath) {
			return aliasPath, nil
		}
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
			filepath.Join(homeDir, ".claude", "local", "claude.exe"),
			filepath.Join(homeDir, ".npm-global", "claude.cmd"),
			filepath.Join(homeDir, ".npm-global", "claude.exe"),
			filepath.Join(homeDir, ".volta", "bin", "claude.exe"),
			filepath.Join(homeDir, ".volta", "bin", "claude.cmd"),
			filepath.Join(homeDir, "AppData", "Local", "Programs", "claude", "claude.exe"),
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "claude.cmd"),
			filepath.Join(homeDir, "AppData", "Roaming", "npm", "claude.exe"),
		}
	} else {
		// Unix-like paths
		commonPaths = []string{
			filepath.Join(homeDir, ".claude/local/claude"),
			filepath.Join(homeDir, ".npm-global/bin/claude"),
			filepath.Join(homeDir, ".volta/bin/claude"),
			"/usr/local/bin/claude",
			"/opt/homebrew/bin/claude",
			filepath.Join(homeDir, ".local/bin/claude"),
		}
	}

	// Add npm global prefix bin directory (for nvm and other npm managers)
	if npmPrefix := getNpmPrefix(); npmPrefix != "" {
		if runtime.GOOS == "windows" {
			// On Windows, npm installs .cmd files
			commonPaths = append(commonPaths,
				filepath.Join(npmPrefix, "claude.cmd"),
				filepath.Join(npmPrefix, "claude.exe"))
		} else {
			npmClaudePath := filepath.Join(npmPrefix, "bin", "claude")
			commonPaths = append(commonPaths, npmClaudePath)
		}
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

// resolveClaudeAlias attempts to resolve claude alias from shell configuration
func resolveClaudeAlias() (string, error) {
	// Skip on Windows as Unix shell commands won't work
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("shell alias resolution not supported on Windows")
	}

	// Detect the user's shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	// Extract shell name from path
	shellName := filepath.Base(shell)

	// Try to get alias definition using shell built-in alias command
	var cmd *exec.Cmd
	switch shellName {
	case "bash", "zsh":
		// Use type command which is more reliable for alias resolution
		cmd = exec.Command(shell, "-i", "-c", "type -p claude 2>/dev/null || alias claude 2>/dev/null | grep -oE \"='[^']*'|=\\\"[^\\\"]*\\\"|=[^[:space:]]+\" | sed 's/^=//' | sed 's/^[\"'\"'\"']//;s/[\"'\"'\"']$//'")
	case "fish":
		// Fish shell has different syntax
		cmd = exec.Command(shell, "-c", "functions -D claude 2>/dev/null && which claude 2>/dev/null")
	default:
		// For other shells, try generic approach
		cmd = exec.Command(shell, "-i", "-c", "type -p claude 2>/dev/null")
	}

	output, err := cmd.Output()
	if err != nil {
		// Try alternative method: check common shell config files directly
		return checkShellConfigFiles()
	}

	aliasOutput := strings.TrimSpace(string(output))
	if aliasOutput == "" {
		// Shell command succeeded but returned empty output
		// Try alternative method: check common shell config files directly
		return checkShellConfigFiles()
	}

	// Parse the alias output
	aliasPath := parseAliasOutput(aliasOutput)
	if aliasPath == "" {
		return "", fmt.Errorf("could not parse alias output")
	}

	// Resolve the full path of the aliased command
	if !filepath.IsAbs(aliasPath) {
		resolvedPath, err := exec.LookPath(aliasPath)
		if err == nil {
			aliasPath = resolvedPath
		}
	}

	return aliasPath, nil
}

// parseAliasOutput extracts the actual command from various alias formats
func parseAliasOutput(output string) string {
	// Remove any alias prefix
	output = strings.TrimPrefix(output, "alias claude=")

	// First preserve the original output for complex commands
	originalOutput := output

	// Handle various quote formats
	output = strings.Trim(output, "'\"")

	// Special handling for paths with spaces
	if strings.Contains(originalOutput, "\"") && strings.Contains(output, " ") {
		// This was a quoted path with spaces
		return output
	}

	// If it's a Windows path (contains backslash), return as-is
	if strings.Contains(output, "\\") {
		return output
	}

	// If the alias contains arguments or complex commands, extract just the executable
	// Handle cases like: node /path/to/claude.js, npx @anthropic-ai/claude-code, etc.
	parts := strings.Fields(output)
	if len(parts) == 0 {
		return ""
	}

	// First part is the command
	command := parts[0]

	// If command is a interpreter (node, python, etc.) and has a script path, return full command
	interpreters := []string{"node", "node.exe", "python", "python.exe", "python3", "python3.exe"}
	for _, interp := range interpreters {
		if command == interp && len(parts) > 1 {
			// Check if the second part is a file path
			scriptPath := parts[1]
			if filepath.IsAbs(scriptPath) || strings.Contains(scriptPath, "/") || strings.Contains(scriptPath, "\\") {
				return output // Return the full command with interpreter
			}
		}
	}

	return command
}

// checkShellConfigFiles directly reads shell configuration files for alias definitions
func checkShellConfigFiles() (string, error) {
	// Skip on Windows as Unix shell config files don't exist
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("shell config file checking not supported on Windows")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to HOME env var
		home = os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE") // Windows fallback
		}
		if home == "" {
			return "", fmt.Errorf("unable to determine home directory")
		}
	}

	// Common shell config files to check
	configFiles := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".bash_profile"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".config/fish/config.fish"),
	}

	for _, configFile := range configFiles {
		if aliasPath, found := searchAliasInFile(configFile); found {
			return aliasPath, nil
		}
	}

	return "", fmt.Errorf("no alias found in shell config files")
}

// searchAliasInFile searches for claude alias in a specific config file
func searchAliasInFile(filepath string) (string, bool) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", false
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Check for alias definition
		if strings.HasPrefix(line, "alias claude=") {
			aliasValue := strings.TrimPrefix(line, "alias claude=")
			aliasPath := parseAliasOutput(aliasValue)
			if aliasPath != "" {
				return aliasPath, true
			}
		}
	}

	return "", false
}

// isValidClaudeCLI verifies that the found executable is actually Claude CLI
func isValidClaudeCLI(path string) bool {
	// Handle interpreter-based commands (e.g., "node /path/to/claude.js")
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

	// Skip symlink creation on Windows as it requires admin privileges
	if runtime.GOOS == "windows" {
		// On Windows, we rely on claude being in PATH or using full path
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
	// Skip on Windows as we don't create symlinks there
	if runtime.GOOS == "windows" {
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	symlinkPath := filepath.Join(homeDir, ".local/bin", "claude")

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

// getNpmPrefix gets the npm global installation prefix
func getNpmPrefix() string {
	cmd := exec.Command("npm", "config", "get", "prefix")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	prefix := strings.TrimSpace(string(output))
	if prefix == "" || prefix == "undefined" {
		return ""
	}

	return prefix
}

// Execute runs Claude with the given input
func (c *RealClaudeClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	args := []string{}
	if outputFormat != "" {
		args = append(args, "--output-format", outputFormat)
	}

	var cmd *exec.Cmd

	// Check if claudePath contains interpreter command (e.g., "node /path/to/claude.js")
	if strings.Contains(c.claudePath, " ") {
		// Parse command with interpreter
		parts := strings.Fields(c.claudePath)
		if len(parts) >= 2 {
			// First part is interpreter, rest are script and its args
			interpreter := parts[0]
			scriptAndArgs := append(parts[1:], args...)
			cmd = exec.CommandContext(ctx, interpreter, scriptAndArgs...)
		} else {
			// Fallback to direct execution
			cmd = exec.CommandContext(ctx, c.claudePath, args...)
		}
	} else {
		// Direct command execution
		cmd = exec.CommandContext(ctx, c.claudePath, args...)
	}

	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Remove process group setting to allow proper signal handling
	// cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Run(); err != nil {
		// Check if it's a context cancellation
		if ctx.Err() == context.Canceled {
			return "", ctx.Err()
		}
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
