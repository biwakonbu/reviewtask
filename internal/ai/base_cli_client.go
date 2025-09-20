package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reviewtask/internal/config"
	"runtime"
	"strings"
	"time"
)

// BaseCLIClient provides common CLI functionality for AI providers
type BaseCLIClient struct {
	cliPath      string
	model        string
	providerConf ProviderConfig
}

// NewBaseCLIClient creates a new base CLI client with the given provider configuration
func NewBaseCLIClient(cfg *config.Config, providerConf ProviderConfig) (*BaseCLIClient, error) {
	// Find CLI path
	cliPath, err := findCLI(providerConf)
	if err != nil {
		return nil, fmt.Errorf("%s command not found: %w\n\nTo resolve this issue:\n1. Install %s CLI: npm install -g %s\n2. Or ensure %s is in your PATH\n3. Or create an alias '%s' pointing to your %s CLI installation",
			providerConf.CommandName, err, providerConf.Name, providerConf.CommandName,
			providerConf.CommandName, providerConf.CommandName, providerConf.Name)
	}

	// Check if CLI is in PATH, create symlink if needed
	if _, pathErr := exec.LookPath(providerConf.CommandName); pathErr != nil {
		log.Printf("ℹ️  %s CLI found at %s but not in PATH", providerConf.Name, cliPath)
		if err := ensureCLIAvailable(cliPath, providerConf.CommandName); err != nil {
			return nil, fmt.Errorf("failed to ensure %s availability: %w", providerConf.CommandName, err)
		}
		log.Printf("✓ Created symlink at ~/.local/bin/%s for reviewtask compatibility", providerConf.CommandName)
	}

	// Get model from config or use provider default
	model := providerConf.DefaultModel
	if cfg != nil && cfg.AISettings.Model != "" {
		if cfg.AISettings.Model == "auto" {
			// For "auto", use provider's default model
			model = providerConf.DefaultModel
		} else {
			model = cfg.AISettings.Model
		}
	}

	client := &BaseCLIClient{
		cliPath:      cliPath,
		model:        model,
		providerConf: providerConf,
	}

	// Check authentication unless skipped
	skipAuthCheck := os.Getenv(providerConf.AuthEnvVar) == "true"
	if cfg != nil && cfg.AISettings.SkipClaudeAuthCheck {
		skipAuthCheck = true
	}

	if !skipAuthCheck {
		if err := client.CheckAuthentication(); err != nil {
			return nil, fmt.Errorf("%s CLI authentication check failed: %w\n\nTo authenticate:\n1. Run: %s\n2. Follow the authentication prompts\n\nOr skip this check by setting:\n- Environment variable: %s=true\n- Config file: \"skip_claude_auth_check\": true",
				providerConf.Name, err, providerConf.LoginCommand, providerConf.AuthEnvVar)
		}
	}

	return client, nil
}

// CheckAuthentication verifies that the CLI is properly authenticated
func (c *BaseCLIClient) CheckAuthentication() error {
	// Skip if environment variable is set
	if os.Getenv(c.providerConf.AuthEnvVar) == "true" {
		return nil
	}

	ctx := context.Background()
	testInput := "test"

	// Build command
	args := []string{}
	if c.model != "" {
		args = append(args, "--model", c.model)
	}
	args = append(args, "--print", "--output-format", "json")

	cmd := c.buildCommand(ctx, args)
	cmd.Stdin = strings.NewReader(testInput)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command - we expect it might fail if not authenticated
	_ = cmd.Run() // Ignore the error, we'll check the output

	// Parse the response to check for authentication error
	var response struct {
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &response); err == nil {
		if response.IsError && (strings.Contains(strings.ToLower(response.Result), "api key") ||
			strings.Contains(strings.ToLower(response.Result), "login") ||
			strings.Contains(strings.ToLower(response.Result), "logged")) {
			return fmt.Errorf("%s CLI is not authenticated: %s", c.providerConf.Name, response.Result)
		}
	}

	// Special case for cursor-agent status command
	if c.providerConf.Name == "cursor" {
		cmd = exec.CommandContext(ctx, c.cliPath, "status")
		output, err := cmd.CombinedOutput()
		outputStr := strings.ToLower(string(output))
		if err != nil || !strings.Contains(outputStr, "logged in") {
			return fmt.Errorf("%s is not authenticated. Please run: %s",
				c.providerConf.CommandName, c.providerConf.LoginCommand)
		}
	}

	return nil
}

// executeCursorWithTimeout executes cursor-agent with automatic termination when JSON response is complete
func (c *BaseCLIClient) executeCursorWithTimeout(cmd *exec.Cmd, stdout, stderr *bytes.Buffer, timeoutSeconds int) error {
	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cursor-agent: %w", err)
	}

	// Create channels for monitoring output and completion
	done := make(chan error, 1)
	outputReady := make(chan bool, 1)

	// Monitor stdout for JSON response completion
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		// Allow up to 10MB JSON lines to handle large/minified JSON
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		accumulatedOutput := ""

		for scanner.Scan() {
			line := scanner.Text()
			stdout.WriteString(line + "\n")
			accumulatedOutput += line

			// Check if we received a complete JSON response
			if c.isCompleteJSONResponse(accumulatedOutput) {
				outputReady <- true
				return
			}
		}
	}()

	// Monitor stderr separately
	go func() {
		io.Copy(stderr, stderrPipe)
	}()

	// Wait for completion with timeout
	go func() {
		done <- cmd.Wait()
	}()

	timeout := time.Duration(timeoutSeconds) * time.Second
	select {
	case <-outputReady:
		// Got complete JSON response, terminate the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		// Wait a moment for graceful shutdown
		time.Sleep(100 * time.Millisecond)
		return nil
	case err := <-done:
		// Process finished naturally
		return err
	case <-time.After(timeout):
		// Timeout reached, kill the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("cursor-agent execution timed out after %d seconds", timeoutSeconds)
	}
}

// isCompleteJSONResponse checks if the accumulated output contains a complete JSON response
func (c *BaseCLIClient) isCompleteJSONResponse(output string) bool {
	// First check for cursor-agent specific response patterns
	if strings.Contains(output, `"type":"result"`) {
		// Look for complete JSON structure
		openBraces := strings.Count(output, "{")
		closeBraces := strings.Count(output, "}")

		// Must have at least one complete JSON object
		if openBraces > 0 && closeBraces >= openBraces {
			// Try to parse as JSON to verify completeness
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(output), &response); err == nil {
				// Check if it has the expected result field
				if result, exists := response["result"]; exists && result != nil {
					return true
				}
			}
		}
	}

	// Also check for simple JSON array responses (for task generation)
	if strings.HasPrefix(strings.TrimSpace(output), "[") {
		openBrackets := strings.Count(output, "[")
		closeBrackets := strings.Count(output, "]")

		if openBrackets > 0 && closeBrackets >= openBrackets {
			var jsonArray []interface{}
			if err := json.Unmarshal([]byte(output), &jsonArray); err == nil {
				return true
			}
		}
	}

	return false
}

// Execute runs the CLI with the given input
func (c *BaseCLIClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	args := []string{}

	// Provider-specific command building
	if c.providerConf.Name == "cursor" {
		// cursor-agent: use -p option for prompt input
		args = append(args, "-p", input)

		// Add model parameter if specified and not "auto" (cursor-agent uses auto by default)
		if c.model != "" && c.model != "auto" {
			args = append(args, "--model", c.model)
		}

		// Use text format for cursor-agent to avoid streaming issues
		// JSON format causes the process to hang waiting for stream termination
		if outputFormat == "json" || outputFormat == "" {
			args = append(args, "--output-format", "text")
		} else {
			args = append(args, "--output-format", outputFormat)
		}
	} else {
		// claude: use traditional stdin approach
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
	}

	cmd := c.buildCommand(ctx, args)

	// Set stdin only for claude provider
	if c.providerConf.Name != "cursor" {
		cmd.Stdin = strings.NewReader(input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Debug logging
	if os.Getenv("REVIEWTASK_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: Executing %s command: %s %s\n",
			c.providerConf.Name, cmd.Path, strings.Join(cmd.Args[1:], " "))
	}

	// Handle cursor-agent specific execution (doesn't auto-terminate with JSON)
	if c.providerConf.Name == "cursor" && outputFormat == "json" {
		// For cursor with JSON output (which we changed to text), use standard execution
		// since we're now using text format to avoid streaming issues
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("%s execution failed: %w, stderr: %s, stdout: %.200s",
				c.providerConf.CommandName, err, stderr.String(), stdout.String())
		}
	} else {
		// Standard execution for other providers
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("%s execution failed: %w, stderr: %s, stdout: %.200s",
				c.providerConf.CommandName, err, stderr.String(), stdout.String())
		}
	}

	output := stdout.String()

	// If output format is JSON, extract the result field
	if outputFormat == "json" || (c.providerConf.Name == "cursor" && outputFormat == "") {
		// Handle cursor-agent specific response format
		if c.providerConf.Name == "cursor" {
			// Since we're using text format for cursor to avoid hanging,
			// return the text output directly
			return output, nil
		} else {
			// Handle claude response format
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
				return "", fmt.Errorf("%s API error: %s", c.providerConf.Name, response.Error)
			}

			return response.Result, nil
		}
	}

	return output, nil
}

// buildCommand creates an exec.Cmd with the appropriate setup
func (c *BaseCLIClient) buildCommand(ctx context.Context, args []string) *exec.Cmd {
	return exec.CommandContext(ctx, c.cliPath, args...)
}

// findCLI searches for the CLI executable in common locations
func findCLI(providerConf ProviderConfig) (string, error) {
	// Try standard PATH first
	if path, err := exec.LookPath(providerConf.CommandName); err == nil {
		return path, nil
	}

	// Try common installation locations
	for _, path := range providerConf.CommonPaths {
		if _, err := os.Stat(path); err == nil {
			// Verify it's executable by running version check
			if isValidCLI(path, providerConf.Name) {
				return path, nil
			}
		}
	}

	// Add npm global prefix bin directory
	if npmPrefix := getNpmPrefix(); npmPrefix != "" {
		if runtime.GOOS == "windows" {
			// On Windows, npm installs .cmd files
			npmPaths := []string{
				filepath.Join(npmPrefix, providerConf.CommandName+".cmd"),
				filepath.Join(npmPrefix, providerConf.CommandName+".exe"),
			}
			for _, path := range npmPaths {
				if _, err := os.Stat(path); err == nil && isValidCLI(path, providerConf.Name) {
					return path, nil
				}
			}
		} else {
			npmPath := filepath.Join(npmPrefix, "bin", providerConf.CommandName)
			if _, err := os.Stat(npmPath); err == nil && isValidCLI(npmPath, providerConf.Name) {
				return npmPath, nil
			}
		}
	}

	return "", fmt.Errorf("%s CLI not found in PATH or common installation locations", providerConf.CommandName)
}

// isValidCLI verifies that the found executable is actually the expected CLI
func isValidCLI(path string, providerName string) bool {
	// Validate provider name is not empty
	if providerName == "" {
		return false
	}

	cmd := exec.Command(path, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	outputStr := strings.ToLower(string(output))
	// Only check if output contains the provider name or command name
	return strings.Contains(outputStr, strings.ToLower(providerName)) ||
		strings.Contains(outputStr, strings.ToLower(filepath.Base(path)))
}

// ensureCLIAvailable creates symlink if CLI is not in PATH
func ensureCLIAvailable(cliPath string, commandName string) error {
	// Check if already available in PATH
	if _, err := exec.LookPath(commandName); err == nil {
		return nil
	}

	// Skip symlink creation on Windows as it requires admin privileges
	if runtime.GOOS == "windows" {
		return nil
	}

	// Create symlink in ~/.local/bin (which is commonly in PATH)
	homeDir := getHomeDir()
	localBin := filepath.Join(homeDir, ".local/bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return fmt.Errorf("failed to create ~/.local/bin directory: %w", err)
	}

	symlinkPath := filepath.Join(localBin, commandName)

	// Remove existing symlink if it exists
	if fi, err := os.Lstat(symlinkPath); err == nil {
		// Only remove if it's a symlink, not a regular file or directory
		if fi.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to overwrite non-symlink path: %s", symlinkPath)
		}
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	if err := os.Symlink(cliPath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}
