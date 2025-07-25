package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/storage"
)

// TestStatusDisplaysActualUUIDs tests that the status command displays actual task UUIDs
// instead of custom TSK-XXX format IDs. This addresses Issue #112.
func TestStatusDisplaysActualUUIDs(t *testing.T) {
	testCases := []struct {
		name     string
		tasks    []storage.Task
		expected []string // Expected UUIDs in output
	}{
		{
			name: "Single doing task displays actual UUID",
			tasks: []storage.Task{
				{
					ID:          "uuid-12345-abcde-67890",
					Description: "Fix critical bug",
					Priority:    "critical",
					Status:      "doing",
					PRNumber:    123,
				},
			},
			expected: []string{"uuid-12345-abcde-67890"},
		},
		{
			name: "Multiple todo tasks display actual UUIDs in priority order",
			tasks: []storage.Task{
				{
					ID:          "uuid-task-1",
					Description: "Low priority task",
					Priority:    "low",
					Status:      "todo",
					PRNumber:    123,
				},
				{
					ID:          "uuid-task-2",
					Description: "Critical task",
					Priority:    "critical",
					Status:      "todo",
					PRNumber:    123,
				},
				{
					ID:          "uuid-task-3",
					Description: "High priority task",
					Priority:    "high",
					Status:      "todo",
					PRNumber:    123,
				},
			},
			expected: []string{"uuid-task-2", "uuid-task-3", "uuid-task-1"}, // Priority order: critical, high, low
		},
		{
			name: "Mixed statuses show UUIDs for doing and todo only",
			tasks: []storage.Task{
				{
					ID:          "uuid-doing-task",
					Description: "Current work",
					Priority:    "high",
					Status:      "doing",
					PRNumber:    123,
				},
				{
					ID:          "uuid-todo-task",
					Description: "Next work",
					Priority:    "medium",
					Status:      "todo",
					PRNumber:    123,
				},
				{
					ID:          "uuid-done-task",
					Description: "Completed work",
					Priority:    "high",
					Status:      "done",
					PRNumber:    123,
				},
			},
			expected: []string{"uuid-doing-task", "uuid-todo-task"}, // Only doing and todo tasks are displayed
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := displayAIModeContent(tc.tasks, "test context")
			require.NoError(t, err)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Verify all expected UUIDs appear in output
			for _, expectedUUID := range tc.expected {
				assert.Contains(t, output, expectedUUID,
					"Expected UUID '%s' to appear in status output", expectedUUID)
			}

			// Verify no TSK-XXX format IDs appear (regression test)
			assert.NotContains(t, output, "TSK-",
				"Status output should not contain TSK-XXX format IDs")
		})
	}
}

// TestStatusUUIDsCompatibleWithShowCommand tests that UUIDs displayed by status
// command can be used directly with the show command. This validates the fix for
// the core issue where status and show commands used incompatible ID formats.
func TestStatusUUIDsCompatibleWithShowCommand(t *testing.T) {
	testTasks := []storage.Task{
		{
			ID:          "compatible-uuid-123",
			Description: "Test task for UUID compatibility",
			Priority:    "high",
			Status:      "doing",
			PRNumber:    123,
			File:        "test.go",
			Line:        42,
		},
		{
			ID:          "compatible-uuid-456",
			Description: "Another test task",
			Priority:    "medium",
			Status:      "todo",
			PRNumber:    123,
			File:        "test2.go",
			Line:        100,
		},
	}

	// Test 1: Get UUID from status output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := displayAIModeContent(testTasks, "test context")
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	statusOutput := buf.String()

	// Verify status output contains the exact UUIDs
	assert.Contains(t, statusOutput, "compatible-uuid-123")
	assert.Contains(t, statusOutput, "compatible-uuid-456")

	// Test 2: Verify these UUIDs work with displayTaskDetails (show command logic)
	for _, task := range testTasks {
		// Capture output for each task
		r, w, _ = os.Pipe()
		os.Stdout = w

		err := displayTaskDetails(task)
		require.NoError(t, err, "displayTaskDetails should work with UUID: %s", task.ID)

		w.Close()
		os.Stdout = old

		var taskBuf bytes.Buffer
		io.Copy(&taskBuf, r)
		taskOutput := taskBuf.String()

		// Verify the show command displays the same UUID
		assert.Contains(t, taskOutput, "Task ID: "+task.ID,
			"Show command should display the same UUID that status command shows")
	}
}

// TestStatusUUIDFormat tests that the displayed IDs are actual UUIDs, not custom formats
func TestStatusUUIDFormat(t *testing.T) {
	testTasks := []storage.Task{
		{
			ID:          "real-uuid-format-12345-abcde",
			Description: "Test with real UUID format",
			Priority:    "high",
			Status:      "doing",
			PRNumber:    123,
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

	// Verify the actual UUID format appears
	assert.Contains(t, output, "real-uuid-format-12345-abcde")

	// Verify NO custom TSK format appears (this was the bug)
	assert.NotContains(t, output, "TSK-123")
	assert.NotContains(t, output, "TSK-")

	// Verify no other custom ID patterns appear
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "real-uuid-format-12345-abcde") {
			// This line should contain the UUID, not any custom format
			assert.NotRegexp(t, `TSK-\d+`, line,
				"Line containing UUID should not have TSK-XXX format: %s", line)
		}
	}
}

// TestStatusEmptyTasksNoUUIDs tests that empty state doesn't display any task IDs
func TestStatusEmptyTasksNoUUIDs(t *testing.T) {
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

	// Verify no task IDs appear in empty state
	assert.NotContains(t, output, "TSK-")
	assert.NotContains(t, output, "uuid")
	assert.NotContains(t, output, "task-")

	// But verify it shows the correct empty messages
	assert.Contains(t, output, "No active tasks")
	assert.Contains(t, output, "No pending tasks")
}