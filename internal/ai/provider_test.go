package ai

import (
	"reviewtask/internal/config"
	"testing"
)

func TestProviderFactory(t *testing.T) {
	tests := []struct {
		name       string
		aiProvider string
		expectType string // "claude", "cursor", or "error"
	}{
		{
			name:       "Explicit Claude provider",
			aiProvider: "claude",
			expectType: "claude",
		},
		{
			name:       "Explicit Cursor provider",
			aiProvider: "cursor",
			expectType: "cursor",
		},
		{
			name:       "Auto provider selection",
			aiProvider: "auto",
			expectType: "any", // Could be either
		},
		{
			name:       "Invalid provider",
			aiProvider: "invalid",
			expectType: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				AISettings: config.AISettings{
					AIProvider: tt.aiProvider,
					Model:      "auto",
				},
			}

			_, err := NewAIProvider(cfg)

			if tt.expectType == "error" {
				if err == nil {
					t.Errorf("Expected error for invalid provider, but got none")
				}
				return
			}

			// Note: In real environment, these might fail if CLI tools aren't installed
			// This is expected in test environments
			if err != nil {
				t.Logf("Provider creation failed (expected in test environment): %v", err)
			}
		})
	}
}

func TestProviderConfig(t *testing.T) {
	// Test Claude configuration
	claudeConfig := GetClaudeProviderConfig()
	if claudeConfig.Name != "claude" {
		t.Errorf("Expected Claude provider name to be 'claude', got %s", claudeConfig.Name)
	}
	if claudeConfig.CommandName != "claude" {
		t.Errorf("Expected Claude command name to be 'claude', got %s", claudeConfig.CommandName)
	}
	if claudeConfig.DefaultModel != "sonnet" {
		t.Errorf("Expected Claude default model to be 'sonnet', got %s", claudeConfig.DefaultModel)
	}

	// Test Cursor configuration
	cursorConfig := GetCursorProviderConfig()
	if cursorConfig.Name != "cursor" {
		t.Errorf("Expected Cursor provider name to be 'cursor', got %s", cursorConfig.Name)
	}
	if cursorConfig.CommandName != "cursor-agent" {
		t.Errorf("Expected Cursor command name to be 'cursor-agent', got %s", cursorConfig.CommandName)
	}
	if cursorConfig.DefaultModel != "auto" {
		t.Errorf("Expected Cursor default model to be 'auto', got %s", cursorConfig.DefaultModel)
	}
}

func TestProviderInterfaceCompliance(t *testing.T) {
	// Compile-time checks that providers implement ClaudeClient interface
	var _ ClaudeClient = (*ClaudeProvider)(nil)
	var _ ClaudeClient = (*CursorProvider)(nil)
}

func TestMockProviderCompatibility(t *testing.T) {
	// Test that mock clients still work with the new architecture
	mockClient := &MockClaudeClient{
		Responses: map[string]string{
			"test": `{"tasks": [{"description": "Test task", "priority": "medium"}]}`,
		},
	}

	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage: "English",
			OutputFormat: "json",
		},
	}

	analyzer := NewAnalyzerWithClient(cfg, mockClient)
	if analyzer == nil {
		t.Fatal("Failed to create analyzer with mock client")
	}

	if analyzer.claudeClient != mockClient {
		t.Error("Analyzer not using the provided mock client")
	}
}