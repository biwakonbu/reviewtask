package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"reviewtask/internal/version"
)

var (
	checkUpdate bool
	showLatest  bool
)

var versionCmd = &cobra.Command{
	Use:   "version [VERSION]",
	Short: "Show version information or change to a different version",
	Long: `Display version, build information, and runtime details for reviewtask.
When VERSION is provided, change to the specified version.

Arguments:
  VERSION      Version to change to (e.g., v1.2.3, latest)

Options:
  --check      Check for available updates

Examples:
  reviewtask version           # Show current version and check for updates
  reviewtask version v1.2.3    # Change to version v1.2.3
  reviewtask version latest    # Change to latest version`,
	RunE: runVersion,
}

func init() {
	versionCmd.Flags().BoolVar(&checkUpdate, "check", false, "Check for available updates")
}

func runVersion(cmd *cobra.Command, args []string) error {
	// If version argument provided, handle version change
	if len(args) > 0 {
		targetVersion := args[0]
		return changeToVersion(targetVersion)
	}

	// Show basic version information
	fmt.Printf("reviewtask version %s\n", appVersion)
	fmt.Printf("Commit: %s\n", appCommitHash)
	fmt.Printf("Built: %s\n", appBuildDate)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Always check for updates when showing version info
	fmt.Println()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	checker := version.NewChecker(0)
	latestVersion, err := checker.GetLatestVersion(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to check for updates: %v\n", err)
		return nil // Don't fail the command
	}

	comparison, err := checker.CompareVersions(appVersion, latestVersion.TagName)
	if err != nil {
		fmt.Printf("❌ Failed to compare versions: %v\n", err)
		return nil
	}

	switch comparison {
	case version.VersionOlder:
		fmt.Printf("Latest version: %s\n", latestVersion.TagName)
		fmt.Printf("\nUpdate available! Run 'reviewtask version latest' to upgrade.\n")
	case version.VersionSame:
		fmt.Printf("✅ You're running the latest version!\n")
	case version.VersionNewer:
		fmt.Printf("ℹ️  You're running a development version newer than the latest release\n")
		fmt.Printf("Latest stable version: %s\n", latestVersion.TagName)
	}

	return nil
}

// changeToVersion handles changing to a different version
func changeToVersion(targetVersion string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	checker := version.NewChecker(0)
	updater := version.NewBinaryUpdater()
	
	// Handle 'latest' version
	if targetVersion == "latest" {
		latestVersion, err := checker.GetLatestVersion(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
		targetVersion = latestVersion.TagName
	}
	
	// Show current and target versions
	fmt.Printf("Current version: %s\n", appVersion)
	fmt.Printf("Target version: %s", targetVersion)
	
	// Compare versions to determine if it's upgrade or downgrade
	comparison, err := checker.CompareVersions(appVersion, targetVersion)
	if err != nil {
		return fmt.Errorf("failed to compare versions: %w", err)
	}
	
	switch comparison {
	case version.VersionSame:
		fmt.Printf(" (no change)\n")
		fmt.Printf("✅ You're already running version %s\n", targetVersion)
		return nil
	case version.VersionOlder:
		fmt.Printf(" (upgrade)\n")
	case version.VersionNewer:
		fmt.Printf(" (downgrade)\n")
		fmt.Printf("⚠️  This will downgrade reviewtask. Continue? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("❌ Downgrade cancelled")
			return nil
		}
	}
	
	// Detect platform
	osName, arch := version.DetectPlatform()
	fmt.Printf("Platform: %s/%s\n", osName, arch)
	
	// Get current binary path
	currentBinaryPath, err := version.GetCurrentBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to get current binary path: %w", err)
	}
	
	// Create backup path
	backupPath := currentBinaryPath + ".backup." + appVersion
	
	fmt.Printf("Downloading reviewtask-%s-%s-%s... ", targetVersion, osName, arch)
	
	// Download new binary
	binaryData, err := updater.DownloadBinary(ctx, targetVersion, osName, arch)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("download failed: %w", err)
	}
	fmt.Printf("✓\n")
	
	fmt.Printf("Verifying checksum... ")
	
	// Verify checksum
	err = updater.VerifyChecksum(ctx, targetVersion, osName, arch, binaryData)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	fmt.Printf("✓\n")
	
	// Extract binary from tar.gz (simplified - assuming tar.gz contains single binary)
	// TODO: Implement proper tar.gz extraction
	fmt.Printf("Installing version %s... ", targetVersion)
	
	// Backup current binary
	err = version.BackupCurrentBinary(currentBinaryPath, backupPath)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("failed to backup current binary: %w", err)
	}
	
	// For now, assume binaryData is the raw binary (needs tar.gz extraction implementation)
	// TODO: Extract from tar.gz properly
	fmt.Printf("⚠️  Binary extraction from tar.gz not yet implemented\n")
	fmt.Printf("Please download manually from: %s\n", updater.GetAssetURL(targetVersion, osName, arch))
	
	// Clean up backup on cancellation
	defer func() {
		os.Remove(backupPath)
	}()
	
	return nil
}
