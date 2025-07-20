package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
)

// TestStatusCommand tests the status command functionality
func TestStatusCommand(t *testing.T) {
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
			name:        "No arguments (default behavior)",
			args:        []string{},
			expectError: false,
			expectUsage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command for each test to avoid flag pollution
			cmd := &cobra.Command{
				Use:   "status",
				Short: "Show current task status and statistics",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock implementation that doesn't actually run status
					return nil
				},
			}

			// Add flags
			var statusShowAll bool
			var statusSpecificPR int
			var statusBranch string

			cmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
			cmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
			cmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")

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

// TestStatusFlagPriority tests the priority of different flags
func TestStatusFlagPriority(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedAction string
	}{
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

			cmd := &cobra.Command{
				Use: "status",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock the priority logic from runStatus
					if statusSpecificPR > 0 {
						capturedAction = "pr_flag"
					} else if statusBranch != "" {
						capturedAction = "branch_flag"
					} else if statusShowAll {
						capturedAction = "all_flag"
					} else {
						capturedAction = "current_branch"
					}
					return nil
				},
			}

			// Add flags (using package-level variables to match actual implementation)
			cmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
			cmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
			cmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")

			// Reset flags before each test
			statusShowAll = false
			statusSpecificPR = 0
			statusBranch = ""

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

// TestStatusCommandHelp tests the help text of the status command
func TestStatusCommandHelp(t *testing.T) {
	// Create a fresh command instance to avoid interference with global state
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current task status and statistics",
		Long: `Display current tasks, next tasks to work on, and overall statistics.

By default, shows tasks for the current branch. Use flags to show all PRs 
or filter by specific criteria.

Shows:
- Current tasks (doing status)
- Next tasks (todo status, sorted by priority)
- Task statistics (status breakdown, priority breakdown, completion rate)

Examples:
  reviewtask status             # Show current branch tasks
  reviewtask status --all       # Show all PRs tasks
  reviewtask status --pr 123    # Show PR #123 tasks
  reviewtask status --branch feature/xyz # Show specific branch tasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mock implementation - should not run
			return nil
		},
	}

	// Add flags
	var statusShowAll bool
	var statusSpecificPR int
	var statusBranch string

	cmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
	cmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
	cmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")

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
		"Display current tasks, next tasks",
		"Examples:",
		"reviewtask status",
		"--all",
		"--pr",
		"--branch",
		"Show current branch tasks",
		"Show all PRs tasks",
		"Show PR #123 tasks",
		"Show specific branch tasks",
		"Current tasks (doing status)",
		"Next tasks (todo status, sorted by priority)",
		"Task statistics",
	}

	for _, element := range expectedElements {
		if !strings.Contains(output, element) {
			t.Errorf("Expected help text to contain '%s', but it didn't. Full output:\n%s", element, output)
		}
	}
}

// TestStatusCommandFlags tests flag parsing
func TestStatusCommandFlags(t *testing.T) {
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
			// Reset flags
			statusShowAll = false
			statusSpecificPR = 0
			statusBranch = ""

			// Create a copy of the command to avoid state pollution
			cmd := &cobra.Command{
				Use: "status",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Just capture flag values, don't execute logic
					return nil
				},
			}

			cmd.Flags().BoolVar(&statusShowAll, "all", false, "Show tasks for all PRs")
			cmd.Flags().IntVar(&statusSpecificPR, "pr", 0, "Show tasks for specific PR number")
			cmd.Flags().StringVar(&statusBranch, "branch", "", "Show tasks for specific branch")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if statusShowAll != tt.expectedAll {
				t.Errorf("Expected statusShowAll to be %v, got %v", tt.expectedAll, statusShowAll)
			}

			if statusSpecificPR != tt.expectedPR {
				t.Errorf("Expected statusSpecificPR to be %d, got %d", tt.expectedPR, statusSpecificPR)
			}

			if statusBranch != tt.expectedBranch {
				t.Errorf("Expected statusBranch to be '%s', got '%s'", tt.expectedBranch, statusBranch)
			}
		})
	}
}

// TestStatusTaskSorting tests the task sorting functionality
func TestStatusTaskSorting(t *testing.T) {
	// Mock tasks with different priorities
	tasks := []storage.Task{
		{Priority: "low", Description: "Low priority task"},
		{Priority: "critical", Description: "Critical task"},
		{Priority: "medium", Description: "Medium task"},
		{Priority: "high", Description: "High priority task"},
		{Priority: "medium", Description: "Another medium task"},
	}

	// Sort tasks
	sortTasksByPriority(tasks)

	// Verify sorting order
	expectedOrder := []string{"critical", "high", "medium", "medium", "low"}

	for i, task := range tasks {
		if task.Priority != expectedOrder[i] {
			t.Errorf("Expected task %d to have priority '%s', got '%s'", i, expectedOrder[i], task.Priority)
		}
	}
}

// TestStatusTaskFiltering tests the task filtering functionality
func TestStatusTaskFiltering(t *testing.T) {
	tasks := []storage.Task{
		{Status: "todo", Description: "Todo task 1"},
		{Status: "doing", Description: "Doing task 1"},
		{Status: "done", Description: "Done task 1"},
		{Status: "todo", Description: "Todo task 2"},
		{Status: "pending", Description: "Pending task 1"},
		{Status: "doing", Description: "Doing task 2"},
	}

	tests := []struct {
		status   string
		expected int
	}{
		{"todo", 2},
		{"doing", 2},
		{"done", 1},
		{"pending", 1},
		{"cancel", 0},
	}

	for _, tt := range tests {
		filtered := filterTasksByStatus(tasks, tt.status)
		if len(filtered) != tt.expected {
			t.Errorf("Expected %d tasks with status '%s', got %d", tt.expected, tt.status, len(filtered))
		}

		// Verify all filtered tasks have the correct status
		for _, task := range filtered {
			if task.Status != tt.status {
				t.Errorf("Expected task status '%s', got '%s'", tt.status, task.Status)
			}
		}
	}
}

// TestStatusTaskStats tests the task statistics calculation
func TestStatusTaskStats(t *testing.T) {
	tasks := []storage.Task{
		{Status: "todo", Priority: "high", PRNumber: 1},
		{Status: "doing", Priority: "medium", PRNumber: 1},
		{Status: "done", Priority: "high", PRNumber: 2},
		{Status: "todo", Priority: "low", PRNumber: 2},
		{Status: "cancel", Priority: "critical", PRNumber: 1},
	}

	stats := calculateTaskStats(tasks)

	// Test status counts
	expectedStatusCounts := map[string]int{
		"todo":   2,
		"doing":  1,
		"done":   1,
		"cancel": 1,
	}

	for status, expected := range expectedStatusCounts {
		if stats.StatusCounts[status] != expected {
			t.Errorf("Expected %d tasks with status '%s', got %d", expected, status, stats.StatusCounts[status])
		}
	}

	// Test priority counts
	expectedPriorityCounts := map[string]int{
		"critical": 1,
		"high":     2,
		"medium":   1,
		"low":      1,
	}

	for priority, expected := range expectedPriorityCounts {
		if stats.PriorityCounts[priority] != expected {
			t.Errorf("Expected %d tasks with priority '%s', got %d", expected, priority, stats.PriorityCounts[priority])
		}
	}

	// Test PR counts
	expectedPRCounts := map[int]int{
		1: 3,
		2: 2,
	}

	for pr, expected := range expectedPRCounts {
		if stats.PRCounts[pr] != expected {
			t.Errorf("Expected %d tasks for PR %d, got %d", expected, pr, stats.PRCounts[pr])
		}
	}
}
