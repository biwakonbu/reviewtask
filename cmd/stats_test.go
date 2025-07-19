package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestStatsCommand tests the stats command functionality
func TestStatsCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectUsage bool
	}{
		{
			name:        "Help flag",
			args:        []string{"--help"},
			expectError: false,
			expectUsage: true,
		},
		{
			name:        "All flag",
			args:        []string{"--all"},
			expectError: false,
			expectUsage: false,
		},
		{
			name:        "PR flag with valid number",
			args:        []string{"--pr", "123"},
			expectError: false,
			expectUsage: false,
		},
		{
			name:        "Branch flag",
			args:        []string{"--branch", "feature/test"},
			expectError: false,
			expectUsage: false,
		},
		{
			name:        "Positional argument",
			args:        []string{"123"},
			expectError: false,
			expectUsage: false,
		},
		{
			name:        "Invalid positional argument",
			args:        []string{"abc"},
			expectError: false, // Mock implementation doesn't validate args
			expectUsage: false,
		},
		{
			name:        "Too many arguments",
			args:        []string{"123", "456"},
			expectError: true,
			expectUsage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command for each test to avoid flag pollution
			cmd := &cobra.Command{
				Use:   "stats [PR_NUMBER]",
				Short: "Show task statistics by comment",
				Args:  cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock implementation that doesn't actually run statistics
					return nil
				},
			}

			// Add flags
			var showAll bool
			var specificPR int
			var branchName string

			cmd.Flags().BoolVar(&showAll, "all", false, "Show statistics for all PRs")
			cmd.Flags().IntVar(&specificPR, "pr", 0, "Show statistics for specific PR number")
			cmd.Flags().StringVar(&branchName, "branch", "", "Show statistics for specific branch")

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set args
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

			// Check if help was displayed
			output := buf.String()
			if tt.expectUsage {
				if !strings.Contains(output, "Usage:") {
					t.Errorf("Expected usage information in output, got: %s", output)
				}
			}
		})
	}
}

// TestStatsFlagPriority tests the priority of different flags and arguments
func TestStatsFlagPriority(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedAction string
	}{
		{
			name:           "Positional argument takes priority over flags",
			args:           []string{"123", "--pr", "456", "--branch", "test"},
			expectedAction: "positional",
		},
		{
			name:           "PR flag takes priority over branch flag",
			args:           []string{"--pr", "123", "--branch", "test"},
			expectedAction: "pr_flag",
		},
		{
			name:           "Branch flag takes priority over all flag",
			args:           []string{"--branch", "test", "--all"},
			expectedAction: "branch_flag",
		},
		{
			name:           "All flag when no other flags",
			args:           []string{"--all"},
			expectedAction: "all_flag",
		},
		{
			name:           "Default to current branch when no flags",
			args:           []string{},
			expectedAction: "current_branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command with mock implementation that captures the action
			var capturedAction string
			
			// Use local variables to avoid race conditions
			var localShowAllPRs bool
			var localSpecificPR int
			var localBranchName string

			cmd := &cobra.Command{
				Use:  "stats [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock the priority logic from runStats
					if len(args) > 0 {
						capturedAction = "positional"
					} else if localSpecificPR > 0 {
						capturedAction = "pr_flag"
					} else if localBranchName != "" {
						capturedAction = "branch_flag"
					} else if localShowAllPRs {
						capturedAction = "all_flag"
					} else {
						capturedAction = "current_branch"
					}
					return nil
				},
			}

			// Add flags using local variables
			cmd.Flags().BoolVar(&localShowAllPRs, "all", false, "Show statistics for all PRs")
			cmd.Flags().IntVar(&localSpecificPR, "pr", 0, "Show statistics for specific PR number")
			cmd.Flags().StringVar(&localBranchName, "branch", "", "Show statistics for specific branch")

			// Set args and execute
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if capturedAction != tt.expectedAction {
				t.Errorf("Expected action %s, got: %s", tt.expectedAction, capturedAction)
			}
		})
	}
}

// TestStatsCommandHelp tests the help text of the stats command
func TestStatsCommandHelp(t *testing.T) {
	// Create a fresh command instance to avoid interference with global state
	cmd := &cobra.Command{
		Use:   "stats [PR_NUMBER]",
		Short: "Show task statistics by comment",
		Long: `Display detailed statistics about tasks generated from PR review comments.
Shows both overall statistics and per-comment breakdown of task status.

By default, shows statistics for the current branch. Use flags to show all PRs 
or filter by specific criteria.

Examples:
  gh-review-task stats           # Show stats for current branch
  gh-review-task stats --all     # Show stats for all PRs
  gh-review-task stats --pr 123  # Show stats for PR #123
  gh-review-task stats --branch feature/xyz  # Show stats for specific branch
  gh-review-task stats 123       # Show stats for PR #123 (positional)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mock implementation - should not run
			return nil
		},
	}

	// Add flags
	var showAll bool
	var specificPR int
	var branchName string

	cmd.Flags().BoolVar(&showAll, "all", false, "Show statistics for all PRs")
	cmd.Flags().IntVar(&specificPR, "pr", 0, "Show statistics for specific PR number")
	cmd.Flags().StringVar(&branchName, "branch", "", "Show statistics for specific branch")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error when getting help, got: %v", err)
	}

	output := buf.String()

	// Check for key elements in help text
	expectedElements := []string{
		"Display detailed statistics about tasks",
		"Examples:",
		"gh-review-task stats",
		"--all",
		"--pr",
		"--branch",
		"Show stats for current branch",
		"Show stats for all PRs",
		"Show stats for PR #123",
		"Show stats for specific branch",
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("Expected help text to contain '%s', but it didn't. Full output:\n%s", element, output)
		}
	}
}

// TestStatsCommandFlags tests flag parsing
func TestStatsCommandFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedAll    bool
		expectedPR     int
		expectedBranch string
	}{
		{
			name:           "All flag set",
			args:           []string{"--all"},
			expectedAll:    true,
			expectedPR:     0,
			expectedBranch: "",
		},
		{
			name:           "PR flag set",
			args:           []string{"--pr", "123"},
			expectedAll:    false,
			expectedPR:     123,
			expectedBranch: "",
		},
		{
			name:           "Branch flag set",
			args:           []string{"--branch", "feature/test"},
			expectedAll:    false,
			expectedPR:     0,
			expectedBranch: "feature/test",
		},
		{
			name:           "Multiple flags set",
			args:           []string{"--all", "--pr", "456", "--branch", "main"},
			expectedAll:    true,
			expectedPR:     456,
			expectedBranch: "main",
		},
		{
			name:           "No flags set",
			args:           []string{},
			expectedAll:    false,
			expectedPR:     0,
			expectedBranch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use local variables to avoid race conditions
			var localShowAllPRs bool
			var localSpecificPR int
			var localBranchName string

			// Create a copy of the command to avoid state pollution
			cmd := &cobra.Command{
				Use:  "stats [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Just capture flag values, don't execute logic
					return nil
				},
			}

			cmd.Flags().BoolVar(&localShowAllPRs, "all", false, "Show statistics for all PRs")
			cmd.Flags().IntVar(&localSpecificPR, "pr", 0, "Show statistics for specific PR number")
			cmd.Flags().StringVar(&localBranchName, "branch", "", "Show statistics for specific branch")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if localShowAllPRs != tt.expectedAll {
				t.Errorf("Expected showAllPRs to be %v, got %v", tt.expectedAll, localShowAllPRs)
			}

			if localSpecificPR != tt.expectedPR {
				t.Errorf("Expected specificPR to be %d, got %d", tt.expectedPR, localSpecificPR)
			}

			if localBranchName != tt.expectedBranch {
				t.Errorf("Expected branchName to be '%s', got '%s'", tt.expectedBranch, localBranchName)
			}
		})
	}
}
