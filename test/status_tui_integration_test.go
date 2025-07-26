package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatusCommandIntegration tests the status command functionality in both modes
func TestStatusCommandIntegration(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "1" {
		t.Skip("Skipping integration tests")
	}

	// Create a temporary test directory
	testDir, err := os.MkdirTemp("", "reviewtask-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Set up git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create test PR data
	prDir := filepath.Join(testDir, ".pr-review", "PR-123")
	err = os.MkdirAll(prDir, 0755)
	require.NoError(t, err)

	// Create test tasks
	tasksContent := `{
		"generated_at": "2024-01-01T10:00:00Z",
		"tasks": [
			{
				"id": "task1",
				"description": "認証トークンの検証処理を修正",
				"priority": "high",
				"status": "doing",
				"pr_number": 123,
				"file": "auth.go",
				"line": 45
			},
			{
				"id": "task2",
				"description": "APIドキュメントの更新",
				"priority": "medium",
				"status": "todo",
				"pr_number": 123,
				"file": "README.md",
				"line": 10
			},
			{
				"id": "task3",
				"description": "ユニットテストを追加",
				"priority": "high",
				"status": "todo",
				"pr_number": 123,
				"file": "test.go",
				"line": 100
			},
			{
				"id": "task4",
				"description": "データベース層のリファクタリング",
				"priority": "low",
				"status": "done",
				"pr_number": 123,
				"file": "db.go",
				"line": 200
			},
			{
				"id": "task5",
				"description": "非推奨APIを削除",
				"priority": "medium",
				"status": "cancel",
				"pr_number": 123,
				"file": "api.go",
				"line": 150
			}
		]
	}`

	tasksFile := filepath.Join(prDir, "tasks.json")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	require.NoError(t, err)

	// Create branch mapping
	branchFile := filepath.Join(testDir, ".pr-review", "branches", "feature-test")
	err = os.MkdirAll(filepath.Dir(branchFile), 0755)
	require.NoError(t, err)
	err = os.WriteFile(branchFile, []byte("123"), 0644)
	require.NoError(t, err)

	// Build the reviewtask binary with proper extension for Windows
	binaryName := "reviewtask"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	// Build from the module root directory
	moduleRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(testDir, binaryName))
	buildCmd.Dir = moduleRoot
	err = buildCmd.Run()
	require.NoError(t, err)

	reviewtaskPath := filepath.Join(testDir, binaryName)

	t.Run("AI Mode Output", func(t *testing.T) {
		cmd := exec.Command(reviewtaskPath, "status", "--pr", "123")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", output)

		outputStr := string(output)

		// Check AI mode format
		assert.Contains(t, outputStr, "ReviewTask Status - 40.0% Complete (2/5)")
		assert.Contains(t, outputStr, "Progress:")
		assert.Contains(t, outputStr, "█") // Filled progress
		assert.Contains(t, outputStr, "░") // Empty progress

		// Check task summary
		assert.Contains(t, outputStr, "Task Summary:")
		assert.Contains(t, outputStr, "todo: 2    doing: 1    done: 1    pending: 0    cancel: 1")

		// Check current task
		assert.Contains(t, outputStr, "Current Task:")
		assert.Contains(t, outputStr, "task1") // Use actual task ID instead of TSK-123
		assert.Contains(t, outputStr, "HIGH")
		assert.Contains(t, outputStr, "認証トークンの検証処理を修正")

		// Check next tasks
		assert.Contains(t, outputStr, "Next Tasks (up to 5):")
		assert.Contains(t, outputStr, "1. task3  HIGH    ユニットテストを追加")     // Use actual task ID
		assert.Contains(t, outputStr, "2. task2  MEDIUM    APIドキュメントの更新") // Use actual task ID

		// Check timestamp
		assert.Contains(t, outputStr, "Last updated:")
	})

	t.Run("Japanese Character Display", func(t *testing.T) {
		cmd := exec.Command(reviewtaskPath, "status", "--pr", "123")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)

		// Verify Japanese text is properly displayed
		assert.Contains(t, outputStr, "認証トークンの検証処理を修正")
		assert.Contains(t, outputStr, "APIドキュメントの更新")
		assert.Contains(t, outputStr, "ユニットテストを追加")
		// Note: done and cancel status tasks are not displayed in the main output
	})

	t.Run("Empty State Display", func(t *testing.T) {
		// Create empty PR
		emptyPRDir := filepath.Join(testDir, ".pr-review", "PR-999")
		err = os.MkdirAll(emptyPRDir, 0755)
		require.NoError(t, err)

		emptyTasksContent := `{
			"generated_at": "2024-01-01T10:00:00Z",
			"tasks": []
		}`

		emptyTasksFile := filepath.Join(emptyPRDir, "tasks.json")
		err = os.WriteFile(emptyTasksFile, []byte(emptyTasksContent), 0644)
		require.NoError(t, err)

		cmd := exec.Command(reviewtaskPath, "status", "--pr", "999")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)

		// Check empty state
		assert.Contains(t, outputStr, "ReviewTask Status - 0% Complete")
		assert.Contains(t, outputStr, strings.Repeat("░", 80))
		assert.Contains(t, outputStr, "todo: 0    doing: 0    done: 0    pending: 0    cancel: 0")
		assert.Contains(t, outputStr, "No active tasks - all completed!")
		assert.Contains(t, outputStr, "No pending tasks")
	})

	t.Run("Watch Flag Recognition", func(t *testing.T) {
		// Test that --watch flag is recognized (even though TUI won't work in test env)
		cmd := exec.Command(reviewtaskPath, "status", "--help")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "--watch")
		assert.Contains(t, outputStr, "-w")
		assert.Contains(t, outputStr, "Human mode: rich TUI dashboard with real-time updates")
	})

	t.Run("Priority Sorting", func(t *testing.T) {
		// Create PR with mixed priority tasks
		mixedPRDir := filepath.Join(testDir, ".pr-review", "PR-456")
		err = os.MkdirAll(mixedPRDir, 0755)
		require.NoError(t, err)

		mixedTasksContent := `{
			"generated_at": "2024-01-01T10:00:00Z",
			"tasks": [
				{
					"id": "task1",
					"description": "Low priority task",
					"priority": "low",
					"status": "todo",
					"pr_number": 456
				},
				{
					"id": "task2",
					"description": "Critical task",
					"priority": "critical",
					"status": "todo",
					"pr_number": 456
				},
				{
					"id": "task3",
					"description": "Medium task",
					"priority": "medium",
					"status": "todo",
					"pr_number": 456
				},
				{
					"id": "task4",
					"description": "High priority task",
					"priority": "high",
					"status": "todo",
					"pr_number": 456
				}
			]
		}`

		mixedTasksFile := filepath.Join(mixedPRDir, "tasks.json")
		err = os.WriteFile(mixedTasksFile, []byte(mixedTasksContent), 0644)
		require.NoError(t, err)

		cmd := exec.Command(reviewtaskPath, "status", "--pr", "456")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)

		// Check tasks are sorted by priority
		lines := strings.Split(outputStr, "\n")
		var taskLines []string
		inNextTasks := false
		for _, line := range lines {
			if strings.Contains(line, "Next Tasks") {
				inNextTasks = true
				continue
			}
			if inNextTasks && strings.TrimSpace(line) != "" && (strings.Contains(line, "task") || strings.Contains(line, ".")) {
				// Look for lines that contain task information (task IDs or numbered list items)
				if strings.Contains(line, "CRITICAL") || strings.Contains(line, "HIGH") ||
					strings.Contains(line, "MEDIUM") || strings.Contains(line, "LOW") {
					taskLines = append(taskLines, line)
				}
			}
			if inNextTasks && strings.TrimSpace(line) == "" {
				break
			}
		}

		// Verify order: critical > high > medium > low
		require.Len(t, taskLines, 4)
		assert.Contains(t, taskLines[0], "CRITICAL")
		assert.Contains(t, taskLines[1], "HIGH")
		assert.Contains(t, taskLines[2], "MEDIUM")
		assert.Contains(t, taskLines[3], "LOW")
	})
}

// TestCharacterWidthCalculation tests the character width calculation for CJK support
func TestCharacterWidthCalculation(t *testing.T) {
	testCases := []struct {
		text          string
		expectedWidth int
		description   string
	}{
		{"Hello World", 11, "ASCII characters"},
		{"認証トークン", 12, "Japanese characters (6 chars × 2 width)"},
		{"Fix 認証 token", 14, "Mixed ASCII and Japanese"},
		{"データベース層のリファクタリング", 32, "Long Japanese text"},
		{"API更新", 7, "Short mixed text"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actualWidth := runewidth.StringWidth(tc.text)
			assert.Equal(t, tc.expectedWidth, actualWidth, "Width calculation mismatch for: %s", tc.text)
		})
	}
}

// TestProgressBarColorTerminalCompatibility tests progress bar colors in different terminal environments
func TestProgressBarColorTerminalCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping color terminal integration test in short mode")
	}

	if os.Getenv("SKIP_INTEGRATION_TESTS") == "1" {
		t.Skip("Skipping integration tests")
	}

	// Set up test environment (reuse the same setup as other integration tests)
	testDir, err := os.MkdirTemp("", "reviewtask-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Set up git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = testDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create test PR data
	prDir := filepath.Join(testDir, ".pr-review", "PR-123")
	err = os.MkdirAll(prDir, 0755)
	require.NoError(t, err)

	// Create tasks.json with sample data
	tasksContent := `{
		"tasks": [
			{
				"id": "task1",
				"description": "Fix authentication bug",
				"priority": "high",
				"status": "done",
				"pr_number": 123,
				"file": "auth.go",
				"line": 45
			},
			{
				"id": "task2",
				"description": "Update documentation",
				"priority": "medium",
				"status": "todo",
				"pr_number": 123,
				"file": "README.md",
				"line": 1
			}
		]
	}`

	tasksFile := filepath.Join(prDir, "tasks.json")
	err = os.WriteFile(tasksFile, []byte(tasksContent), 0644)
	require.NoError(t, err)

	// Build binary
	binaryName := "reviewtask"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	moduleRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(testDir, binaryName))
	buildCmd.Dir = moduleRoot
	err = buildCmd.Run()
	require.NoError(t, err)

	reviewtaskPath := filepath.Join(testDir, binaryName)

	testCases := []struct {
		name    string
		env     map[string]string
		cmdArgs []string
	}{
		{
			name: "Standard terminal with color support",
			env: map[string]string{
				"TERM":        "xterm-256color",
				"FORCE_COLOR": "1",
			},
			cmdArgs: []string{"status", "--pr", "123"},
		},
		{
			name: "No color terminal",
			env: map[string]string{
				"TERM":     "dumb",
				"NO_COLOR": "1",
			},
			cmdArgs: []string{"status", "--pr", "123"},
		},
		{
			name: "Basic terminal",
			env: map[string]string{
				"TERM": "xterm",
			},
			cmdArgs: []string{"status", "--pr", "123"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(reviewtaskPath, tc.cmdArgs...)
			cmd.Dir = testDir

			// Set environment variables
			cmd.Env = os.Environ()
			for key, value := range tc.env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
			}

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Command should succeed in terminal environment: %s", tc.name)

			outputStr := string(output)

			// All environments should show progress bar with proper characters
			assert.Contains(t, outputStr, "Progress:", "Progress bar should be present")

			// Should contain either filled blocks or empty blocks (or both)
			hasFilledBlocks := strings.Contains(outputStr, "█")
			hasEmptyBlocks := strings.Contains(outputStr, "░")
			assert.True(t, hasFilledBlocks || hasEmptyBlocks, "Progress bar should contain progress characters")

			// Basic functionality should work regardless of color support
			assert.Contains(t, outputStr, "Task Summary:")
			assert.Contains(t, outputStr, "ReviewTask Status")

			// Should not crash or produce empty output
			assert.NotEmpty(t, outputStr, "Output should not be empty")

			// Should contain meaningful content regardless of color support
			assert.Contains(t, outputStr, "todo:")
			assert.Contains(t, outputStr, "doing:")
			assert.Contains(t, outputStr, "done:")
		})
	}
}
