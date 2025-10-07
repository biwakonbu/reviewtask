package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestHelpCommand tests the help command functionality
func TestHelpCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		shouldError    bool
	}{
		{
			name: "Help flag on root",
			args: []string{"--help"},
			expectedOutput: []string{
				"reviewtask fetches GitHub Pull Request reviews",
				"Usage:",
				"reviewtask [PR_NUMBER] [flags]",
				"Available Commands:",
				"auth",
				"fetch",
				"init",
				"prompt",
				"show",
				"stats",
				"status",
				"update",
				"version",
			},
			shouldError: false,
		},
		{
			name: "Help command",
			args: []string{"help"},
			expectedOutput: []string{
				"reviewtask fetches GitHub Pull Request reviews",
				"Usage:",
				"Available Commands:",
			},
			shouldError: false,
		},
		{
			name: "Help for specific command",
			args: []string{"help", "status"},
			expectedOutput: []string{
				"Display current tasks",
				"Usage:",
				"reviewtask status",
			},
			shouldError: false,
		},
		{
			name: "Help flag on subcommand",
			args: []string{"status", "--help"},
			expectedOutput: []string{
				"Display current tasks",
				"Usage:",
				"reviewtask status",
			},
			shouldError: false,
		},
		{
			name: "Help for non-existent command",
			args: []string{"help", "nonexistent"},
			expectedOutput: []string{
				"reviewtask fetches GitHub Pull Request reviews",
				"Usage:",
				"Available Commands:",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			buf := new(bytes.Buffer)

			// Create fresh command for each test
			root := NewRootCmd()
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			// Execute the command
			err := root.Execute()

			// Check error expectation
			if tt.shouldError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
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

// TestAllCommandsHaveHelp tests that all commands have help text
func TestAllCommandsHaveHelp(t *testing.T) {
	root := NewRootCmd()

	// Test each command
	for _, cmd := range root.Commands() {
		t.Run(cmd.Name(), func(t *testing.T) {
			// Create a buffer to capture output
			buf := new(bytes.Buffer)

			// Set up the command
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{cmd.Name(), "--help"})

			// Execute
			err := root.Execute()
			if err != nil {
				t.Errorf("Error getting help for command '%s': %v", cmd.Name(), err)
			}

			// Check output contains expected elements
			output := buf.String()

			// Should contain the command name in usage
			if !strings.Contains(output, cmd.Name()) {
				t.Errorf("Help for '%s' doesn't contain command name", cmd.Name())
			}

			// Should contain "Usage:"
			if !strings.Contains(output, "Usage:") {
				t.Errorf("Help for '%s' doesn't contain Usage section", cmd.Name())
			}

			// Should contain either the short or long description
			// (Cobra shows Long description if available, otherwise Short)
			hasDescription := false
			if cmd.Long != "" && strings.Contains(output, cmd.Long) {
				hasDescription = true
			} else if cmd.Short != "" && strings.Contains(output, cmd.Short) {
				hasDescription = true
			}

			if !hasDescription && (cmd.Short != "" || cmd.Long != "") {
				t.Errorf("Help for '%s' doesn't contain expected description", cmd.Name())
			}
		})
	}
}

// TestHelpListsAllCommands tests that help output lists all registered commands
func TestHelpListsAllCommands(t *testing.T) {
	// Create a buffer to capture output
	buf := new(bytes.Buffer)

	// Set up the root command
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})

	// Execute
	err := root.Execute()
	if err != nil {
		t.Fatalf("Error executing help: %v", err)
	}

	// Get the output
	output := buf.String()

	// Check that all commands are listed (except deprecated ones which are hidden)
	for _, cmd := range root.Commands() {
		// Skip deprecated commands as they are hidden from help output
		if cmd.Deprecated != "" {
			continue
		}

		if !strings.Contains(output, cmd.Name()) {
			t.Errorf("Command '%s' not listed in help output", cmd.Name())
		}

		// Also check that the short description is shown
		if cmd.Short != "" && !strings.Contains(output, cmd.Short) {
			t.Errorf("Short description for '%s' not shown in help output", cmd.Name())
		}
	}
}
