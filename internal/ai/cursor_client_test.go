package ai

import (
	"context"
	"reviewtask/internal/config"
	"testing"
)

// MockCursorClient implements ClaudeClient for testing
type MockCursorClient struct {
	ExecuteFunc func(ctx context.Context, input string, outputFormat string) (string, error)
}

func (m *MockCursorClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, input, outputFormat)
	}
	return "", nil
}

func (m *MockCursorClient) CheckAuthentication() error {
	return nil
}

func TestMockCursorClient_Execute(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		outputFormat   string
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "Basic execution",
			input:          "test input",
			outputFormat:   "json",
			expectedOutput: `{"tasks": []}`,
			expectError:    false,
		},
		{
			name:           "Text format",
			input:          "test input",
			outputFormat:   "text",
			expectedOutput: "test output",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockCursorClient{
				ExecuteFunc: func(ctx context.Context, input string, outputFormat string) (string, error) {
					if input == tt.input && outputFormat == tt.outputFormat {
						return tt.expectedOutput, nil
					}
					return "", nil
				},
			}

			output, err := client.Execute(context.Background(), tt.input, tt.outputFormat)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if output != tt.expectedOutput {
				t.Errorf("Expected output %s, got %s", tt.expectedOutput, output)
			}
		})
	}
}

func TestAnalyzerWithCursorClient(t *testing.T) {
	// Create a mock cursor client
	mockClient := &MockCursorClient{
		ExecuteFunc: func(ctx context.Context, input string, outputFormat string) (string, error) {
			// Return a simple task response
			return `{
				"tasks": [
					{
						"description": "Test task from Cursor",
						"priority": "medium"
					}
				]
			}`, nil
		},
	}

	// Create test config
	validationTrue := true
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage:      "English",
			OutputFormat:      "json",
			MaxRetries:        5,
			AIProvider:        "cursor",
			Model:             "auto",
			ValidationEnabled: &validationTrue,
			QualityThreshold:  0.8,
		},
	}
	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	if analyzer == nil {
		t.Fatal("Failed to create analyzer with cursor client")
	}

	if analyzer.claudeClient != mockClient {
		t.Error("Analyzer not using the provided cursor client")
	}
}

func TestCursorClientInterfaceCompliance(t *testing.T) {
	// This test ensures MockCursorClient implements ClaudeClient interface
	var _ ClaudeClient = (*MockCursorClient)(nil)

	// Also test that RealCursorClient would implement ClaudeClient
	// (This is a compile-time check)
	var _ ClaudeClient = (*RealCursorClient)(nil)
}
