package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SimplifiedConfig represents the minimal configuration format
type SimplifiedConfig struct {
	// Level 1: Minimal Configuration (just 2 fields)
	Language   string `json:"language,omitempty"`
	AIProvider string `json:"ai_provider,omitempty"`

	// Level 2: Basic Configuration (optional)
	Model      string                 `json:"model,omitempty"`
	Priorities map[string]interface{} `json:"priorities,omitempty"`

	// Level 3: Advanced Configuration (optional)
	AI       map[string]interface{} `json:"ai,omitempty"`
	Advanced map[string]interface{} `json:"advanced,omitempty"`
}

// LoadSimplified loads a simplified config and converts it to full config
func LoadSimplified() (*Config, error) {
	// Try to load simplified config first
	if config, err := tryLoadSimplifiedConfig(); err == nil && config != nil {
		return config, nil
	}

	// Fall back to standard config loading
	return Load()
}

// tryLoadSimplifiedConfig attempts to parse config as simplified format
func tryLoadSimplifiedConfig() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		return nil, err
	}

	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}

	// Try to unmarshal as simplified config
	var simplified SimplifiedConfig
	if err := json.Unmarshal(data, &simplified); err != nil {
		return nil, err
	}

	// Check if this looks like a simplified config
	// (has language/ai_provider but no priority_rules structure)
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		// If we can't parse as generic map, it's not a simplified config
		return nil, err
	}

	_, hasLanguage := rawConfig["language"]
	_, hasProvider := rawConfig["ai_provider"]
	_, hasPriorityRules := rawConfig["priority_rules"]

	if (hasLanguage || hasProvider) && !hasPriorityRules {
		// Convert simplified to full config
		return convertSimplifiedToFull(&simplified), nil
	}

	return nil, fmt.Errorf("not a simplified config")
}

// convertSimplifiedToFull converts a simplified config to full config format
func convertSimplifiedToFull(simplified *SimplifiedConfig) *Config {
	config := defaultConfig()

	// Apply Level 1 settings
	if simplified.Language != "" {
		config.AISettings.UserLanguage = simplified.Language
	}

	if simplified.AIProvider != "" {
		config.AISettings.AIProvider = simplified.AIProvider
	}

	// Apply Level 2 settings
	if simplified.Model != "" {
		config.AISettings.Model = simplified.Model
	}

	if simplified.Priorities != nil {
		applySimplifiedPriorities(config, simplified.Priorities)
	}

	// Apply Level 3 settings (AI)
	if simplified.AI != nil {
		applyAdvancedAISettings(config, simplified.AI)
	}

	// Apply Level 3 settings (Advanced)
	if simplified.Advanced != nil {
		applyAdvancedSettings(config, simplified.Advanced)
	}

	return config
}

// applySimplifiedPriorities applies simplified priority configuration
func applySimplifiedPriorities(config *Config, priorities map[string]interface{}) {
	// Check for project_specific sub-map
	if projectSpecific, ok := priorities["project_specific"].(map[string]interface{}); ok {
		if critical, ok := projectSpecific["critical"].(string); ok {
			config.ProjectSpecific.Critical = critical
		}
		if high, ok := projectSpecific["high"].(string); ok {
			config.ProjectSpecific.High = high
		}
		if medium, ok := projectSpecific["medium"].(string); ok {
			config.ProjectSpecific.Medium = medium
		}
		if low, ok := projectSpecific["low"].(string); ok {
			config.ProjectSpecific.Low = low
		}
	}
}

// applyAdvancedAISettings applies advanced AI configuration
func applyAdvancedAISettings(config *Config, aiSettings map[string]interface{}) {
	if provider, ok := aiSettings["provider"].(string); ok {
		config.AISettings.AIProvider = provider
	}

	if model, ok := aiSettings["model"].(string); ok {
		config.AISettings.Model = model
	}

	if profile, ok := aiSettings["prompt_profile"].(string); ok {
		config.AISettings.PromptProfile = profile
	}

	// Support both "verbose" and "verbose_mode" for backward compatibility
	if verbose, ok := aiSettings["verbose"].(bool); ok {
		config.AISettings.VerboseMode = verbose
	}
	if verboseMode, ok := aiSettings["verbose_mode"].(bool); ok {
		config.AISettings.VerboseMode = verboseMode
	}

	// Support both "validation" and "validation_enabled" for backward compatibility
	if validation, ok := aiSettings["validation"].(bool); ok {
		config.AISettings.ValidationEnabled = &validation
	}
	if validationEnabled, ok := aiSettings["validation_enabled"].(bool); ok {
		config.AISettings.ValidationEnabled = &validationEnabled
	}

	// Additional AI settings
	if streamProcessing, ok := aiSettings["stream_processing_enabled"].(bool); ok {
		config.AISettings.StreamProcessingEnabled = streamProcessing
	}

	if realtimeSaving, ok := aiSettings["realtime_saving_enabled"].(bool); ok {
		config.AISettings.RealtimeSavingEnabled = realtimeSaving
	}

	if skipClaudeAuthCheck, ok := aiSettings["skip_claude_auth_check"].(bool); ok {
		config.AISettings.SkipClaudeAuthCheck = skipClaudeAuthCheck
	}
}

// applyAdvancedSettings applies advanced general configuration
func applyAdvancedSettings(config *Config, advanced map[string]interface{}) {
	if maxRetries, ok := advanced["max_retries"].(float64); ok {
		config.AISettings.MaxRetries = int(maxRetries)
	}

	if timeoutSeconds, ok := advanced["timeout_seconds"].(float64); ok {
		config.VerificationSettings.TimeoutMinutes = int(timeoutSeconds) / 60
	}

	if dedupThreshold, ok := advanced["deduplication_threshold"].(float64); ok {
		config.AISettings.SimilarityThreshold = dedupThreshold
	}
}

// DetectProjectType detects the project type based on files in the repository
func DetectProjectType() string {
	// Check for various project markers
	if fileExists("go.mod") {
		return "go"
	}
	if fileExists("package.json") {
		return "node"
	}
	if fileExists("Cargo.toml") {
		return "rust"
	}
	if fileExists("requirements.txt") || fileExists("setup.py") || fileExists("pyproject.toml") {
		return "python"
	}
	if fileExists("pom.xml") || fileExists("build.gradle") {
		return "java"
	}
	if fileExists("Gemfile") {
		return "ruby"
	}
	if fileExists("composer.json") {
		return "php"
	}
	if fileExists("*.csproj") || fileExists("*.sln") {
		return "dotnet"
	}

	return "generic"
}

// fileExists checks if a file or pattern exists
func fileExists(pattern string) bool {
	// Handle glob patterns
	if strings.Contains(pattern, "*") {
		matches, _ := filepath.Glob(pattern)
		return len(matches) > 0
	}

	// Check single file
	_, err := os.Stat(pattern)
	return err == nil
}

// GetDefaultCommandsForProject returns default commands based on project type
func GetDefaultCommandsForProject(projectType string) (build, test, lint string) {
	switch projectType {
	case "go":
		return "go build ./...", "go test ./...", "golangci-lint run"
	case "node":
		return "npm run build", "npm test", "npm run lint"
	case "rust":
		return "cargo build", "cargo test", "cargo clippy"
	case "python":
		return "python -m py_compile .", "pytest", "pylint ."
	case "java":
		if fileExists("pom.xml") {
			return "mvn compile", "mvn test", "mvn checkstyle:check"
		}
		return "gradle build", "gradle test", "gradle check"
	case "ruby":
		return "bundle install", "bundle exec rspec", "rubocop"
	case "php":
		return "composer install", "phpunit", "phpcs"
	case "dotnet":
		return "dotnet build", "dotnet test", "dotnet format --verify-no-changes"
	default:
		return "make build", "make test", "make lint"
	}
}

// ApplySmartDefaults applies intelligent defaults to configuration
func ApplySmartDefaults(config *Config) {
	// Detect project type and apply appropriate defaults
	projectType := DetectProjectType()

	// Only override if commands are empty or still at Go defaults for non-Go projects
	if projectType != "go" {
		build, test, lint := GetDefaultCommandsForProject(projectType)

		// Check if still using Go defaults in non-Go project
		if config.VerificationSettings.BuildCommand == "go build ./..." {
			config.VerificationSettings.BuildCommand = build
		}
		if config.VerificationSettings.TestCommand == "go test ./..." {
			config.VerificationSettings.TestCommand = test
		}
		if config.VerificationSettings.LintCommand == "golangci-lint run" {
			config.VerificationSettings.LintCommand = lint
		}
	}

	// Apply intelligent AI provider selection
	if config.AISettings.AIProvider == "auto" {
		// Auto-detect available AI providers
		cursorAvailable := CheckCursorAvailable()
		claudeAvailable := CheckClaudeAvailable()

		if cursorAvailable {
			config.AISettings.AIProvider = "cursor"
			if config.AISettings.Model == "auto" || config.AISettings.Model == "" {
				config.AISettings.Model = "grok"
			}
		} else if claudeAvailable {
			config.AISettings.AIProvider = "claude"
			if config.AISettings.Model == "auto" || config.AISettings.Model == "" {
				config.AISettings.Model = "sonnet"
			}
		}
	}
}

// CheckCursorAvailable checks if Cursor CLI is available
func CheckCursorAvailable() bool {
	paths := []string{
		"cursor",
		"/usr/local/bin/cursor",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "cursor"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Check in PATH
	pathEnv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathEnv, ":") {
		cursorPath := filepath.Join(dir, "cursor")
		if _, err := os.Stat(cursorPath); err == nil {
			return true
		}
	}

	return false
}

// CheckClaudeAvailable checks if Claude CLI is available
func CheckClaudeAvailable() bool {
	paths := []string{
		"claude",
		"/usr/local/bin/claude",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "claude"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Check in PATH
	pathEnv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathEnv, ":") {
		claudePath := filepath.Join(dir, "claude")
		if _, err := os.Stat(claudePath); err == nil {
			return true
		}
	}

	return false
}

// GetProviderDisplayName returns the display name for the AI provider
func GetProviderDisplayName(provider, model string) string {
	providerNames := map[string]string{
		"cursor": "Cursor CLI",
		"claude": "Claude Code",
	}

	pName := providerNames[provider]
	if pName == "" {
		pName = provider
	}

	// Use model name exactly as configured
	if model != "" && model != "auto" {
		return fmt.Sprintf("%s (%s)", pName, model)
	}
	return pName
}

// CreateSimplifiedConfig creates a minimal configuration file
func CreateSimplifiedConfig(simplified *SimplifiedConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(ConfigFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(simplified, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile, data, 0644)
}

// ValidateConfig checks the current configuration for issues
func ValidateConfig() (*ValidationReport, error) {
	report := &ValidationReport{
		IsValid:     true,
		Suggestions: []string{},
		Warnings:    []string{},
		Errors:      []string{},
	}

	// Load current config
	config, err := Load()
	if err != nil {
		report.IsValid = false
		report.Errors = append(report.Errors, fmt.Sprintf("Failed to load configuration: %v", err))
		return report, nil
	}

	// Check AI provider configuration
	if config.AISettings.AIProvider == "" || config.AISettings.AIProvider == "auto" {
		cursorAvailable := CheckCursorAvailable()
		claudeAvailable := CheckClaudeAvailable()

		if !cursorAvailable && !claudeAvailable {
			report.Warnings = append(report.Warnings, "No AI providers detected. Please install Cursor CLI or Claude Code.")
		}
	}

	// Check for deprecated or unused settings
	if config.AISettings.PartialResponseThreshold != 0 && config.AISettings.PartialResponseThreshold != 0.7 {
		report.Suggestions = append(report.Suggestions, "Consider removing unused 'partial_response_threshold' setting")
	}

	// Check verification settings
	if !config.VerificationSettings.Enabled {
		report.Warnings = append(report.Warnings, "'verification.enabled' is false, verification commands will be ignored")
	}

	// Check project type match
	projectType := DetectProjectType()
	if projectType == "node" && strings.Contains(config.VerificationSettings.BuildCommand, "go build") {
		report.Suggestions = append(report.Suggestions, "Detected Node.js project but using Go build commands. Consider updating verification commands.")
	}

	return report, nil
}

// ValidationReport contains the results of config validation
type ValidationReport struct {
	IsValid     bool
	Suggestions []string
	Warnings    []string
	Errors      []string
}

// MigrateToSimplified migrates a full config to simplified format
func MigrateToSimplified(config *Config) (*SimplifiedConfig, error) {
	simplified := &SimplifiedConfig{
		Language:   config.AISettings.UserLanguage,
		AIProvider: config.AISettings.AIProvider,
	}

	// Only add model if not default
	if config.AISettings.Model != "" && config.AISettings.Model != "auto" {
		simplified.Model = config.AISettings.Model
	}

	// Only add priorities if customized
	if config.ProjectSpecific.Critical != "" ||
		config.ProjectSpecific.High != "" ||
		config.ProjectSpecific.Medium != "" ||
		config.ProjectSpecific.Low != "" {
		simplified.Priorities = map[string]interface{}{
			"project_specific": map[string]interface{}{},
		}
		projectSpecific := simplified.Priorities["project_specific"].(map[string]interface{})

		if config.ProjectSpecific.Critical != "" {
			projectSpecific["critical"] = config.ProjectSpecific.Critical
		}
		if config.ProjectSpecific.High != "" {
			projectSpecific["high"] = config.ProjectSpecific.High
		}
		if config.ProjectSpecific.Medium != "" {
			projectSpecific["medium"] = config.ProjectSpecific.Medium
		}
		if config.ProjectSpecific.Low != "" {
			projectSpecific["low"] = config.ProjectSpecific.Low
		}
	}

	// Only add advanced settings if customized from defaults
	defaults := defaultConfig()
	needsAdvanced := false
	advanced := make(map[string]interface{})

	if config.AISettings.MaxRetries != defaults.AISettings.MaxRetries {
		advanced["max_retries"] = config.AISettings.MaxRetries
		needsAdvanced = true
	}

	if config.AISettings.SimilarityThreshold != defaults.AISettings.SimilarityThreshold {
		advanced["deduplication_threshold"] = config.AISettings.SimilarityThreshold
		needsAdvanced = true
	}

	if needsAdvanced {
		simplified.Advanced = advanced
	}

	return simplified, nil
}
