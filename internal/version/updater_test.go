package version

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	os, arch := DetectPlatform()

	// Should return valid OS names
	validOS := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
	}

	if !validOS[os] {
		t.Errorf("DetectPlatform() returned invalid OS: %s", os)
	}

	// Should return valid arch names
	validArch := map[string]bool{
		"amd64": true,
		"arm64": true,
	}

	if !validArch[arch] {
		t.Errorf("DetectPlatform() returned invalid arch: %s", arch)
	}

	// Should match current runtime
	if os != runtime.GOOS {
		t.Errorf("DetectPlatform() OS mismatch: got %s, expected %s", os, runtime.GOOS)
	}

	// Arch should be normalized or match
	expectedArch := runtime.GOARCH
	if expectedArch != "amd64" && expectedArch != "arm64" {
		expectedArch = "amd64" // Fallback
	}

	if arch != expectedArch {
		t.Errorf("DetectPlatform() arch mismatch: got %s, expected %s", arch, expectedArch)
	}
}

func TestGetBinaryName(t *testing.T) {
	tests := []struct {
		os       string
		expected string
	}{
		{"windows", "reviewtask.exe"},
		{"linux", "reviewtask"},
		{"darwin", "reviewtask"},
		{"freebsd", "reviewtask"}, // Fallback case
	}

	for _, test := range tests {
		result := GetBinaryName(test.os)
		if result != test.expected {
			t.Errorf("GetBinaryName(%s) = %s, expected %s", test.os, result, test.expected)
		}
	}
}

func TestNewBinaryUpdater(t *testing.T) {
	updater := NewBinaryUpdater()

	if updater.owner != "biwakonbu" {
		t.Errorf("Expected owner 'biwakonbu', got %s", updater.owner)
	}

	if updater.repo != "reviewtask" {
		t.Errorf("Expected repo 'reviewtask', got %s", updater.repo)
	}

	if updater.timeout.Seconds() != 30 {
		t.Errorf("Expected timeout 30s, got %v", updater.timeout)
	}
}

func TestGetAssetURL(t *testing.T) {
	updater := NewBinaryUpdater()

	tests := []struct {
		version  string
		os       string
		arch     string
		expected string
	}{
		{
			"v1.2.3", "linux", "amd64",
			"https://github.com/biwakonbu/reviewtask/releases/download/v1.2.3/reviewtask-v1.2.3-linux-amd64.tar.gz",
		},
		{
			"1.2.3", "darwin", "arm64",
			"https://github.com/biwakonbu/reviewtask/releases/download/v1.2.3/reviewtask-v1.2.3-darwin-arm64.tar.gz",
		},
		{
			"v2.0.0", "windows", "amd64",
			"https://github.com/biwakonbu/reviewtask/releases/download/v2.0.0/reviewtask-v2.0.0-windows-amd64.tar.gz",
		},
	}

	for _, test := range tests {
		result := updater.GetAssetURL(test.version, test.os, test.arch)
		if result != test.expected {
			t.Errorf("GetAssetURL(%s, %s, %s) = %s, expected %s",
				test.version, test.os, test.arch, result, test.expected)
		}
	}
}

func TestGetChecksumURL(t *testing.T) {
	updater := NewBinaryUpdater()

	tests := []struct {
		version  string
		expected string
	}{
		{
			"v1.2.3",
			"https://github.com/biwakonbu/reviewtask/releases/download/v1.2.3/checksums.txt",
		},
		{
			"1.2.3",
			"https://github.com/biwakonbu/reviewtask/releases/download/v1.2.3/checksums.txt",
		},
	}

	for _, test := range tests {
		result := updater.GetChecksumURL(test.version)
		if result != test.expected {
			t.Errorf("GetChecksumURL(%s) = %s, expected %s", test.version, result, test.expected)
		}
	}
}

func createMockTarGz(t *testing.T, filename string, content []byte) []byte {
	var buf bytes.Buffer

	// Create gzip writer
	gzWriter := gzip.NewWriter(&buf)

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)

	// Add file to tar
	header := &tar.Header{
		Name: filename,
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}

	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	// Close tar writer
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}

	// Close gzip writer
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	return buf.Bytes()
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	updater := NewBinaryUpdater()

	// Mock binary content
	mockBinary := []byte("mock binary content")

	// Test Linux extraction
	linuxTarGz := createMockTarGz(t, "reviewtask", mockBinary)

	extracted, err := updater.ExtractBinaryFromTarGz(linuxTarGz, "linux")
	if err != nil {
		t.Fatalf("ExtractBinaryFromTarGz failed: %v", err)
	}

	if !bytes.Equal(extracted, mockBinary) {
		t.Errorf("Extracted binary content mismatch: got %s, expected %s", string(extracted), string(mockBinary))
	}

	// Test Windows extraction
	windowsTarGz := createMockTarGz(t, "reviewtask.exe", mockBinary)

	extracted, err = updater.ExtractBinaryFromTarGz(windowsTarGz, "windows")
	if err != nil {
		t.Fatalf("ExtractBinaryFromTarGz failed: %v", err)
	}

	if !bytes.Equal(extracted, mockBinary) {
		t.Errorf("Extracted binary content mismatch: got %s, expected %s", string(extracted), string(mockBinary))
	}

	// Test binary not found
	emptyTarGz := createMockTarGz(t, "other-file", mockBinary)

	_, err = updater.ExtractBinaryFromTarGz(emptyTarGz, "linux")
	if err == nil {
		t.Error("ExtractBinaryFromTarGz should fail when binary not found")
	}
}

func TestBackupCurrentBinary(t *testing.T) {
	// Create a temporary file to simulate current binary
	tempDir := t.TempDir()
	currentBinary := filepath.Join(tempDir, "reviewtask")
	backupBinary := filepath.Join(tempDir, "reviewtask.backup")

	// Write mock binary content
	mockContent := []byte("mock binary content")
	if err := os.WriteFile(currentBinary, mockContent, 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Test backup
	err := BackupCurrentBinary(currentBinary, backupBinary)
	if err != nil {
		t.Fatalf("BackupCurrentBinary failed: %v", err)
	}

	// Verify backup exists and has correct content
	backupContent, err := os.ReadFile(backupBinary)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if !bytes.Equal(backupContent, mockContent) {
		t.Errorf("Backup content mismatch: got %s, expected %s", string(backupContent), string(mockContent))
	}

	// Verify backup has executable permissions
	info, err := os.Stat(backupBinary)
	if err != nil {
		t.Fatalf("Failed to stat backup file: %v", err)
	}

	if info.Mode()&0755 != 0755 {
		t.Errorf("Backup file permissions incorrect: got %v, expected 0755", info.Mode())
	}
}

func TestRestoreFromBackup(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	backupBinary := filepath.Join(tempDir, "reviewtask.backup")
	targetBinary := filepath.Join(tempDir, "reviewtask")

	// Write mock backup content
	mockContent := []byte("backup binary content")
	if err := os.WriteFile(backupBinary, mockContent, 0755); err != nil {
		t.Fatalf("Failed to create mock backup: %v", err)
	}

	// Test restore
	err := RestoreFromBackup(backupBinary, targetBinary)
	if err != nil {
		t.Fatalf("RestoreFromBackup failed: %v", err)
	}

	// Verify target exists and has correct content
	targetContent, err := os.ReadFile(targetBinary)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if !bytes.Equal(targetContent, mockContent) {
		t.Errorf("Restored content mismatch: got %s, expected %s", string(targetContent), string(mockContent))
	}

	// Verify target has executable permissions
	info, err := os.Stat(targetBinary)
	if err != nil {
		t.Fatalf("Failed to stat target file: %v", err)
	}

	if info.Mode()&0755 != 0755 {
		t.Errorf("Target file permissions incorrect: got %v, expected 0755", info.Mode())
	}
}

func TestAtomicReplace(t *testing.T) {
	// Create temporary directory and binary
	tempDir := t.TempDir()
	targetBinary := filepath.Join(tempDir, "reviewtask")

	// Create original binary
	originalContent := []byte("original binary content")
	if err := os.WriteFile(targetBinary, originalContent, 0755); err != nil {
		t.Fatalf("Failed to create original binary: %v", err)
	}

	// Test atomic replacement
	newContent := []byte("new binary content")
	err := AtomicReplace(targetBinary, newContent)
	if err != nil {
		t.Fatalf("AtomicReplace failed: %v", err)
	}

	// Verify content was replaced
	actualContent, err := os.ReadFile(targetBinary)
	if err != nil {
		t.Fatalf("Failed to read replaced binary: %v", err)
	}

	if !bytes.Equal(actualContent, newContent) {
		t.Errorf("Replaced content mismatch: got %s, expected %s", string(actualContent), string(newContent))
	}

	// Verify permissions are correct
	info, err := os.Stat(targetBinary)
	if err != nil {
		t.Fatalf("Failed to stat replaced binary: %v", err)
	}

	if info.Mode()&0755 != 0755 {
		t.Errorf("Replaced binary permissions incorrect: got %v, expected 0755", info.Mode())
	}
}

func TestValidateNewBinary(t *testing.T) {
	// Create temporary directory and binary
	tempDir := t.TempDir()
	validBinary := filepath.Join(tempDir, "reviewtask")

	// Create valid binary
	if err := os.WriteFile(validBinary, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("Failed to create valid binary: %v", err)
	}

	// Test valid binary
	err := ValidateNewBinary(validBinary)
	if err != nil {
		t.Errorf("ValidateNewBinary failed for valid binary: %v", err)
	}

	// Test non-existent binary
	err = ValidateNewBinary(filepath.Join(tempDir, "nonexistent"))
	if err == nil {
		t.Error("ValidateNewBinary should fail for non-existent binary")
	}

	// Test non-executable binary
	nonExecBinary := filepath.Join(tempDir, "non-executable")
	if err := os.WriteFile(nonExecBinary, []byte("mock binary"), 0644); err != nil {
		t.Fatalf("Failed to create non-executable binary: %v", err)
	}

	err = ValidateNewBinary(nonExecBinary)
	if err == nil {
		t.Error("ValidateNewBinary should fail for non-executable binary")
	}
}

func TestFindChecksumForFile(t *testing.T) {
	updater := NewBinaryUpdater()

	checksumContent := `abc123  reviewtask-v1.2.3-linux-amd64.tar.gz
def456  reviewtask-v1.2.3-darwin-amd64.tar.gz  
ghi789  reviewtask-v1.2.3-windows-amd64.tar.gz`

	tests := []struct {
		filename string
		expected string
	}{
		{"reviewtask-v1.2.3-linux-amd64.tar.gz", "abc123"},
		{"reviewtask-v1.2.3-darwin-amd64.tar.gz", "def456"},
		{"reviewtask-v1.2.3-windows-amd64.tar.gz", "ghi789"},
		{"nonexistent-file.tar.gz", ""},
	}

	for _, test := range tests {
		result := updater.findChecksumForFile(checksumContent, test.filename)
		if result != test.expected {
			t.Errorf("findChecksumForFile(%s) = %s, expected %s", test.filename, result, test.expected)
		}
	}
}
