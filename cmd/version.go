package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
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

// Function variables for testing
var changeToVersion = changeToVersionImpl

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

// changeToVersionImpl handles changing to a different version (implementation)
func changeToVersionImpl(targetVersion string) error {
	// Validate version argument
	if err := validateVersionArgument(targetVersion); err != nil {
		return err
	}

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
	archiveData, err := updater.DownloadBinary(ctx, targetVersion, osName, arch)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("download failed: %w", err)
	}
	fmt.Printf("✓\n")

	fmt.Printf("Verifying checksum... ")

	// Verify checksum
	err = updater.VerifyChecksum(ctx, targetVersion, osName, arch, archiveData)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	fmt.Printf("✓\n")

	fmt.Printf("Extracting binary... ")

	// Extract binary from tar.gz
	binaryData, err := updater.ExtractBinaryFromTarGz(archiveData, osName)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	fmt.Printf("✓\n")

	fmt.Printf("Installing version %s... ", targetVersion)

	// Backup current binary
	err = version.BackupCurrentBinary(currentBinaryPath, backupPath)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Set up rollback on failure
	rollbackOnError := true
	defer func() {
		if rollbackOnError {
			// Restore from backup if something went wrong
			if restoreErr := version.RestoreFromBackup(backupPath, currentBinaryPath); restoreErr != nil {
				fmt.Printf("\n❌ Failed to restore backup: %v\n", restoreErr)
				fmt.Printf("Manual restore required from: %s\n", backupPath)
			} else {
				fmt.Printf("\n🔄 Restored previous version from backup\n")
			}
		}
		// Clean up backup
		os.Remove(backupPath)
	}()

	// Perform atomic replacement
	err = version.AtomicReplace(currentBinaryPath, binaryData)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Validate new binary
	err = version.ValidateNewBinary(currentBinaryPath)
	if err != nil {
		fmt.Printf("❌\n")
		return fmt.Errorf("binary validation failed: %w", err)
	}

	// Success - don't rollback
	rollbackOnError = false
	fmt.Printf("✓\n")

	fmt.Printf("\n✅ Version change completed successfully!\n")
	fmt.Printf("Updated to reviewtask version %s\n", targetVersion)

	return nil
}

// validateVersionArgument validates the version argument format
func validateVersionArgument(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Allow 'latest' as a special case
	if version == "latest" {
		return nil
	}

	// Check if it starts with 'v' and remove it for validation
	cleanVersion := version
	if strings.HasPrefix(version, "v") {
		cleanVersion = version[1:]
	}

	// Basic semantic version validation
	parts := strings.Split(cleanVersion, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid version format '%s': expected format v1.2.3 or 1.2.3", version)
	}

	// Validate each part is a number
	for i, part := range parts {
		// Remove prerelease suffix for validation (e.g., 1.0.0-beta1)
		if i == 2 && strings.Contains(part, "-") {
			part = strings.Split(part, "-")[0]
		}

		if part == "" {
			return fmt.Errorf("invalid version format '%s': empty version part", version)
		}

		// Check if it's a valid number
		for _, char := range part {
			if char < '0' || char > '9' {
				return fmt.Errorf("invalid version format '%s': non-numeric version part '%s'", version, part)
			}
		}
	}

	return nil
}
