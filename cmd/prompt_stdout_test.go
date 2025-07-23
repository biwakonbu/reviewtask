package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPromptStdoutCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectInOutput []string
		notInOutput    []string
	}{
		{
			name:        "Valid pr-review target",
			args:        []string{"pr-review"},
			expectError: false,
			expectInOutput: []string{
				"review-task-workflow",
				"Execute PR review tasks systematically",
				"reviewtask show",
				"reviewtask update",
				"doing",
				"todo",
				"pending",
				"Workflow Steps",
			},
			notInOutput: []string{},
		},
		{
			name:           "Invalid target",
			args:           []string{"invalid-target"},
			expectError:    true,
			expectInOutput: []string{"unknown target: invalid-target"},
			notInOutput:    []string{},
		},
		{
			name:           "No arguments",
			args:           []string{},
			expectError:    true,
			expectInOutput: []string{},
			notInOutput:    []string{},
		},
		{
			name:           "Too many arguments",
			args:           []string{"pr-review", "extra"},
			expectError:    true,
			expectInOutput: []string{},
			notInOutput:    []string{},
		},
		{
			name:        "Help flag",
			args:        []string{"--help"},
			expectError: false,
			expectInOutput: []string{
				"Output AI provider prompts to standard output",
				"redirect the output to any file or pipe",
				"Examples:",
				"reviewtask prompt stdout pr-review",
				"> my-workflow.md",
				"pbcopy",
			},
			notInOutput: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command instance for each test
			cmd := &cobra.Command{
				Use:   "stdout <target>",
				Short: "Output AI provider prompts to stdout for redirection or piping",
				Long: `Output AI provider prompts to standard output instead of writing to files.
This allows you to redirect the output to any file or pipe it to other tools.

Available targets:
  pr-review    Output PR review workflow prompt

Examples:
  reviewtask prompt stdout pr-review                    # Display prompt on stdout
  reviewtask prompt stdout pr-review > my-workflow.md   # Save to custom file
  reviewtask prompt stdout pr-review | pbcopy           # Copy to clipboard (macOS)
  reviewtask prompt stdout pr-review | xclip            # Copy to clipboard (Linux)`,
				Args: cobra.ExactArgs(1),
				RunE: runPromptStdout,
			}

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check output content
			output := buf.String()

			for _, expected := range tt.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
				}
			}
			for _, notExpected := range tt.notInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output NOT to contain '%s', but it did. Output:\n%s", notExpected, output)
				}
			}
		})
	}
}

func TestPromptStdoutIntegration(t *testing.T) {
	// Test the actual prompt stdout functionality
	t.Run("PR review prompt output", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{
			Use:  "stdout",
			Args: cobra.ExactArgs(1),
			RunE: runPromptStdout,
		}
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{"pr-review"})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		output := buf.String()

		// Verify the output is complete and well-formed
		requiredSections := []string{
			"---\nname: review-task-workflow",
			"## Initial Setup",
			"## Workflow Steps",
			"## Important Notes",
			"Execute this workflow now",
		}

		for _, section := range requiredSections {
			if !strings.Contains(output, section) {
				t.Errorf("Missing required section: %s", section)
			}
		}

		// Ensure output doesn't contain file creation messages
		if strings.Contains(output, "Created file") || strings.Contains(output, ".claude/commands") {
			t.Error("Output should not contain file creation messages")
		}
	})
}

func TestPromptStdoutTargets(t *testing.T) {
	// Test all available targets
	validTargets := []string{"pr-review"}

	for _, target := range validTargets {
		t.Run("Target: "+target, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cobra.Command{
				Use:  "stdout",
				Args: cobra.ExactArgs(1),
				RunE: runPromptStdout,
			}
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{target})

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Failed to execute with target %s: %v", target, err)
			}

			output := buf.String()
			if len(output) == 0 {
				t.Errorf("Expected output for target %s, but got empty output", target)
			}
		})
	}
}
