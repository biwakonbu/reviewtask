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
	PriorityRules   PriorityRules   `json:"priority_rules"`
	ProjectSpecific ProjectSpecific `json:"project_specific"`
	TaskSettings    TaskSettings    `json:"task_settings"`
	AISettings      AISettings      `json:"ai_settings"`
	UpdateCheck     UpdateCheck     `json:"update_check"`
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
	UserLanguage         string  `json:"user_language"`         // e.g., "Japanese", "English"
	OutputFormat         string  `json:"output_format"`         // "json"
	MaxRetries           int     `json:"max_retries"`           // Validation retry attempts (default: 5)
	ValidationEnabled    *bool   `json:"validation_enabled"`    // Enable two-stage validation
	QualityThreshold     float64 `json:"quality_threshold"`     // Minimum score to accept (0.0-1.0)
	DebugMode            bool    `json:"debug_mode"`            // Enable debug information (PATH, command locations)
	ClaudePath           string  `json:"claude_path"`           // Custom path to Claude CLI (overrides default search)
	MaxTasksPerComment   int     `json:"max_tasks_per_comment"` // Maximum tasks to generate per comment (default: 2)
	DeduplicationEnabled bool    `json:"deduplication_enabled"` // Enable task deduplication (default: true)
	SimilarityThreshold  float64 `json:"similarity_threshold"`  // Task similarity threshold for deduplication (0.0-1.0)
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
			UserLanguage:         "English",
			OutputFormat:         "json",
			MaxRetries:           5,
			ValidationEnabled:    &validationTrue,
			QualityThreshold:     0.8,
			DebugMode:            false,
			ClaudePath:           "", // Empty means use default search paths
			MaxTasksPerComment:   2,
			DeduplicationEnabled: true,
			SimilarityThreshold:  0.8,
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
	// Note: DeduplicationEnabled is a boolean that defaults to true
	// We only set it if the loaded config has zero value (false) and the field is missing
	// This is handled by checking if other fields indicate a partial config
	if !config.AISettings.DeduplicationEnabled && config.AISettings.UserLanguage == "" {
		// If UserLanguage is empty, it likely means this is a partial config or old version
		config.AISettings.DeduplicationEnabled = defaults.AISettings.DeduplicationEnabled
	}

	// Merge boolean pointer fields
	if config.AISettings.ValidationEnabled == nil {
		config.AISettings.ValidationEnabled = defaults.AISettings.ValidationEnabled
	}

	// Merge update check settings
	if config.UpdateCheck.IntervalHours == 0 {
		config.UpdateCheck.IntervalHours = defaults.UpdateCheck.IntervalHours
	}

	// Note: DebugMode is NOT merged with defaults - explicit false values are preserved
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
