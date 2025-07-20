package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// VersionComparison represents the result of comparing two versions
type VersionComparison int

const (
	VersionNewer VersionComparison = iota // Current version is newer than compared version
	VersionSame                           // Versions are the same
	VersionOlder                          // Current version is older than compared version
)

// Release represents a GitHub release
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
}

// VersionChecker interface for dependency injection in tests
type VersionChecker interface {
	GetLatestVersion(ctx context.Context) (*Release, error)
	CompareVersions(current, latest string) (VersionComparison, error)
	CheckAndNotify(ctx context.Context, currentVersion string, notifyPrereleases bool) (string, error)
}

// Checker handles version checking operations
type Checker struct {
	owner string
	repo  string
}

// NewChecker creates a new version checker
func NewChecker() *Checker {
	return &Checker{
		owner: "biwakonbu",
		repo:  "reviewtask",
	}
}

// GetLatestVersion fetches the latest release from GitHub
func (c *Checker) GetLatestVersion(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", c.owner, c.repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "reviewtask-version-checker")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

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

// CompareVersions compares two semantic versions
func (c *Checker) CompareVersions(current, latest string) (VersionComparison, error) {
	currentVersion, err := parseVersion(current)
	if err != nil {
		return VersionSame, fmt.Errorf("failed to parse current version: %w", err)
	}

	latestVersion, err := parseVersion(latest)
	if err != nil {
		return VersionSame, fmt.Errorf("failed to parse latest version: %w", err)
	}

	result := compareSemanticVersions(currentVersion, latestVersion)
	switch result {
	case 1:
		return VersionNewer, nil
	case 0:
		return VersionSame, nil
	case -1:
		return VersionOlder, nil
	default:
		return VersionSame, nil
	}
}

// semanticVersion represents a parsed semantic version
type semanticVersion struct {
	major int
	minor int
	patch int
}

// parseVersion parses a semantic version string
func parseVersion(version string) (*semanticVersion, error) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Handle development version
	if version == "dev" {
		return &semanticVersion{major: 999, minor: 999, patch: 999}, nil
	}

	// Handle prerelease versions by removing prerelease suffix
	if strings.Contains(version, "-") {
		version = strings.Split(version, "-")[0]
	}

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s (expected major.minor.patch)", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return &semanticVersion{
		major: major,
		minor: minor,
		patch: patch,
	}, nil
}

// compareSemanticVersions compares two semantic versions
// Returns: 1 if a > b, 0 if a == b, -1 if a < b
func compareSemanticVersions(a, b *semanticVersion) int {
	if a.major != b.major {
		if a.major > b.major {
			return 1
		}
		return -1
	}

	if a.minor != b.minor {
		if a.minor > b.minor {
			return 1
		}
		return -1
	}

	if a.patch != b.patch {
		if a.patch > b.patch {
			return 1
		}
		return -1
	}

	return 0
}

// ShouldCheckForUpdates determines if an update check is needed based on configuration
func ShouldCheckForUpdates(enabled bool, intervalHours int, lastCheck time.Time) bool {
	if !enabled {
		return false
	}

	if lastCheck.IsZero() {
		return true // Never checked before
	}

	elapsed := time.Since(lastCheck)
	return elapsed >= time.Duration(intervalHours)*time.Hour
}

// CheckAndNotify performs an update check and returns notification message if update available
func (c *Checker) CheckAndNotify(ctx context.Context, currentVersion string, notifyPrereleases bool) (string, error) {
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

