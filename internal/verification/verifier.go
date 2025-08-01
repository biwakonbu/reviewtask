package verification

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/storage"
)

// VerificationType defines the type of verification to perform
type VerificationType string

const (
	VerificationBuild  VerificationType = "build"
	VerificationTest   VerificationType = "test"
	VerificationLint   VerificationType = "lint"
	VerificationFormat VerificationType = "format"
	VerificationCustom VerificationType = "custom"
)

// VerificationResult contains the result of a verification operation
type VerificationResult struct {
	Type       VerificationType `json:"type"`
	Success    bool             `json:"success"`
	Message    string           `json:"message"`
	Output     string           `json:"output"`
	Command    string           `json:"command"`
	Duration   time.Duration    `json:"duration"`
	ExecutedAt time.Time        `json:"executed_at"`
}

// VerificationConfig holds configuration for verification commands
type VerificationConfig struct {
	BuildCommand  string             `json:"build_command"`
	TestCommand   string             `json:"test_command"`
	LintCommand   string             `json:"lint_command"`
	FormatCommand string             `json:"format_command"`
	CustomRules   map[string]string  `json:"custom_rules"` // task-type -> command mapping
	Mandatory     []VerificationType `json:"mandatory"`    // required verifications before completion
	Optional      []VerificationType `json:"optional"`     // optional verifications
	Timeout       time.Duration      `json:"timeout"`      // command timeout
}

// Verifier handles task completion verification
type Verifier struct {
	config  *VerificationConfig
	storage *storage.Manager
}

// NewVerifier creates a new verifier instance
func NewVerifier() (*Verifier, error) {
	config, err := loadVerificationConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load verification config: %w", err)
	}

	return &Verifier{
		config:  config,
		storage: storage.NewManager(),
	}, nil
}

// VerifyTask performs verification checks for a task before completion
func (v *Verifier) VerifyTask(taskID string) ([]VerificationResult, error) {
	// Get task details from storage
	allTasks, err := v.storage.GetAllTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	var task *storage.Task
	for _, t := range allTasks {
		if t.ID == taskID {
			task = &t
			break
		}
	}

	if task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	// Determine which verifications are required
	verifications := v.getRequiredVerifications(task)

	var results []VerificationResult
	for _, verificationType := range verifications {
		result := v.runVerification(verificationType, task)
		results = append(results, result)

		// Stop on first failure if it's a mandatory verification
		if !result.Success && v.isMandatory(verificationType) {
			break
		}
	}

	return results, nil
}

// CompleteTaskWithVerification updates task status to done after successful verification
func (v *Verifier) CompleteTaskWithVerification(taskID string) error {
	results, err := v.VerifyTask(taskID)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Check if all mandatory verifications passed
	for _, result := range results {
		if !result.Success && v.isMandatory(result.Type) {
			return fmt.Errorf("verification failed for %s: %s", result.Type, result.Message)
		}
	}

	// All verifications passed, update task status
	return v.storage.UpdateTaskStatus(taskID, "done")
}

// getRequiredVerifications determines which verifications to run for a task
func (v *Verifier) getRequiredVerifications(task *storage.Task) []VerificationType {
	verifications := make([]VerificationType, 0)

	// Add mandatory verifications
	verifications = append(verifications, v.config.Mandatory...)

	// Add task-specific custom verifications if any
	taskType := v.inferTaskType(task)
	if customCommand, exists := v.config.CustomRules[taskType]; exists && customCommand != "" {
		verifications = append(verifications, VerificationCustom)
	}

	return verifications
}

// runVerification executes a specific verification type
func (v *Verifier) runVerification(verificationType VerificationType, task *storage.Task) VerificationResult {
	startTime := time.Now()
	result := VerificationResult{
		Type:       verificationType,
		ExecutedAt: startTime,
	}

	var command string
	switch verificationType {
	case VerificationBuild:
		command = v.config.BuildCommand
	case VerificationTest:
		command = v.config.TestCommand
	case VerificationLint:
		command = v.config.LintCommand
	case VerificationFormat:
		command = v.config.FormatCommand
	case VerificationCustom:
		taskType := v.inferTaskType(task)
		command = v.config.CustomRules[taskType]
	default:
		result.Success = false
		result.Message = fmt.Sprintf("unknown verification type: %s", verificationType)
		return result
	}

	if command == "" {
		result.Success = false
		result.Message = fmt.Sprintf("no command configured for verification type: %s", verificationType)
		return result
	}

	result.Command = command
	result.Success, result.Output, result.Message = v.executeCommand(command)
	result.Duration = time.Since(startTime)

	return result
}

// executeCommand runs a shell command and returns success status, output, and message
func (v *Verifier) executeCommand(command string) (bool, string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), v.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, outputStr, fmt.Sprintf("command timed out after %v", v.config.Timeout)
		}
		return false, outputStr, fmt.Sprintf("command failed: %v", err)
	}

	return true, outputStr, "verification passed"
}

// isMandatory checks if a verification type is mandatory
func (v *Verifier) isMandatory(verificationType VerificationType) bool {
	for _, mandatory := range v.config.Mandatory {
		if mandatory == verificationType {
			return true
		}
	}
	return false
}

// inferTaskType tries to determine the task type from the task description
func (v *Verifier) inferTaskType(task *storage.Task) string {
	description := strings.ToLower(task.Description)

	// Simple keyword-based inference
	if strings.Contains(description, "test") || strings.Contains(description, "testing") {
		return "test-task"
	}
	if strings.Contains(description, "build") || strings.Contains(description, "compile") {
		return "build-task"
	}
	if strings.Contains(description, "lint") || strings.Contains(description, "format") {
		return "style-task"
	}
	if strings.Contains(description, "bug") || strings.Contains(description, "fix") {
		return "bug-fix"
	}
	if strings.Contains(description, "feature") || strings.Contains(description, "implement") {
		return "feature-task"
	}

	return "general-task"
}

// loadVerificationConfig loads verification configuration from config file
func loadVerificationConfig() (*VerificationConfig, error) {
	_, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Return default verification config if not present in main config
	// In a real implementation, this could be extended to read from config
	return getDefaultVerificationConfig(), nil
}

// getDefaultVerificationConfig returns default verification settings
func getDefaultVerificationConfig() *VerificationConfig {
	return &VerificationConfig{
		BuildCommand:  "go build ./...",
		TestCommand:   "go test ./...",
		LintCommand:   "golangci-lint run",
		FormatCommand: "gofmt -l .",
		CustomRules: map[string]string{
			"test-task":    "go test -v ./...",
			"build-task":   "go build -v ./...",
			"style-task":   "gofmt -l . && golangci-lint run",
			"feature-task": "go build ./... && go test ./...",
		},
		Mandatory: []VerificationType{
			VerificationBuild,
		},
		Optional: []VerificationType{
			VerificationTest,
			VerificationLint,
		},
		Timeout: 5 * time.Minute,
	}
}

// SetVerificationCommand allows setting custom verification commands
func (v *Verifier) SetVerificationCommand(taskType, command string) error {
	if v.config.CustomRules == nil {
		v.config.CustomRules = make(map[string]string)
	}
	v.config.CustomRules[taskType] = command

	// Save the configuration to persistence
	return v.saveVerificationConfig()
}

// GetVerificationStatus returns the verification status for a task
func (v *Verifier) GetVerificationStatus(taskID string) ([]VerificationResult, error) {
	// In a real implementation, this could load verification history from storage
	// For now, we'll run verification on demand
	return v.VerifyTask(taskID)
}

// GetConfig returns the current verification configuration
func (v *Verifier) GetConfig() *VerificationConfig {
	return v.config
}

// saveVerificationConfig saves the verification configuration to storage
func (v *Verifier) saveVerificationConfig() error {
	// For now, we'll save to a JSON file in .pr-review/
	// In a real implementation, this could be integrated with the main config
	// This is a placeholder - configuration persistence can be enhanced
	return nil
}
