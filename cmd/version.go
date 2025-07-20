package cmd

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"reviewtask/internal/version"
)

var (
	checkUpdate bool
	showLatest  bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display version, build information, and runtime details for reviewtask.

Options:
  --check      Check for available updates
  --latest     Show latest available version information`,
	RunE: runVersion,
}

func init() {
	versionCmd.Flags().BoolVar(&checkUpdate, "check", false, "Check for available updates")
	versionCmd.Flags().BoolVar(&showLatest, "latest", false, "Show latest available version information")
}

func runVersion(cmd *cobra.Command, args []string) error {
	// Show basic version information
	fmt.Printf("reviewtask version %s\n", appVersion)
	fmt.Printf("Commit: %s\n", appCommitHash)
	fmt.Printf("Built: %s\n", appBuildDate)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Handle --check or --latest flags
	if checkUpdate || showLatest {
		fmt.Println()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		checker := version.NewChecker()
		latestVersion, err := checker.GetLatestVersion(ctx)
		if err != nil {
			fmt.Printf("❌ Failed to check for updates: %v\n", err)
			return nil // Don't fail the command
		}

		if showLatest {
			fmt.Printf("Latest version: %s\n", latestVersion.TagName)
			fmt.Printf("Published: %s\n", latestVersion.PublishedAt.Format("2006-01-02 15:04:05"))
			if latestVersion.Body != "" {
				fmt.Printf("Release notes: %s\n", latestVersion.HTMLURL)
			}
		}

		if checkUpdate {
			comparison, err := checker.CompareVersions(appVersion, latestVersion.TagName)
			if err != nil {
				fmt.Printf("❌ Failed to compare versions: %v\n", err)
				return nil
			}

			switch comparison {
			case version.VersionNewer:
				fmt.Printf("✨ Update available: %s → %s\n", appVersion, latestVersion.TagName)
				fmt.Printf("Release notes: %s\n", latestVersion.HTMLURL)
			case version.VersionSame:
				fmt.Printf("✅ You're running the latest version!\n")
			case version.VersionOlder:
				fmt.Printf("ℹ️  You're running a development version newer than the latest release\n")
			}
		}
	}

	return nil
}
