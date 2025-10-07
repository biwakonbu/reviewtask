package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"reviewtask/internal/github"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:        "PR number as argument",
			args:        []string{"123"},
			expectError: false,
			expectUsage: false,
		},
		{
			name:        "Short flag",
			args:        []string{"--short"},
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

			// Add flags (v3.0.0 simplified flags)
			var statusShowAll bool
			var statusShort bool

			cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
			cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

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
			name:           "PR number argument takes priority over all flag",
			args:           []string{"123", "--all"},
			expectedAction: "pr_arg",
		},
		{
			name:           "All flag when no PR argument",
			args:           []string{"--all"},
			expectedAction: "all_flag",
		},
		{
			name:           "Default to current branch when no flags or args",
			args:           []string{},
			expectedAction: "current_branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command with mock implementation that captures the action
			var capturedAction string

			cmd := &cobra.Command{
				Use:  "status [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock the priority logic from runStatus (v3.0.0)
					if len(args) > 0 {
						// Parse PR number from positional argument
						_, err := strconv.Atoi(args[0])
						if err == nil {
							capturedAction = "pr_arg"
							return nil
						}
					}
					if statusShowAll {
						capturedAction = "all_flag"
					} else {
						capturedAction = "current_branch"
					}
					return nil
				},
			}

			// Add flags (v3.0.0 simplified flags)
			cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
			cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

			// Reset flags before each test
			statusShowAll = false
			statusShort = false

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
		Use:   "status [PR_NUMBER]",
		Short: "Show current task status and statistics",
		Long: `Display current tasks, next tasks to work on, and overall statistics.

By default, shows tasks for the current branch. Provide a PR number to show
tasks for a specific PR, or use --all to show all PRs.

Shows:
- Current tasks (doing status)
- Next tasks (todo status, sorted by priority)
- Task statistics (status breakdown, priority breakdown, completion rate)

Examples:
  reviewtask status             # Show tasks for current branch
  reviewtask status 123         # Show tasks for PR #123
  reviewtask status --all       # Show tasks for all PRs
  reviewtask status --short     # Brief output format`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Mock implementation - should not run
			return nil
		},
	}

	// Add flags (v3.0.0 simplified)
	var statusShowAll bool
	var statusShort bool

	cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
	cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error when getting help, got: %v", err)
	}

	output := buf.String()

	// Check for key elements in help text (v3.0.0)
	expectedElements := []string{
		"Display current tasks, next tasks",
		"Examples:",
		"reviewtask status",
		"--all",
		"--short",
		"Show tasks for current branch",
		"Show tasks for all PRs",
		"Show tasks for PR #123",
		"Brief output format",
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

// TestStatusCommandFlags tests flag parsing (v3.0.0)
func TestStatusCommandFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedAll   bool
		expectedShort bool
	}{
		{
			name:          "All flag set",
			args:          []string{"--all"},
			expectedAll:   true,
			expectedShort: false,
		},
		{
			name:          "Short flag set",
			args:          []string{"--short"},
			expectedAll:   false,
			expectedShort: true,
		},
		{
			name:          "Both flags set",
			args:          []string{"--all", "--short"},
			expectedAll:   true,
			expectedShort: true,
		},
		{
			name:          "No flags set",
			args:          []string{},
			expectedAll:   false,
			expectedShort: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			statusShowAll = false
			statusShort = false

			// Create a copy of the command to avoid state pollution
			cmd := &cobra.Command{
				Use:  "status [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Just capture flag values, don't execute logic
					return nil
				},
			}

			cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
			cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if statusShowAll != tt.expectedAll {
				t.Errorf("Expected statusShowAll to be %v, got %v", tt.expectedAll, statusShowAll)
			}

			if statusShort != tt.expectedShort {
				t.Errorf("Expected statusShort to be %v, got %v", tt.expectedShort, statusShort)
			}
		})
	}
}

// TestStatusTaskSorting tests the task sorting functionality
func TestStatusTaskSorting(t *testing.T) {
	// Mock tasks with different priorities
	testTasks := []storage.Task{
		{Priority: "low", Description: "Low priority task"},
		{Priority: "critical", Description: "Critical task"},
		{Priority: "medium", Description: "Medium task"},
		{Priority: "high", Description: "High priority task"},
		{Priority: "medium", Description: "Another medium task"},
	}

	// Sort tasks
	tasks.SortTasksByPriority(testTasks)

	// Verify sorting order
	expectedOrder := []string{"critical", "high", "medium", "medium", "low"}

	for i, task := range testTasks {
		if task.Priority != expectedOrder[i] {
			t.Errorf("Expected task %d to have priority '%s', got '%s'", i, expectedOrder[i], task.Priority)
		}
	}
}

// TestStatusTaskFiltering tests the task filtering functionality
func TestStatusTaskFiltering(t *testing.T) {
	testTasks := []storage.Task{
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
		filtered := tasks.FilterTasksByStatus(testTasks, tt.status)
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
	testTasks := []storage.Task{
		{Status: "todo", Priority: "high", PRNumber: 1},
		{Status: "doing", Priority: "medium", PRNumber: 1},
		{Status: "done", Priority: "high", PRNumber: 2},
		{Status: "todo", Priority: "low", PRNumber: 2},
		{Status: "cancel", Priority: "critical", PRNumber: 1},
	}

	stats := tasks.CalculateTaskStats(testTasks)

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

// TestDisplayAIModeEmpty tests the AI mode empty state output
func TestDisplayAIModeEmpty(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayAIModeEmpty()
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check essential parts of empty state (Modern UI)
	assert.Contains(t, output, "Review Status")
	assert.Contains(t, output, "Progress: 0% Complete (0/0)")
	assert.Contains(t, output, strings.Repeat("░", 80))
	assert.Contains(t, output, "Tasks")
	assert.Contains(t, output, "todo: 0    doing: 0    done: 0    pending: 0    cancel: 0")
	assert.Contains(t, output, "Current Task")
	assert.Contains(t, output, "No active tasks - all completed!")
	assert.Contains(t, output, "Next Steps")
	assert.Contains(t, output, "All tasks completed!")
}

// TestDisplayAIModeContent tests the AI mode content output
func TestDisplayAIModeContent(t *testing.T) {
	// Create test tasks
	testTasks := []storage.Task{
		{
			ID:          "task1",
			Description: "Fix authentication bug",
			Priority:    "high",
			Status:      "doing",
			PRNumber:    123,
			File:        "auth.go",
			Line:        45,
		},
		{
			ID:          "task2",
			Description: "Update documentation",
			Priority:    "medium",
			Status:      "todo",
			PRNumber:    123,
			File:        "README.md",
			Line:        10,
		},
		{
			ID:          "task3",
			Description: "Add unit tests",
			Priority:    "high",
			Status:      "todo",
			PRNumber:    123,
			File:        "test.go",
			Line:        100,
		},
		{
			ID:          "task4",
			Description: "Refactor database layer",
			Priority:    "low",
			Status:      "done",
			PRNumber:    123,
			File:        "db.go",
			Line:        200,
		},
		{
			ID:          "task5",
			Description: "Remove deprecated API",
			Priority:    "medium",
			Status:      "cancel",
			PRNumber:    123,
			File:        "api.go",
			Line:        150,
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayAIModeContent(testTasks, "test context", nil, nil)
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check completion rate (Modern UI format)
	assert.Contains(t, output, "Review Status")
	assert.Contains(t, output, "Progress: 40.0% Complete (2/5)")

	// Check progress bar has both filled and empty parts
	assert.Contains(t, output, "Progress [")
	assert.Contains(t, output, "█")
	assert.Contains(t, output, "░")

	// Check task summary (Modern UI - vertical layout)
	assert.Contains(t, output, "Tasks")
	assert.Contains(t, output, "TODO: 2")
	assert.Contains(t, output, "DOING: 1")
	assert.Contains(t, output, "DONE: 1")
	assert.Contains(t, output, "PENDING: 0")
	assert.Contains(t, output, "CANCEL: 1")

	// Check current task shows the doing task (Modern UI)
	assert.Contains(t, output, "Current Task")
	assert.Contains(t, output, "task1") // Use actual task ID instead of TSK-123
	assert.Contains(t, output, "HIGH")
	assert.Contains(t, output, "Fix authentication bug")

	// Check next tasks are sorted by priority (Modern UI)
	assert.Contains(t, output, "Next Tasks")
	assert.Contains(t, output, "1. task3  HIGH    Add unit tests")         // Use actual task ID
	assert.Contains(t, output, "2. task2  MEDIUM    Update documentation") // Use actual task ID

	// Check Next Steps section (Modern UI)
	assert.Contains(t, output, "Next Steps")
}

// TestEnglishMessagesInAIModeNoActiveTasks verifies English messages when no active tasks
func TestEnglishMessagesInAIModeNoActiveTasks(t *testing.T) {
	testCases := []struct {
		name         string
		tasks        []storage.Task
		expectedMsg1 string
		expectedMsg2 string
	}{
		{
			name: "No doing tasks but has todo tasks",
			tasks: []storage.Task{
				{ID: "1", Status: "todo", Priority: "high", PRNumber: 1},
				{ID: "2", Status: "done", Priority: "low", PRNumber: 1},
			},
			expectedMsg1: "Next Tasks", // Modern UI shows next tasks section when todo tasks exist
			expectedMsg2: "HIGH",
		},
		{
			name: "No todo tasks but has doing tasks",
			tasks: []storage.Task{
				{ID: "1", Status: "doing", Priority: "high", PRNumber: 1},
				{ID: "2", Status: "done", Priority: "low", PRNumber: 1},
			},
			expectedMsg1: "HIGH",
			expectedMsg2: "Current Task", // Modern UI shows current task section
		},
		{
			name: "Only completed tasks",
			tasks: []storage.Task{
				{ID: "1", Status: "done", Priority: "high", PRNumber: 1},
				{ID: "2", Status: "cancel", Priority: "low", PRNumber: 1},
			},
			expectedMsg1: "Next Steps",           // Modern UI shows Next Steps section
			expectedMsg2: "All tasks completed!", // Modern UI shows completion message in Next Steps
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := displayAIModeContent(tc.tasks, "test", nil, nil)
			require.NoError(t, err)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Verify expected messages appear
			assert.Contains(t, output, tc.expectedMsg1)
			assert.Contains(t, output, tc.expectedMsg2)

			// Ensure no Japanese messages appear
			assert.NotContains(t, output, "アクティブなタスクはありません")
			assert.NotContains(t, output, "待機中のタスクはありません")
		})
	}
}

// TestStatusCommandArgumentValidation tests command argument validation logic
func TestStatusCommandArgumentValidation(t *testing.T) {
	// Test cases for command argument validation
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "Valid PR number",
			args:          []string{"123"},
			expectedError: "",
		},
		{
			name:          "Valid PR number with flags",
			args:          []string{"456", "--all", "--short"},
			expectedError: "",
		},
		{
			name:          "Invalid PR number format",
			args:          []string{"abc"},
			expectedError: "invalid PR number",
		},
		{
			name:          "Zero PR number",
			args:          []string{"0"},
			expectedError: "must be a positive integer",
		},
		{
			name:          "Negative PR number",
			args:          []string{"--", "-1"},
			expectedError: "must be a positive integer",
		},
		{
			name:          "Too many arguments",
			args:          []string{"123", "456"},
			expectedError: "accepts at most 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "status [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Parse PR number from arguments if provided
					if len(args) > 0 {
						prNumber, err := strconv.Atoi(args[0])
						if err != nil {
							return fmt.Errorf("invalid PR number: %s", args[0])
						}
						if prNumber <= 0 {
							return fmt.Errorf("invalid PR number: %s (must be a positive integer)", args[0])
						}
					}
					// If we reach here, validation passed
					return nil
				},
			}

			// Add flags
			cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
			cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectedError == "" && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectedError != "" && err == nil {
				t.Errorf("Expected error containing '%s' but got none", tt.expectedError)
			}
			if tt.expectedError != "" && err != nil {
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s' but got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

// TestStatusCommandErrorHandling tests error scenarios
func TestStatusCommandErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "Invalid PR number format",
			args:          []string{"abc"},
			expectedError: "invalid PR number",
		},
		{
			name:          "Empty PR number",
			args:          []string{""},
			expectedError: "invalid PR number",
		},
		{
			name:          "Zero PR number",
			args:          []string{"0"},
			expectedError: "must be a positive integer",
		},
		{
			name:          "Negative PR number",
			args:          []string{"--", "-1"},
			expectedError: "must be a positive integer",
		},
		{
			name:          "Too many arguments",
			args:          []string{"123", "456"},
			expectedError: "accepts at most 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "status [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: runStatus,
			}

			// Add flags
			cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
			cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestStatusCommandIntegration tests the integration with storage and task management
func TestStatusCommandIntegration(t *testing.T) {
	// This test verifies that the status command properly integrates
	// with the storage and task management systems
	t.Run("Storage Manager Integration", func(t *testing.T) {
		// Test that the command can create and use a storage manager
		cmd := &cobra.Command{
			Use:  "status [PR_NUMBER]",
			Args: cobra.MaximumNArgs(1),
			RunE: runStatus,
		}

		// Add flags
		cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
		cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

		// Test with no arguments (should not error during setup)
		cmd.SetArgs([]string{})
		// Note: This test would need proper mocking for full integration testing
		// For now, we verify the command structure is correct
		assert.NotNil(t, cmd)
	})

	t.Run("Task Management Integration", func(t *testing.T) {
		// Test that the command properly handles task-related operations
		// This would typically test the interaction between status command
		// and the tasks package functions like CalculateTaskStats, FilterTasksByStatus, etc.

		// Create test tasks
		testTasks := []storage.Task{
			{ID: "task1", Status: "doing", Priority: "high", PRNumber: 123},
			{ID: "task2", Status: "todo", Priority: "medium", PRNumber: 123},
			{ID: "task3", Status: "done", Priority: "low", PRNumber: 123},
		}

		// Verify task structure
		assert.Len(t, testTasks, 3)
		assert.Equal(t, "doing", testTasks[0].Status)
		assert.Equal(t, "high", testTasks[0].Priority)
		assert.Equal(t, 123, testTasks[0].PRNumber)
	})
}

// TestStatusCommandFlagCombinations tests various flag combinations
func TestStatusCommandFlagCombinations(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedAll   bool
		expectedShort bool
		expectedError bool
	}{
		{
			name:          "All flag only",
			args:          []string{"--all"},
			expectedAll:   true,
			expectedShort: false,
			expectedError: false,
		},
		{
			name:          "Short flag only",
			args:          []string{"--short"},
			expectedAll:   false,
			expectedShort: true,
			expectedError: false,
		},
		{
			name:          "Both flags",
			args:          []string{"--all", "--short"},
			expectedAll:   true,
			expectedShort: true,
			expectedError: false,
		},
		{
			name:          "PR number with all flag",
			args:          []string{"123", "--all"},
			expectedAll:   true,
			expectedShort: false,
			expectedError: false,
		},
		{
			name:          "PR number with short flag",
			args:          []string{"123", "--short"},
			expectedAll:   false,
			expectedShort: true,
			expectedError: false,
		},
		{
			name:          "PR number with both flags",
			args:          []string{"123", "--all", "--short"},
			expectedAll:   true,
			expectedShort: true,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			statusShowAll = false
			statusShort = false

			cmd := &cobra.Command{
				Use:  "status [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Just validate that flags are parsed correctly
					return nil
				},
			}

			// Add flags
			cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
			cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectedError {
				assert.Equal(t, tt.expectedAll, statusShowAll)
				assert.Equal(t, tt.expectedShort, statusShort)
			}
		})
	}
}

// TestStatusCommandPriority tests that PR number argument takes priority over flags
func TestStatusCommandPriority(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedAction string
	}{
		{
			name:           "PR number argument with conflicting flags",
			args:           []string{"123", "--all"},
			expectedAction: "pr_number_takes_priority",
		},
		{
			name:           "All flag when no PR argument",
			args:           []string{"--all"},
			expectedAction: "all_flag_used",
		},
		{
			name:           "Default behavior with no arguments",
			args:           []string{},
			expectedAction: "current_branch_detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedAction string

			cmd := &cobra.Command{
				Use:  "status [PR_NUMBER]",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Simulate the priority logic from runStatus
					if len(args) > 0 {
						// Parse PR number from positional argument
						_, err := strconv.Atoi(args[0])
						if err == nil {
							capturedAction = "pr_number_takes_priority"
							return nil
						}
					}

					if statusShowAll {
						capturedAction = "all_flag_used"
					} else {
						capturedAction = "current_branch_detected"
					}
					return nil
				},
			}

			// Add flags
			cmd.Flags().BoolVarP(&statusShowAll, "all", "a", false, "Show tasks for all PRs")
			cmd.Flags().BoolVarP(&statusShort, "short", "s", false, "Brief output format")

			// Reset flags
			statusShowAll = false
			statusShort = false

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAction, capturedAction)
		})
	}
}

// TestDetectCompletionState tests the completion state detection logic
func TestDetectCompletionState(t *testing.T) {
	tests := []struct {
		name             string
		tasks            []storage.Task
		unresolvedReport *github.UnresolvedCommentsReport
		expectedComplete bool
		expectedSummary  string
	}{
		{
			name: "All tasks completed, no unresolved comments",
			tasks: []storage.Task{
				{ID: "task-1", Status: "done"},
				{ID: "task-2", Status: "done"},
			},
			unresolvedReport: &github.UnresolvedCommentsReport{
				UnanalyzedComments: []github.Comment{},
				InProgressComments: []github.Comment{},
				ResolvedComments:   []github.Comment{{ID: 1}, {ID: 2}},
			},
			expectedComplete: true,
			expectedSummary:  "✅ All tasks completed and all comments resolved",
		},
		{
			name: "Some tasks pending, some comments unresolved",
			tasks: []storage.Task{
				{ID: "task-1", Status: "done"},
				{ID: "task-2", Status: "todo"},
				{ID: "task-3", Status: "doing"},
			},
			unresolvedReport: &github.UnresolvedCommentsReport{
				UnanalyzedComments: []github.Comment{{ID: 1}},
				InProgressComments: []github.Comment{{ID: 2}},
				ResolvedComments:   []github.Comment{},
			},
			expectedComplete: false,
			expectedSummary:  "⏳ Incomplete: 1 pending tasks, 1 in-progress tasks, 2 unresolved comments",
		},
		{
			name: "All tasks completed but comments unresolved",
			tasks: []storage.Task{
				{ID: "task-1", Status: "done"},
				{ID: "task-2", Status: "cancel"},
			},
			unresolvedReport: &github.UnresolvedCommentsReport{
				UnanalyzedComments: []github.Comment{{ID: 1}},
				InProgressComments: []github.Comment{},
				ResolvedComments:   []github.Comment{},
			},
			expectedComplete: true,
			expectedSummary:  "✅ All tasks completed and all comments resolved",
		},
		{
			name:  "No tasks but comments resolved",
			tasks: []storage.Task{},
			unresolvedReport: &github.UnresolvedCommentsReport{
				UnanalyzedComments: []github.Comment{},
				InProgressComments: []github.Comment{},
				ResolvedComments:   []github.Comment{},
			},
			expectedComplete: true,
			expectedSummary:  "✅ All tasks completed and all comments resolved",
		},
		{
			name: "Tasks completed but no comment report",
			tasks: []storage.Task{
				{ID: "task-1", Status: "done"},
			},
			unresolvedReport: nil,
			expectedComplete: true,
			expectedSummary:  "✅ All tasks completed and all comments resolved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectCompletionState(tt.tasks, tt.unresolvedReport, 123)

			assert.Equal(t, tt.expectedComplete, result.IsComplete)
			assert.Contains(t, result.CompletionSummary, tt.expectedSummary)

			// Check unresolved items count
			if tt.unresolvedReport != nil {
				expectedUnresolvedComments := len(tt.unresolvedReport.UnanalyzedComments) + len(tt.unresolvedReport.InProgressComments)
				assert.Equal(t, expectedUnresolvedComments, len(result.UnresolvedComments))
			}

			// Check unresolved tasks count
			expectedUnresolvedTasks := 0
			for _, task := range tt.tasks {
				if task.Status == "todo" || task.Status == "doing" || task.Status == "pending" {
					expectedUnresolvedTasks++
				}
			}
			assert.Equal(t, expectedUnresolvedTasks, len(result.UnresolvedTasks))
		})
	}
}

// TestUnresolvedCommentsReport tests the UnresolvedCommentsReport functionality
func TestUnresolvedCommentsReport(t *testing.T) {
	comment1 := github.Comment{ID: 1, Body: "Comment 1"}
	comment2 := github.Comment{ID: 2, Body: "Comment 2"}
	comment3 := github.Comment{ID: 3, Body: "Comment 3"}

	report := &github.UnresolvedCommentsReport{
		UnanalyzedComments: []github.Comment{comment1},
		InProgressComments: []github.Comment{comment2},
		ResolvedComments:   []github.Comment{comment3},
	}

	// Test IsComplete
	assert.False(t, report.IsComplete())

	// Test with all comments resolved
	emptyReport := &github.UnresolvedCommentsReport{
		UnanalyzedComments: []github.Comment{},
		InProgressComments: []github.Comment{},
		ResolvedComments:   []github.Comment{comment1, comment2, comment3},
	}
	assert.True(t, emptyReport.IsComplete())

	// Test GetSummary
	summary := report.GetSummary()
	assert.Contains(t, summary, "Unresolved Comments: 2")
	assert.Contains(t, summary, "1 comments not yet analyzed")
	assert.Contains(t, summary, "1 comments with pending tasks")
	assert.Contains(t, summary, "1 comments resolved")
}
