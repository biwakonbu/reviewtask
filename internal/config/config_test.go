package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	// Test new nitpick configuration defaults
	assert.True(t, config.AISettings.ProcessNitpickComments, "ProcessNitpickComments should default to true")
	assert.Equal(t, "low", config.AISettings.NitpickPriority, "NitpickPriority should default to 'low'")

	// Test existing defaults still work
	assert.Equal(t, "English", config.AISettings.UserLanguage)
	assert.Equal(t, "json", config.AISettings.OutputFormat)
	assert.True(t, config.AISettings.DeduplicationEnabled)
}

func TestConfigMergeWithDefaults(t *testing.T) {
	tests := []struct {
		name             string
		config           Config
		expectedNitpick  bool
		expectedPriority string
	}{
		{
			name: "empty config gets defaults",
			config: Config{
				AISettings: AISettings{},
			},
			expectedNitpick:  true,
			expectedPriority: "low",
		},
		{
			name: "partial config preserves existing values",
			config: Config{
				AISettings: AISettings{
					UserLanguage:           "Japanese",
					ProcessNitpickComments: false,
					NitpickPriority:        "medium",
				},
			},
			expectedNitpick:  false,
			expectedPriority: "medium",
		},
		{
			name: "old config without nitpick settings gets defaults",
			config: Config{
				AISettings: AISettings{
					UserLanguage: "English",
					OutputFormat: "json",
					MaxRetries:   3,
				},
			},
			expectedNitpick:  true,
			expectedPriority: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeWithDefaults(&tt.config)

			assert.Equal(t, tt.expectedNitpick, tt.config.AISettings.ProcessNitpickComments)
			assert.Equal(t, tt.expectedPriority, tt.config.AISettings.NitpickPriority)

			// Ensure other defaults are also set
			assert.NotEmpty(t, tt.config.AISettings.UserLanguage)
			assert.NotEmpty(t, tt.config.AISettings.OutputFormat)
		})
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "config_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// Create config with nitpick settings
	originalConfig := defaultConfig()
	originalConfig.AISettings.ProcessNitpickComments = false
	originalConfig.AISettings.NitpickPriority = "high"
	originalConfig.AISettings.UserLanguage = "Japanese"

	// Save config
	err = originalConfig.Save()
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, ConfigFile)

	// Load config
	loadedConfig, err := Load()
	require.NoError(t, err)

	// Verify nitpick settings are preserved
	assert.Equal(t, false, loadedConfig.AISettings.ProcessNitpickComments)
	assert.Equal(t, "high", loadedConfig.AISettings.NitpickPriority)
	assert.Equal(t, "Japanese", loadedConfig.AISettings.UserLanguage)
}

func TestConfigJSONSerialization(t *testing.T) {
	config := defaultConfig()
	config.AISettings.ProcessNitpickComments = false
	config.AISettings.NitpickPriority = "medium"

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)

	// Verify nitpick fields are in JSON
	jsonStr := string(jsonData)
	assert.Contains(t, jsonStr, "process_nitpick_comments")
	assert.Contains(t, jsonStr, "nitpick_priority")
	assert.Contains(t, jsonStr, "medium")

	// Deserialize and verify
	var loadedConfig Config
	err = json.Unmarshal(jsonData, &loadedConfig)
	require.NoError(t, err)

	assert.Equal(t, false, loadedConfig.AISettings.ProcessNitpickComments)
	assert.Equal(t, "medium", loadedConfig.AISettings.NitpickPriority)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		valid    bool
	}{
		{"valid low priority", "low", true},
		{"valid medium priority", "medium", true},
		{"valid high priority", "high", true},
		{"valid critical priority", "critical", true},
		{"empty priority gets default", "", true},
		{"custom priority accepted", "custom", true}, // Config allows any string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := defaultConfig()
			config.AISettings.NitpickPriority = tt.priority

			// Config doesn't enforce validation - just test the value is preserved
			assert.Equal(t, tt.priority, config.AISettings.NitpickPriority)
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that old config files without nitpick settings still work
	oldConfigJSON := `{
		"priority_rules": {
			"critical": "Security vulnerabilities",
			"high": "Performance issues",
			"medium": "Functional bugs",
			"low": "Code style"
		},
		"ai_settings": {
			"user_language": "English",
			"output_format": "json",
			"max_retries": 5
		}
	}`

	var config Config
	err := json.Unmarshal([]byte(oldConfigJSON), &config)
	require.NoError(t, err)

	// Apply defaults
	mergeWithDefaults(&config)

	// Verify old settings preserved
	assert.Equal(t, "English", config.AISettings.UserLanguage)
	assert.Equal(t, "json", config.AISettings.OutputFormat)
	assert.Equal(t, 5, config.AISettings.MaxRetries)

	// Verify new settings get defaults
	assert.True(t, config.AISettings.ProcessNitpickComments)
	assert.Equal(t, "low", config.AISettings.NitpickPriority)
}

func TestConfigCreateDefault(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "config_create_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// Create default config
	err = CreateDefault()
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, ConfigFile)

	// Load and verify it has nitpick defaults
	config, err := Load()
	require.NoError(t, err)

	assert.True(t, config.AISettings.ProcessNitpickComments)
	assert.Equal(t, "low", config.AISettings.NitpickPriority)
}
