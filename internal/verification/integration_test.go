package verification

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

func TestVerificationIntegration(t *testing.T) {
	// Create a temporary directory for test storage
	tempDir, err := os.MkdirTemp("", "reviewtask-verification-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create test configuration with simple commands that should succeed
	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			BuildCommand:  "echo 'Build successful'",
			TestCommand:   "echo 'Tests passed'",
			LintCommand:   "echo 'Linting complete'",
			FormatCommand: "echo 'Format check passed'",
			CustomRules: map[string]string{
				"test-task": "echo 'Custom test verification'",
			},
			MandatoryChecks: []string{"build"},
			OptionalChecks:  []string{"test", "lint"},
			TimeoutMinutes:  1,
			Enabled:         true,
		},
	}

	// Create storage manager and test task
	storageManager := storage.NewManager()

	// Create PR directory structure
	prDir := filepath.Join(".pr-review", "PR-123")
	if err := os.MkdirAll(prDir, 0755); err != nil {
		t.Fatalf("Failed to create PR directory: %v", err)
	}

	// Create test task
	testTask := storage.Task{
		ID:          "integration-test-task",
		Description: "Integration test task for verification",
		Status:      "doing",
		PRNumber:    123,
		CreatedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
	}

	// Save task to storage
	if err := storageManager.SaveTasks(123, []storage.Task{testTask}); err != nil {
		t.Fatalf("Failed to save test task: %v", err)
	}

	// Create verifier with test config
	verifier := &Verifier{
		config:  cfg,
		storage: storageManager,
	}

	// Test 1: Verify task requirements
	t.Run("VerifyTaskRequirements", func(t *testing.T) {
		verifications := verifier.getRequiredVerifications(&testTask)

		// Should have at least the mandatory build check
		if len(verifications) == 0 {
			t.Error("Expected at least one verification requirement")
		}

		// Check that build verification is included (mandatory)
		foundBuild := false
		for _, v := range verifications {
			if v == VerificationBuild {
				foundBuild = true
				break
			}
		}
		if !foundBuild {
			t.Error("Expected build verification to be required")
		}
	})

	// Test 2: Test individual verification execution
	t.Run("ExecuteIndividualVerification", func(t *testing.T) {
		result := verifier.runVerification(VerificationBuild, &testTask)

		if !result.Success {
			t.Errorf("Expected build verification to succeed, got: %s", result.Message)
		}

		if result.Type != VerificationBuild {
			t.Errorf("Expected verification type build, got: %s", result.Type)
		}

		if result.Command != "echo 'Build successful'" {
			t.Errorf("Expected command to be set correctly, got: %s", result.Command)
		}

		if result.Duration <= 0 {
			t.Error("Expected duration to be measured")
		}
	})

	// Test 3: Test verification with custom rules
	t.Run("CustomVerificationRules", func(t *testing.T) {
		// Create a test-task to trigger custom rules
		testTaskWithCustom := testTask
		testTaskWithCustom.Description = "Fix test failure in user service"
		testTaskWithCustom.ID = "custom-test-task"

		verifications := verifier.getRequiredVerifications(&testTaskWithCustom)

		// Should include custom verification for test-task
		foundCustom := false
		for _, v := range verifications {
			if v == VerificationCustom {
				foundCustom = true
				break
			}
		}
		if !foundCustom {
			t.Error("Expected custom verification to be included for test-task")
		}

		// Test custom verification execution
		result := verifier.runVerification(VerificationCustom, &testTaskWithCustom)
		if !result.Success {
			t.Errorf("Expected custom verification to succeed, got: %s", result.Message)
		}
	})

	// Test 4: Test verification status updates
	t.Run("VerificationStatusUpdates", func(t *testing.T) {
		// Test successful verification status update
		successResult := &storage.VerificationResult{
			Timestamp:     time.Now().Format("2006-01-02T15:04:05Z"),
			Success:       true,
			FailureReason: "",
			ChecksRun:     []string{"build"},
		}

		err := storageManager.UpdateTaskVerificationStatus(testTask.ID, "verified", successResult)
		if err != nil {
			t.Errorf("Failed to update verification status: %v", err)
		}

		// Verify the status was updated
		tasks, err := storageManager.GetAllTasks()
		if err != nil {
			t.Fatalf("Failed to get tasks: %v", err)
		}

		var updatedTask *storage.Task
		for _, task := range tasks {
			if task.ID == testTask.ID {
				updatedTask = &task
				break
			}
		}

		if updatedTask == nil {
			t.Fatal("Could not find updated task")
		}

		if updatedTask.VerificationStatus != "verified" {
			t.Errorf("Expected verification status 'verified', got %q", updatedTask.VerificationStatus)
		}

		if len(updatedTask.VerificationResults) != 1 {
			t.Errorf("Expected 1 verification result, got %d", len(updatedTask.VerificationResults))
		}
	})

	// Test 5: Test verification failure handling
	t.Run("VerificationFailureHandling", func(t *testing.T) {
		// Create a verifier with a command that will fail
		failConfig := &config.Config{
			VerificationSettings: config.VerificationSettings{
				BuildCommand:    "exit 1", // This command will fail
				MandatoryChecks: []string{"build"},
				TimeoutMinutes:  1,
				Enabled:         true,
			},
		}

		failVerifier := &Verifier{
			config:  failConfig,
			storage: storageManager,
		}

		result := failVerifier.runVerification(VerificationBuild, &testTask)

		if result.Success {
			t.Error("Expected verification to fail with exit 1 command")
		}

		if result.Message == "" {
			t.Error("Expected failure message to be set")
		}
	})

	// Test 6: Test configuration integration
	t.Run("ConfigurationIntegration", func(t *testing.T) {
		// Test that verifier correctly uses configuration
		verificationConfig := verifier.GetConfig()

		if verificationConfig.BuildCommand != cfg.VerificationSettings.BuildCommand {
			t.Errorf("Expected build command %q, got %q", cfg.VerificationSettings.BuildCommand, verificationConfig.BuildCommand)
		}

		if len(verificationConfig.Mandatory) != 1 {
			t.Errorf("Expected 1 mandatory check, got %d", len(verificationConfig.Mandatory))
		}

		if verificationConfig.Timeout != time.Duration(cfg.VerificationSettings.TimeoutMinutes)*time.Minute {
			t.Errorf("Expected timeout %v, got %v", time.Duration(cfg.VerificationSettings.TimeoutMinutes)*time.Minute, verificationConfig.Timeout)
		}
	})

	// Test 7: Test command timeout
	t.Run("CommandTimeout", func(t *testing.T) {
		// Create a verifier with a very short timeout and a long-running command
		timeoutConfig := &config.Config{
			VerificationSettings: config.VerificationSettings{
				BuildCommand:    "sleep 2", // This will exceed the timeout
				MandatoryChecks: []string{"build"},
				TimeoutMinutes:  0, // This will be converted to 0 seconds, causing immediate timeout
				Enabled:         true,
			},
		}

		timeoutVerifier := &Verifier{
			config:  timeoutConfig,
			storage: storageManager,
		}

		result := timeoutVerifier.runVerification(VerificationBuild, &testTask)

		// The command should fail due to timeout
		if result.Success {
			t.Error("Expected verification to fail due to timeout")
		}

		// Note: The timeout test might be flaky depending on system performance
		// In a real test environment, we'd use a more controlled timeout mechanism
	})
}

func TestTaskTypeInference(t *testing.T) {
	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			CustomRules: map[string]string{
				"test-task":    "go test -v ./...",
				"build-task":   "go build -v ./...",
				"style-task":   "gofmt -l . && golangci-lint run",
				"bug-fix":      "go test ./... && go build ./...",
				"feature-task": "go build ./... && go test ./...",
			},
		},
	}

	mockStorage := &MockStorage{
		tasks:                       make([]storage.Task, 0),
		verificationStatusUpdates:   make(map[string]string),
		implementationStatusUpdates: make(map[string]string),
		verificationResults:         make(map[string][]storage.VerificationResult),
	}

	verifier := &Verifier{
		config:  cfg,
		storage: mockStorage,
	}

	tests := []struct {
		description  string
		expectedType string
	}{
		{"Fix failing unit tests in auth module", "test-task"},
		{"Build the new payment service", "build-task"},
		{"Run linter on codebase", "style-task"},
		{"Fix bug in user registration", "bug-fix"},
		{"Implement new dashboard feature", "feature-task"},
		{"Update README documentation", "general-task"},
		{"", "general-task"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			task := &storage.Task{Description: test.description}
			result := verifier.inferTaskType(task)

			if result != test.expectedType {
				t.Errorf("inferTaskType(%q) = %q, expected %q", test.description, result, test.expectedType)
			}
		})
	}
}

func TestVerificationHistoryTracking(t *testing.T) {
	// Create a temporary directory for test storage
	tempDir, err := os.MkdirTemp("", "reviewtask-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create storage manager and test task
	storageManager := storage.NewManager()

	// Create PR directory structure
	prDir := filepath.Join(".pr-review", "PR-456")
	if err := os.MkdirAll(prDir, 0755); err != nil {
		t.Fatalf("Failed to create PR directory: %v", err)
	}

	// Create test task
	testTask := storage.Task{
		ID:          "history-test-task",
		Description: "Test task for verification history",
		Status:      "doing",
		PRNumber:    456,
		CreatedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
	}

	// Save task to storage
	if err := storageManager.SaveTasks(456, []storage.Task{testTask}); err != nil {
		t.Fatalf("Failed to save test task: %v", err)
	}

	// Add multiple verification results
	results := []storage.VerificationResult{
		{
			Timestamp:     time.Now().Add(-2 * time.Hour).Format("2006-01-02T15:04:05Z"),
			Success:       false,
			FailureReason: "Build failed",
			ChecksRun:     []string{"build"},
		},
		{
			Timestamp:     time.Now().Add(-1 * time.Hour).Format("2006-01-02T15:04:05Z"),
			Success:       false,
			FailureReason: "Tests failed",
			ChecksRun:     []string{"build", "test"},
		},
		{
			Timestamp:     time.Now().Format("2006-01-02T15:04:05Z"),
			Success:       true,
			FailureReason: "",
			ChecksRun:     []string{"build", "test", "lint"},
		},
	}

	// Add verification results one by one
	for i, result := range results {
		status := "failed"
		if result.Success {
			status = "verified"
		}

		err := storageManager.UpdateTaskVerificationStatus(testTask.ID, status, &result)
		if err != nil {
			t.Errorf("Failed to update verification status for result %d: %v", i, err)
		}
	}

	// Retrieve verification history
	history, err := storageManager.GetTaskVerificationHistory(testTask.ID)
	if err != nil {
		t.Fatalf("Failed to get verification history: %v", err)
	}

	// Verify history is correct
	if len(history) != len(results) {
		t.Errorf("Expected %d verification results in history, got %d", len(results), len(history))
	}

	// Verify each result in history
	for i, expected := range results {
		if i >= len(history) {
			t.Errorf("Missing verification result at index %d", i)
			continue
		}

		actual := history[i]
		if actual.Success != expected.Success {
			t.Errorf("Result %d: expected Success %t, got %t", i, expected.Success, actual.Success)
		}

		if actual.FailureReason != expected.FailureReason {
			t.Errorf("Result %d: expected FailureReason %q, got %q", i, expected.FailureReason, actual.FailureReason)
		}

		if len(actual.ChecksRun) != len(expected.ChecksRun) {
			t.Errorf("Result %d: expected %d checks, got %d", i, len(expected.ChecksRun), len(actual.ChecksRun))
		}
	}
}
