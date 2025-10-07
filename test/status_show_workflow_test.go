package test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/cmd"
	"reviewtask/internal/storage"
)

// TestStatusShowWorkflowUnitTest tests the core workflow without building binaries.
// This validates that the status command displays UUIDs that work with show command.
func TestStatusShowWorkflowUnitTest(t *testing.T) {
	// Test data: realistic tasks with UUIDs
	testTasks := []storage.Task{
		{
			ID:              "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Description:     "Fix authentication vulnerability in login handler",
			OriginText:      "The login handler is vulnerable to timing attacks. Please implement constant-time comparison.",
			Priority:        "critical",
			Status:          "doing",
			SourceReviewID:  12345,
			SourceCommentID: 67890,
			File:            "auth/login.go",
			Line:            42,
			PRNumber:        123,
			CreatedAt:       "2023-12-01T10:00:00Z",
			UpdatedAt:       "2023-12-01T11:00:00Z",
		},
		{
			ID:              "b2c3d4e5-f6g7-8901-bcde-f12345678901",
			Description:     "Add unit tests for user validation",
			OriginText:      "Please add comprehensive unit tests for the user validation logic to ensure edge cases are covered.",
			Priority:        "high",
			Status:          "todo",
			SourceReviewID:  12345,
			SourceCommentID: 67891,
			File:            "user/validation.go",
			Line:            15,
			PRNumber:        123,
			CreatedAt:       "2023-12-01T10:00:00Z",
			UpdatedAt:       "2023-12-01T10:00:00Z",
		},
		{
			ID:              "c3d4e5f6-g7h8-9012-cdef-123456789012",
			Description:     "Update API documentation for new endpoints",
			OriginText:      "The new endpoints need to be documented in the API specification.",
			Priority:        "medium",
			Status:          "todo",
			SourceReviewID:  12345,
			SourceCommentID: 67892,
			File:            "docs/api.md",
			Line:            100,
			PRNumber:        123,
			CreatedAt:       "2023-12-01T10:00:00Z",
			UpdatedAt:       "2023-12-01T10:00:00Z",
		},
	}

	// Test 1: Get output from status command display function
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Use the internal displayAIModeContent function directly
	err := cmd.DisplayAIModeContentForTest(testTasks, "test context")
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	statusOutput := buf.String()

	t.Logf("Status command output:\n%s", statusOutput)

	// Verify status output contains actual UUIDs
	assert.Contains(t, statusOutput, "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"Status should display the actual UUID for doing task")
	assert.Contains(t, statusOutput, "b2c3d4e5-f6g7-8901-bcde-f12345678901",
		"Status should display the actual UUID for high priority todo task")
	assert.Contains(t, statusOutput, "c3d4e5f6-g7h8-9012-cdef-123456789012",
		"Status should display the actual UUID for medium priority todo task")

	// Verify status output does NOT contain TSK-XXX format (the bug we fixed)
	assert.NotContains(t, statusOutput, "TSK-123",
		"Status should not display TSK-XXX format IDs")
	assert.NotContains(t, statusOutput, "TSK-",
		"Status should not display any TSK- prefix IDs")

	// Test 2: Verify these UUIDs work with show command display function
	for _, task := range testTasks {
		if task.Status == "doing" || task.Status == "todo" {
			// Capture output for each task's show display
			r, w, _ = os.Pipe()
			os.Stdout = w

			err := cmd.DisplayTaskDetailsForTest(task)
			require.NoError(t, err, "displayTaskDetails should work with UUID: %s", task.ID)

			w.Close()
			os.Stdout = old

			var taskBuf bytes.Buffer
			io.Copy(&taskBuf, r)
			taskOutput := taskBuf.String()

			// Verify the show command displays the same UUID (Modern UI format)
			assert.Contains(t, taskOutput, "ID: "+task.ID,
				"Show command should display the same UUID that status command shows")
			assert.Contains(t, taskOutput, task.Description,
				"Show command should display the correct task description")
		}
	}

	// Test 3: Verify priority ordering in status output
	lines := strings.Split(statusOutput, "\n")
	var taskLines []string

	// Find lines containing todo task UUIDs
	for _, line := range lines {
		if strings.Contains(line, "b2c3d4e5-f6g7-8901-bcde-f12345678901") ||
			strings.Contains(line, "c3d4e5f6-g7h8-9012-cdef-123456789012") {
			taskLines = append(taskLines, strings.TrimSpace(line))
		}
	}

	// Verify we found todo task lines and they're in priority order
	assert.Len(t, taskLines, 2, "Should display 2 todo tasks")

	// High priority should come before medium priority
	highPriorityLineIndex := -1
	mediumPriorityLineIndex := -1

	for i, line := range taskLines {
		if strings.Contains(line, "b2c3d4e5-f6g7-8901-bcde-f12345678901") { // high priority
			highPriorityLineIndex = i
		}
		if strings.Contains(line, "c3d4e5f6-g7h8-9012-cdef-123456789012") { // medium priority
			mediumPriorityLineIndex = i
		}
	}

	assert.True(t, highPriorityLineIndex < mediumPriorityLineIndex,
		"High priority task should appear before medium priority task in status output")
}

// TestStatusShowUUIDCompatibility tests that all displayed UUIDs are compatible
// with the show command, addressing the core issue from Issue #112.
func TestStatusShowUUIDCompatibility(t *testing.T) {
	testCases := []struct {
		name string
		task storage.Task
	}{
		{
			name: "Critical doing task with full UUID",
			task: storage.Task{
				ID:              "11111111-2222-3333-4444-555555555555",
				Description:     "Critical security fix",
				Priority:        "critical",
				Status:          "doing",
				PRNumber:        123,
				File:            "security.go",
				Line:            10,
				SourceReviewID:  1,
				SourceCommentID: 1,
			},
		},
		{
			name: "High priority todo task with UUID",
			task: storage.Task{
				ID:              "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				Description:     "High priority feature",
				Priority:        "high",
				Status:          "todo",
				PRNumber:        123,
				File:            "feature.go",
				Line:            20,
				SourceReviewID:  2,
				SourceCommentID: 2,
			},
		},
		{
			name: "Medium priority todo task with short UUID",
			task: storage.Task{
				ID:              "abc-123-def-456",
				Description:     "Medium priority improvement",
				Priority:        "medium",
				Status:          "todo",
				PRNumber:        123,
				File:            "improvement.go",
				Line:            30,
				SourceReviewID:  3,
				SourceCommentID: 3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that the task's UUID appears in status output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := cmd.DisplayAIModeContentForTest([]storage.Task{tc.task}, "test")
			require.NoError(t, err)

			w.Close()
			os.Stdout = old

			var statusBuf bytes.Buffer
			io.Copy(&statusBuf, r)
			statusOutput := statusBuf.String()

			// Verify the UUID appears in status output
			assert.Contains(t, statusOutput, tc.task.ID,
				"Status output should contain the task UUID: %s", tc.task.ID)

			// Test that the same UUID works with show command
			r, w, _ = os.Pipe()
			os.Stdout = w

			err = cmd.DisplayTaskDetailsForTest(tc.task)
			require.NoError(t, err)

			w.Close()
			os.Stdout = old

			var showBuf bytes.Buffer
			io.Copy(&showBuf, r)
			showOutput := showBuf.String()

			// Verify show command accepts and displays the UUID correctly (Modern UI format)
			assert.Contains(t, showOutput, "ID: "+tc.task.ID,
				"Show command should display the task UUID: %s", tc.task.ID)
			assert.Contains(t, showOutput, tc.task.Description,
				"Show command should display the task description")
			assert.Contains(t, showOutput, strings.ToUpper(tc.task.Priority),
				"Show command should display the task priority")
		})
	}
}

// TestNoTSKFormatRegression tests that the TSK-XXX format never appears in status output.
// This is a regression test for the bug fixed in Issue #112.
func TestNoTSKFormatRegression(t *testing.T) {
	// Create tasks with various PR numbers to test GenerateTaskID scenarios
	testTasks := []storage.Task{
		{ID: "uuid-1", Priority: "critical", Status: "doing", PRNumber: 1},
		{ID: "uuid-2", Priority: "high", Status: "todo", PRNumber: 42},
		{ID: "uuid-3", Priority: "medium", Status: "todo", PRNumber: 999},
		{ID: "uuid-4", Priority: "low", Status: "todo", PRNumber: 1234},
	}

	// Capture status output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.DisplayAIModeContentForTest(testTasks, "regression test")
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// These are the TSK formats that GenerateTaskID would have created (the bug)
	forbiddenFormats := []string{
		"TSK-001",  // PR 1
		"TSK-042",  // PR 42
		"TSK-999",  // PR 999
		"TSK-1234", // PR 1234
		"TSK-",     // Any TSK prefix
	}

	for _, forbidden := range forbiddenFormats {
		assert.NotContains(t, output, forbidden,
			"Status output must not contain TSK format: %s", forbidden)
	}

	// Verify the actual UUIDs DO appear
	expectedUUIDs := []string{"uuid-1", "uuid-2", "uuid-3", "uuid-4"}
	for _, uuid := range expectedUUIDs {
		assert.Contains(t, output, uuid,
			"Status output should contain actual UUID: %s", uuid)
	}
}
