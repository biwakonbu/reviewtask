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

// Execute runs the CLI with the given input
func (c *BaseCLIClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	// Special handling for cursor-agent - optimized for single JSON response
	if c.providerConf.Name == "cursor" {
		// Build arguments for cursor-agent
		cursorArgs := []string{"-p"}

		// Add model parameter if specified and not "auto" (cursor-agent uses auto by default)
		if c.model != "" && c.model != "auto" {
			cursorArgs = append(cursorArgs, "--model", c.model)
		}

		// Use JSON format for cursor-agent
		if outputFormat == "json" || outputFormat == "" {
			cursorArgs = append(cursorArgs, "--output-format", "json")
		} else {
			cursorArgs = append(cursorArgs, "--output-format", outputFormat)
		}

		// Validate cursor-agent binary before execution
		if err := c.validateCursorAgent(); err != nil {
			return "", fmt.Errorf("cursor-agent validation failed: %w", err)
		}

		// Execute cursor-agent with reasonable timeout
		// cursor-agent outputs single JSON response, so shorter timeout is fine
		timeoutSeconds := 120 // 2 minutes - sufficient for single API call
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()

		cmd := exec.CommandContext(timeoutCtx, c.cliPath, cursorArgs...)
		cmd.Stdin = strings.NewReader(input)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Debug logging (only when REVIEWTASK_DEBUG=true)
		if os.Getenv("REVIEWTASK_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "DEBUG: Executing cursor-agent (input: %d bytes, timeout: %d seconds)\n", len(input), timeoutSeconds)
		}

		// Execute and wait for completion
		err := cmd.Run()

		// Check if this was a timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			if os.Getenv("REVIEWTASK_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "DEBUG: Command timed out, force killing process\n")
			}
			if cmd.Process != nil {
				// Process is still running, kill it
				cmd.Process.Kill()
			}
			return "", fmt.Errorf("%s execution timed out after %d seconds (input length: %d bytes)",
				c.providerConf.CommandName, timeoutSeconds, len(input))
		}

		if err != nil {
			// Analyze the error for better error messages
			stderrStr := stderr.String()
			stdoutStr := stdout.String()

			if os.Getenv("REVIEWTASK_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "DEBUG: Command failed - stdout: %s, stderr: %s\n", stdoutStr, stderrStr)
			}

			// Check for common cursor-agent errors
			if c.providerConf.Name == "cursor" {
				if strings.Contains(stderrStr, "API key") || strings.Contains(stderrStr, "authentication") {
					return "", fmt.Errorf("%s authentication failed: please run 'cursor-agent login' and try again", c.providerConf.CommandName)
				}
				if strings.Contains(stderrStr, "network") || strings.Contains(stderrStr, "connection") {
					return "", fmt.Errorf("%s network error: please check your internet connection and try again", c.providerConf.CommandName)
				}
				if strings.Contains(stderrStr, "rate limit") {
					return "", fmt.Errorf("%s rate limit exceeded: please wait a moment and try again", c.providerConf.CommandName)
				}
			}

			return "", fmt.Errorf("%s execution failed: %w\nstdout: %.200s\nstderr: %.200s",
				c.providerConf.CommandName, err, stdoutStr, stderrStr)
		}

		outputStr := stdout.String()
		if os.Getenv("REVIEWTASK_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "DEBUG: cursor-agent completed successfully (output length: %d bytes)\n", len(outputStr))
			fmt.Fprintf(os.Stderr, "DEBUG: AI Response: %s\n", outputStr)
		}

		// For cursor-agent, extract the actual result from the JSON response
		if outputFormat == "json" || outputFormat == "" {
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(outputStr), &response); err == nil {
				if result, exists := response["result"]; exists && result != nil {
					// Return the result field content directly
					if resultStr, ok := result.(string); ok {
						return resultStr, nil
					}
				}
			}
		}

		return outputStr, nil
	}

	// Standard execution for claude and other providers
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

	cmd := c.buildCommand(ctx, args)

	// Set stdin for non-cursor providers
	if c.providerConf.Name != "cursor" {
		cmd.Stdin = strings.NewReader(input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Debug logging
	if os.Getenv("REVIEWTASK_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: Executing %s command: %s %s (input length: %d)\n",
			c.providerConf.Name, cmd.Path, strings.Join(cmd.Args[1:], " "), len(input))
	}

	// Standard execution for all providers
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s execution failed: %w, stderr: %s, stdout: %.200s",
			c.providerConf.CommandName, err, stderr.String(), stdout.String())
	}

	output := stdout.String()

	// If output format is JSON, extract the result field
	if outputFormat == "json" || (c.providerConf.Name == "cursor" && outputFormat == "") {
		// Handle cursor-agent specific response format
		if c.providerConf.Name == "cursor" {
			var cursorResponse struct {
				Type    string `json:"type"`
				Subtype string `json:"subtype"`
				IsError bool   `json:"is_error"`
				Result  string `json:"result"`
				Error   string `json:"error,omitempty"`
			}

			if err := json.Unmarshal([]byte(output), &cursorResponse); err != nil {
				if os.Getenv("REVIEWTASK_DEBUG") == "true" {
					fmt.Fprintf(os.Stderr, "DEBUG: Failed to parse cursor-agent JSON response: %v\n", err)
					fmt.Fprintf(os.Stderr, "DEBUG: Raw response: %.500s\n", output)
				}
				// If unmarshaling fails, try to extract result from partial JSON
				if strings.Contains(output, `"result"`) {
					// Try to extract result field manually
					resultStart := strings.Index(output, `"result"`)
					if resultStart != -1 {
						resultStart = strings.Index(output[resultStart:], `"`) + resultStart
						if resultStart != -1 {
							resultEnd := strings.Index(output[resultStart+1:], `"`)
							if resultEnd != -1 {
								resultEnd += resultStart + 1
								extractedResult := output[resultStart+1 : resultEnd]
								if os.Getenv("REVIEWTASK_DEBUG") == "true" {
									fmt.Fprintf(os.Stderr, "DEBUG: Extracted result from malformed JSON: %s\n", extractedResult)
								}
								return extractedResult, nil
							}
						}
					}
				}
				return "", fmt.Errorf("cursor-agent returned invalid JSON response: %w\nResponse: %.500s", err, output)
			}

			if cursorResponse.IsError {
				errorMsg := cursorResponse.Error
				if errorMsg == "" {
					errorMsg = cursorResponse.Result
				}
				return "", fmt.Errorf("%s API error: %s", c.providerConf.Name, errorMsg)
			}

			return cursorResponse.Result, nil
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

// validateCursorAgent validates that cursor-agent binary exists and is executable
func (c *BaseCLIClient) validateCursorAgent() error {
	if c.providerConf.Name != "cursor" {
		return nil // Skip validation for non-cursor providers
	}

	// Check if binary exists
	if _, err := os.Stat(c.cliPath); os.IsNotExist(err) {
		return fmt.Errorf("cursor-agent binary not found at %s: %w", c.cliPath, err)
	}

	// Check if binary is executable
	if info, err := os.Stat(c.cliPath); err != nil {
		return fmt.Errorf("cannot stat cursor-agent binary: %w", err)
	} else if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("cursor-agent binary is not executable: %s", c.cliPath)
	}

	// Try to run cursor-agent --version to verify it's working
	testCmd := exec.Command(c.cliPath, "--version")
	testCmd.Stdout = nil
	testCmd.Stderr = nil
	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("cursor-agent --version failed: %w (binary may be corrupted or incompatible)", err)
	}

	return nil
}
