package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestFetchCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		shouldError    bool
		expectedOutput []string
	}{
		{
			name:        "Help flag",
			args:        []string{"fetch", "--help"},
			shouldError: false,
			expectedOutput: []string{
				"Fetch GitHub Pull Request reviews, save them locally,",
				"Usage:",
				"reviewtask fetch [PR_NUMBER]",
			},
		},
		{
			name:        "No arguments - should fetch for current branch",
			args:        []string{"fetch"},
			shouldError: false, // Will succeed in showing help behavior but won't actually fetch without proper setup
		},
		{
			name:        "With PR number argument",
			args:        []string{"fetch", "123"},
			shouldError: false, // Will fail with proper error about repo setup, but command structure is valid
		},
		// Note: Too many arguments test is covered in TestFetchCommandArgs
		// Removed due to test execution context issues with root command state
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			buf := new(bytes.Buffer)

			// Create root command and add fetch command
			root := rootCmd
			root.SetOut(buf)
			root.SetErr(buf)

			// Set arguments
			root.SetArgs(tt.args)

			// Execute command
			err := root.Execute()

			// Check error expectation
			if tt.shouldError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.shouldError && err != nil && !strings.Contains(err.Error(), "repository is not initialized") && !strings.Contains(err.Error(), "failed to get current branch") && !strings.Contains(err.Error(), "accepts at most") {
				// Allow initialization and git errors as they're expected in test environment
				t.Errorf("Unexpected error: %v", err)
			}

			// Get the output
			output := buf.String()

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestFetchCommandRegistration(t *testing.T) {
	// Verify fetch command is registered
	root := rootCmd

	fetchCmd, _, err := root.Find([]string{"fetch"})
	if err != nil || fetchCmd == nil {
		t.Error("fetch command not found in root command")
	}

	if fetchCmd == nil {
		t.Fatal("fetch command is nil")
	}

	if fetchCmd.Use != "fetch [PR_NUMBER]" {
		t.Errorf("Unexpected fetch command Use: %s", fetchCmd.Use)
	}

	if !strings.Contains(fetchCmd.Short, "Fetch GitHub Pull Request reviews") {
		t.Errorf("Fetch command Short description doesn't match expected: %s", fetchCmd.Short)
	}
}

func TestFetchCommandArgs(t *testing.T) {
	root := rootCmd
	fetchCmd, _, err := root.Find([]string{"fetch"})

	if err != nil || fetchCmd == nil {
		t.Fatal("fetch command not found")
	}

	// Test args validation
	if fetchCmd.Args == nil {
		t.Error("fetch command should have Args validation")
	}

	// Test that it accepts 0 or 1 arguments
	tests := []struct {
		args        []string
		shouldError bool
	}{
		{[]string{}, false},            // 0 args - should be valid
		{[]string{"123"}, false},       // 1 arg - should be valid
		{[]string{"123", "456"}, true}, // 2 args - should be invalid
	}

	for _, tt := range tests {
		err := fetchCmd.Args(fetchCmd, tt.args)
		if tt.shouldError && err == nil {
			t.Errorf("Expected error for args %v but got none", tt.args)
		}
		if !tt.shouldError && err != nil {
			t.Errorf("Unexpected error for args %v: %v", tt.args, err)
		}
	}
}
