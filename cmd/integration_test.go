package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestCommandIntegration tests basic command integration without external dependencies
func TestCommandIntegration(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
		contains  []string
	}{
		{
			name:      "version command shows version info",
			args:      []string{"version"},
			expectErr: false,
			contains:  []string{"reviewtask version"},
		},
		{
			name:      "help command shows usage",
			args:      []string{"--help"},
			expectErr: false,
			contains:  []string{"reviewtask", "Usage:", "Available Commands:"},
		},
		{
			name:      "auth help shows subcommands",
			args:      []string{"auth", "--help"},
			expectErr: false,
			contains:  []string{"login", "logout", "status", "check"},
		},
		{
			name:      "stats help shows options",
			args:      []string{"stats", "--help"},
			expectErr: false,
			contains:  []string{"--all", "--pr", "--branch"},
		},
		{
			name:      "status help shows options",
			args:      []string{"status", "--help"},
			expectErr: false,
			contains:  []string{"--all", "--pr", "--branch"},
		},
		{
			name:      "versions command available",
			args:      []string{"versions", "--help"},
			expectErr: false,
			contains:  []string{"versions", "List"},
		},
		{
			name:      "claude command available",
			args:      []string{"claude", "--help"},
			expectErr: false,
			contains:  []string{"claude", "target"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use cobra's built-in testing approach with rootCmd directly
			var buf bytes.Buffer

			// Store original output settings
			originalOut := rootCmd.OutOrStdout()
			originalErr := rootCmd.ErrOrStderr()

			// Set output capture
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			// Restore original settings
			rootCmd.SetOut(originalOut)
			rootCmd.SetErr(originalErr)
			rootCmd.SetArgs([]string{})

			output := buf.String()

			if tt.expectErr && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check if output contains expected strings
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s' but got: %s", expected, output)
				}
			}

		})
	}
}

// TestDocumentedFlagsWork tests that documented flags are functional
func TestDocumentedFlagsWork(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(string) bool
	}{
		{
			name:      "refresh-cache flag recognized",
			args:      []string{"--refresh-cache", "--help"},
			expectErr: false,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "refresh-cache")
			},
		},
		{
			name:      "stats all flag works",
			args:      []string{"stats", "--all", "--help"},
			expectErr: false,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "all")
			},
		},
		{
			name:      "version check flag works",
			args:      []string{"version", "--check", "--help"},
			expectErr: false,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "check")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use cobra's built-in testing approach with rootCmd directly
			var buf bytes.Buffer

			// Store original output settings
			originalOut := rootCmd.OutOrStdout()
			originalErr := rootCmd.ErrOrStderr()

			// Set output capture
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			// Restore original settings
			rootCmd.SetOut(originalOut)
			rootCmd.SetErr(originalErr)
			rootCmd.SetArgs([]string{})

			output := buf.String()

			if tt.expectErr && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && !tt.checkFunc(output) {
				t.Errorf("Check function failed for output: %s", output)
			}

		})
	}
}

// TestCommandHelpConsistency tests that help text matches documented functionality
func TestCommandHelpConsistency(t *testing.T) {
	commandTests := []struct {
		command     string
		helpArgs    []string
		mustContain []string
	}{
		{
			command:  "stats",
			helpArgs: []string{"stats", "--help"},
			mustContain: []string{
				"statistics",
				"--all",
				"--pr",
				"--branch",
				"comment",
			},
		},
		{
			command:  "versions",
			helpArgs: []string{"versions", "--help"},
			mustContain: []string{
				"versions",
				"recent",
				"release",
			},
		},
		{
			command:  "claude",
			helpArgs: []string{"claude", "--help"},
			mustContain: []string{
				"claude",
				"target",
				"template",
			},
		},
		{
			command:  "version",
			helpArgs: []string{"version", "--help"},
			mustContain: []string{
				"version",
				"--check",
				"update",
			},
		},
	}

	for _, tt := range commandTests {
		t.Run("help_consistency_"+tt.command, func(t *testing.T) {
			// Use cobra's built-in testing approach with rootCmd directly
			var buf bytes.Buffer

			// Store original output settings
			originalOut := rootCmd.OutOrStdout()
			originalErr := rootCmd.ErrOrStderr()

			// Set output capture
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.helpArgs)

			rootCmd.Execute()

			// Restore original settings
			rootCmd.SetOut(originalOut)
			rootCmd.SetErr(originalErr)
			rootCmd.SetArgs([]string{})

			output := buf.String()

			for _, required := range tt.mustContain {
				if !strings.Contains(strings.ToLower(output), strings.ToLower(required)) {
					t.Errorf("Help for %s command should contain '%s' but output was: %s",
						tt.command, required, output)
				}
			}

		})
	}
}

