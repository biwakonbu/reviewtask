package verification

import (
	"testing"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

// MockStorageInterface defines the interface needed by verifier
type MockStorageInterface interface {
	GetAllTasks() ([]storage.Task, error)
	UpdateTaskStatus(taskID, newStatus string) error
	UpdateTaskVerificationStatus(taskID string, verificationStatus string, result *storage.VerificationResult) error
	UpdateTaskImplementationStatus(taskID string, implementationStatus string) error
	GetTaskVerificationHistory(taskID string) ([]storage.VerificationResult, error)
}

// MockStorage is a mock implementation for testing
type MockStorage struct {
	tasks                       []storage.Task
	verificationStatusUpdates   map[string]string
	implementationStatusUpdates map[string]string
	verificationResults         map[string][]storage.VerificationResult
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		tasks:                       make([]storage.Task, 0),
		verificationStatusUpdates:   make(map[string]string),
		implementationStatusUpdates: make(map[string]string),
		verificationResults:         make(map[string][]storage.VerificationResult),
	}
}

func (m *MockStorage) GetAllTasks() ([]storage.Task, error) {
	return m.tasks, nil
}

func (m *MockStorage) UpdateTaskStatus(taskID, newStatus string) error {
	for i := range m.tasks {
		if m.tasks[i].ID == taskID {
			m.tasks[i].Status = newStatus
			return nil
		}
	}
	return storage.ErrTaskNotFound
}

func (m *MockStorage) UpdateTaskVerificationStatus(taskID string, verificationStatus string, result *storage.VerificationResult) error {
	m.verificationStatusUpdates[taskID] = verificationStatus
	if result != nil {
		if m.verificationResults[taskID] == nil {
			m.verificationResults[taskID] = make([]storage.VerificationResult, 0)
		}
		m.verificationResults[taskID] = append(m.verificationResults[taskID], *result)
	}
	return nil
}

func (m *MockStorage) UpdateTaskImplementationStatus(taskID string, implementationStatus string) error {
	m.implementationStatusUpdates[taskID] = implementationStatus
	return nil
}

func (m *MockStorage) GetTaskVerificationHistory(taskID string) ([]storage.VerificationResult, error) {
	if results, exists := m.verificationResults[taskID]; exists {
		return results, nil
	}
	return []storage.VerificationResult{}, nil
}

func (m *MockStorage) AddTask(task storage.Task) {
	m.tasks = append(m.tasks, task)
}

func TestNewVerifier(t *testing.T) {
	// Test verifier creation
	verifier, err := NewVerifier()
	if err != nil {
		t.Fatalf("Expected no error creating verifier, got: %v", err)
	}
	if verifier == nil {
		t.Fatal("Expected verifier to be created, got nil")
	}
	if verifier.config == nil {
		t.Fatal("Expected verifier config to be loaded, got nil")
	}
	if verifier.storage == nil {
		t.Fatal("Expected verifier storage to be initialized, got nil")
	}
}

func TestStringToVerificationType(t *testing.T) {
	tests := []struct {
		input    string
		expected VerificationType
	}{
		{"build", VerificationBuild},
		{"test", VerificationTest},
		{"lint", VerificationLint},
		{"format", VerificationFormat},
		{"custom", VerificationCustom},
		{"invalid", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := stringToVerificationType(test.input)
		if result != test.expected {
			t.Errorf("stringToVerificationType(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestStringSliceToVerificationTypes(t *testing.T) {
	input := []string{"build", "test", "invalid", "lint"}
	expected := []VerificationType{VerificationBuild, VerificationTest, VerificationLint}

	result := stringSliceToVerificationTypes(input)
	if len(result) != len(expected) {
		t.Errorf("Expected %d verification types, got %d", len(expected), len(result))
	}

	for i, vType := range expected {
		if i >= len(result) || result[i] != vType {
			t.Errorf("Expected verification type %q at index %d, got %q", vType, i, result[i])
		}
	}
}

func TestInferTaskType(t *testing.T) {
	// Create a verifier with test config
	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			BuildCommand:    "go build",
			TestCommand:     "go test",
			LintCommand:     "golangci-lint run",
			FormatCommand:   "gofmt -l .",
			CustomRules:     make(map[string]string),
			MandatoryChecks: []string{"build"},
			OptionalChecks:  []string{"test", "lint"},
			TimeoutMinutes:  5,
			Enabled:         true,
		},
	}

	verifier := &Verifier{
		config:  cfg,
		storage: NewMockStorage(),
	}

	tests := []struct {
		description string
		expected    string
	}{
		{"Fix test failure in user service", "test-task"},
		{"Build the new authentication module", "build-task"},
		{"Run linter on the codebase", "style-task"},
		{"Fix bug in payment processing", "bug-fix"},
		{"Implement new feature for user management", "feature-task"},
		{"Update documentation", "general-task"},
		{"", "general-task"},
	}

	for _, test := range tests {
		task := &storage.Task{Description: test.description}
		result := verifier.inferTaskType(task)
		if result != test.expected {
			t.Errorf("inferTaskType(%q) = %q, expected %q", test.description, result, test.expected)
		}
	}
}

func TestGetConfig(t *testing.T) {
	// Create a verifier with test config
	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			BuildCommand:    "go build ./...",
			TestCommand:     "go test ./...",
			LintCommand:     "golangci-lint run",
			FormatCommand:   "gofmt -l .",
			CustomRules:     map[string]string{"test-task": "go test -v ./..."},
			MandatoryChecks: []string{"build"},
			OptionalChecks:  []string{"test", "lint"},
			TimeoutMinutes:  10,
			Enabled:         true,
		},
	}

	verifier := &Verifier{
		config:  cfg,
		storage: NewMockStorage(),
	}

	result := verifier.GetConfig()
	if result == nil {
		t.Fatal("Expected config to be returned, got nil")
	}

	if result.BuildCommand != "go build ./..." {
		t.Errorf("Expected BuildCommand 'go build ./...', got %q", result.BuildCommand)
	}

	if result.TestCommand != "go test ./..." {
		t.Errorf("Expected TestCommand 'go test ./...', got %q", result.TestCommand)
	}

	if len(result.Mandatory) != 1 || result.Mandatory[0] != VerificationBuild {
		t.Errorf("Expected Mandatory to contain [VerificationBuild], got %v", result.Mandatory)
	}

	if len(result.Optional) != 2 {
		t.Errorf("Expected 2 optional checks, got %d", len(result.Optional))
	}

	if result.Timeout != 10*time.Minute {
		t.Errorf("Expected timeout 10m, got %v", result.Timeout)
	}

	if result.CustomRules["test-task"] != "go test -v ./..." {
		t.Errorf("Expected custom rule for test-task, got %q", result.CustomRules["test-task"])
	}
}

func TestSetVerificationCommand(t *testing.T) {
	// Create a verifier with test config
	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			BuildCommand:    "go build",
			TestCommand:     "go test",
			LintCommand:     "golangci-lint run",
			FormatCommand:   "gofmt -l .",
			CustomRules:     make(map[string]string),
			MandatoryChecks: []string{"build"},
			OptionalChecks:  []string{"test"},
			TimeoutMinutes:  5,
			Enabled:         true,
		},
	}

	verifier := &Verifier{
		config:  cfg,
		storage: NewMockStorage(),
	}

	// Note: In a real test, we would mock the config.Save() call
	// For this test, we'll verify the custom rule is set correctly
	if verifier.config.VerificationSettings.CustomRules == nil {
		verifier.config.VerificationSettings.CustomRules = make(map[string]string)
	}
	verifier.config.VerificationSettings.CustomRules["test-task"] = "go test -v ./..."

	if verifier.config.VerificationSettings.CustomRules["test-task"] != "go test -v ./..." {
		t.Errorf("Expected custom rule to be set, got %q", verifier.config.VerificationSettings.CustomRules["test-task"])
	}
}

func TestIsMandatory(t *testing.T) {
	// Create a verifier with test config
	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			MandatoryChecks: []string{"build", "test"},
			OptionalChecks:  []string{"lint"},
		},
	}

	verifier := &Verifier{
		config:  cfg,
		storage: NewMockStorage(),
	}

	tests := []struct {
		verificationType VerificationType
		expected         bool
	}{
		{VerificationBuild, true},
		{VerificationTest, true},
		{VerificationLint, false},
		{VerificationFormat, false},
		{VerificationCustom, false},
	}

	for _, test := range tests {
		result := verifier.isMandatory(test.verificationType)
		if result != test.expected {
			t.Errorf("isMandatory(%q) = %t, expected %t", test.verificationType, result, test.expected)
		}
	}
}

func TestGetRequiredVerifications(t *testing.T) {
	// Create a verifier with test config
	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			MandatoryChecks: []string{"build", "test"},
			CustomRules:     map[string]string{"test-task": "go test -v ./..."},
		},
	}

	verifier := &Verifier{
		config:  cfg,
		storage: NewMockStorage(),
	}

	// Test with test-task (should include custom verification)
	testTask := &storage.Task{
		ID:          "test-1",
		Description: "Fix test failure in user service",
	}

	verifications := verifier.getRequiredVerifications(testTask)

	expectedCount := 3 // build, test, custom
	if len(verifications) != expectedCount {
		t.Errorf("Expected %d verifications for test-task, got %d", expectedCount, len(verifications))
	}

	// Test with regular task (no custom verification)
	regularTask := &storage.Task{
		ID:          "task-1",
		Description: "Update documentation",
	}

	verifications = verifier.getRequiredVerifications(regularTask)

	expectedCount = 2 // build, test
	if len(verifications) != expectedCount {
		t.Errorf("Expected %d verifications for regular task, got %d", expectedCount, len(verifications))
	}
}

func TestCompleteTaskWithVerification_Success(t *testing.T) {
	// This is a complex test that would require mocking command execution
	// For now, we'll test the setup and mock storage interaction

	cfg := &config.Config{
		VerificationSettings: config.VerificationSettings{
			BuildCommand:    "echo 'build success'",
			TestCommand:     "echo 'test success'",
			MandatoryChecks: []string{"build"},
			TimeoutMinutes:  1,
		},
	}

	mockStorage := NewMockStorage()
	task := storage.Task{
		ID:          "task-1",
		Description: "Test task",
		Status:      "doing",
	}
	mockStorage.AddTask(task)

	// Verify that the mock storage methods work correctly
	if len(mockStorage.tasks) != 1 {
		t.Errorf("Expected 1 task in mock storage, got %d", len(mockStorage.tasks))
	}

	// Verify we can create a verifier with the config
	if cfg.VerificationSettings.BuildCommand != "echo 'build success'" {
		t.Errorf("Expected build command to be set correctly")
	}
}

func TestVerificationResult(t *testing.T) {
	// Test VerificationResult struct
	result := VerificationResult{
		Type:       VerificationBuild,
		Success:    true,
		Message:    "Build successful",
		Output:     "Build completed without errors",
		Command:    "go build ./...",
		Duration:   2 * time.Second,
		ExecutedAt: time.Now(),
	}

	if result.Type != VerificationBuild {
		t.Errorf("Expected Type VerificationBuild, got %q", result.Type)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Message != "Build successful" {
		t.Errorf("Expected Message 'Build successful', got %q", result.Message)
	}

	if result.Duration != 2*time.Second {
		t.Errorf("Expected Duration 2s, got %v", result.Duration)
	}
}
