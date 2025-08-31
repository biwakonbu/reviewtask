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

// StorageInterface defines the storage operations needed by verifier
type StorageInterface interface {
	GetAllTasks() ([]storage.Task, error)
	UpdateTaskStatus(taskID, newStatus string) error
	UpdateTaskVerificationStatus(taskID string, verificationStatus string, result *storage.VerificationResult) error
	UpdateTaskImplementationStatus(taskID string, implementationStatus string) error
	GetTaskVerificationHistory(taskID string) ([]storage.VerificationResult, error)
}

// Verifier handles task completion verification
type Verifier struct {
	config  *config.Config
	storage StorageInterface
}

// NewVerifier creates a new verifier instance
func NewVerifier() (*Verifier, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &Verifier{
		config:  cfg,
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
		// Update verification status to failed
		failureResult := &storage.VerificationResult{
			Timestamp:     time.Now().Format("2006-01-02T15:04:05Z"),
			Success:       false,
			FailureReason: err.Error(),
			ChecksRun:     []string{},
		}
		_ = v.storage.UpdateTaskVerificationStatus(taskID, "failed", failureResult)
		return fmt.Errorf("verification failed: %w", err)
	}

	// Check if all mandatory verifications passed
	var failedChecks []string
	var allChecks []string
	for _, result := range results {
		allChecks = append(allChecks, string(result.Type))
		if !result.Success && v.isMandatory(result.Type) {
			failedChecks = append(failedChecks, string(result.Type))
		}
	}

	if len(failedChecks) > 0 {
		// Update verification status to failed with details
		failureResult := &storage.VerificationResult{
			Timestamp:     time.Now().Format("2006-01-02T15:04:05Z"),
			Success:       false,
			FailureReason: fmt.Sprintf("Mandatory verification checks failed: %s", strings.Join(failedChecks, ", ")),
			ChecksRun:     allChecks,
		}
		_ = v.storage.UpdateTaskVerificationStatus(taskID, "failed", failureResult)
		return fmt.Errorf("verification failed for %s", strings.Join(failedChecks, ", "))
	}

	// All verifications passed, update both verification and task status
	successResult := &storage.VerificationResult{
		Timestamp:     time.Now().Format("2006-01-02T15:04:05Z"),
		Success:       true,
		FailureReason: "",
		ChecksRun:     allChecks,
	}

	// Update verification status first
	if err := v.storage.UpdateTaskVerificationStatus(taskID, "verified", successResult); err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	// Update implementation status
	if err := v.storage.UpdateTaskImplementationStatus(taskID, "implemented"); err != nil {
		return fmt.Errorf("failed to update implementation status: %w", err)
	}

	// Finally update task status to done
	return v.storage.UpdateTaskStatus(taskID, "done")
}

// getRequiredVerifications determines which verifications to run for a task
func (v *Verifier) getRequiredVerifications(task *storage.Task) []VerificationType {
	verifications := make([]VerificationType, 0)

	// Add mandatory verifications
	for _, checkType := range v.config.VerificationSettings.MandatoryChecks {
		if vType := stringToVerificationType(checkType); vType != "" {
			verifications = append(verifications, vType)
		}
	}

	// Add task-specific custom verifications if any
	taskType := v.inferTaskType(task)
	if customCommand, exists := v.config.VerificationSettings.CustomRules[taskType]; exists && customCommand != "" {
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
		command = v.config.VerificationSettings.BuildCommand
	case VerificationTest:
		command = v.config.VerificationSettings.TestCommand
	case VerificationLint:
		command = v.config.VerificationSettings.LintCommand
	case VerificationFormat:
		command = v.config.VerificationSettings.FormatCommand
	case VerificationCustom:
		taskType := v.inferTaskType(task)
		command = v.config.VerificationSettings.CustomRules[taskType]
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
	timeout := time.Duration(v.config.VerificationSettings.TimeoutMinutes) * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, outputStr, fmt.Sprintf("command timed out after %v", timeout)
		}
		return false, outputStr, fmt.Sprintf("command failed: %v", err)
	}

	return true, outputStr, "verification passed"
}

// isMandatory checks if a verification type is mandatory
func (v *Verifier) isMandatory(verificationType VerificationType) bool {
	for _, mandatory := range v.config.VerificationSettings.MandatoryChecks {
		if stringToVerificationType(mandatory) == verificationType {
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

// SetVerificationCommand allows setting custom verification commands
func (v *Verifier) SetVerificationCommand(taskType, command string) error {
	if v.config.VerificationSettings.CustomRules == nil {
		v.config.VerificationSettings.CustomRules = make(map[string]string)
	}
	v.config.VerificationSettings.CustomRules[taskType] = command

	// Save the configuration to persistence
	return v.config.Save()
}

// GetVerificationStatus returns the verification status for a task
func (v *Verifier) GetVerificationStatus(taskID string) ([]VerificationResult, error) {
	// In a real implementation, this could load verification history from storage
	// For now, we'll run verification on demand
	return v.VerifyTask(taskID)
}

// GetVerificationHistory returns the stored verification history for a task
func (v *Verifier) GetVerificationHistory(taskID string) ([]storage.VerificationResult, error) {
	return v.storage.GetTaskVerificationHistory(taskID)
}

// GetConfig returns the current verification configuration
func (v *Verifier) GetConfig() *VerificationConfig {
	return &VerificationConfig{
		BuildCommand:  v.config.VerificationSettings.BuildCommand,
		TestCommand:   v.config.VerificationSettings.TestCommand,
		LintCommand:   v.config.VerificationSettings.LintCommand,
		FormatCommand: v.config.VerificationSettings.FormatCommand,
		CustomRules:   v.config.VerificationSettings.CustomRules,
		Mandatory:     stringSliceToVerificationTypes(v.config.VerificationSettings.MandatoryChecks),
		Optional:      stringSliceToVerificationTypes(v.config.VerificationSettings.OptionalChecks),
		Timeout:       time.Duration(v.config.VerificationSettings.TimeoutMinutes) * time.Minute,
	}
}

// stringToVerificationType converts string to VerificationType
func stringToVerificationType(s string) VerificationType {
	switch s {
	case "build":
		return VerificationBuild
	case "test":
		return VerificationTest
	case "lint":
		return VerificationLint
	case "format":
		return VerificationFormat
	case "custom":
		return VerificationCustom
	default:
		return ""
	}
}

// stringSliceToVerificationTypes converts string slice to VerificationType slice
func stringSliceToVerificationTypes(strings []string) []VerificationType {
	types := make([]VerificationType, 0, len(strings))
	for _, s := range strings {
		if vType := stringToVerificationType(s); vType != "" {
			types = append(types, vType)
		}
	}
	return types
}
