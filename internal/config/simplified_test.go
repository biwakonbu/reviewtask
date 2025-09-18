package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSimplifiedConfigLoading(t *testing.T) {
	// Create a temp directory
	tempDir := t.TempDir()
	originalConfigFile := ConfigFile
	ConfigFile = filepath.Join(tempDir, ".pr-review", "config.json")
	defer func() { ConfigFile = originalConfigFile }()

	// Create directory
	os.MkdirAll(filepath.Dir(ConfigFile), 0755)

	tests := []struct {
		name        string
		configJSON  string
		wantError   bool
		checkResult func(*testing.T, *Config)
	}{
		{
			name: "minimal_config",
			configJSON: `{
				"language": "English",
				"ai_provider": "cursor"
			}`,
			wantError: false,
			checkResult: func(t *testing.T, cfg *Config) {
				if cfg.AISettings.UserLanguage != "English" {
					t.Errorf("Expected language to be English, got %s", cfg.AISettings.UserLanguage)
				}
				if cfg.AISettings.AIProvider != "cursor" {
					t.Errorf("Expected AI provider to be cursor, got %s", cfg.AISettings.AIProvider)
				}
				// Should have defaults for other fields
				if cfg.AISettings.MaxRetries != 5 {
					t.Errorf("Expected MaxRetries default to be 5, got %d", cfg.AISettings.MaxRetries)
				}
			},
		},
		{
			name: "basic_config_with_model",
			configJSON: `{
				"language": "Japanese",
				"ai_provider": "claude",
				"model": "opus"
			}`,
			wantError: false,
			checkResult: func(t *testing.T, cfg *Config) {
				if cfg.AISettings.UserLanguage != "Japanese" {
					t.Errorf("Expected language to be Japanese, got %s", cfg.AISettings.UserLanguage)
				}
				if cfg.AISettings.AIProvider != "claude" {
					t.Errorf("Expected AI provider to be claude, got %s", cfg.AISettings.AIProvider)
				}
				if cfg.AISettings.Model != "opus" {
					t.Errorf("Expected model to be opus, got %s", cfg.AISettings.Model)
				}
			},
		},
		{
			name: "config_with_priorities",
			configJSON: `{
				"language": "English",
				"ai_provider": "auto",
				"priorities": {
					"project_specific": {
						"critical": "Security issues",
						"high": "Performance problems"
					}
				}
			}`,
			wantError: false,
			checkResult: func(t *testing.T, cfg *Config) {
				if cfg.ProjectSpecific.Critical != "Security issues" {
					t.Errorf("Expected critical priority to be 'Security issues', got %s", cfg.ProjectSpecific.Critical)
				}
				if cfg.ProjectSpecific.High != "Performance problems" {
					t.Errorf("Expected high priority to be 'Performance problems', got %s", cfg.ProjectSpecific.High)
				}
			},
		},
		{
			name: "full_config_backward_compatible",
			configJSON: `{
				"priority_rules": {
					"critical": "Security vulnerabilities",
					"high": "Performance issues",
					"medium": "Functional bugs",
					"low": "Style issues"
				},
				"ai_settings": {
					"user_language": "English",
					"ai_provider": "cursor",
					"model": "grok"
				}
			}`,
			wantError: false,
			checkResult: func(t *testing.T, cfg *Config) {
				if cfg.AISettings.UserLanguage != "English" {
					t.Errorf("Expected language to be English, got %s", cfg.AISettings.UserLanguage)
				}
				if cfg.PriorityRules.Critical != "Security vulnerabilities" {
					t.Errorf("Expected critical rule to be preserved, got %s", cfg.PriorityRules.Critical)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file
			err := os.WriteFile(ConfigFile, []byte(tt.configJSON), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Load config
			cfg, err := Load()
			if (err != nil) != tt.wantError {
				t.Errorf("Load() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.checkResult != nil {
				tt.checkResult(t, cfg)
			}
		})
	}
}

func TestProjectTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name:     "go_project",
			files:    map[string]string{"go.mod": "module test"},
			expected: "go",
		},
		{
			name:     "node_project",
			files:    map[string]string{"package.json": "{}"},
			expected: "node",
		},
		{
			name:     "rust_project",
			files:    map[string]string{"Cargo.toml": "[package]"},
			expected: "rust",
		},
		{
			name:     "python_project",
			files:    map[string]string{"requirements.txt": "flask==2.0.0"},
			expected: "python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and files
			tempDir := t.TempDir()
			originalWd, _ := os.Getwd()
			defer os.Chdir(originalWd)

			os.Chdir(tempDir)

			for filename, content := range tt.files {
				err := os.WriteFile(filename, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			result := DetectProjectType()
			if result != tt.expected {
				t.Errorf("DetectProjectType() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestGetDefaultCommandsForProject(t *testing.T) {
	tests := []struct {
		projectType string
		wantBuild   string
		wantTest    string
		wantLint    string
	}{
		{
			projectType: "go",
			wantBuild:   "go build ./...",
			wantTest:    "go test ./...",
			wantLint:    "golangci-lint run",
		},
		{
			projectType: "node",
			wantBuild:   "npm run build",
			wantTest:    "npm test",
			wantLint:    "npm run lint",
		},
		{
			projectType: "rust",
			wantBuild:   "cargo build",
			wantTest:    "cargo test",
			wantLint:    "cargo clippy",
		},
		{
			projectType: "python",
			wantBuild:   "python -m py_compile .",
			wantTest:    "pytest",
			wantLint:    "pylint .",
		},
	}

	for _, tt := range tests {
		t.Run(tt.projectType, func(t *testing.T) {
			build, test, lint := GetDefaultCommandsForProject(tt.projectType)
			if build != tt.wantBuild {
				t.Errorf("Build command = %s, want %s", build, tt.wantBuild)
			}
			if test != tt.wantTest {
				t.Errorf("Test command = %s, want %s", test, tt.wantTest)
			}
			if lint != tt.wantLint {
				t.Errorf("Lint command = %s, want %s", lint, tt.wantLint)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	// Create a temp directory
	tempDir := t.TempDir()
	originalConfigFile := ConfigFile
	ConfigFile = filepath.Join(tempDir, ".pr-review", "config.json")
	defer func() { ConfigFile = originalConfigFile }()

	// Create directory
	os.MkdirAll(filepath.Dir(ConfigFile), 0755)

	tests := []struct {
		name            string
		config          *Config
		wantValid       bool
		wantSuggestions int
		wantWarnings    int
		wantErrors      int
	}{
		{
			name: "valid_minimal_config",
			config: &Config{
				AISettings: AISettings{
					UserLanguage: "English",
					AIProvider:   "cursor",
				},
				VerificationSettings: VerificationSettings{
					Enabled: true,
				},
			},
			wantValid:       true,
			wantSuggestions: 0,
			wantWarnings:    0,
			wantErrors:      0,
		},
		{
			name: "config_with_disabled_verification",
			config: &Config{
				AISettings: AISettings{
					UserLanguage: "English",
					AIProvider:   "claude",
				},
				VerificationSettings: VerificationSettings{
					Enabled: false,
				},
			},
			wantValid:       true,
			wantSuggestions: 0,
			wantWarnings:    1, // Should warn about disabled verification
			wantErrors:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save config
			data, _ := json.Marshal(tt.config)
			os.WriteFile(ConfigFile, data, 0644)

			report, err := ValidateConfig()
			if err != nil {
				t.Fatalf("ValidateConfig() error = %v", err)
			}

			if report.IsValid != tt.wantValid {
				t.Errorf("IsValid = %v, want %v", report.IsValid, tt.wantValid)
			}

			if len(report.Suggestions) != tt.wantSuggestions {
				t.Errorf("Suggestions count = %d, want %d", len(report.Suggestions), tt.wantSuggestions)
			}

			if len(report.Warnings) != tt.wantWarnings {
				t.Errorf("Warnings count = %d, want %d", len(report.Warnings), tt.wantWarnings)
			}

			if len(report.Errors) != tt.wantErrors {
				t.Errorf("Errors count = %d, want %d", len(report.Errors), tt.wantErrors)
			}
		})
	}
}

func TestMigrateToSimplified(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   *SimplifiedConfig
	}{
		{
			name: "minimal_migration",
			config: &Config{
				AISettings: AISettings{
					UserLanguage: "English",
					AIProvider:   "auto",
					Model:        "auto",
					MaxRetries:   5, // default value
				},
			},
			want: &SimplifiedConfig{
				Language:   "English",
				AIProvider: "auto",
			},
		},
		{
			name: "migration_with_custom_model",
			config: &Config{
				AISettings: AISettings{
					UserLanguage: "Japanese",
					AIProvider:   "claude",
					Model:        "opus",
				},
			},
			want: &SimplifiedConfig{
				Language:   "Japanese",
				AIProvider: "claude",
				Model:      "opus",
			},
		},
		{
			name: "migration_with_priorities",
			config: &Config{
				AISettings: AISettings{
					UserLanguage: "English",
					AIProvider:   "cursor",
				},
				ProjectSpecific: ProjectSpecific{
					Critical: "Auth bugs",
					High:     "Data loss",
				},
			},
			want: &SimplifiedConfig{
				Language:   "English",
				AIProvider: "cursor",
				Priorities: map[string]interface{}{
					"project_specific": map[string]interface{}{
						"critical": "Auth bugs",
						"high":     "Data loss",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MigrateToSimplified(tt.config)
			if err != nil {
				t.Fatalf("MigrateToSimplified() error = %v", err)
			}

			// Compare results
			if result.Language != tt.want.Language {
				t.Errorf("Language = %s, want %s", result.Language, tt.want.Language)
			}

			if result.AIProvider != tt.want.AIProvider {
				t.Errorf("AIProvider = %s, want %s", result.AIProvider, tt.want.AIProvider)
			}

			if result.Model != tt.want.Model {
				t.Errorf("Model = %s, want %s", result.Model, tt.want.Model)
			}

			// Check priorities if present
			if tt.want.Priorities != nil {
				if result.Priorities == nil {
					t.Error("Expected priorities to be migrated")
				}
			}
		})
	}
}

func TestGetProviderDisplayName(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		want     string
	}{
		{"cursor", "grok", "Cursor CLI (grok)"},
		{"claude", "sonnet", "Claude Code (sonnet)"},
		{"cursor", "auto", "Cursor CLI"},
		{"claude", "", "Claude Code"},
		{"custom", "model", "custom (model)"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.model, func(t *testing.T) {
			result := GetProviderDisplayName(tt.provider, tt.model)
			if result != tt.want {
				t.Errorf("GetProviderDisplayName(%s, %s) = %s, want %s", tt.provider, tt.model, result, tt.want)
			}
		})
	}
}

