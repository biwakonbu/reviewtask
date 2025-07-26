package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
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

	// Check essential parts of empty state
	assert.Contains(t, output, "ReviewTask Status - 0% Complete")
	assert.Contains(t, output, strings.Repeat("░", 80))
	assert.Contains(t, output, "todo: 0    doing: 0    done: 0    pending: 0    cancel: 0")
	assert.Contains(t, output, "No active tasks - all completed!")
	assert.Contains(t, output, "No pending tasks")
	assert.Contains(t, output, "Last updated:")
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

	err := displayAIModeContent(testTasks, "test context")
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check completion rate (2 completed out of 5 = 40%)
	assert.Contains(t, output, "ReviewTask Status - 40.0% Complete (2/5)")

	// Check progress bar has both filled and empty parts
	assert.Contains(t, output, "█")
	assert.Contains(t, output, "░")

	// Check task summary
	assert.Contains(t, output, "todo: 2    doing: 1    done: 1    pending: 0    cancel: 1")

	// Check current task shows the doing task
	assert.Contains(t, output, "Current Task:")
	assert.Contains(t, output, "task1") // Use actual task ID instead of TSK-123
	assert.Contains(t, output, "HIGH")
	assert.Contains(t, output, "Fix authentication bug")

	// Check next tasks are sorted by priority
	assert.Contains(t, output, "Next Tasks (up to 5):")
	assert.Contains(t, output, "1. task3  HIGH    Add unit tests")         // Use actual task ID
	assert.Contains(t, output, "2. task2  MEDIUM    Update documentation") // Use actual task ID
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
			expectedMsg1: "No active tasks",
			expectedMsg2: "HIGH",
		},
		{
			name: "No todo tasks but has doing tasks",
			tasks: []storage.Task{
				{ID: "1", Status: "doing", Priority: "high", PRNumber: 1},
				{ID: "2", Status: "done", Priority: "low", PRNumber: 1},
			},
			expectedMsg1: "HIGH",
			expectedMsg2: "No pending tasks",
		},
		{
			name: "Only completed tasks",
			tasks: []storage.Task{
				{ID: "1", Status: "done", Priority: "high", PRNumber: 1},
				{ID: "2", Status: "cancel", Priority: "low", PRNumber: 1},
			},
			expectedMsg1: "No active tasks",
			expectedMsg2: "No pending tasks",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := displayAIModeContent(tc.tasks, "test")
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

// TestGenerateTaskID tests task ID generation
func TestGenerateTaskID(t *testing.T) {
	testCases := []struct {
		prNumber int
		expected string
	}{
		{42, "TSK-042"},
		{1234, "TSK-1234"},
		{1, "TSK-001"},
		{999, "TSK-999"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("PR_%d", tc.prNumber), func(t *testing.T) {
			task := storage.Task{PRNumber: tc.prNumber}
			id := tasks.GenerateTaskID(task)
			assert.Equal(t, tc.expected, id)
		})
	}
}

// TestCalculateTaskStatsNormalization tests the normalization of "cancelled" to "cancel"
func TestCalculateTaskStatsNormalization(t *testing.T) {
	testTasks := []storage.Task{
		{Status: "todo", Priority: "high", PRNumber: 1},
		{Status: "doing", Priority: "medium", PRNumber: 1},
		{Status: "done", Priority: "high", PRNumber: 2},
		{Status: "cancelled", Priority: "low", PRNumber: 2}, // Should be normalized to "cancel"
		{Status: "cancel", Priority: "critical", PRNumber: 1},
		{Status: "pending", Priority: "medium", PRNumber: 3},
	}

	stats := tasks.CalculateTaskStats(testTasks)

	// Check status counts
	assert.Equal(t, 1, stats.StatusCounts["todo"])
	assert.Equal(t, 1, stats.StatusCounts["doing"])
	assert.Equal(t, 1, stats.StatusCounts["done"])
	assert.Equal(t, 2, stats.StatusCounts["cancel"]) // Both "cancel" and "cancelled"
	assert.Equal(t, 1, stats.StatusCounts["pending"])
	assert.Equal(t, 0, stats.StatusCounts["cancelled"]) // Should not exist

	// Check priority counts
	assert.Equal(t, 1, stats.PriorityCounts["critical"])
	assert.Equal(t, 2, stats.PriorityCounts["high"])
	assert.Equal(t, 2, stats.PriorityCounts["medium"])
	assert.Equal(t, 1, stats.PriorityCounts["low"])

	// Check PR counts
	assert.Equal(t, 3, stats.PRCounts[1])
	assert.Equal(t, 2, stats.PRCounts[2])
	assert.Equal(t, 1, stats.PRCounts[3])
}

// TestWatchFlag tests the watch flag functionality
func TestWatchFlag(t *testing.T) {
	// Reset flags
	statusWatch = false

	cmd := &cobra.Command{
		Use: "status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Just check if watch mode is triggered
			if statusWatch {
				fmt.Fprint(cmd.OutOrStdout(), "Human Mode (TUI Dashboard) - Coming Soon!")
			} else {
				fmt.Fprint(cmd.OutOrStdout(), "AI Mode Output")
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Human mode: rich TUI dashboard with real-time updates")

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "Without watch flag",
			args:     []string{},
			expected: "AI Mode Output",
		},
		{
			name:     "With watch flag",
			args:     []string{"--watch"},
			expected: "Human Mode (TUI Dashboard) - Coming Soon!",
		},
		{
			name:     "With short watch flag",
			args:     []string{"-w"},
			expected: "Human Mode (TUI Dashboard) - Coming Soon!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusWatch = false // Reset before each test
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.expected)
		})
	}
}

// TestGenerateColoredProgressBar tests the colored progress bar generation
func TestGenerateColoredProgressBar(t *testing.T) {
	testCases := []struct {
		name             string
		stats            tasks.TaskStats
		width            int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Empty stats",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{},
			},
			width:            10,
			shouldContain:    []string{"░"},
			shouldNotContain: []string{"█"},
		},
		{
			name: "All done tasks",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"done":    5,
					"todo":    0,
					"doing":   0,
					"pending": 0,
					"cancel":  0,
				},
			},
			width:            10,
			shouldContain:    []string{"█"},
			shouldNotContain: []string{"░"},
		},
		{
			name: "Mixed task states",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"done":    2,
					"doing":   1,
					"todo":    1,
					"pending": 1,
					"cancel":  0,
				},
			},
			width:            10,
			shouldContain:    []string{"█", "░"},
			shouldNotContain: []string{},
		},
		{
			name: "Only incomplete tasks",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"done":    0,
					"doing":   2,
					"todo":    2,
					"pending": 1,
					"cancel":  0,
				},
			},
			width:            10,
			shouldContain:    []string{"░"},
			shouldNotContain: []string{"█"},
		},
		{
			name: "With cancelled tasks",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{
					"done":    1,
					"doing":   1,
					"todo":    1,
					"pending": 1,
					"cancel":  1,
				},
			},
			width:            10,
			shouldContain:    []string{"█", "░"},
			shouldNotContain: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generateColoredProgressBar(tc.stats, tc.width)

			// Check that result is not empty
			assert.NotEmpty(t, result)

			// Check for expected characters
			for _, expected := range tc.shouldContain {
				assert.Contains(t, result, expected, "Expected to contain '%s'", expected)
			}

			// Check for unexpected characters
			for _, unexpected := range tc.shouldNotContain {
				assert.NotContains(t, result, unexpected, "Expected NOT to contain '%s'", unexpected)
			}
		})
	}
}

// TestGenerateColoredProgressBarWidth tests that progress bar respects width constraints
func TestGenerateColoredProgressBarWidth(t *testing.T) {
	stats := tasks.TaskStats{
		StatusCounts: map[string]int{
			"done":    3,
			"doing":   2,
			"todo":    3,
			"pending": 1,
			"cancel":  1,
		},
	}

	widths := []int{10, 20, 50, 80}

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			result := generateColoredProgressBar(stats, width)

			// Count visible characters (█ and ░)
			visibleChars := strings.Count(result, "█") + strings.Count(result, "░")

			// Should match the requested width (allowing for ANSI color codes)
			assert.Equal(t, width, visibleChars, "Progress bar should have exactly %d visible characters", width)
		})
	}
}

// TestGenerateColoredProgressBarEdgeCases tests edge cases
func TestGenerateColoredProgressBarEdgeCases(t *testing.T) {
	testCases := []struct {
		name  string
		stats tasks.TaskStats
		width int
	}{
		{
			name: "Zero width",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{"done": 1},
			},
			width: 0,
		},
		{
			name: "Single character width",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{"done": 1, "todo": 1},
			},
			width: 1,
		},
		{
			name: "Large width",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{"done": 1, "todo": 1},
			},
			width: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			result := generateColoredProgressBar(tc.stats, tc.width)

			if tc.width > 0 {
				assert.NotEmpty(t, result)
				visibleChars := strings.Count(result, "█") + strings.Count(result, "░")
				assert.Equal(t, tc.width, visibleChars)
			}
		})
	}
}
