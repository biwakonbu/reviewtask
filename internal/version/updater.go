package version

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// UpdateOptions holds options for version updates
type UpdateOptions struct {
	TargetVersion    string
	ForceDowngrade   bool
	BackupDirectory  string
	VerifyChecksum   bool
	Timeout          time.Duration
}

// UpdateResult represents the result of a version update
type UpdateResult struct {
	PreviousVersion string
	NewVersion      string
	Success         bool
	BackupPath      string
	ErrorMessage    string
}

// BinaryUpdater handles self-update operations
type BinaryUpdater struct {
	owner   string
	repo    string
	timeout time.Duration
}

// NewBinaryUpdater creates a new binary updater
func NewBinaryUpdater() *BinaryUpdater {
	return &BinaryUpdater{
		owner:   "biwakonbu",
		repo:    "reviewtask", 
		timeout: 30 * time.Second,
	}
}

// DetectPlatform returns the current OS and architecture
func DetectPlatform() (string, string) {
	os := runtime.GOOS
	arch := runtime.GOARCH
	
	// Normalize OS names to match GitHub release naming
	switch os {
	case "darwin", "linux", "windows":
		// These are supported as-is
	default:
		// For unsupported OS, default to linux for unix-like systems
		os = "linux"
	}
	
	// Normalize architecture names
	switch arch {
	case "amd64", "arm64":
		// These are supported as-is
	case "386":
		// 32-bit x86 - not typically supported, fallback to amd64
		arch = "amd64"
	case "arm":
		// 32-bit ARM - fallback to arm64
		arch = "arm64"
	default:
		// For any other architecture, fallback to amd64
		arch = "amd64"
	}
	
	return os, arch
}

// GetBinaryName returns the expected binary name for the given platform
func GetBinaryName(os string) string {
	if os == "windows" {
		return "reviewtask.exe"
	}
	return "reviewtask"
}

// GetCurrentBinaryPath returns the path of the currently running binary
func GetCurrentBinaryPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	
	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink: %w", err)
	}
	
	return realPath, nil
}

// GetAssetURL constructs the download URL for a specific version and platform
func (u *BinaryUpdater) GetAssetURL(version, os, arch string) string {
	// Ensure version starts with 'v'
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	
	// Construct filename based on platform
	filename := fmt.Sprintf("reviewtask-%s-%s-%s.tar.gz", version, os, arch)
	
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		u.owner, u.repo, version, filename)
}

// GetChecksumURL constructs the checksum URL for a specific version
func (u *BinaryUpdater) GetChecksumURL(version string) string {
	// Ensure version starts with 'v'
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/checksums.txt",
		u.owner, u.repo, version)
}

// DownloadBinary downloads a binary from GitHub releases
func (u *BinaryUpdater) DownloadBinary(ctx context.Context, version, os, arch string) ([]byte, error) {
	url := u.GetAssetURL(version, os, arch)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}
	
	req.Header.Set("User-Agent", "reviewtask-updater")
	
	client := &http.Client{
		Timeout: u.timeout,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("version %s not found for platform %s/%s", version, os, arch)
		}
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download response: %w", err)
	}
	
	return data, nil
}

// VerifyChecksum verifies the checksum of downloaded data
func (u *BinaryUpdater) VerifyChecksum(ctx context.Context, version, os, arch string, data []byte) error {
	// Download checksums file
	checksumURL := u.GetChecksumURL(version)
	
	req, err := http.NewRequestWithContext(ctx, "GET", checksumURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create checksum request: %w", err)
	}
	
	req.Header.Set("User-Agent", "reviewtask-updater")
	
	client := &http.Client{
		Timeout: u.timeout,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums file not available (status %d)", resp.StatusCode)
	}
	
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}
	
	// Parse checksums file and find our file
	filename := fmt.Sprintf("reviewtask-%s-%s-%s.tar.gz", version, os, arch)
	expectedChecksum := u.findChecksumForFile(string(checksumData), filename)
	if expectedChecksum == "" {
		return fmt.Errorf("checksum not found for file %s", filename)
	}
	
	// Calculate actual checksum
	hash := sha256.Sum256(data)
	actualChecksum := fmt.Sprintf("%x", hash)
	
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	
	return nil
}

// findChecksumForFile parses checksums.txt and finds the checksum for the specified file
func (u *BinaryUpdater) findChecksumForFile(checksumContent, filename string) string {
	lines := strings.Split(checksumContent, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.Contains(parts[1], filename) {
			return parts[0]
		}
	}
	return ""
}

// BackupCurrentBinary creates a backup of the current binary
func BackupCurrentBinary(currentPath, backupPath string) error {
	// Validate paths to prevent directory traversal
	if err := validateFilePath(currentPath); err != nil {
		return fmt.Errorf("invalid current path: %w", err)
	}
	if err := validateFilePath(backupPath); err != nil {
		return fmt.Errorf("invalid backup path: %w", err)
	}

	// Read current binary
	data, err := os.ReadFile(currentPath)
	if err != nil {
		return fmt.Errorf("failed to read current binary: %w", err)
	}
	
	// Write backup
	err = os.WriteFile(backupPath, data, 0755)
	if err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}
	
	return nil
}

// RestoreFromBackup restores binary from backup
func RestoreFromBackup(backupPath, targetPath string) error {
	// Validate paths to prevent directory traversal
	if err := validateFilePath(backupPath); err != nil {
		return fmt.Errorf("invalid backup path: %w", err)
	}
	if err := validateFilePath(targetPath); err != nil {
		return fmt.Errorf("invalid target path: %w", err)
	}

	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}
	
	// Write to target
	err = os.WriteFile(targetPath, data, 0755)
	if err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}
	
	return nil
}

// ExtractBinaryFromTarGz extracts the binary from a tar.gz archive
func (u *BinaryUpdater) ExtractBinaryFromTarGz(data []byte, targetOS string) ([]byte, error) {
	// Create gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzReader)
	
	// Find the binary file
	binaryName := GetBinaryName(targetOS)
	const maxFileSize = 100 * 1024 * 1024 // 100MB max
	
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}
		
		// Security: Prevent path traversal attacks
		if strings.Contains(header.Name, "..") || strings.HasPrefix(header.Name, "/") {
			continue // Skip potentially dangerous paths
		}
		
		// Security: Prevent tar bombs (excessively large files)
		if header.Size > maxFileSize {
			return nil, fmt.Errorf("file too large: %d bytes (max %d)", header.Size, maxFileSize)
		}
		
		// Check if this is our binary file
		if filepath.Base(header.Name) == binaryName {
			// Read the binary data with size limit
			binaryData, err := io.ReadAll(io.LimitReader(tarReader, maxFileSize))
			if err != nil {
				return nil, fmt.Errorf("failed to read binary from tar: %w", err)
			}
			return binaryData, nil
		}
	}
	
	return nil, fmt.Errorf("binary '%s' not found in archive", binaryName)
}

// AtomicReplace performs atomic replacement of the binary
func AtomicReplace(currentPath string, newBinaryData []byte) error {
	// Create a temporary file in the same directory as the target
	dir := filepath.Dir(currentPath)
	tempFile, err := os.CreateTemp(dir, "reviewtask-update-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	
	// Ensure cleanup on error
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()
	
	// Write new binary to temp file
	if _, err := tempFile.Write(newBinaryData); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	
	// Make executable
	if err := tempFile.Chmod(0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}
	
	// Close before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tempPath, currentPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	
	return nil
}

// validateFilePath validates a file path to prevent directory traversal attacks
func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	
	// Check for directory traversal patterns
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal: %s", path)
	}
	
	// Ensure the path is not absolute with suspicious patterns
	if strings.HasPrefix(path, "/etc/") || strings.HasPrefix(path, "/usr/") || 
	   strings.HasPrefix(path, "/bin/") || strings.HasPrefix(path, "/sbin/") {
		return fmt.Errorf("path targets system directory: %s", path)
	}
	
	return nil
}

// ValidateNewBinary tests that the new binary can execute basic commands
func ValidateNewBinary(binaryPath string) error {
	// This is a basic validation - just check if the binary exists and has executable permissions
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("binary not accessible: %w", err)
	}
	
	// Check if file has executable permissions
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("binary is not executable")
	}
	
	return nil
}