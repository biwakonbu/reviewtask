//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCursorCommandIntegration(t *testing.T) {
	// Clean up any existing .cursor directory
	os.RemoveAll(".cursor")
	defer os.RemoveAll(".cursor")

	// Reset flags
	cursorAllFlag = false
	cursorStdoutFlag = false

	tests := []struct {
		name               string
		args               string
		expectInOutput     []string
		expectNotInOutput  []string
		expectFiles        []string
		expectFileContents map[string][]string // file -> expected content snippets
	}{
		{
			name: "Generate all templates and verify content",
			args: "--all",
			expectInOutput: []string{
				"Created Cursor IDE command template",
				"All Cursor IDE command templates have been generated successfully",
			},
			expectFiles: []string{
				".cursor/commands/pr-review/review-task-workflow.md",
				".cursor/commands/issue-to-pr/issue-to-pr.md",
				".cursor/commands/label-issues/label-issues.md",
			},
			expectFileContents: map[string][]string{
				".cursor/commands/pr-review/review-task-workflow.md": {
					"review-task-workflow",
					"Execute PR review tasks systematically",
					"reviewtask status",
					"reviewtask show",
				},
				".cursor/commands/issue-to-pr/issue-to-pr.md": {
					"Issue to PR Workflow",
					"GitHub Issues",
					"Draft PR",
				},
				".cursor/commands/label-issues/label-issues.md": {
					"Label Issues",
					"GitHub",
					"issue",
				},
			},
		},
		{
			name: "Output all to stdout and verify content",
			args: "--all --stdout",
			expectInOutput: []string{
				"=== PR REVIEW WORKFLOW ===",
				"review-task-workflow",
				"=== ISSUE TO PR WORKFLOW ===",
				"Issue to PR Workflow",
				"=== LABEL ISSUES WORKFLOW ===",
				"Label Issues",
			},
			expectNotInOutput: []string{
				"Created Cursor IDE command template",
			},
			expectFiles: []string{}, // No files should be created
		},
		{
			name: "Generate specific template with stdout",
			args: "pr-review --stdout",
			expectInOutput: []string{
				"review-task-workflow",
				"Execute PR review tasks systematically",
				"reviewtask verify",
				"reviewtask complete",
			},
			expectNotInOutput: []string{
				"Issue to PR Workflow",
				"Label Issues",
			},
			expectFiles: []string{}, // No files should be created
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(".cursor")

			// Create root command with cursor subcommand
			rootCmd := &cobra.Command{Use: "reviewtask"}

			// Create cursor command
			cursorCmd := &cobra.Command{
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
			cursorCmd.Flags().BoolVar(&cursorAllFlag, "all", false, "Generate all available command templates")
			cursorCmd.Flags().BoolVar(&cursorStdoutFlag, "stdout", false, "Output to standard output instead of files")

			rootCmd.AddCommand(cursorCmd)

			// Prepare command arguments
			args := strings.Fields("cursor " + tt.args)
			rootCmd.SetArgs(args)

			// Capture output
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)

			// Execute command
			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check output
			// Normalize path separators for cross-platform compatibility
			output := strings.ReplaceAll(buf.String(), "\\", "/")
			for _, expected := range tt.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
				}
			}

			for _, notExpected := range tt.expectNotInOutput {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output NOT to contain %q, but it did.\nOutput: %s", notExpected, output)
				}
			}

			// Check files
			for _, file := range tt.expectFiles {
				p := filepath.FromSlash(file)
				if _, err := os.Stat(p); os.IsNotExist(err) {
					t.Errorf("Expected file %s to be created, but it wasn't", file)
				}
			}

			// Check file contents
			for file, expectedContents := range tt.expectFileContents {
				p := filepath.FromSlash(file)
				content, err := os.ReadFile(p)
				if err != nil {
					t.Errorf("Failed to read file %s: %v", file, err)
					continue
				}

				// Debug: Show first 200 chars of content if test fails
				contentStr := string(content)
				for _, expected := range expectedContents {
					if !strings.Contains(contentStr, expected) {
						preview := contentStr
						if len(preview) > 200 {
							preview = preview[:200] + "..."
						}
						t.Errorf("Expected file %s to contain %q, but it didn't. File starts with: %s", file, expected, preview)
					}
				}
			}
		})
	}
}

func TestCursorCommandEndToEnd(t *testing.T) {
	// This test simulates a complete user workflow
	// Clean up any existing .cursor directory
	os.RemoveAll(".cursor")
	defer os.RemoveAll(".cursor")

	// Step 1: Generate all templates
	rootCmd := &cobra.Command{Use: "reviewtask"}
	cursorCmd := &cobra.Command{
		Use:   "cursor [TARGET]",
		Short: "Output command templates for Cursor IDE to .cursor/commands directory",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCursor,
	}
	cursorCmd.Flags().BoolVar(&cursorAllFlag, "all", false, "Generate all available command templates")
	cursorCmd.Flags().BoolVar(&cursorStdoutFlag, "stdout", false, "Output to standard output instead of files")
	rootCmd.AddCommand(cursorCmd)

	// Execute with --all flag
	cursorAllFlag = true
	cursorStdoutFlag = false

	rootCmd.SetArgs([]string{"cursor", "--all"})
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to generate all templates: %v", err)
	}

	// Verify all template directories exist
	expectedDirs := []string{
		".cursor/commands/pr-review",
		".cursor/commands/issue-to-pr",
		".cursor/commands/label-issues",
	}

	for _, dir := range expectedDirs {
		if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
			t.Errorf("Expected directory %s to exist, but it doesn't", dir)
		}
	}

	// Step 2: Verify templates can be read and contain valid content
	templateFiles := map[string]string{
		".cursor/commands/pr-review/review-task-workflow.md": "review-task-workflow",
		".cursor/commands/issue-to-pr/issue-to-pr.md":        "Issue to PR Workflow",
		".cursor/commands/label-issues/label-issues.md":      "Label Issues",
	}

	for file, expectedContent := range templateFiles {
		p := filepath.FromSlash(file)
		content, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("Failed to read template file %s: %v", file, err)
			continue
		}

		if !strings.Contains(string(content), expectedContent) {
			t.Errorf("Template file %s doesn't contain expected content %q", file, expectedContent)
		}

		// Verify the file is valid markdown
		if !strings.HasSuffix(p, ".md") {
			t.Errorf("Template file %s should have .md extension", file)
		}

		// Verify file has reasonable size (not empty, not too large)
		fileInfo, err := os.Stat(p)
		if err != nil {
			t.Errorf("Failed to stat file %s: %v", file, err)
			continue
		}
		size := fileInfo.Size()
		if size < 100 {
			t.Errorf("Template file %s seems too small (%d bytes)", file, size)
		}
		if size > 1024*1024 { // 1MB
			t.Errorf("Template file %s seems too large (%d bytes)", file, size)
		}
	}

	// Step 3: Test regeneration (should overwrite existing files)
	// Modify a file and regenerate to ensure it gets overwritten
	testFile := filepath.FromSlash(".cursor/commands/pr-review/review-task-workflow.md")
	if err := os.WriteFile(testFile, []byte("MODIFIED CONTENT"), 0644); err != nil {
		t.Fatalf("Failed to modify %s: %v", testFile, err)
	}

	// Regenerate
	rootCmd.SetArgs([]string{"cursor", "pr-review"})
	buf.Reset()
	cursorAllFlag = false

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to regenerate template: %v", err)
	}

	// Verify file was overwritten
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", testFile, err)
	}
	if strings.Contains(string(content), "MODIFIED CONTENT") {
		t.Errorf("Template file was not overwritten during regeneration")
	}
	if !strings.Contains(string(content), "review-task-workflow") {
		t.Errorf("Regenerated template doesn't contain expected content")
	}
}

func TestCursorCommandWithTemplateFiles(t *testing.T) {
	// This test verifies that templates are correctly read from .claude/commands/
	// and that the command gracefully handles missing template files

	// Ensure .claude/commands/ templates exist
	requiredTemplates := []string{
		".claude/commands/issue-to-pr.md",
		".claude/commands/label-issues.md",
	}

	for _, template := range requiredTemplates {
		if _, err := os.Stat(template); os.IsNotExist(err) {
			t.Skipf("Skipping test: required template %s not found", template)
		}
	}

	// Clean up any existing .cursor directory
	os.RemoveAll(".cursor")
	defer os.RemoveAll(".cursor")

	// Generate templates that depend on .claude/commands/ files
	rootCmd := &cobra.Command{Use: "reviewtask"}
	cursorCmd := &cobra.Command{
		Use:   "cursor [TARGET]",
		Short: "Output command templates for Cursor IDE to .cursor/commands directory",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCursor,
	}
	cursorCmd.Flags().BoolVar(&cursorAllFlag, "all", false, "Generate all available command templates")
	cursorCmd.Flags().BoolVar(&cursorStdoutFlag, "stdout", false, "Output to standard output instead of files")
	rootCmd.AddCommand(cursorCmd)

	// Test issue-to-pr template generation
	rootCmd.SetArgs([]string{"cursor", "issue-to-pr"})
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to generate issue-to-pr template: %v", err)
	}

	// Verify the generated file matches source
	sourceContent, _ := os.ReadFile(filepath.FromSlash(".claude/commands/issue-to-pr.md"))
	generatedContent, _ := os.ReadFile(filepath.FromSlash(".cursor/commands/issue-to-pr/issue-to-pr.md"))

	if string(sourceContent) != string(generatedContent) {
		t.Errorf("Generated issue-to-pr template doesn't match source template")
	}

	// Test label-issues template generation
	rootCmd.SetArgs([]string{"cursor", "label-issues"})
	buf.Reset()

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to generate label-issues template: %v", err)
	}

	// Verify the generated file matches source
	sourceContent, _ = os.ReadFile(filepath.FromSlash(".claude/commands/label-issues.md"))
	generatedContent, _ = os.ReadFile(filepath.FromSlash(".cursor/commands/label-issues/label-issues.md"))

	if string(sourceContent) != string(generatedContent) {
		t.Errorf("Generated label-issues template doesn't match source template")
	}
}
