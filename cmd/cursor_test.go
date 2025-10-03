package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCursorCommand(t *testing.T) {
	// Save original flags
	originalAll := cursorAllFlag
	originalStdout := cursorStdoutFlag
	defer func() {
		cursorAllFlag = originalAll
		cursorStdoutFlag = originalStdout
	}()

	tests := []struct {
		name           string
		args           []string
		allFlag        bool
		stdoutFlag     bool
		expectError    bool
		expectInOutput []string
		notInOutput    []string
		checkFiles     []string // Files that should be created
	}{
		{
			name:        "Valid pr-review target to files",
			args:        []string{"pr-review"},
			allFlag:     false,
			stdoutFlag:  false,
			expectError: false,
			expectInOutput: []string{
				"Created Cursor IDE command template",
				".cursor/commands/pr-review",
				"review-task-workflow",
			},
			notInOutput: []string{
				"issue-to-pr",
				"label-issues",
			},
			checkFiles: []string{
				".cursor/commands/pr-review/review-task-workflow.md",
			},
		},
		{
			name:        "Valid issue-to-pr target to files",
			args:        []string{"issue-to-pr"},
			allFlag:     false,
			stdoutFlag:  false,
			expectError: false,
			expectInOutput: []string{
				"Created Cursor IDE command template",
				".cursor/commands/issue-to-pr",
			},
			notInOutput: []string{
				"pr-review",
				"label-issues",
			},
			checkFiles: []string{
				".cursor/commands/issue-to-pr/issue-to-pr.md",
			},
		},
		{
			name:        "Valid label-issues target to files",
			args:        []string{"label-issues"},
			allFlag:     false,
			stdoutFlag:  false,
			expectError: false,
			expectInOutput: []string{
				"Created Cursor IDE command template",
				".cursor/commands/label-issues",
			},
			notInOutput: []string{
				"pr-review",
				"issue-to-pr",
			},
			checkFiles: []string{
				".cursor/commands/label-issues/label-issues.md",
			},
		},
		{
			name:        "Generate all templates",
			args:        []string{},
			allFlag:     true,
			stdoutFlag:  false,
			expectError: false,
			expectInOutput: []string{
				"Created Cursor IDE command template",
				".cursor/commands/pr-review",
				".cursor/commands/issue-to-pr",
				".cursor/commands/label-issues",
				"All Cursor IDE command templates have been generated successfully",
			},
			notInOutput: []string{},
			checkFiles: []string{
				".cursor/commands/pr-review/review-task-workflow.md",
				".cursor/commands/issue-to-pr/issue-to-pr.md",
				".cursor/commands/label-issues/label-issues.md",
			},
		},
		{
			name:        "Output pr-review to stdout",
			args:        []string{"pr-review"},
			allFlag:     false,
			stdoutFlag:  true,
			expectError: false,
			expectInOutput: []string{
				"review-task-workflow",
				"Execute PR review tasks systematically",
				"reviewtask show",
			},
			notInOutput: []string{
				"Created Cursor IDE command template",
				"Issue to PR Workflow",
				"Label Issues",
			},
			checkFiles: []string{}, // No files should be created with stdout
		},
		{
			name:        "Output all to stdout",
			args:        []string{},
			allFlag:     true,
			stdoutFlag:  true,
			expectError: false,
			expectInOutput: []string{
				"=== PR REVIEW WORKFLOW ===",
				"review-task-workflow",
				"=== ISSUE TO PR WORKFLOW ===",
				"Issue to PR Workflow",
				"=== LABEL ISSUES WORKFLOW ===",
				"Label Issues",
			},
			notInOutput: []string{
				"Created Cursor IDE command template",
			},
			checkFiles: []string{}, // No files should be created with stdout
		},
		{
			name:           "Invalid target",
			args:           []string{"invalid-target"},
			allFlag:        false,
			stdoutFlag:     false,
			expectError:    true,
			expectInOutput: []string{"unknown target: invalid-target"},
			notInOutput:    []string{},
			checkFiles:     []string{},
		},
		{
			name:           "No arguments without --all flag",
			args:           []string{},
			allFlag:        false,
			stdoutFlag:     false,
			expectError:    true,
			expectInOutput: []string{"target argument required when not using --all flag"},
			notInOutput:    []string{},
			checkFiles:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing .cursor directory
			os.RemoveAll(".cursor")
			defer os.RemoveAll(".cursor")

			// Set flags
			cursorAllFlag = tt.allFlag
			cursorStdoutFlag = tt.stdoutFlag

			// Create a fresh command instance for each test
			cmd := &cobra.Command{
				Use:   "cursor [TARGET]",
				Short: "Output command templates for Cursor IDE to .cursor/commands directory",
				Long: `Output command templates for Cursor IDE to .cursor/commands directory for better organization and discoverability.

Available targets:
  pr-review      Output PR review workflow command template
  issue-to-pr    Output Issue-to-PR workflow command template
  label-issues   Output Label Issues workflow command template

Examples:
  reviewtask cursor pr-review          # Output review-task-workflow command template for Cursor IDE
  reviewtask cursor issue-to-pr        # Output issue-to-pr workflow command template
  reviewtask cursor label-issues       # Output label-issues workflow command template
  reviewtask cursor --all              # Output all available command templates
  reviewtask cursor pr-review --stdout # Output to standard output instead of files
  reviewtask cursor --all --stdout     # Output all templates to standard output`,
				Args: cobra.MaximumNArgs(1),
				RunE: runCursor,
			}

			// Add flags
			cmd.Flags().BoolVar(&cursorAllFlag, "all", tt.allFlag, "Generate all available command templates")
			cmd.Flags().BoolVar(&cursorStdoutFlag, "stdout", tt.stdoutFlag, "Output to standard output instead of files")

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
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			// Normalize path separators for cross-platform compatibility
			output := strings.ReplaceAll(buf.String(), "\\", "/")
			for _, expected := range tt.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
				}
			}

			for _, notExpected := range tt.notInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output NOT to contain %q, but it did.\nOutput: %s", notExpected, output)
				}
			}

			// Check if files were created (only when not using stdout)
			if !tt.stdoutFlag {
				for _, file := range tt.checkFiles {
					if _, err := os.Stat(file); os.IsNotExist(err) {
						t.Errorf("Expected file %s to be created, but it wasn't", file)
					}
				}
			}
		})
	}
}

func TestCursorCommandHelp(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "cursor [TARGET]",
		Short: "Output command templates for Cursor IDE to .cursor/commands directory",
		Long: `Output command templates for Cursor IDE to .cursor/commands directory for better organization and discoverability.

Available targets:
  pr-review      Output PR review workflow command template
  issue-to-pr    Output Issue-to-PR workflow command template
  label-issues   Output Label Issues workflow command template

Examples:
  reviewtask cursor pr-review          # Output review-task-workflow command template for Cursor IDE
  reviewtask cursor issue-to-pr        # Output issue-to-pr workflow command template
  reviewtask cursor label-issues       # Output label-issues workflow command template
  reviewtask cursor --all              # Output all available command templates
  reviewtask cursor pr-review --stdout # Output to standard output instead of files
  reviewtask cursor --all --stdout     # Output all templates to standard output`,
		Args: cobra.MaximumNArgs(1),
		RunE: runCursor,
	}

	// Add flags
	cmd.Flags().BoolVar(&cursorAllFlag, "all", false, "Generate all available command templates")
	cmd.Flags().BoolVar(&cursorStdoutFlag, "stdout", false, "Output to standard output instead of files")

	// Test help output
	cmd.SetArgs([]string{"--help"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"Output command templates for Cursor IDE",
		"Available targets:",
		"pr-review",
		"issue-to-pr",
		"label-issues",
		"--all",
		"--stdout",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help output to contain %q, but it didn't.\nOutput: %s", expected, output)
		}
	}
}

func TestCursorTemplateFileContent(t *testing.T) {
	// Clean up any existing .cursor directory
	os.RemoveAll(".cursor")
	defer os.RemoveAll(".cursor")

	// Test that pr-review template is created with correct content
	cursorAllFlag = false
	cursorStdoutFlag = false

	cmd := &cobra.Command{
		Use:   "cursor [TARGET]",
		Short: "Output command templates for Cursor IDE to .cursor/commands directory",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCursor,
	}

	cmd.SetArgs([]string{"pr-review"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that the file was created and contains expected content
	filePath := filepath.Join(".cursor", "commands", "pr-review", "review-task-workflow.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read created file: %v", err)
	}

	expectedContent := []string{
		"review-task-workflow",
		"Execute PR review tasks systematically",
		"reviewtask show",
		"reviewtask update",
		"Workflow Steps",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Expected file content to contain %q, but it didn't", expected)
		}
	}
}
