package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"reviewtask/internal/version"
)

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "List recent versions with GitHub releases link",
	Long: `Display recent versions of reviewtask with release information.
Shows the 5 most recent versions with release dates and descriptions.

Examples:
  reviewtask versions         # List recent versions with GitHub link`,
	RunE: runVersions,
}

func runVersions(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("Available recent versions:")
	fmt.Println()

	// Get recent releases from GitHub API
	releases, err := getRecentReleases(ctx, 5)
	if err != nil {
		return fmt.Errorf("failed to get recent versions: %w", err)
	}

	if len(releases) == 0 {
		fmt.Println("No releases found.")
		return nil
	}

	// Display releases
	for i, release := range releases {
		// Mark latest and current versions
		status := ""
		if i == 0 {
			status = " (latest)"
		}
		if release.TagName == appVersion {
			status = " (current)"
		}

		fmt.Printf("%s%s    %s  %s\n",
			release.TagName,
			status,
			release.PublishedAt.Format("2006-01-02"),
			truncateReleaseNotes(release.Name, 50))
	}

	fmt.Println()
	fmt.Printf("For all versions, visit: https://github.com/biwakonbu/reviewtask/releases\n")
	fmt.Println()
	fmt.Printf("To change version: reviewtask version <VERSION>\n")

	return nil
}

// getRecentReleases fetches recent releases from GitHub API
func getRecentReleases(ctx context.Context, count int) ([]*version.Release, error) {
	// Use GitHub API to get multiple releases
	url := fmt.Sprintf("https://api.github.com/repos/biwakonbu/reviewtask/releases?per_page=%d", count)

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
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("no releases found for biwakonbu/reviewtask")
		}
		if resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("GitHub API rate limit exceeded")
		}
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []*version.Release
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return releases, nil
}

// truncateReleaseNotes truncates release notes to specified length
func truncateReleaseNotes(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	// Truncate and add ellipsis
	return text[:maxLength-3] + "..."
}
