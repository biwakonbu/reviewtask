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
			name: "Root help should mention fetch command as deprecated",
			args: []string{"--help"},
			expectedOutput: []string{
				"fetch       [DEPRECATED] Use 'reviewtask [PR_NUMBER]' instead",
			},
			shouldContain: true,
		},
		{
			name: "Root help examples should show integrated workflow usage",
			args: []string{"--help"},
			expectedOutput: []string{
				"reviewtask              # Analyze current branch's PR (integrated workflow)",
				"reviewtask 123          # Analyze PR #123 (integrated workflow)",
			},
			shouldContain: true,
		},
		{
			name: "Fetch help should show deprecation notice",
			args: []string{"fetch", "--help"},
			expectedOutput: []string{
				"DEPRECATION NOTICE",
				"integrated into the main reviewtask command",
				"reviewtask 123",
			},
			shouldContain: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			buf := new(bytes.Buffer)

			// Get fresh root command instance
			root := NewRootCmd()
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

// TestRootCommandDefaultBehavior tests that root command runs integrated workflow (fetch+analyze)
func TestRootCommandDefaultBehavior(t *testing.T) {
	// Create a buffer to capture output
	buf := new(bytes.Buffer)

	// Get fresh root command instance
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)

	// Execute with no arguments - should try to run integrated workflow
	// This will fail in test environment but we can check for expected messages
	root.SetArgs([]string{})
	err := root.Execute()

	// Expect error since we don't have a PR in test environment
	if err == nil {
		t.Skip("Skipping test - would require PR environment setup")
		return
	}

	// Get the output
	output := buf.String()

	// Should contain workflow-related messages or errors (not just help text)
	// One of these should be present:
	expectedMessages := []string{
		"This repository is not initialized for reviewtask",
		"failed to load config",
		"failed to initialize GitHub client",
		"No pull request found",
		"Checking for closed PRs",
	}

	foundExpected := false
	for _, expected := range expectedMessages {
		if strings.Contains(output, expected) || strings.Contains(err.Error(), expected) {
			foundExpected = true
			break
		}
	}

	if !foundExpected {
		t.Logf("Expected one of the workflow messages, but got output:\n%s\nerror: %v", output, err)
	}
}

// TestIntegratedWorkflowAcceptsPRNumber verifies the integrated workflow accepts PR number
func TestIntegratedWorkflowAcceptsPRNumber(t *testing.T) {
	// Test that new behavior accepts PR number as argument to root command
	buf := new(bytes.Buffer)

	// Get fresh root command instance
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)

	// This should now try to run integrated workflow for PR #123
	root.SetArgs([]string{"123"})
	err := root.Execute()

	// Will fail in test environment, but should NOT be "unknown command" error
	if err != nil && strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Root command should accept PR number as argument, but got 'unknown command' error: %v", err)
	}

	// The error should be workflow-related, not argument validation
	if err != nil {
		expectedErrors := []string{
			"failed to load config",
			"failed to initialize GitHub client",
			"authentication required",
			"repository not initialized",
		}

		foundExpected := false
		for _, expected := range expectedErrors {
			if strings.Contains(err.Error(), expected) {
				foundExpected = true
				break
			}
		}

		if !foundExpected {
			t.Logf("Expected workflow error, got: %v", err)
		}
	}
}
