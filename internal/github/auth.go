package github

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AuthConfig struct {
	Token string `json:"github_token"`
}

// Simple YAML parser for gh config (avoiding external dependencies)
type GHConfigData struct {
	OAuthToken string
	User       string
}

// GetGitHubToken retrieves GitHub token from various sources in priority order
func GetGitHubToken() (string, error) {
	// 1. Environment variable (highest priority)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}

	// 2. Local auth configuration (higher priority than gh CLI)
	if token, err := getLocalToken(); err == nil && token != "" {
		return token, nil
	}

	// 3. gh CLI configuration
	if token, err := getGHToken(); err == nil && token != "" {
		return token, nil
	}

	// 4. No interactive prompt - return error to guide user to auth command
	return "", fmt.Errorf("no GitHub token found")
}

// getGHToken reads token from gh CLI configuration using simple parsing
func getGHToken() (string, error) {
	// Check if GH_CONFIG_DIR is set (for testing)
	configDir := os.Getenv("GH_CONFIG_DIR")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", "gh")
	}

	configPath := filepath.Join(configDir, "hosts.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	// Simple YAML parsing for oauth_token under github.com
	lines := strings.Split(string(data), "\n")
	inGithubSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "github.com:" {
			inGithubSection = true
			continue
		}

		if inGithubSection && strings.HasPrefix(line, "oauth_token:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				token := strings.TrimSpace(parts[1])
				return token, nil
			}
		}

		// Reset if we hit another top-level section
		if inGithubSection && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && line != "" {
			inGithubSection = false
		}
	}

	return "", fmt.Errorf("oauth_token not found in gh config")
}

// getLocalToken reads token from local auth configuration
func getLocalToken() (string, error) {
	authPath := filepath.Join(".pr-review", "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		return "", err
	}

	var config AuthConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	return config.Token, nil
}

// saveLocalToken saves token to local auth configuration
func saveLocalToken(token string) error {
	// Ensure directory exists
	if err := os.MkdirAll(".pr-review", 0755); err != nil {
		return err
	}

	config := AuthConfig{Token: token}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	authPath := filepath.Join(".pr-review", "auth.json")
	return os.WriteFile(authPath, data, 0600) // 600 permissions for security
}

// promptForToken interactively prompts user for GitHub token
// Deprecated: replaced by PromptForTokenWithSave. Kept commented to avoid unused warnings.
// func promptForToken() (string, error) { return PromptForTokenWithSave() }

// GetTokenWithSource returns both the token and its source
func GetTokenWithSource() (string, string, error) {
	// 1. Environment variable (highest priority)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return "environment variable", token, nil
	}

	// 2. Local auth configuration (higher priority than gh CLI)
	if token, err := getLocalToken(); err == nil && token != "" {
		return fmt.Sprintf("local config (%s)", filepath.Join(".pr-review", "auth.json")), token, nil
	}

	// 3. gh CLI configuration
	if token, err := getGHToken(); err == nil && token != "" {
		return "gh CLI config", token, nil
	}

	return "", "", fmt.Errorf("no authentication found")
}

// PromptForTokenWithSave forces interactive token input with save
func PromptForTokenWithSave() (string, error) {
	fmt.Println("Please provide a GitHub Personal Access Token.")
	fmt.Println("You can create one at: https://github.com/settings/tokens")
	fmt.Println("Required scopes: repo, pull_requests")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter GitHub token: ")

	tokenInput, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read token: %w", err)
	}

	token := strings.TrimSpace(tokenInput)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// Always save for auth login command
	if err := saveLocalToken(token); err != nil {
		return "", fmt.Errorf("failed to save token: %w", err)
	}

	return token, nil
}

// RemoveLocalToken removes the locally stored token
func RemoveLocalToken() error {
	authPath := filepath.Join(".pr-review", "auth.json")
	err := os.Remove(authPath)
	if os.IsNotExist(err) {
		return nil // Already removed
	}
	return err
}
