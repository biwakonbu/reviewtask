package cmd

import (
	"context"
	"fmt"
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
	// This would typically use a paginated API call to get multiple releases
	// For now, we'll use a simplified approach that gets the latest and mock others
	
	checker := version.NewChecker(0)
	latest, err := checker.GetLatestVersion(ctx)
	if err != nil {
		return nil, err
	}

	// Return just the latest for now
	// TODO: Implement proper GitHub API pagination to get multiple releases
	return []*version.Release{latest}, nil
}

// truncateReleaseNotes truncates release notes to specified length
func truncateReleaseNotes(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	// Truncate and add ellipsis
	return text[:maxLength-3] + "..."
}