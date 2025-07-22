package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/storage"
)

// TestCancelTaskIntegration tests the full workflow of cancelling tasks
func TestCancelTaskIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Build the binary
	// Find project root by looking for go.mod
	// Start from the executable's directory rather than working directory
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	projectRoot := filepath.Dir(testDir) // go up from test/ to project root

	// Windows requires .exe extension
	execName := "reviewtask"
	if runtime.GOOS == "windows" {
		execName = "reviewtask.exe"
	}

	// Build executable path
	execPath := filepath.Join(tempDir, execName)
	buildCmd := exec.Command("go", "build", "-o", execPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Prepare the command to run the executable
	execCmd := "./" + execName
	if runtime.GOOS == "windows" {
		execCmd = ".\\" + execName
	}

	// Initialize git repository
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = tempDir
	if output, err := gitInitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git repo: %v\nOutput: %s", err, output)
	}

	// Configure git user
	gitConfigUserCmd := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigUserCmd.Dir = tempDir
	if output, err := gitConfigUserCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.email: %v\nOutput: %s", err, output)
	}

	gitConfigNameCmd := exec.Command("git", "config", "user.name", "Test User")
	gitConfigNameCmd.Dir = tempDir
	if output, err := gitConfigNameCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.name: %v\nOutput: %s", err, output)
	}

	// Create and checkout a branch
	gitCheckoutCmd := exec.Command("git", "checkout", "-b", "test-branch")
	gitCheckoutCmd.Dir = tempDir
	if output, err := gitCheckoutCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to checkout branch: %v\nOutput: %s", err, output)
	}

	// Create test data
	prDir := filepath.Join(".pr-review", "PR-1")
	require.NoError(t, os.MkdirAll(prDir, 0755))

	// Create PR info file to link branch with PR
	prInfo := map[string]interface{}{
		"pr_number": 1,
		"title":     "Test PR",
		"author":    "testuser",
		"branch":    "test-branch",
		"state":     "open",
	}
	infoData, err := json.MarshalIndent(prInfo, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(prDir, "info.json"), infoData, 0644))

	// Create test tasks
	tasks := []storage.Task{
		{
			ID:              "task-1",
			Description:     "Test task 1",
			Status:          "todo",
			Priority:        "high",
			SourceCommentID: 100,
			PRNumber:        1,
		},
		{
			ID:              "task-2",
			Description:     "Test task 2",
			Status:          "doing",
			Priority:        "medium",
			SourceCommentID: 100,
			PRNumber:        1,
		},
		{
			ID:              "task-3",
			Description:     "Test task 3",
			Status:          "done",
			Priority:        "low",
			SourceCommentID: 100,
			PRNumber:        1,
		},
	}

	// Save tasks
	tasksData := storage.TasksFile{
		GeneratedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		Tasks:       tasks,
	}
	tasksFile := filepath.Join(prDir, "tasks.json")
	data, err := json.MarshalIndent(tasksData, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tasksFile, data, 0644))

	t.Run("update_task_to_cancel_status", func(t *testing.T) {
		// Update task-1 to cancel status
		cmd := exec.Command(execCmd, "update", "task-1", "cancel")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", output)

		// Verify the task was updated
		assert.Contains(t, string(output), "✓ Updated task 'task-1' status to 'cancel'")

		// Load tasks and verify status
		updatedData, err := os.ReadFile(tasksFile)
		require.NoError(t, err)

		var updatedTasksFile storage.TasksFile
		require.NoError(t, json.Unmarshal(updatedData, &updatedTasksFile))

		// Find task-1
		var task1 *storage.Task
		for i := range updatedTasksFile.Tasks {
			if updatedTasksFile.Tasks[i].ID == "task-1" {
				task1 = &updatedTasksFile.Tasks[i]
				break
			}
		}

		require.NotNil(t, task1, "Task-1 not found")
		assert.Equal(t, "cancel", task1.Status, "Task should be marked as 'cancel'")
	})

	t.Run("status_command_shows_cancelled_tasks", func(t *testing.T) {
		// Run status command
		cmd := exec.Command(execCmd, "status")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", output)

		outputStr := string(output)

		// Should show cancel: 1 in status breakdown
		assert.Contains(t, outputStr, "cancel: 1", "Status should show 1 cancelled task")

		// Completion rate should include cancelled tasks
		// We have: 1 done, 1 cancel, 1 doing = 2/3 completed
		assert.Contains(t, outputStr, "66.7%", "Completion rate should be 66.7%")
		assert.Contains(t, outputStr, "(2/3)", "Should show 2 out of 3 tasks completed")
	})

	t.Run("show_command_skips_cancelled_tasks", func(t *testing.T) {
		// First cancel the doing task
		cmd := exec.Command(execCmd, "update", "task-2", "cancel")
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Run show command
		cmd = exec.Command("./reviewtask", "show")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", output)

		outputStr := string(output)

		// Should not show cancelled tasks
		assert.NotContains(t, outputStr, "task-1", "Should not show cancelled task-1")
		assert.NotContains(t, outputStr, "task-2", "Should not show cancelled task-2")

		// Should indicate all tasks are completed or cancelled
		assert.Contains(t, outputStr, "No current or next tasks found", "Should indicate no actionable tasks")
	})
}

// TestBackwardCompatibilityIntegration tests handling of old "cancelled" status
func TestBackwardCompatibilityIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Build the binary
	// Find project root by looking for go.mod
	// Start from the executable's directory rather than working directory
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	projectRoot := filepath.Dir(testDir) // go up from test/ to project root

	// Windows requires .exe extension
	execName := "reviewtask"
	if runtime.GOOS == "windows" {
		execName = "reviewtask.exe"
	}

	// Build executable path
	execPath := filepath.Join(tempDir, execName)
	buildCmd := exec.Command("go", "build", "-o", execPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Prepare the command to run the executable
	execCmd := "./" + execName
	if runtime.GOOS == "windows" {
		execCmd = ".\\" + execName
	}

	// Initialize git repository
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = tempDir
	if output, err := gitInitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git repo: %v\nOutput: %s", err, output)
	}

	// Configure git user
	gitConfigUserCmd := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigUserCmd.Dir = tempDir
	if output, err := gitConfigUserCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.email: %v\nOutput: %s", err, output)
	}

	gitConfigNameCmd := exec.Command("git", "config", "user.name", "Test User")
	gitConfigNameCmd.Dir = tempDir
	if output, err := gitConfigNameCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.name: %v\nOutput: %s", err, output)
	}

	// Create and checkout a branch
	gitCheckoutCmd := exec.Command("git", "checkout", "-b", "test-branch")
	gitCheckoutCmd.Dir = tempDir
	if output, err := gitCheckoutCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to checkout branch: %v\nOutput: %s", err, output)
	}

	// Create test data with old "cancelled" status
	prDir := filepath.Join(".pr-review", "PR-1")
	require.NoError(t, os.MkdirAll(prDir, 0755))

	// Create PR info file to link branch with PR
	prInfo := map[string]interface{}{
		"pr_number": 1,
		"title":     "Test PR",
		"author":    "testuser",
		"branch":    "test-branch",
		"state":     "open",
	}
	infoData, err := json.MarshalIndent(prInfo, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(prDir, "info.json"), infoData, 0644))

	// Create tasks with mixed cancel/cancelled statuses
	tasksData := storage.TasksFile{
		GeneratedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		Tasks: []storage.Task{
			{
				ID:       "task-1",
				Status:   "cancelled", // Old format
				Priority: "high",
				PRNumber: 1,
			},
			{
				ID:       "task-2",
				Status:   "cancel", // New format
				Priority: "medium",
				PRNumber: 1,
			},
			{
				ID:       "task-3",
				Status:   "todo",
				Priority: "low",
				PRNumber: 1,
			},
		},
	}

	tasksFile := filepath.Join(prDir, "tasks.json")
	data, err := json.MarshalIndent(tasksData, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tasksFile, data, 0644))

	// Run status command
	cmd := exec.Command(execCmd, "status")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed: %s", output)

	outputStr := string(output)

	// Should count both "cancel" and "cancelled" statuses
	assert.Contains(t, outputStr, "cancel: 2", "Should count both cancel and cancelled statuses")

	// Completion rate should be 66.7% (2 cancelled + 0 done) / 3 total
	assert.Contains(t, outputStr, "66.7%", "Completion rate should be 66.7%")
}

// TestMergeWithCancelledTasks tests the merge behavior when comments are deleted
func TestMergeWithCancelledTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Initialize git repository
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = tempDir
	if output, err := gitInitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git repo: %v\nOutput: %s", err, output)
	}

	// Configure git user
	gitConfigUserCmd := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigUserCmd.Dir = tempDir
	if output, err := gitConfigUserCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.email: %v\nOutput: %s", err, output)
	}

	gitConfigNameCmd := exec.Command("git", "config", "user.name", "Test User")
	gitConfigNameCmd.Dir = tempDir
	if output, err := gitConfigNameCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.name: %v\nOutput: %s", err, output)
	}

	// Create and checkout a branch
	gitCheckoutCmd := exec.Command("git", "checkout", "-b", "test-branch")
	gitCheckoutCmd.Dir = tempDir
	if output, err := gitCheckoutCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to checkout branch: %v\nOutput: %s", err, output)
	}

	// Create test data
	prDir := filepath.Join(".pr-review", "PR-1")
	require.NoError(t, os.MkdirAll(prDir, 0755))

	// Simulate initial tasks from a comment
	initialTasks := []storage.Task{
		{
			ID:              "task-1",
			Description:     "Fix the bug",
			Status:          "todo",
			Priority:        "high",
			SourceCommentID: 12345,
			PRNumber:        1,
			OriginText:      "Please fix this bug",
		},
		{
			ID:              "task-2",
			Description:     "Add tests",
			Status:          "doing",
			Priority:        "medium",
			SourceCommentID: 12345,
			PRNumber:        1,
			OriginText:      "Please fix this bug",
		},
	}

	// Save initial tasks
	sm := storage.NewManager()
	err := sm.SaveTasks(1, initialTasks)
	require.NoError(t, err)

	// Simulate comment deletion by merging with empty tasks
	err = sm.MergeTasks(1, []storage.Task{})
	require.NoError(t, err)

	// Load tasks and verify they're cancelled
	tasks, err := sm.GetTasksByPR(1)
	require.NoError(t, err)

	assert.Len(t, tasks, 2, "Should still have 2 tasks")

	for _, task := range tasks {
		assert.Equal(t, "cancel", task.Status, "All tasks should be marked as 'cancel'")
		assert.NotEmpty(t, task.UpdatedAt, "UpdatedAt should be set")
	}

	// Also verify via file content
	tasksFile := filepath.Join(prDir, "tasks.json")
	data, err := os.ReadFile(tasksFile)
	require.NoError(t, err)

	// Should contain "cancel" not "cancelled"
	assert.Contains(t, string(data), `"status": "cancel"`, "File should contain 'cancel' status")
	assert.NotContains(t, string(data), `"status": "cancelled"`, "File should not contain 'cancelled' status")
}

// TestShowCommandPriority tests that show command correctly prioritizes non-cancelled tasks
func TestShowCommandPriority(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// Build the binary
	// Find project root by looking for go.mod
	// Start from the executable's directory rather than working directory
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	projectRoot := filepath.Dir(testDir) // go up from test/ to project root

	// Windows requires .exe extension
	execName := "reviewtask"
	if runtime.GOOS == "windows" {
		execName = "reviewtask.exe"
	}

	// Build executable path
	execPath := filepath.Join(tempDir, execName)
	buildCmd := exec.Command("go", "build", "-o", execPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Prepare the command to run the executable
	execCmd := "./" + execName
	if runtime.GOOS == "windows" {
		execCmd = ".\\" + execName
	}

	// Initialize git repository
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = tempDir
	if output, err := gitInitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git repo: %v\nOutput: %s", err, output)
	}

	// Configure git user
	gitConfigUserCmd := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigUserCmd.Dir = tempDir
	if output, err := gitConfigUserCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.email: %v\nOutput: %s", err, output)
	}

	gitConfigNameCmd := exec.Command("git", "config", "user.name", "Test User")
	gitConfigNameCmd.Dir = tempDir
	if output, err := gitConfigNameCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to config git user.name: %v\nOutput: %s", err, output)
	}

	// Create and checkout a branch
	gitCheckoutCmd := exec.Command("git", "checkout", "-b", "test-branch")
	gitCheckoutCmd.Dir = tempDir
	if output, err := gitCheckoutCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to checkout branch: %v\nOutput: %s", err, output)
	}

	// Create test data
	prDir := filepath.Join(".pr-review", "PR-1")
	require.NoError(t, os.MkdirAll(prDir, 0755))

	// Create PR info file to link branch with PR
	prInfo := map[string]interface{}{
		"pr_number": 1,
		"title":     "Test PR",
		"author":    "testuser",
		"branch":    "test-branch",
		"state":     "open",
	}
	infoData, err := json.MarshalIndent(prInfo, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(prDir, "info.json"), infoData, 0644))

	// Create tasks with cancelled high priority and active medium priority
	tasksData := storage.TasksFile{
		GeneratedAt: time.Now().Format("2006-01-02T15:04:05Z"),
		Tasks: []storage.Task{
			{
				ID:          "task-1",
				Description: "Cancelled high priority task",
				Status:      "cancel",
				Priority:    "high",
				PRNumber:    1,
			},
			{
				ID:          "task-2",
				Description: "Active medium priority task",
				Status:      "todo",
				Priority:    "medium",
				PRNumber:    1,
			},
			{
				ID:          "task-3",
				Description: "Cancelled critical task",
				Status:      "cancel",
				Priority:    "critical",
				PRNumber:    1,
			},
		},
	}

	tasksFile := filepath.Join(prDir, "tasks.json")
	data, err := json.MarshalIndent(tasksData, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tasksFile, data, 0644))

	// Run show command
	cmd := exec.Command(execCmd, "show")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed: %s", output)

	outputStr := string(output)

	// Should show the medium priority task (only non-cancelled task)
	assert.Contains(t, outputStr, "task-2", "Should show task-2")
	assert.Contains(t, outputStr, "Active medium priority task", "Should show task-2 description")

	// Should not show cancelled tasks
	assert.NotContains(t, outputStr, "task-1", "Should not show cancelled task-1")
	assert.NotContains(t, outputStr, "task-3", "Should not show cancelled task-3")
}
