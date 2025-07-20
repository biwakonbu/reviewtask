package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expected    *semanticVersion
		expectError bool
	}{
		{
			name:     "valid version",
			version:  "1.2.3",
			expected: &semanticVersion{major: 1, minor: 2, patch: 3},
		},
		{
			name:     "valid version with v prefix",
			version:  "v2.0.1",
			expected: &semanticVersion{major: 2, minor: 0, patch: 1},
		},
		{
			name:     "dev version",
			version:  "dev",
			expected: &semanticVersion{major: 999, minor: 999, patch: 999},
		},
		{
			name:        "invalid format",
			version:     "1.2",
			expectError: true,
		},
		{
			name:        "invalid major",
			version:     "a.2.3",
			expectError: true,
		},
		{
			name:        "invalid minor",
			version:     "1.b.3",
			expectError: true,
		},
		{
			name:        "invalid patch",
			version:     "1.2.c",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVersion(tt.version)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result.major != tt.expected.major || result.minor != tt.expected.minor || result.patch != tt.expected.patch {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestCompareSemanticVersions(t *testing.T) {
	tests := []struct {
		name     string
		a        *semanticVersion
		b        *semanticVersion
		expected int
	}{
		{
			name:     "a > b (major)",
			a:        &semanticVersion{major: 2, minor: 0, patch: 0},
			b:        &semanticVersion{major: 1, minor: 9, patch: 9},
			expected: 1,
		},
		{
			name:     "a < b (major)",
			a:        &semanticVersion{major: 1, minor: 9, patch: 9},
			b:        &semanticVersion{major: 2, minor: 0, patch: 0},
			expected: -1,
		},
		{
			name:     "a > b (minor)",
			a:        &semanticVersion{major: 1, minor: 2, patch: 0},
			b:        &semanticVersion{major: 1, minor: 1, patch: 9},
			expected: 1,
		},
		{
			name:     "a < b (minor)",
			a:        &semanticVersion{major: 1, minor: 1, patch: 9},
			b:        &semanticVersion{major: 1, minor: 2, patch: 0},
			expected: -1,
		},
		{
			name:     "a > b (patch)",
			a:        &semanticVersion{major: 1, minor: 1, patch: 2},
			b:        &semanticVersion{major: 1, minor: 1, patch: 1},
			expected: 1,
		},
		{
			name:     "a < b (patch)",
			a:        &semanticVersion{major: 1, minor: 1, patch: 1},
			b:        &semanticVersion{major: 1, minor: 1, patch: 2},
			expected: -1,
		},
		{
			name:     "a == b",
			a:        &semanticVersion{major: 1, minor: 2, patch: 3},
			b:        &semanticVersion{major: 1, minor: 2, patch: 3},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSemanticVersions(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	checker := NewChecker()
	
	tests := []struct {
		name        string
		current     string
		latest      string
		expected    VersionComparison
		expectError bool
	}{
		{
			name:     "current newer",
			current:  "1.2.0",
			latest:   "1.1.0",
			expected: VersionNewer,
		},
		{
			name:     "current same",
			current:  "1.2.0",
			latest:   "1.2.0",
			expected: VersionSame,
		},
		{
			name:     "current older",
			current:  "1.1.0",
			latest:   "1.2.0",
			expected: VersionOlder,
		},
		{
			name:     "dev version newer",
			current:  "dev",
			latest:   "1.2.0",
			expected: VersionNewer,
		},
		{
			name:        "invalid current version",
			current:     "invalid",
			latest:      "1.2.0",
			expectError: true,
		},
		{
			name:        "invalid latest version",
			current:     "1.2.0",
			latest:      "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checker.CompareVersions(tt.current, tt.latest)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestShouldCheckForUpdates(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twentyFourHoursAgo := now.Add(-24 * time.Hour)
	twentyFiveHoursAgo := now.Add(-25 * time.Hour)
	
	tests := []struct {
		name          string
		enabled       bool
		intervalHours int
		lastCheck     time.Time
		expected      bool
	}{
		{
			name:          "disabled",
			enabled:       false,
			intervalHours: 24,
			lastCheck:     oneHourAgo,
			expected:      false,
		},
		{
			name:          "never checked",
			enabled:       true,
			intervalHours: 24,
			lastCheck:     time.Time{},
			expected:      true,
		},
		{
			name:          "checked recently",
			enabled:       true,
			intervalHours: 24,
			lastCheck:     oneHourAgo,
			expected:      false,
		},
		{
			name:          "checked exactly at interval",
			enabled:       true,
			intervalHours: 24,
			lastCheck:     twentyFourHoursAgo,
			expected:      true, // exactly at boundary should trigger check
		},
		{
			name:          "checked past interval",
			enabled:       true,
			intervalHours: 24,
			lastCheck:     twentyFiveHoursAgo,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldCheckForUpdates(tt.enabled, tt.intervalHours, tt.lastCheck)
			if result != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/biwakonbu/reviewtask/releases/latest" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("expected Accept header 'application/vnd.github.v3+json', got '%s'", r.Header.Get("Accept"))
		}
		
		if r.Header.Get("User-Agent") != "reviewtask-version-checker" {
			t.Errorf("expected User-Agent header 'reviewtask-version-checker', got '%s'", r.Header.Get("User-Agent"))
		}

		release := Release{
			TagName:     "v1.2.3",
			Name:        "Version 1.2.3",
			Body:        "Bug fixes and improvements",
			Prerelease:  false,
			PublishedAt: time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
			HTMLURL:     "https://github.com/biwakonbu/reviewtask/releases/tag/v1.2.3",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Create checker with test server URL
	checker := &testChecker{
		serverURL: server.URL,
	}

	ctx := context.Background()
	release, err := checker.GetLatestVersion(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if release.TagName != "v1.2.3" {
		t.Errorf("expected tag name 'v1.2.3', got '%s'", release.TagName)
	}
	
	if release.Name != "Version 1.2.3" {
		t.Errorf("expected name 'Version 1.2.3', got '%s'", release.Name)
	}
	
	if release.Prerelease {
		t.Errorf("expected prerelease to be false, got true")
	}
}

// testChecker implements VersionChecker for testing
type testChecker struct {
	serverURL string
}

func (c *testChecker) GetLatestVersion(ctx context.Context) (*Release, error) {
	url := c.serverURL + "/repos/biwakonbu/reviewtask/releases/latest"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "reviewtask-version-checker")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &release, nil
}

func (c *testChecker) CompareVersions(current, latest string) (VersionComparison, error) {
	checker := NewChecker()
	return checker.CompareVersions(current, latest)
}

func (c *testChecker) CheckAndNotify(ctx context.Context, currentVersion string, notifyPrereleases bool) (string, error) {
	latestVersion, err := c.GetLatestVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to check for updates: %w", err)
	}
	
	// Skip prereleases if not enabled
	if latestVersion.Prerelease && !notifyPrereleases {
		return "", nil
	}
	
	comparison, err := c.CompareVersions(currentVersion, latestVersion.TagName)
	if err != nil {
		return "", fmt.Errorf("failed to compare versions: %w", err)
	}
	
	if comparison == VersionOlder {
		return fmt.Sprintf("✨ Update available: %s → %s\nRelease notes: %s", 
			currentVersion, latestVersion.TagName, latestVersion.HTMLURL), nil
	}
	
	return "", nil
}

func TestCheckAndNotify(t *testing.T) {
	// Mock server for testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := Release{
			TagName:     "v1.2.0",
			Name:        "Version 1.2.0",
			Body:        "New features",
			Prerelease:  false,
			PublishedAt: time.Now(),
			HTMLURL:     "https://github.com/biwakonbu/reviewtask/releases/tag/v1.2.0",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	checker := &testChecker{
		serverURL: server.URL,
	}

	ctx := context.Background()
	
	// Test update available
	notification, err := checker.CheckAndNotify(ctx, "1.1.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if notification == "" {
		t.Errorf("expected notification for available update, got empty string")
	}
	
	if !contains(notification, "Update available") {
		t.Errorf("expected notification to contain 'Update available', got: %s", notification)
	}
	
	// Test no update needed
	notification, err = checker.CheckAndNotify(ctx, "1.2.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if notification != "" {
		t.Errorf("expected no notification for same version, got: %s", notification)
	}
}

func TestCheckAndNotifyWithPrerelease(t *testing.T) {
	// Mock server for prerelease testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := Release{
			TagName:     "v1.3.0-beta1",
			Name:        "Version 1.3.0 Beta 1",
			Body:        "Beta release",
			Prerelease:  true,
			PublishedAt: time.Now(),
			HTMLURL:     "https://github.com/biwakonbu/reviewtask/releases/tag/v1.3.0-beta1",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	checker := &testChecker{
		serverURL: server.URL,
	}

	ctx := context.Background()
	
	// Test with prereleases disabled
	notification, err := checker.CheckAndNotify(ctx, "1.1.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if notification != "" {
		t.Errorf("expected no notification for prerelease when disabled, got: %s", notification)
	}
	
	// Test with prereleases enabled
	notification, err = checker.CheckAndNotify(ctx, "1.1.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if notification == "" {
		t.Errorf("expected notification for prerelease when enabled, got empty string")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}