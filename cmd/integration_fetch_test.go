package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestFetchCommandIntegration tests the fetch command integration with the root command
func TestFetchCommandIntegration(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		shouldContain  bool
	}{
		{
			name: "Root help should mention fetch command",
			args: []string{"--help"},
			expectedOutput: []string{
				"fetch       Fetch GitHub Pull Request reviews and generate tasks",
			},
			shouldContain: true,
		},
		{
			name: "Root help examples should show fetch usage",
			args: []string{"--help"},
			expectedOutput: []string{
				"reviewtask fetch        # Check reviews for current branch's PR",
				"reviewtask fetch 123    # Check reviews for PR #123",
			},
			shouldContain: true,
		},
		{
			name: "Fetch help should be accessible via root help",
			args: []string{"help", "fetch"},
			expectedOutput: []string{
				"Fetch GitHub Pull Request reviews, save them locally,",
				"Usage:",
				"reviewtask fetch [PR_NUMBER]",
			},
			shouldContain: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			buf := new(bytes.Buffer)

			// Get root command
			root := rootCmd
			root.SetOut(buf)
			root.SetErr(buf)

			// Set arguments
			root.SetArgs(tt.args)

			// Execute command
			err := root.Execute()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Get the output
			output := buf.String()

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if tt.shouldContain && !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expected, output)
				}
			}
		})
	}
}

// TestRootCommandDefaultBehavior tests that root command now shows help instead of fetching
func TestRootCommandDefaultBehavior(t *testing.T) {
	// Create a buffer to capture output
	buf := new(bytes.Buffer)

	// Get root command
	root := rootCmd
	root.SetOut(buf)
	root.SetErr(buf)

	// Execute with no arguments - should show help
	root.SetArgs([]string{})
	err := root.Execute()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Get the output
	output := buf.String()

	// Should contain help information
	expectedHelp := []string{
		"reviewtask fetches GitHub Pull Request reviews",
		"Usage:",
		"Available Commands:",
		"fetch       Fetch GitHub Pull Request reviews and generate tasks",
	}

	for _, expected := range expectedHelp {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help output to contain '%s', but got:\n%s", expected, output)
		}
	}

	// Should NOT contain initialization prompts or review fetching behavior
	notExpected := []string{
		"This repository is not initialized for reviewtask",
		"Fetching reviews for PR",
		"Processing comments",
	}

	for _, notExp := range notExpected {
		if strings.Contains(output, notExp) {
			t.Errorf("Help output should not contain '%s', but got:\n%s", notExp, output)
		}
	}
}

// TestBackwardCompatibilityBreaking verifies the breaking change behavior
func TestBackwardCompatibilityBreaking(t *testing.T) {
	// Test that old behavior (reviewtask without subcommand doing PR number) no longer works
	buf := new(bytes.Buffer)

	root := rootCmd
	root.SetOut(buf)
	root.SetErr(buf)

	// This should now show help instead of trying to fetch PR #123
	root.SetArgs([]string{"123"})
	err := root.Execute()

	// Should get an error because root command doesn't accept arguments
	if err == nil {
		t.Error("Expected error when providing PR number to root command, but got none")
	}

	// Should contain error about unknown command
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, but got: %v", err)
	}
}
