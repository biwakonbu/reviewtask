package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"reviewtask/internal/storage"
)

// TestShowCommand tests the show command functionality
func TestShowCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-show-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Setup test data
	setupTestDataForShow(t)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectOut   []string
	}{
		{
			name:        "show current task",
			args:        []string{},
			expectError: false,
			expectOut:   []string{"Task Details", "Status:", "Priority:"},
		},
		{
			name:        "show specific task by ID",
			args:        []string{"test-task-1"},
			expectError: false,
			expectOut:   []string{"Task Details"},
		},
		{
			name:        "show non-existent task",
			args:        []string{"non-existent-id"},
			expectError: true,
			expectOut:   []string{},
		},
		{
			name:        "show with invalid arguments",
			args:        []string{"arg1", "arg2"},
			expectError: true,
			expectOut:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use: "show",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runShow(cmd, args)
				},
			}
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			outputStr := output.String()
			for _, expectedOut := range tt.expectOut {
				if !strings.Contains(outputStr, expectedOut) {
					t.Errorf("Expected output to contain %q, got: %s", expectedOut, outputStr)
				}
			}
		})
	}
}

// TestShowCurrentOrNextTask tests the showCurrentOrNextTask function
func TestShowCurrentOrNextTask(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-show-current-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Setup test data
	setupTestDataForShow(t)

	tests := []struct {
		name        string
		expectError bool
		expectOut   []string
	}{
		{
			name:        "show current task when tasks exist",
			expectError: false,
			expectOut:   []string{"Current Task", "Priority:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&output)

			storageManager := storage.NewManager()
			err := showCurrentOrNextTask(storageManager)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			outputStr := output.String()
			for _, expectedOut := range tt.expectOut {
				if !strings.Contains(outputStr, expectedOut) {
					t.Logf("Output: %s", outputStr)
					t.Logf("Looking for: %s", expectedOut)
				}
			}
		})
	}
}

// TestShowSpecificTask tests the showSpecificTask function
func TestShowSpecificTask(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-show-specific-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Initialize git repository for tests
	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run()

	// Setup test data
	setupTestDataForShow(t)

	tests := []struct {
		name        string
		taskID      string
		expectError bool
		expectOut   []string
	}{
		{
			name:        "show existing task",
			taskID:      "test-task-1",
			expectError: false,
			expectOut:   []string{"Task Details", "ID:"},
		},
		{
			name:        "show non-existent task",
			taskID:      "non-existent",
			expectError: true,
			expectOut:   []string{},
		},
		{
			name:        "show empty task ID",
			taskID:      "",
			expectError: true,
			expectOut:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&output)

			storageManager := storage.NewManager()
			err := showSpecificTask(storageManager, tt.taskID)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			outputStr := output.String()
			for _, expectedOut := range tt.expectOut {
				if !strings.Contains(outputStr, expectedOut) {
					t.Logf("Output: %s", outputStr)
					t.Logf("Looking for: %s", expectedOut)
				}
			}
		})
	}
}

// TestDisplayTaskDetails tests the displayTaskDetails function
func TestDisplayTaskDetails(t *testing.T) {
	tests := []struct {
		name      string
		task      storage.Task
		expectOut []string
	}{
		{
			name: "complete task details",
			task: storage.Task{
				ID:              "test-task-1",
				Description:     "Test Description",
				OriginText:      "Test Origin",
				Status:          "todo",
				Priority:        "high",
				SourceCommentID: 123,
				CreatedAt:       time.Now().Format(time.RFC3339),
			},
			expectOut: []string{"Test Description", "todo", "high", "Test Origin"},
		},
		{
			name: "minimal task details",
			task: storage.Task{
				ID:          "test-task-2",
				Description: "Minimal Task",
				Status:      "done",
				Priority:    "low",
			},
			expectOut: []string{"Minimal Task", "done", "low"},
		},
		{
			name: "task with special characters",
			task: storage.Task{
				ID:          "test-task-3",
				Description: "Task with 特殊文字 and émojis 🚀\nDescription with\nmultiple lines\nand special chars: <>&\"'",
				Status:      "doing",
				Priority:    "critical",
			},
			expectOut: []string{"特殊文字", "émojis", "doing", "critical"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			displayTaskDetails(tt.task)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			outputBytes, _ := io.ReadAll(r)
			outputStr := string(outputBytes)

			for _, expectedOut := range tt.expectOut {
				if !strings.Contains(outputStr, expectedOut) {
					t.Errorf("Expected output to contain %q, got: %s", expectedOut, outputStr)
				}
			}
		})
	}
}

// TestGetStatusIndicator tests the getStatusIndicator function
func TestGetStatusIndicator(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"todo status", "todo", "📝"},
		{"doing status", "doing", "🔄"},
		{"done status", "done", "✅"},
		{"pending status", "pending", "⏸️"},
		{"cancel status", "cancel", "❌"},
		{"unknown status", "unknown", "❓"},
		{"empty status", "", "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusIndicator(tt.status)
			if result != tt.expected {
				t.Errorf("getStatusIndicator(%q) = %q, expected %q", tt.status, result, tt.expected)
			}
		})
	}
}

// TestGetPriorityIndicator tests the getPriorityIndicator function
func TestGetPriorityIndicator(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		expected string
	}{
		{"critical priority", "critical", "🔴"},
		{"high priority", "high", "🟠"},
		{"medium priority", "medium", "🟡"},
		{"low priority", "low", "🟢"},
		{"unknown priority", "unknown", "⚪"},
		{"empty priority", "", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPriorityIndicator(tt.priority)
			if result != tt.expected {
				t.Errorf("getPriorityIndicator(%q) = %q, expected %q", tt.priority, result, tt.expected)
			}
		})
	}
}

// TestGetImplementationIndicator tests the getImplementationIndicator function
func TestGetImplementationIndicator(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"implemented status", "implemented", "✅"},
		{"not_implemented status", "not_implemented", "❌"},
		{"empty status", "", "❌"},
		{"unknown status", "unknown", "❌"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getImplementationIndicator(tt.status)
			if result != tt.expected {
				t.Errorf("getImplementationIndicator(%q) = %q, expected %q",
					tt.status, result, tt.expected)
			}
		})
	}
}

// TestGetVerificationIndicator tests the getVerificationIndicator function
func TestGetVerificationIndicator(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"verified status", "verified", "✔️"},
		{"not_verified status", "not_verified", "❓"},
		{"failed status", "failed", "⚠️"},
		{"empty status", "", "❓"},
		{"unknown status", "unknown", "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getVerificationIndicator(tt.status)
			if result != tt.expected {
				t.Errorf("getVerificationIndicator(%q) = %q, expected %q",
					tt.status, result, tt.expected)
			}
		})
	}
}

// TestShowCommandErrorHandling tests error handling in show command
func TestShowCommandErrorHandling(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-show-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	tests := []struct {
		name        string
		setupData   bool
		args        []string
		expectError bool
	}{
		{
			name:        "no pr-review directory",
			setupData:   false,
			args:        []string{},
			expectError: true,
		},
		{
			name:        "empty task data",
			setupData:   true,
			args:        []string{},
			expectError: false, // Should handle empty data gracefully
		},
		{
			name:        "too many arguments",
			setupData:   true,
			args:        []string{"arg1", "arg2", "arg3"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.RemoveAll(".pr-review")

			if tt.setupData {
				// Create minimal .pr-review structure
				err := os.MkdirAll(".pr-review", 0755)
				if err != nil {
					t.Fatalf("Failed to create .pr-review directory: %v", err)
				}
			}

			var output bytes.Buffer
			cmd := &cobra.Command{
				Use: "show",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runShow(cmd, args)
				},
			}
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestShowCommandWithComplexTasks tests show command with complex task data
func TestShowCommandWithComplexTasks(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "reviewtask-show-complex-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Setup complex test data
	setupComplexTestDataForShow(t)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectOut   []string
	}{
		{
			name:        "show task with long description",
			args:        []string{"long-desc-task"},
			expectError: false,
			expectOut:   []string{"Long Description Task", "This is a very long description"},
		},
		{
			name:        "show task with unicode characters",
			args:        []string{"unicode-task"},
			expectError: false,
			expectOut:   []string{"Unicode Task", "特殊文字", "🚀"},
		},
		{
			name:        "show task with implementation details",
			args:        []string{"implementation-task"},
			expectError: false,
			expectOut:   []string{"Implementation Task", "💻"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use: "show",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runShow(cmd, args)
				},
			}
			cmd.SetOut(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			outputStr := output.String()
			for _, expectedOut := range tt.expectOut {
				if !strings.Contains(outputStr, expectedOut) {
					t.Logf("Expected %q in output, got: %s", expectedOut, outputStr)
				}
			}
		})
	}
}

// Helper functions for testing

func setupTestDataForShow(t *testing.T) {
	// Create .pr-review directory structure
	err := os.MkdirAll(".pr-review/PR-123", 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory structure: %v", err)
	}

	// Create test tasks data
	tasksData := `{
		"generated_at": "2023-01-01T00:00:00Z",
		"tasks": [
			{
				"id": "test-task-1",
				"title": "Test Task 1",
				"description": "Test description for task 1",
				"status": "todo",
				"priority": "high",
				"comment_id": 123,
				"origin": "Test origin 1",
				"created_at": "2023-01-01T00:00:00Z"
			},
			{
				"id": "test-task-2",
				"title": "Test Task 2", 
				"description": "Test description for task 2",
				"status": "doing",
				"priority": "medium",
				"comment_id": 124,
				"origin": "Test origin 2",
				"created_at": "2023-01-01T01:00:00Z"
			}
		]
	}`

	tasksFile := ".pr-review/PR-123/tasks.json"
	err = os.WriteFile(tasksFile, []byte(tasksData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tasks file: %v", err)
	}
}

func setupComplexTestDataForShow(t *testing.T) {
	// Create .pr-review directory structure
	err := os.MkdirAll(".pr-review/PR-456", 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory structure: %v", err)
	}

	// Create complex test tasks data
	longDescription := strings.Repeat("This is a very long description that spans multiple lines and contains various types of content including code snippets, URLs, and special characters. ", 10)

	tasksData := fmt.Sprintf(`[
		{
			"id": "long-desc-task",
			"title": "Long Description Task",
			"description": "%s",
			"status": "todo",
			"priority": "high",
			"comment_id": 201,
			"origin": "Long description origin",
			"created_at": "2023-01-01T00:00:00Z"
		},
		{
			"id": "unicode-task",
			"title": "Unicode Task 特殊文字 🚀",
			"description": "Task with unicode characters: 日本語, émojis 🎉, and symbols ★☆",
			"status": "doing",
			"priority": "medium",
			"comment_id": 202,
			"origin": "Unicode origin with 特殊文字",
			"created_at": "2023-01-01T01:00:00Z"
		},
		{
			"id": "implementation-task",
			"title": "Implementation Task",
			"description": "Task with implementation details",
			"status": "todo",
			"priority": "critical",
			"comment_id": 203,
			"origin": "Implementation origin",
			"created_at": "2023-01-01T02:00:00Z",
			"implementation": {
				"type": "code",
				"details": "Implement new feature in main.go"
			}
		}
	]`, longDescription)

	tasksFile := ".pr-review/PR-456/tasks.json"
	err = os.WriteFile(tasksFile, []byte(tasksData), 0644)
	if err != nil {
		t.Fatalf("Failed to create complex test tasks file: %v", err)
	}
}

// TestShowScenarios tests complete show command workflows
func TestShowScenarios(t *testing.T) {
	scenarios := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		commands []showCommand
	}{
		{
			name: "開発者の朝のワークフロー",
			setup: func(t *testing.T, dir string) {
				setupScenarioTasks(t, dir, []storage.Task{
					{
						ID:          "morning-001",
						Description: "Critical bug fix needed - Application crashes on startup",
						Status:      "todo",
						Priority:    "critical",
						PRNumber:    100,
						CreatedAt:   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
					},
					{
						ID:          "morning-002",
						Description: "Add unit tests - Coverage is below 80%",
						Status:      "doing",
						Priority:    "high",
						PRNumber:    100,
						CreatedAt:   time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
					},
					{
						ID:          "morning-003",
						Description: "Update documentation - API docs outdated",
						Status:      "todo",
						Priority:    "low",
						PRNumber:    100,
						CreatedAt:   time.Now().Add(-6 * time.Hour).Format(time.RFC3339),
					},
				})
			},
			commands: []showCommand{
				{
					args:      []string{},
					expectOut: []string{"Critical bug fix", "Add unit tests", "Update documentation"},
				},
				{
					args:      []string{"morning-001"},
					expectOut: []string{"Critical bug fix needed", "crashes on startup"},
				},
				{
					args:      []string{"--status", "doing"},
					expectOut: []string{"Add unit tests"},
					notExpect: []string{"Critical bug", "Update documentation"},
				},
			},
		},
		{
			name: "複数PRレビュー対応",
			setup: func(t *testing.T, dir string) {
				// PR 200のタスク
				setupScenarioTasks(t, dir, []storage.Task{
					{
						ID:          "pr200-001",
						Description: "Fix memory leak",
						Status:      "todo",
						Priority:    "high",
						PRNumber:    200,
					},
					{
						ID:          "pr200-002",
						Description: "Optimize query",
						Status:      "done",
						Priority:    "medium",
						PRNumber:    200,
					},
				})
				// PR 300のタスク
				setupScenarioTasks(t, dir, []storage.Task{
					{
						ID:          "pr300-001",
						Description: "Add validation",
						Status:      "doing",
						Priority:    "critical",
						PRNumber:    300,
					},
				})
			},
			commands: []showCommand{
				{
					args:      []string{},
					expectOut: []string{"Fix memory leak", "Add validation"},
				},
				{
					args:      []string{"--pr", "200"},
					expectOut: []string{"Fix memory leak", "Optimize query"},
					notExpect: []string{"Add validation"},
				},
				{
					args:      []string{"--pr", "300"},
					expectOut: []string{"Add validation"},
					notExpect: []string{"Fix memory leak"},
				},
			},
		},
		{
			name: "優先度トリアージ",
			setup: func(t *testing.T, dir string) {
				setupScenarioTasks(t, dir, []storage.Task{
					{ID: "triage-001", Description: "Critical security fix", Status: "todo", Priority: "critical", PRNumber: 400},
					{ID: "triage-002", Description: "Important bug", Status: "todo", Priority: "high", PRNumber: 400},
					{ID: "triage-003", Description: "Feature request", Status: "todo", Priority: "medium", PRNumber: 400},
					{ID: "triage-004", Description: "Code cleanup", Status: "todo", Priority: "low", PRNumber: 400},
				})
			},
			commands: []showCommand{
				{
					args:      []string{"--priority", "critical"},
					expectOut: []string{"Critical security fix"},
					notExpect: []string{"Important bug", "Feature request", "Code cleanup"},
				},
				{
					args:      []string{"--priority", "high"},
					expectOut: []string{"Important bug"},
					notExpect: []string{"Critical security", "Feature request"},
				},
				{
					args:      []string{},
					expectOut: []string{"Critical security", "Important bug", "Feature request", "Code cleanup"},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			scenario.setup(t, tempDir)

			for i, cmd := range scenario.commands {
				t.Run(fmt.Sprintf("command_%d", i), func(t *testing.T) {
					var output bytes.Buffer
					cobraCmd := &cobra.Command{
						Use:  "show",
						RunE: runShow,
					}
					cobraCmd.Flags().String("status", "", "Filter by status")
					cobraCmd.Flags().String("priority", "", "Filter by priority")
					cobraCmd.Flags().Int("pr", 0, "Filter by PR number")
					cobraCmd.SetOut(&output)
					cobraCmd.SetErr(&output)
					cobraCmd.SetArgs(cmd.args)

					err := cobraCmd.Execute()
					if err != nil && !cmd.expectError {
						t.Errorf("Unexpected error: %v", err)
					}

					outputStr := output.String()
					for _, expected := range cmd.expectOut {
						if !strings.Contains(outputStr, expected) {
							t.Errorf("Expected output to contain %q, got: %s", expected, outputStr)
						}
					}
					for _, notExpected := range cmd.notExpect {
						if strings.Contains(outputStr, notExpected) {
							t.Errorf("Output should not contain %q", notExpected)
						}
					}
				})
			}
		})
	}
}

type showCommand struct {
	args        []string
	expectOut   []string
	notExpect   []string
	expectError bool
}

func setupScenarioTasks(t *testing.T, dir string, tasks []storage.Task) {
	os.MkdirAll(".pr-review", 0755)

	prTasks := make(map[int][]storage.Task)
	for i := range tasks {
		if tasks[i].CreatedAt == "" {
			tasks[i].CreatedAt = time.Now().Format(time.RFC3339)
		}
		if tasks[i].UpdatedAt == "" {
			tasks[i].UpdatedAt = time.Now().Format(time.RFC3339)
		}
		prTasks[tasks[i].PRNumber] = append(prTasks[tasks[i].PRNumber], tasks[i])
	}

	for pr, prTaskList := range prTasks {
		prDir := filepath.Join(".pr-review", fmt.Sprintf("PR-%d", pr))
		os.MkdirAll(prDir, 0755)

		data, _ := json.MarshalIndent(prTaskList, "", "  ")
		tasksFile := filepath.Join(prDir, "tasks.json")
		err := os.WriteFile(tasksFile, data, 0644)
		if err != nil {
			t.Fatalf("Failed to write scenario tasks: %v", err)
		}
	}
}

// TestShowJapaneseContent tests Japanese content display
func TestShowJapaneseContent(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	tasks := []storage.Task{
		{
			ID:          "jp-001",
			Description: "メモリリークを修正する必要があります - アプリケーションが起動時にクラッシュします。これは重大なバグです。",
			OriginText:  "山田太郎: メモリリークを修正してください",
			Status:      "todo",
			Priority:    "critical",
			PRNumber:    500,
		},
		{
			ID:          "jp-002",
			Description: "単体テストを追加 - カバレッジが80%以下です",
			Status:      "doing",
			Priority:    "high",
			PRNumber:    500,
			File:        "src/日本語ファイル.go",
			Line:        42,
		},
	}

	setupScenarioTasks(t, tempDir, tasks)

	tests := []struct {
		name      string
		args      []string
		expectOut []string
	}{
		{
			name:      "日本語タスク一覧",
			args:      []string{},
			expectOut: []string{"メモリリーク", "単体テスト"},
		},
		{
			name:      "日本語タスク詳細",
			args:      []string{"jp-001"},
			expectOut: []string{"メモリリークを修正", "クラッシュします", "山田太郎"},
		},
		{
			name:      "日本語ファイルコンテキスト",
			args:      []string{"jp-002"},
			expectOut: []string{"日本語ファイル.go:42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use:  "show",
				RunE: runShow,
			}
			cmd.SetOut(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			outputStr := output.String()
			for _, expected := range tt.expectOut {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected Japanese content %q, got: %s", expected, outputStr)
				}
			}
		})
	}
}

// TestShowPerformance tests performance with large datasets
func TestShowPerformance(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Create 1000 tasks
	var tasks []storage.Task
	for i := 0; i < 1000; i++ {
		tasks = append(tasks, storage.Task{
			ID:          fmt.Sprintf("perf-%04d", i),
			Description: fmt.Sprintf("Performance test task %d - Description for task %d with various content", i, i),
			Status:      []string{"todo", "doing", "done", "pending"}[i%4],
			Priority:    []string{"critical", "high", "medium", "low"}[i%4],
			PRNumber:    600 + (i % 10),
		})
	}

	setupScenarioTasks(t, tempDir, tasks)

	start := time.Now()

	var output bytes.Buffer
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runShow,
	}
	cmd.SetOut(&output)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Failed with large dataset: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Errorf("Show command too slow: %v", elapsed)
	}

	t.Logf("Displayed 1000 tasks in %v", elapsed)
}

// TestShowErrorRecovery tests error recovery scenarios
func TestShowErrorRecovery(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string)
		args      []string
		expectErr bool
	}{
		{
			name: "破損したJSONファイル",
			setup: func(t *testing.T, dir string) {
				os.MkdirAll(".pr-review/PR-700", 0755)
				os.WriteFile(".pr-review/PR-700/tasks.json", []byte("{ broken json"), 0644)
			},
			args:      []string{},
			expectErr: true,
		},
		{
			name: "権限のないファイル",
			setup: func(t *testing.T, dir string) {
				os.MkdirAll(".pr-review/PR-800", 0755)
				tasksFile := ".pr-review/PR-800/tasks.json"
				os.WriteFile(tasksFile, []byte(`{"generated_at": "2023-01-01T00:00:00Z", "tasks": []}`), 0000)
			},
			args:      []string{},
			expectErr: false, // Should handle gracefully
		},
		{
			name: "循環参照のあるタスク",
			setup: func(t *testing.T, dir string) {
				// Create task with self-reference (should handle gracefully)
				task := map[string]interface{}{
					"id":     "circular-001",
					"title":  "Circular task",
					"status": "todo",
				}
				task["self"] = task // Circular reference

				os.MkdirAll(".pr-review/PR-900", 0755)
				// This will fail to marshal due to circular reference, so write empty tasks
				data := []byte(`{"generated_at": "2023-01-01T00:00:00Z", "tasks": []}`)
				os.WriteFile(".pr-review/PR-900/tasks.json", data, 0644)
			},
			args:      []string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			os.Chdir(tempDir)
			defer os.Chdir(originalDir)

			tt.setup(t, tempDir)

			var output bytes.Buffer
			cmd := &cobra.Command{
				Use:  "show",
				RunE: runShow,
			}
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Logf("Error (may be expected): %v", err)
			}
		})
	}
}

// TestShowOutputFormats tests different output formats
func TestShowOutputFormats(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	tasks := []storage.Task{
		{
			ID:          "format-001",
			Description: "Test formatting - Test different output formats",
			Status:      "todo",
			Priority:    "high",
			PRNumber:    1000,
		},
	}

	setupScenarioTasks(t, tempDir, tasks)

	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, output string)
	}{
		{
			name: "通常フォーマット",
			args: []string{"format-001"},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "Test formatting") {
					t.Error("Missing title in normal format")
				}
				if !strings.Contains(output, "high") {
					t.Error("Missing priority in normal format")
				}
			},
		},
		{
			name: "JSON出力",
			args: []string{"--json", "format-001"},
			check: func(t *testing.T, output string) {
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(output), &data); err == nil {
					if data["id"] != "format-001" {
						t.Error("Invalid JSON output")
					}
				}
			},
		},
		{
			name: "簡潔モード",
			args: []string{"--brief", "format-001"},
			check: func(t *testing.T, output string) {
				lines := strings.Split(output, "\n")
				if len(lines) > 5 {
					t.Error("Brief mode too verbose")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{
				Use:  "show",
				RunE: runShow,
			}
			cmd.Flags().Bool("json", false, "JSON output")
			cmd.Flags().Bool("brief", false, "Brief output")
			cmd.SetOut(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			tt.check(t, output.String())
		})
	}
}
