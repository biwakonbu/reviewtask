package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test struct for credentials (the actual implementation doesn't have this)
type Credentials struct {
	Method    string    `json:"method"`
	Token     string    `json:"token"`
	Timestamp time.Time `json:"timestamp"`
}

// TestCredentialsManagement tests basic credentials operations
func TestCredentialsManagement(t *testing.T) {
	tests := []struct {
		name     string
		creds    *Credentials
		validate func(t *testing.T, creds *Credentials)
	}{
		{
			name: "トークン認証の作成",
			creds: &Credentials{
				Method:    "token",
				Token:     "ghp_test123456789",
				Timestamp: time.Now(),
			},
			validate: func(t *testing.T, creds *Credentials) {
				if creds.Method != "token" {
					t.Errorf("Expected method 'token', got %s", creds.Method)
				}
				if creds.Token == "" {
					t.Error("Token should not be empty")
				}
			},
		},
		{
			name: "GH CLI認証の作成",
			creds: &Credentials{
				Method:    "gh_cli",
				Token:     "",
				Timestamp: time.Now(),
			},
			validate: func(t *testing.T, creds *Credentials) {
				if creds.Method != "gh_cli" {
					t.Errorf("Expected method 'gh_cli', got %s", creds.Method)
				}
			},
		},
		{
			name: "環境変数認証の作成",
			creds: &Credentials{
				Method:    "env",
				Token:     "env_token_123",
				Timestamp: time.Now(),
			},
			validate: func(t *testing.T, creds *Credentials) {
				if creds.Method != "env" {
					t.Errorf("Expected method 'env', got %s", creds.Method)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.creds)
		})
	}
}

// TestSaveAndLoadCredentials tests credential persistence
func TestSaveAndLoadCredentials(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Create .pr-review/auth directory
	authDir := filepath.Join(".pr-review", "auth")
	os.MkdirAll(authDir, 0755)

	tests := []struct {
		name        string
		creds       *Credentials
		expectError bool
	}{
		{
			name: "正常な認証情報の保存と読み込み",
			creds: &Credentials{
				Method:    "token",
				Token:     "test-token-save-load",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "空のトークンでの保存",
			creds: &Credentials{
				Method:    "gh_cli",
				Token:     "",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "特殊文字を含むトークン",
			creds: &Credentials{
				Method:    "token",
				Token:     "token!@#$%^&*()_+-=[]{}|;:,.<>?",
				Timestamp: time.Now(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save credentials
			err := SaveCredentials(tt.creds)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				// Load credentials
				loaded, err := LoadCredentials()
				if err != nil {
					t.Errorf("Failed to load credentials: %v", err)
				}

				// Verify loaded credentials match
				if loaded.Method != tt.creds.Method {
					t.Errorf("Method mismatch: expected %s, got %s", tt.creds.Method, loaded.Method)
				}
				if loaded.Token != tt.creds.Token {
					t.Errorf("Token mismatch: expected %s, got %s", tt.creds.Token, loaded.Token)
				}

				// Check file permissions
				credFile := filepath.Join(authDir, "credentials.json")
				info, err := os.Stat(credFile)
				if err != nil {
					t.Errorf("Failed to stat credentials file: %v", err)
				}
				if info.Mode().Perm() != 0600 {
					t.Errorf("Incorrect file permissions: %v", info.Mode().Perm())
				}
			}
		})
	}
}

// TestDetectAuthentication tests auth detection from various sources
func TestDetectAuthentication(t *testing.T) {
	scenarios := []struct {
		name      string
		setup     func() func()
		expected  *Credentials
		expectErr bool
	}{
		{
			name: "環境変数GITHUB_TOKEN",
			setup: func() func() {
				os.Setenv("GITHUB_TOKEN", "env-github-token")
				return func() { os.Unsetenv("GITHUB_TOKEN") }
			},
			expected: &Credentials{
				Method: "env",
				Token:  "env-github-token",
			},
			expectErr: false,
		},
		{
			name: "環境変数GH_TOKEN",
			setup: func() func() {
				os.Setenv("GH_TOKEN", "env-gh-token")
				return func() { os.Unsetenv("GH_TOKEN") }
			},
			expected: &Credentials{
				Method: "env",
				Token:  "env-gh-token",
			},
			expectErr: false,
		},
		{
			name: "ローカル認証ファイル",
			setup: func() func() {
				tempDir := t.TempDir()
				os.Chdir(tempDir)
				os.MkdirAll(filepath.Join(".pr-review", "auth"), 0755)

				creds := &Credentials{
					Method:    "token",
					Token:     "local-file-token",
					Timestamp: time.Now(),
				}
				SaveCredentials(creds)

				return func() {}
			},
			expected: &Credentials{
				Method: "token",
				Token:  "local-file-token",
			},
			expectErr: false,
		},
		{
			name: "認証情報なし",
			setup: func() func() {
				// Create temp directory to ensure no local auth file exists
				tempDir := t.TempDir()
				originalDir, _ := os.Getwd()
				os.Chdir(tempDir)

				os.Unsetenv("GITHUB_TOKEN")
				os.Unsetenv("GH_TOKEN")
				os.Unsetenv("REVIEWTASK_GITHUB_TOKEN")
				return func() {
					os.Chdir(originalDir)
				}
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "複数の認証ソース（優先順位テスト）",
			setup: func() func() {
				tempDir := t.TempDir()
				os.Chdir(tempDir)

				// Setup local file
				os.MkdirAll(filepath.Join(".pr-review", "auth"), 0755)
				localCreds := &Credentials{
					Method:    "token",
					Token:     "local-priority-token",
					Timestamp: time.Now(),
				}
				SaveCredentials(localCreds)

				// Also set environment variable
				os.Setenv("GITHUB_TOKEN", "env-priority-token")

				return func() {
					os.Unsetenv("GITHUB_TOKEN")
				}
			},
			expected: &Credentials{
				Method: "token",
				Token:  "local-priority-token", // Local should win
			},
			expectErr: false,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			creds, err := DetectAuthentication()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expected != nil && creds != nil {
				if creds.Method != tt.expected.Method {
					t.Errorf("Method mismatch: expected %s, got %s", tt.expected.Method, creds.Method)
				}
				if creds.Token != tt.expected.Token {
					t.Errorf("Token mismatch: expected %s, got %s", tt.expected.Token, creds.Token)
				}
			}
		})
	}
}

// TestAuthenticationScenarios tests complete authentication workflows
func TestAuthenticationScenarios(t *testing.T) {
	scenarios := []struct {
		name  string
		steps []authStep
	}{
		{
			name: "初回セットアップフロー",
			steps: []authStep{
				{
					name: "認証情報なしの状態確認",
					action: func() (*Credentials, error) {
						return DetectAuthentication()
					},
					expectError: true,
				},
				{
					name: "環境変数を設定",
					setup: func() {
						os.Setenv("GITHUB_TOKEN", "setup-token")
					},
					action: func() (*Credentials, error) {
						return DetectAuthentication()
					},
					expectError: false,
					validate: func(t *testing.T, creds *Credentials) {
						if creds.Token != "setup-token" {
							t.Errorf("Expected token 'setup-token', got %s", creds.Token)
						}
					},
				},
			},
		},
		{
			name: "認証情報の更新フロー",
			steps: []authStep{
				{
					name: "古い認証情報を保存",
					setup: func() {
						tempDir := t.TempDir()
						os.Chdir(tempDir)
						os.MkdirAll(filepath.Join(".pr-review", "auth"), 0755)

						oldCreds := &Credentials{
							Method:    "token",
							Token:     "old-token",
							Timestamp: time.Now().Add(-48 * time.Hour),
						}
						SaveCredentials(oldCreds)
					},
					action: func() (*Credentials, error) {
						return LoadCredentials()
					},
					expectError: false,
					validate: func(t *testing.T, creds *Credentials) {
						if creds.Token != "old-token" {
							t.Errorf("Expected old token, got %s", creds.Token)
						}
					},
				},
				{
					name: "新しい認証情報で更新",
					action: func() (*Credentials, error) {
						newCreds := &Credentials{
							Method:    "token",
							Token:     "new-token",
							Timestamp: time.Now(),
						}
						err := SaveCredentials(newCreds)
						if err != nil {
							return nil, err
						}
						return LoadCredentials()
					},
					expectError: false,
					validate: func(t *testing.T, creds *Credentials) {
						if creds.Token != "new-token" {
							t.Errorf("Expected new token, got %s", creds.Token)
						}
					},
				},
			},
		},
		{
			name: "CI環境での認証",
			steps: []authStep{
				{
					name: "GitHub Actions環境をシミュレート",
					setup: func() {
						// Create clean temp directory to avoid interference from previous tests
						tempDir := t.TempDir()
						os.Chdir(tempDir)

						os.Setenv("CI", "true")
						os.Setenv("GITHUB_ACTIONS", "true")
						os.Setenv("GITHUB_TOKEN", "gha-token")
					},
					action: func() (*Credentials, error) {
						return DetectAuthentication()
					},
					expectError: false,
					validate: func(t *testing.T, creds *Credentials) {
						if creds.Token != "gha-token" {
							t.Errorf("Expected GHA token, got %s", creds.Token)
						}
					},
					cleanup: func() {
						os.Unsetenv("CI")
						os.Unsetenv("GITHUB_ACTIONS")
						os.Unsetenv("GITHUB_TOKEN")
					},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for _, step := range scenario.steps {
				t.Run(step.name, func(t *testing.T) {
					if step.setup != nil {
						step.setup()
					}

					creds, err := step.action()

					if step.expectError && err == nil {
						t.Error("Expected error but got none")
					}
					if !step.expectError && err != nil {
						t.Errorf("Unexpected error: %v", err)
					}

					if step.validate != nil && creds != nil {
						step.validate(t, creds)
					}

					if step.cleanup != nil {
						step.cleanup()
					}
				})
			}
		})
	}
}

type authStep struct {
	name        string
	setup       func()
	action      func() (*Credentials, error)
	expectError bool
	validate    func(t *testing.T, creds *Credentials)
	cleanup     func()
}

// TestCredentialsValidation tests credential validation logic
func TestCredentialsValidation(t *testing.T) {
	tests := []struct {
		name        string
		creds       *Credentials
		expectValid bool
	}{
		{
			name: "有効なトークン認証",
			creds: &Credentials{
				Method: "token",
				Token:  "ghp_validtoken123",
			},
			expectValid: true,
		},
		{
			name: "有効なGH CLI認証",
			creds: &Credentials{
				Method: "gh_cli",
				Token:  "",
			},
			expectValid: true,
		},
		{
			name: "トークンなしのtoken認証",
			creds: &Credentials{
				Method: "token",
				Token:  "",
			},
			expectValid: false,
		},
		{
			name: "不明な認証方法",
			creds: &Credentials{
				Method: "unknown",
				Token:  "token",
			},
			expectValid: false,
		},
		{
			name:        "nil認証情報",
			creds:       nil,
			expectValid: false,
		},
		{
			name: "空白のみのトークン",
			creds: &Credentials{
				Method: "token",
				Token:  "   ",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := ValidateCredentials(tt.creds)
			if isValid != tt.expectValid {
				t.Errorf("Expected validation result %v, got %v", tt.expectValid, isValid)
			}
		})
	}
}

// TestAuthenticationErrorHandling tests error handling
func TestAuthenticationErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() func()
		operation func() error
		expectErr bool
	}{
		{
			name: "権限のないディレクトリへの保存",
			setup: func() func() {
				tempDir := t.TempDir()
				os.Chdir(tempDir)

				// Create read-only directory
				authDir := filepath.Join(".pr-review", "auth")
				os.MkdirAll(authDir, 0555)

				return func() {
					os.Chmod(authDir, 0755)
				}
			},
			operation: func() error {
				creds := &Credentials{
					Method: "token",
					Token:  "test",
				}
				return SaveCredentials(creds)
			},
			expectErr: true,
		},
		{
			name: "破損した認証ファイルの読み込み",
			setup: func() func() {
				tempDir := t.TempDir()
				os.Chdir(tempDir)

				authDir := filepath.Join(".pr-review", "auth")
				os.MkdirAll(authDir, 0755)

				// Write corrupted JSON
				credFile := filepath.Join(authDir, "credentials.json")
				os.WriteFile(credFile, []byte("{ invalid json }"), 0600)

				return func() {}
			},
			operation: func() error {
				_, err := LoadCredentials()
				return err
			},
			expectErr: true,
		},
		{
			name: "存在しないファイルの読み込み",
			setup: func() func() {
				tempDir := t.TempDir()
				os.Chdir(tempDir)
				return func() {}
			},
			operation: func() error {
				_, err := LoadCredentials()
				return err
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			err := tt.operation()

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestTokenPatterns tests GitHub token pattern recognition
func TestTokenPatterns(t *testing.T) {
	patterns := []struct {
		token       string
		description string
		shouldMatch bool
	}{
		{"ghp_16CharacterToken1234567890ABCDEF", "Classic Personal Access Token", true},
		{"github_pat_22CharToken_1234567890ABCDEFGHIJKLMNOPQRS", "Fine-grained PAT", true},
		{"gho_16CharOAuthToken123456", "OAuth Access Token", true},
		{"ghu_16CharUserToken1234567", "GitHub App user token", true},
		{"ghs_16CharServerToken123456", "GitHub App server token", true},
		{"ghr_16CharRefreshToken12345", "GitHub App refresh token", true},
		{"invalid_token", "Invalid token format", false},
		{"", "Empty token", false},
		{"ghp_", "Incomplete token", false},
	}

	for _, p := range patterns {
		t.Run(p.description, func(t *testing.T) {
			isValid := IsValidGitHubToken(p.token)
			if isValid != p.shouldMatch {
				t.Errorf("Token %q: expected %v, got %v", p.token, p.shouldMatch, isValid)
			}
		})
	}
}

// TestConcurrentAuthOperations tests thread safety
func TestConcurrentAuthOperations(t *testing.T) {
	tempDir := t.TempDir()
	os.Chdir(tempDir)
	os.MkdirAll(filepath.Join(".pr-review", "auth"), 0755)

	// Run concurrent saves
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			creds := &Credentials{
				Method:    "token",
				Token:     fmt.Sprintf("concurrent-token-%d", id),
				Timestamp: time.Now(),
			}
			SaveCredentials(creds)
			done <- true
		}(i)
	}

	// Wait for all operations
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state is consistent
	loaded, err := LoadCredentials()
	if err != nil {
		t.Errorf("Failed to load after concurrent saves: %v", err)
	}
	if loaded == nil {
		t.Error("No credentials after concurrent saves")
	}

	t.Log("Concurrent operations completed successfully")
}

// Helper functions

func ValidateCredentials(creds *Credentials) bool {
	if creds == nil {
		return false
	}

	if creds.Method == "token" && strings.TrimSpace(creds.Token) == "" {
		return false
	}

	validMethods := []string{"token", "gh_cli", "env"}
	for _, m := range validMethods {
		if creds.Method == m {
			return true
		}
	}

	return false
}

func IsValidGitHubToken(token string) bool {
	if token == "" {
		return false
	}

	// GitHub token patterns
	patterns := []string{
		"ghp_",        // Personal access token (classic)
		"github_pat_", // Fine-grained personal access token
		"gho_",        // OAuth access token
		"ghu_",        // GitHub App user token
		"ghs_",        // GitHub App server token
		"ghr_",        // GitHub App refresh token
	}

	for _, prefix := range patterns {
		if strings.HasPrefix(token, prefix) && len(token) > len(prefix)+10 {
			return true
		}
	}

	return false
}

// Mock functions for testing (these would be in the actual implementation)

func SaveCredentials(creds *Credentials) error {
	authDir := filepath.Join(".pr-review", "auth")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		return fmt.Errorf("failed to create auth directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	credFile := filepath.Join(authDir, "credentials.json")
	// Write to temp file first, then rename atomically
	tempFile := credFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	// Atomic rename to avoid concurrent write issues
	if err := os.Rename(tempFile, credFile); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return nil
}

func LoadCredentials() (*Credentials, error) {
	credFile := filepath.Join(".pr-review", "auth", "credentials.json")

	data, err := os.ReadFile(credFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

func DetectAuthentication() (*Credentials, error) {
	// Check local file first
	if creds, err := LoadCredentials(); err == nil {
		return creds, nil
	}

	// Check environment variables
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return &Credentials{
			Method:    "env",
			Token:     token,
			Timestamp: time.Now(),
		}, nil
	}

	if token := os.Getenv("GH_TOKEN"); token != "" {
		return &Credentials{
			Method:    "env",
			Token:     token,
			Timestamp: time.Now(),
		}, nil
	}

	if token := os.Getenv("REVIEWTASK_GITHUB_TOKEN"); token != "" {
		return &Credentials{
			Method:    "env",
			Token:     token,
			Timestamp: time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("no authentication found")
}

// TestAuthenticationIntegration tests full integration scenarios
func TestAuthenticationIntegration(t *testing.T) {
	t.Run("完全な認証ライフサイクル", func(t *testing.T) {
		tempDir := t.TempDir()
		os.Chdir(tempDir)

		// Phase 1: No auth
		_, err := DetectAuthentication()
		if err == nil {
			t.Error("Should fail with no auth")
		}

		// Phase 2: Setup via environment
		os.Setenv("GITHUB_TOKEN", "lifecycle-token")
		defer os.Unsetenv("GITHUB_TOKEN")

		creds, err := DetectAuthentication()
		if err != nil {
			t.Errorf("Should detect env auth: %v", err)
		}
		if creds.Token != "lifecycle-token" {
			t.Errorf("Wrong token detected: %s", creds.Token)
		}

		// Phase 3: Save to local file
		localCreds := &Credentials{
			Method:    "token",
			Token:     "local-lifecycle-token",
			Timestamp: time.Now(),
		}
		err = SaveCredentials(localCreds)
		if err != nil {
			t.Errorf("Failed to save credentials: %v", err)
		}

		// Phase 4: Local should override env
		creds, err = DetectAuthentication()
		if err != nil {
			t.Errorf("Should detect local auth: %v", err)
		}
		if creds.Token != "local-lifecycle-token" {
			t.Errorf("Local token should override env: %s", creds.Token)
		}

		// Phase 5: Verify persistence
		loaded, err := LoadCredentials()
		if err != nil {
			t.Errorf("Failed to load credentials: %v", err)
		}
		if loaded.Token != "local-lifecycle-token" {
			t.Errorf("Persisted token mismatch: %s", loaded.Token)
		}
	})
}
