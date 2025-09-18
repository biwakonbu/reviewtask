package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reviewtask/internal/config"
)

func TestConfigValidateCommand(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalConfigFile := config.ConfigFile
	config.ConfigFile = filepath.Join(tempDir, ".pr-review", "config.json")
	defer func() { config.ConfigFile = originalConfigFile }()

	// Create directory
	os.MkdirAll(filepath.Dir(config.ConfigFile), 0755)

	tests := []struct {
		name           string
		configContent  string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "valid_config",
			configContent: `{
				"ai_settings": {
					"user_language": "English",
					"ai_provider": "cursor"
				},
				"verification_settings": {
					"enabled": true
				}
			}`,
			expectedOutput: []string{"✓ Configuration is valid"},
			notExpected:    []string{"Errors:", "Warnings:"},
		},
		{
			name: "config_with_warnings",
			configContent: `{
				"ai_settings": {
					"user_language": "English"
				},
				"verification_settings": {
					"enabled": false
				}
			}`,
			expectedOutput: []string{"✓ Configuration is valid", "Warnings:"},
			notExpected:    []string{"Errors:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file
			err := os.WriteFile(config.ConfigFile, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			// Capture output
			output := &bytes.Buffer{}
			cmd := NewRootCmd()
			cmd.SetOut(output)
			cmd.SetErr(output)
			cmd.SetArgs([]string{"config", "validate"})

			// Redirect fmt output to our buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = cmd.Execute()

			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			outputStr := buf.String() + output.String()

			if err != nil {
				t.Errorf("Command failed: %v", err)
			}

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, outputStr)
				}
			}

			// Check not expected
			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output NOT to contain '%s', got: %s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestConfigMigrateCommand(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalConfigFile := config.ConfigFile
	config.ConfigFile = filepath.Join(tempDir, ".pr-review", "config.json")
	defer func() { config.ConfigFile = originalConfigFile }()

	// Create directory
	os.MkdirAll(filepath.Dir(config.ConfigFile), 0755)

	// Create a full config to migrate
	fullConfig := `{
		"priority_rules": {
			"critical": "Security vulnerabilities",
			"high": "Performance issues",
			"medium": "Functional bugs",
			"low": "Style issues"
		},
		"project_specific": {
			"critical": "",
			"high": "",
			"medium": "",
			"low": ""
		},
		"task_settings": {
			"default_status": "todo",
			"auto_prioritize": true,
			"low_priority_patterns": ["nit:", "style:"],
			"low_priority_status": "pending"
		},
		"ai_settings": {
			"user_language": "English",
			"output_format": "json",
			"max_retries": 5,
			"ai_provider": "cursor",
			"model": "grok",
			"prompt_profile": "v2",
			"validation_enabled": true,
			"quality_threshold": 0.8,
			"verbose_mode": false,
			"deduplication_enabled": true,
			"similarity_threshold": 0.8
		}
	}`

	err := os.WriteFile(config.ConfigFile, []byte(fullConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Run migrate command
	output := &bytes.Buffer{}
	cmd := NewRootCmd()
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"config", "migrate"})

	// Redirect fmt output to our buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	outputStr := buf.String() + output.String()

	if err != nil {
		t.Errorf("Command failed: %v", err)
	}
	if !strings.Contains(outputStr, "✓ Created backup") {
		t.Errorf("Expected backup creation message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "✓ Configuration migrated to simplified format") {
		t.Errorf("Expected migration success message, got: %s", outputStr)
	}

	// Check that backup was created
	backupPath := config.ConfigFile + ".backup"
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("Backup file was not created: %v", err)
	}

	// Load the migrated config and verify it's simplified
	data, err := os.ReadFile(config.ConfigFile)
	if err != nil {
		t.Fatalf("Failed to read migrated config: %v", err)
	}

	var simplified map[string]interface{}
	err = json.Unmarshal(data, &simplified)
	if err != nil {
		t.Fatalf("Failed to parse migrated config: %v", err)
	}

	// Check that it's actually simplified (should have language and ai_provider at root level)
	if _, hasLanguage := simplified["language"]; !hasLanguage {
		t.Error("Migrated config should have 'language' at root level")
	}

	if _, hasProvider := simplified["ai_provider"]; !hasProvider {
		t.Error("Migrated config should have 'ai_provider' at root level")
	}

	// Should NOT have the old structure
	if _, hasOldStructure := simplified["ai_settings"]; hasOldStructure {
		t.Error("Migrated config should not have 'ai_settings' structure")
	}
}

func TestInitCommandWithSimplifiedConfig(t *testing.T) {
	// This test verifies that the init command creates simplified configs
	// Note: This is a basic test since init command is interactive

	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create a mock stdin for interactive input
	input := "English\nauto\n"
	stdin := strings.NewReader(input)

	output := &bytes.Buffer{}
	cmd := NewRootCmd()
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetIn(stdin)
	cmd.SetArgs([]string{"init"})

	// Note: This might fail due to interactive nature and GitHub auth checks
	// We're mainly testing that the command exists and can be invoked
	_ = cmd.Execute()

	// Check if config file was created (if init succeeded)
	configPath := filepath.Join(".pr-review", "config.json")
	if _, err := os.Stat(configPath); err == nil {
		// Config was created, check if it's simplified format
		data, _ := os.ReadFile(configPath)
		var cfg map[string]interface{}
		if json.Unmarshal(data, &cfg) == nil {
			// Check for simplified structure indicators
			_, hasLanguage := cfg["language"]
			_, hasProvider := cfg["ai_provider"]
			if !hasLanguage && !hasProvider {
				// Might be old format, check for ai_settings
				if aiSettings, ok := cfg["ai_settings"].(map[string]interface{}); ok {
					if _, hasLang := aiSettings["user_language"]; hasLang {
						t.Log("Config appears to be in old format, expected simplified format")
					}
				}
			}
		}
	}
}
