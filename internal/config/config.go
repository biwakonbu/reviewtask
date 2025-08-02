package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ConfigFile = ".pr-review/config.json"
)

type Config struct {
	PriorityRules        PriorityRules        `json:"priority_rules"`
	ProjectSpecific      ProjectSpecific      `json:"project_specific"`
	TaskSettings         TaskSettings         `json:"task_settings"`
	AISettings           AISettings           `json:"ai_settings"`
	VerificationSettings VerificationSettings `json:"verification_settings"`
	UpdateCheck          UpdateCheck          `json:"update_check"`
}

type PriorityRules struct {
	Critical string `json:"critical"`
	High     string `json:"high"`
	Medium   string `json:"medium"`
	Low      string `json:"low"`
}

type ProjectSpecific struct {
	Critical string `json:"critical"`
	High     string `json:"high"`
	Medium   string `json:"medium"`
	Low      string `json:"low"`
}

type TaskSettings struct {
	DefaultStatus       string   `json:"default_status"`
	AutoPrioritize      bool     `json:"auto_prioritize"`
	LowPriorityPatterns []string `json:"low_priority_patterns"`
	LowPriorityStatus   string   `json:"low_priority_status"`
}

type AISettings struct {
	UserLanguage             string  `json:"user_language"`              // e.g., "Japanese", "English"
	OutputFormat             string  `json:"output_format"`              // "json"
	MaxRetries               int     `json:"max_retries"`                // Validation retry attempts (default: 5)
	ValidationEnabled        *bool   `json:"validation_enabled"`         // Enable two-stage validation
	QualityThreshold         float64 `json:"quality_threshold"`          // Minimum score to accept (0.0-1.0)
	VerboseMode              bool    `json:"verbose_mode"`               // Enable verbose output (detailed progress and errors)
	ClaudePath               string  `json:"claude_path"`                // Custom path to Claude CLI (overrides default search)
	MaxTasksPerComment       int     `json:"max_tasks_per_comment"`      // Maximum tasks to generate per comment (default: 2)
	DeduplicationEnabled     bool    `json:"deduplication_enabled"`      // Enable task deduplication (default: true)
	SimilarityThreshold      float64 `json:"similarity_threshold"`       // Task similarity threshold for deduplication (0.0-1.0)
	ProcessNitpickComments   bool    `json:"process_nitpick_comments"`   // Process nitpick comments from review bots (default: true)
	NitpickPriority          string  `json:"nitpick_priority"`           // Priority level for nitpick-generated tasks (default: "low")
	EnableJSONRecovery       bool    `json:"enable_json_recovery"`       // Enable JSON recovery for incomplete Claude API responses (default: true)
	MaxRecoveryAttempts      int     `json:"max_recovery_attempts"`      // Maximum JSON recovery attempts (default: 3)
	PartialResponseThreshold float64 `json:"partial_response_threshold"` // Minimum threshold for accepting partial responses (default: 0.7)
	LogTruncatedResponses    bool    `json:"log_truncated_responses"`    // Log truncated responses for debugging (default: true)
	ProcessSelfReviews       bool    `json:"process_self_reviews"`       // Process self-reviews from PR author (default: false)
}

type VerificationSettings struct {
	BuildCommand    string            `json:"build_command"`    // Command to run for build verification
	TestCommand     string            `json:"test_command"`     // Command to run for test verification
	LintCommand     string            `json:"lint_command"`     // Command to run for lint verification
	FormatCommand   string            `json:"format_command"`   // Command to run for format verification
	CustomRules     map[string]string `json:"custom_rules"`     // Task-type to command mapping
	MandatoryChecks []string          `json:"mandatory_checks"` // Required verification types
	OptionalChecks  []string          `json:"optional_checks"`  // Optional verification types
	TimeoutMinutes  int               `json:"timeout_minutes"`  // Verification timeout in minutes
	Enabled         bool              `json:"enabled"`          // Enable verification functionality
}

type UpdateCheck struct {
	Enabled           bool      `json:"enabled"`            // Enable automatic update checking
	IntervalHours     int       `json:"interval_hours"`     // Check interval in hours (default: 24)
	NotifyPrereleases bool      `json:"notify_prereleases"` // Show prerelease notifications
	LastCheck         time.Time `json:"last_check"`         // Last check timestamp
}

// Default configuration
func defaultConfig() *Config {
	validationTrue := true
	return &Config{
		PriorityRules: PriorityRules{
			Critical: "Security vulnerabilities, authentication bypasses, data exposure risks",
			High:     "Performance bottlenecks, memory leaks, database optimization issues",
			Medium:   "Functional bugs, logic improvements, error handling",
			Low:      "Code style, naming conventions, comment improvements",
		},
		ProjectSpecific: ProjectSpecific{
			Critical: "",
			High:     "",
			Medium:   "",
			Low:      "",
		},
		TaskSettings: TaskSettings{
			DefaultStatus:       "todo",
			AutoPrioritize:      true,
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
			LowPriorityStatus:   "pending",
		},
		AISettings: AISettings{
			UserLanguage:             "English",
			OutputFormat:             "json",
			MaxRetries:               5,
			ValidationEnabled:        &validationTrue,
			QualityThreshold:         0.8,
			VerboseMode:              false,
			ClaudePath:               "", // Empty means use default search paths
			MaxTasksPerComment:       2,
			DeduplicationEnabled:     true,
			SimilarityThreshold:      0.8,
			ProcessNitpickComments:   true,
			NitpickPriority:          "low",
			EnableJSONRecovery:       true,
			MaxRecoveryAttempts:      3,
			PartialResponseThreshold: 0.7,
			LogTruncatedResponses:    true,
			ProcessSelfReviews:       false,
		},
		VerificationSettings: VerificationSettings{
			BuildCommand:    "go build ./...",
			TestCommand:     "go test ./...",
			LintCommand:     "golangci-lint run",
			FormatCommand:   "gofmt -l .",
			CustomRules:     make(map[string]string),
			MandatoryChecks: []string{"build"},
			OptionalChecks:  []string{"test", "lint"},
			TimeoutMinutes:  5,
			Enabled:         true,
		},
		UpdateCheck: UpdateCheck{
			Enabled:           true,
			IntervalHours:     24,
			NotifyPrereleases: false,
			LastCheck:         time.Time{}, // Zero time means never checked
		},
	}
}

func Load() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		// Create default config file
		config := defaultConfig()
		if err := save(config); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	// Load existing config
	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge with defaults for any missing fields
	mergeWithDefaults(&config)

	return &config, nil
}

func save(config *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(ConfigFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile, data, 0644)
}

func mergeWithDefaults(config *Config) {
	defaults := defaultConfig()

	// Merge priority rules
	if config.PriorityRules.Critical == "" {
		config.PriorityRules.Critical = defaults.PriorityRules.Critical
	}
	if config.PriorityRules.High == "" {
		config.PriorityRules.High = defaults.PriorityRules.High
	}
	if config.PriorityRules.Medium == "" {
		config.PriorityRules.Medium = defaults.PriorityRules.Medium
	}
	if config.PriorityRules.Low == "" {
		config.PriorityRules.Low = defaults.PriorityRules.Low
	}

	// Merge task settings
	if config.TaskSettings.DefaultStatus == "" {
		config.TaskSettings.DefaultStatus = defaults.TaskSettings.DefaultStatus
	}
	if len(config.TaskSettings.LowPriorityPatterns) == 0 {
		config.TaskSettings.LowPriorityPatterns = defaults.TaskSettings.LowPriorityPatterns
	}
	if config.TaskSettings.LowPriorityStatus == "" {
		config.TaskSettings.LowPriorityStatus = defaults.TaskSettings.LowPriorityStatus
	}

	// Check if this is likely an old config by looking for any non-zero new fields
	isOldConfig := config.AISettings.MaxTasksPerComment == 0 &&
		config.AISettings.SimilarityThreshold == 0 &&
		config.AISettings.NitpickPriority == "" &&
		config.AISettings.MaxRecoveryAttempts == 0

	// Merge AI settings
	if config.AISettings.UserLanguage == "" {
		config.AISettings.UserLanguage = defaults.AISettings.UserLanguage
	}
	if config.AISettings.OutputFormat == "" {
		config.AISettings.OutputFormat = defaults.AISettings.OutputFormat
	}
	if config.AISettings.MaxRetries == 0 {
		config.AISettings.MaxRetries = defaults.AISettings.MaxRetries
	}
	if config.AISettings.QualityThreshold == 0 {
		config.AISettings.QualityThreshold = defaults.AISettings.QualityThreshold
	}
	if config.AISettings.MaxTasksPerComment == 0 {
		config.AISettings.MaxTasksPerComment = defaults.AISettings.MaxTasksPerComment
	}
	if config.AISettings.SimilarityThreshold == 0 {
		config.AISettings.SimilarityThreshold = defaults.AISettings.SimilarityThreshold
	}
	if config.AISettings.NitpickPriority == "" {
		config.AISettings.NitpickPriority = defaults.AISettings.NitpickPriority
	}
	if config.AISettings.MaxRecoveryAttempts == 0 {
		config.AISettings.MaxRecoveryAttempts = defaults.AISettings.MaxRecoveryAttempts
	}
	if config.AISettings.PartialResponseThreshold == 0 {
		config.AISettings.PartialResponseThreshold = defaults.AISettings.PartialResponseThreshold
	}

	// Note: Boolean fields (DeduplicationEnabled, ProcessNitpickComments, EnableJSONRecovery, LogTruncatedResponses) default to true
	// ProcessSelfReviews defaults to false for backward compatibility
	// Set defaults if the config appears to be missing the new fields (old or empty config)
	if isOldConfig && !config.AISettings.DeduplicationEnabled {
		config.AISettings.DeduplicationEnabled = defaults.AISettings.DeduplicationEnabled
	}
	if isOldConfig && !config.AISettings.ProcessNitpickComments {
		config.AISettings.ProcessNitpickComments = defaults.AISettings.ProcessNitpickComments
	}
	if isOldConfig && !config.AISettings.EnableJSONRecovery {
		config.AISettings.EnableJSONRecovery = defaults.AISettings.EnableJSONRecovery
	}
	if isOldConfig && !config.AISettings.LogTruncatedResponses {
		config.AISettings.LogTruncatedResponses = defaults.AISettings.LogTruncatedResponses
	}
	// ProcessSelfReviews is not set for old configs to maintain backward compatibility (defaults to false)

	// Merge boolean pointer fields
	if config.AISettings.ValidationEnabled == nil {
		config.AISettings.ValidationEnabled = defaults.AISettings.ValidationEnabled
	}

	// Merge verification settings
	if config.VerificationSettings.BuildCommand == "" {
		config.VerificationSettings.BuildCommand = defaults.VerificationSettings.BuildCommand
	}
	if config.VerificationSettings.TestCommand == "" {
		config.VerificationSettings.TestCommand = defaults.VerificationSettings.TestCommand
	}
	if config.VerificationSettings.LintCommand == "" {
		config.VerificationSettings.LintCommand = defaults.VerificationSettings.LintCommand
	}
	if config.VerificationSettings.FormatCommand == "" {
		config.VerificationSettings.FormatCommand = defaults.VerificationSettings.FormatCommand
	}
	if config.VerificationSettings.CustomRules == nil {
		config.VerificationSettings.CustomRules = make(map[string]string)
	}
	if len(config.VerificationSettings.MandatoryChecks) == 0 {
		config.VerificationSettings.MandatoryChecks = defaults.VerificationSettings.MandatoryChecks
	}
	if len(config.VerificationSettings.OptionalChecks) == 0 {
		config.VerificationSettings.OptionalChecks = defaults.VerificationSettings.OptionalChecks
	}
	if config.VerificationSettings.TimeoutMinutes == 0 {
		config.VerificationSettings.TimeoutMinutes = defaults.VerificationSettings.TimeoutMinutes
	}

	// Merge update check settings
	if config.UpdateCheck.IntervalHours == 0 {
		config.UpdateCheck.IntervalHours = defaults.UpdateCheck.IntervalHours
	}

	// Note: VerboseMode is NOT merged with defaults - explicit false values are preserved
	// The JSON unmarshaling process preserves explicit false values from config file
	// Only missing fields get default values during initial config creation
}

// Save saves the configuration to file
func (c *Config) Save() error {
	return save(c)
}

// GetPriorityPrompt returns the full priority context for AI analysis
func (c *Config) GetPriorityPrompt() string {
	prompt := "Priority Guidelines for Task Generation:\n\n"

	prompt += fmt.Sprintf("CRITICAL: %s\n", c.PriorityRules.Critical)
	if c.ProjectSpecific.Critical != "" {
		prompt += fmt.Sprintf("Project-specific critical: %s\n", c.ProjectSpecific.Critical)
	}

	prompt += fmt.Sprintf("\nHIGH: %s\n", c.PriorityRules.High)
	if c.ProjectSpecific.High != "" {
		prompt += fmt.Sprintf("Project-specific high: %s\n", c.ProjectSpecific.High)
	}

	prompt += fmt.Sprintf("\nMEDIUM: %s\n", c.PriorityRules.Medium)
	if c.ProjectSpecific.Medium != "" {
		prompt += fmt.Sprintf("Project-specific medium: %s\n", c.ProjectSpecific.Medium)
	}

	prompt += fmt.Sprintf("\nLOW: %s\n", c.PriorityRules.Low)
	if c.ProjectSpecific.Low != "" {
		prompt += fmt.Sprintf("Project-specific low: %s\n", c.ProjectSpecific.Low)
	}

	return prompt
}

// CreateDefault creates the default configuration file
func CreateDefault() error {
	config := defaultConfig()
	return save(config)
}
