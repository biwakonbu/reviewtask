package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPromptCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectedError  bool
	}{
		{
			name: "Help flag shows available subcommands",
			args: []string{"prompt", "--help"},
			expectedOutput: []string{
				"Output command templates for various AI providers",
				"claude",
				"Examples:",
				"reviewtask prompt claude pr-review",
			},
			expectedError: false,
		},
		{
			name: "No arguments shows help",
			args: []string{"prompt"},
			expectedOutput: []string{
				"Available Commands:",
				"claude",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the command with a buffer to capture output
			buf := &bytes.Buffer{}
			root := NewRootCmd()
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			// Execute
			err := root.Execute()

			// Check error expectation
			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check output
			output := buf.String()
			for _, expectedStr := range tt.expectedOutput {
				if !strings.Contains(output, expectedStr) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expectedStr, output)
				}
			}
		})
	}
}

func TestPromptClaudeCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectedError  bool
	}{
		{
			name: "Help flag shows available targets",
			args: []string{"prompt", "claude", "--help"},
			expectedOutput: []string{
				"Output command templates for Claude Code",
				"pr-review",
				"Examples:",
				"reviewtask prompt claude pr-review",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the command with a buffer to capture output
			buf := &bytes.Buffer{}
			root := NewRootCmd()
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			// Execute
			err := root.Execute()

			// Check error expectation
			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check output
			output := buf.String()
			for _, expectedStr := range tt.expectedOutput {
				if !strings.Contains(output, expectedStr) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expectedStr, output)
				}
			}
		})
	}
}

func TestClaudeCommandDeprecationWarning(t *testing.T) {
	// Test that deprecated claude command shows deprecation warning
	buf := &bytes.Buffer{}
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"claude", "--help"})

	err := root.Execute()
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	output := buf.String()
	expectedWarnings := []string{
		"DEPRECATION WARNING",
		"deprecated",
		"reviewtask prompt claude",
		"future major version",
	}

	for _, warning := range expectedWarnings {
		if !strings.Contains(output, warning) {
			t.Errorf("Expected deprecation warning to contain '%s', but got:\n%s", warning, output)
		}
	}
}

func TestPromptCommandStructure(t *testing.T) {
	root := NewRootCmd()

	// Find the prompt command
	promptCmd, _, err := root.Find([]string{"prompt"})
	if err != nil {
		t.Fatalf("prompt command not found: %v", err)
	}

	// Check that prompt command has claude subcommand
	claudeCmd, _, err := promptCmd.Find([]string{"claude"})
	if err != nil {
		t.Fatalf("claude subcommand not found under prompt: %v", err)
	}

	// Verify claude subcommand properties
	if claudeCmd.Use != "claude [TARGET]" {
		t.Errorf("Expected claude subcommand Use to be 'claude [TARGET]', got '%s'", claudeCmd.Use)
	}

	if !strings.Contains(claudeCmd.Short, "Claude Code") {
		t.Errorf("Expected claude subcommand Short description to mention 'Claude Code', got '%s'", claudeCmd.Short)
	}
}

func TestPromptCommandFutureExtensibility(t *testing.T) {
	root := NewRootCmd()

	// Find the prompt command
	promptCmd, _, err := root.Find([]string{"prompt"})
	if err != nil {
		t.Fatalf("prompt command not found: %v", err)
	}

	// Check that the help mentions future AI providers
	expectedFutureProviders := []string{
		"openai",
		"gemini",
		"future",
	}

	helpText := promptCmd.Long
	for _, provider := range expectedFutureProviders {
		if !strings.Contains(strings.ToLower(helpText), provider) {
			t.Errorf("Expected prompt command help to mention '%s' for future extensibility", provider)
		}
	}
}
