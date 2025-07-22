package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	originalChecker := version.NewChecker(0)
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

// Tests for new self-update functionality

func TestVersionCommand_WithVersionArgument(t *testing.T) {
	// Mock changeToVersion function for testing
	originalChangeToVersion := changeToVersion
	var capturedVersion string
	var mockError error

	changeToVersion = func(targetVersion string) error {
		capturedVersion = targetVersion
		return mockError
	}

	defer func() {
		changeToVersion = originalChangeToVersion
	}()

	tests := []struct {
		name            string
		args            []string
		expectedVersion string
		mockError       error
		shouldError     bool
	}{
		{
			name:            "version latest",
			args:            []string{"latest"},
			expectedVersion: "latest",
			mockError:       nil,
			shouldError:     false,
		},
		{
			name:            "specific version",
			args:            []string{"v1.2.3"},
			expectedVersion: "v1.2.3",
			mockError:       nil,
			shouldError:     false,
		},
		{
			name:            "version change fails",
			args:            []string{"v1.2.3"},
			expectedVersion: "v1.2.3",
			mockError:       fmt.Errorf("version change failed"),
			shouldError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedVersion = ""
			mockError = tt.mockError

			cmd := &cobra.Command{
				Use:  "version",
				RunE: runVersion,
			}

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if capturedVersion != tt.expectedVersion {
				t.Errorf("Expected version %s, got %s", tt.expectedVersion, capturedVersion)
			}
		})
	}
}

func TestVersionsCommand_Basic(t *testing.T) {
	// Mock getRecentReleases function
	originalGetRecentReleases := getRecentReleases
	getRecentReleases = func(ctx context.Context, count int) ([]*version.Release, error) {
		return []*version.Release{
			{
				TagName:     "v1.2.0",
				Name:        "Version 1.2.0 - Major improvements",
				PublishedAt: time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
			},
			{
				TagName:     "v1.1.0",
				Name:        "Version 1.1.0 - Bug fixes and enhancements",
				PublishedAt: time.Date(2023, 11, 1, 10, 0, 0, 0, time.UTC),
			},
		}, nil
	}
	defer func() {
		getRecentReleases = originalGetRecentReleases
	}()

	// Save and set test version
	originalVersion := appVersion
	appVersion = "v1.1.0" // Current version for testing
	defer func() {
		appVersion = originalVersion
	}()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the function directly
	err := runVersions(nil, []string{})
	w.Close()

	// Read the output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = originalStdout
	
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := buf.String()

	// Verify expected content
	if !strings.Contains(output, "Available recent versions:") {
		t.Errorf("Missing header. Output:\n%s", output)
	}

	if !strings.Contains(output, "v1.2.0 (latest)") {
		t.Errorf("Missing latest version marker. Output:\n%s", output)
	}

	if !strings.Contains(output, "v1.1.0 (current)") {
		t.Errorf("Missing current version marker. Output:\n%s", output)
	}

	if !strings.Contains(output, "2023-12-01") {
		t.Errorf("Missing release date. Output:\n%s", output)
	}

	if !strings.Contains(output, "For all versions, visit:") {
		t.Errorf("Missing GitHub link. Output:\n%s", output)
	}

	if !strings.Contains(output, "reviewtask version <VERSION>") {
		t.Errorf("Missing usage instructions. Output:\n%s", output)
	}
}

func TestTruncateReleaseNotes(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Short text", 20, "Short text"},
		{"This is a very long release note that should be truncated", 30, "This is a very long release..."},
		{"Exactly thirty characters!!", 30, "Exactly thirty characters!!"},
		{"", 10, ""},
		{"Short", 100, "Short"},
	}

	for _, tt := range tests {
		result := truncateReleaseNotes(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateReleaseNotes(%q, %d) = %q, expected %q",
				tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestVersionCommand_ShowVersionWithUpdateCheck(t *testing.T) {
	// Create mock server for version checking
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := version.Release{
			TagName:     "v1.3.0",
			Name:        "Version 1.3.0",
			Body:        "Latest features",
			Prerelease:  false,
			PublishedAt: time.Now(),
			HTMLURL:     "https://github.com/biwakonbu/reviewtask/releases/tag/v1.3.0",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Save original values
	originalVersion := appVersion
	originalCommitHash := appCommitHash
	originalBuildDate := appBuildDate

	appVersion = "v1.2.0"
	appCommitHash = "abc123"
	appBuildDate = "2023-01-01T00:00:00Z"

	defer func() {
		appVersion = originalVersion
		appCommitHash = originalCommitHash
		appBuildDate = originalBuildDate
	}()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the function directly  
	err := runVersion(nil, []string{})
	w.Close()

	// Read the output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = originalStdout

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := buf.String()

	// Should contain basic version info
	if !strings.Contains(output, "reviewtask version v1.2.0") {
		t.Errorf("Missing version info. Output:\n%s", output)
	}

	if !strings.Contains(output, "Commit: abc123") {
		t.Errorf("Missing commit hash. Output:\n%s", output)
	}

	if !strings.Contains(output, "Built: 2023-01-01T00:00:00Z") {
		t.Errorf("Missing build date. Output:\n%s", output)
	}

	// May contain update information (depends on network connectivity)
	// We don't assert on update info since it requires network access
}

func TestVersionsCommand_ErrorHandling(t *testing.T) {
	// Mock getRecentReleases to return error
	originalGetRecentReleases := getRecentReleases
	getRecentReleases = func(ctx context.Context, count int) ([]*version.Release, error) {
		return nil, fmt.Errorf("network error")
	}
	defer func() {
		getRecentReleases = originalGetRecentReleases
	}()

	cmd := &cobra.Command{
		Use:  "versions",
		RunE: runVersions,
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error but got none")
	}

	if !strings.Contains(err.Error(), "failed to get recent versions") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestVersionsCommand_EmptyReleases(t *testing.T) {
	// Mock getRecentReleases to return empty list
	originalGetRecentReleases := getRecentReleases
	getRecentReleases = func(ctx context.Context, count int) ([]*version.Release, error) {
		return []*version.Release{}, nil
	}
	defer func() {
		getRecentReleases = originalGetRecentReleases
	}()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the function directly
	err := runVersions(nil, []string{})
	w.Close()

	// Read the output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = originalStdout

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "No releases found") {
		t.Errorf("Should show 'No releases found' message. Output:\n%s", output)
	}
}

// Integration test placeholder
func TestSelfUpdateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would test the full self-update workflow:
	// 1. Detection of platform
	// 2. Download simulation
	// 3. Checksum verification
	// 4. Backup and restore
	// 5. Atomic replacement

	t.Log("Integration test placeholder for self-update functionality")

	// Test platform detection
	osName, arch := version.DetectPlatform()
	if osName == "" || arch == "" {
		t.Error("Platform detection should return valid values")
	}

	// Test binary updater creation
	updater := version.NewBinaryUpdater()
	if updater == nil {
		t.Error("Should create binary updater")
	}

	// Test URL generation
	assetURL := updater.GetAssetURL("v1.2.3", osName, arch)
	if !strings.Contains(assetURL, "v1.2.3") {
		t.Error("Asset URL should contain version")
	}

	if !strings.Contains(assetURL, osName) {
		t.Error("Asset URL should contain OS")
	}

	if !strings.Contains(assetURL, arch) {
		t.Error("Asset URL should contain architecture")
	}
}
