package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
	"reviewtask/internal/ui"
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

// TestWatchFlag has been removed in v3.0.0 since --watch flag is deprecated
// The TUI dashboard functionality has been moved to a separate command

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
			result := ui.GenerateColoredProgressBar(tc.stats, tc.width)

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
			result := ui.GenerateColoredProgressBar(stats, width)

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
			name: "Negative width",
			stats: tasks.TaskStats{
				StatusCounts: map[string]int{"done": 1},
			},
			width: -5,
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
			result := ui.GenerateColoredProgressBar(tc.stats, tc.width)

			if tc.width <= 0 {
				// For zero or negative width, should return empty string
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				visibleChars := strings.Count(result, "█") + strings.Count(result, "░")
				assert.Equal(t, tc.width, visibleChars)
			}
		})
	}
}
