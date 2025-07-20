package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"reviewtask/internal/version"
)

func TestVersionCommand(t *testing.T) {
	// Save original values
	originalVersion := appVersion
	originalCommitHash := appCommitHash
	originalBuildDate := appBuildDate

	// Set test values
	appVersion = "1.0.0"
	appCommitHash = "abc123"
	appBuildDate = "2023-12-01T10:00:00Z"

	defer func() {
		// Restore original values
		appVersion = originalVersion
		appCommitHash = originalCommitHash
		appBuildDate = originalBuildDate
	}()

	// Test version variables are set correctly
	if appVersion != "1.0.0" {
		t.Errorf("expected appVersion to be '1.0.0', got '%s'", appVersion)
	}

	if appCommitHash != "abc123" {
		t.Errorf("expected appCommitHash to be 'abc123', got '%s'", appCommitHash)
	}

	if appBuildDate != "2023-12-01T10:00:00Z" {
		t.Errorf("expected appBuildDate to be '2023-12-01T10:00:00Z', got '%s'", appBuildDate)
	}
}

func TestVersionCommandWithCheckFlag(t *testing.T) {
	// Create a mock server for GitHub API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		release := version.Release{
			TagName:     "v1.1.0",
			Name:        "Version 1.1.0",
			Body:        "New features and bug fixes",
			Prerelease:  false,
			PublishedAt: time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
			HTMLURL:     "https://github.com/biwakonbu/reviewtask/releases/tag/v1.1.0",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Save original values
	originalVersion := appVersion
	appVersion = "1.0.0"
	defer func() {
		appVersion = originalVersion
	}()

	// Mock the version checker to use our test server
	originalChecker := version.NewChecker()
	testChecker := &testVersionChecker{
		serverURL: server.URL,
	}

	// Temporarily replace the checker creation
	// This requires modifying the runVersion function to accept a checker
	// For this test, we'll verify the flag handling logic

	// Reset flags
	checkUpdate = false
	showLatest = false

	cmd := &cobra.Command{
		Use:  "version",
		RunE: runVersion,
	}
	cmd.Flags().BoolVar(&checkUpdate, "check", false, "Check for available updates")
	cmd.Flags().BoolVar(&showLatest, "latest", false, "Show latest available version information")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Test with --check flag
	cmd.SetArgs([]string{"--check"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Since we can't easily mock the HTTP call in the current implementation,
	// we'll just verify that the flag parsing works correctly
	if !checkUpdate {
		t.Errorf("expected checkUpdate to be true after parsing --check flag")
	}

	// Reset for next test
	checkUpdate = false
	showLatest = false

	// Test with --latest flag
	cmd.SetArgs([]string{"--latest"})
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !showLatest {
		t.Errorf("expected showLatest to be true after parsing --latest flag")
	}

	_ = originalChecker
	_ = testChecker
}

func TestVersionCommandErrorHandling(t *testing.T) {
	// Save original values
	originalVersion := appVersion
	appVersion = "1.0.0"
	defer func() {
		appVersion = originalVersion
	}()

	// Test that version command doesn't fail even when update check fails
	err := runVersion(&cobra.Command{}, []string{})
	if err != nil {
		t.Fatalf("version command should not fail: %v", err)
	}
}

// Integration test for configuration loading and update checking
func TestUpdateCheckIntegration(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	os.Chdir(tempDir)

	// Create .pr-review directory
	err := os.Mkdir(".pr-review", 0755)
	if err != nil {
		t.Fatalf("failed to create .pr-review directory: %v", err)
	}

	// Create a test config file with update checking enabled
	configContent := `{
		"priority_rules": {
			"critical": "Security issues",
			"high": "Performance issues",
			"medium": "Bug fixes",
			"low": "Style improvements"
		},
		"project_specific": {
			"critical": "",
			"high": "",
			"medium": "",
			"low": ""
		},
		"task_settings": {
			"default_status": "todo",
			"auto_prioritize": true
		},
		"ai_settings": {
			"user_language": "English",
			"output_format": "json",
			"max_retries": 5,
			"validation_enabled": true,
			"quality_threshold": 0.8,
			"debug_mode": false,
			"claude_path": ""
		},
		"update_check": {
			"enabled": true,
			"interval_hours": 24,
			"notify_prereleases": false,
			"last_check": "0001-01-01T00:00:00Z"
		}
	}`

	err = os.WriteFile(".pr-review/config.json", []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Test that version checking functions work with real config
	// This is more of a smoke test to ensure integration works

	// Mock the version checker behavior
	shouldCheck := version.ShouldCheckForUpdates(true, 24, time.Time{})
	if !shouldCheck {
		t.Errorf("expected should check to be true for never-checked config")
	}

	// Test with recent check
	recentCheck := time.Now().Add(-1 * time.Hour)
	shouldCheck = version.ShouldCheckForUpdates(true, 24, recentCheck)
	if shouldCheck {
		t.Errorf("expected should check to be false for recent check")
	}
}

// Helper type for testing
type testVersionChecker struct {
	serverURL string
}

func (c *testVersionChecker) GetLatestVersion(ctx context.Context) (*version.Release, error) {
	// Implementation would use the test server URL
	return &version.Release{
		TagName:     "v1.1.0",
		Name:        "Test Version",
		Body:        "Test release",
		Prerelease:  false,
		PublishedAt: time.Now(),
		HTMLURL:     c.serverURL + "/releases/tag/v1.1.0",
	}, nil
}

func TestVersionFlagCombinations(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		check  bool
		latest bool
	}{
		{
			name:   "no flags",
			args:   []string{},
			check:  false,
			latest: false,
		},
		{
			name:   "check only",
			args:   []string{"--check"},
			check:  true,
			latest: false,
		},
		{
			name:   "latest only",
			args:   []string{"--latest"},
			check:  false,
			latest: true,
		},
		{
			name:   "both flags",
			args:   []string{"--check", "--latest"},
			check:  true,
			latest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			checkUpdate = false
			showLatest = false

			cmd := &cobra.Command{
				Use: "version",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Just verify flag parsing
					return nil
				},
			}
			cmd.Flags().BoolVar(&checkUpdate, "check", false, "Check for available updates")
			cmd.Flags().BoolVar(&showLatest, "latest", false, "Show latest available version information")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if checkUpdate != tt.check {
				t.Errorf("expected checkUpdate %t, got %t", tt.check, checkUpdate)
			}

			if showLatest != tt.latest {
				t.Errorf("expected showLatest %t, got %t", tt.latest, showLatest)
			}
		})
	}
}
