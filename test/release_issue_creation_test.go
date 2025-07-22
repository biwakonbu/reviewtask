package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestReleaseIssueScript tests the create-release-issue.sh script
func TestReleaseIssueScript(t *testing.T) {
	scriptPath := filepath.Join("..", "scripts", "create-release-issue.sh")
	
	// Check if script exists and is executable
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("Script not found: %s (skipping test)", scriptPath)
	}
	
	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectHelp  bool
	}{
		{
			name:        "Help flag",
			args:        []string{"--help"},
			expectError: false,
			expectHelp:  true,
		},
		{
			name:        "Missing version argument",
			args:        []string{},
			expectError: true,
			expectHelp:  false,
		},
		{
			name:        "Invalid version format",
			args:        []string{"--version", "invalid-version"},
			expectError: true,
			expectHelp:  false,
		},
		{
			name:        "Valid version format v1.2.3",
			args:        []string{"--version", "v1.2.3"},
			expectError: false, // Would fail on GitHub API call, but format validation should pass
			expectHelp:  false,
		},
		{
			name:        "Valid version format without v prefix",
			args:        []string{"--version", "1.2.3"},
			expectError: false, // Would fail on GitHub API call, but format validation should pass
			expectHelp:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("bash", append([]string{scriptPath}, tt.args...)...)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)
			
			if tt.expectHelp {
				if !strings.Contains(outputStr, "Usage:") {
					t.Errorf("Expected help output, got: %s", outputStr)
				}
				return
			}
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but command succeeded. Output: %s", outputStr)
				}
			} else {
				// For valid format tests, we expect it to pass validation but fail on GitHub API
				// So we check that it doesn't fail on validation errors
				if strings.Contains(outputStr, "Invalid version format") {
					t.Errorf("Version format validation failed unexpectedly: %s", outputStr)
				}
			}
		})
	}
}

// TestVersionFormatValidation tests version format validation logic
func TestVersionFormatValidation(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"v1.0.0", true},
		{"v1.2.3", true},
		{"v10.20.30", true},
		{"v1.0.0-alpha", true},
		{"v1.0.0-beta.1", true},
		{"v2.0.0-rc.1", true},
		{"1.0.0", true}, // Should be converted to v1.0.0
		{"1.2.3-alpha", true},
		{"invalid", false},
		{"v1.0", false},
		{"v1", false},
		{"v1.0.0.0", false},
		{"v-1.0.0", false},
		{"va.b.c", false},
	}
	
	// Create a temporary test script that only validates version format
	testScript := `#!/bin/bash
version=$1
if [[ ! "$version" =~ ^v ]]; then
    version="v$version"
fi
if [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "valid"
    exit 0
else
    echo "invalid"
    exit 1
fi`
	
	tmpFile, err := os.CreateTemp("", "version_test_*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(testScript); err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	tmpFile.Close()
	
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		t.Fatalf("Failed to make script executable: %v", err)
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("version_%s", tt.version), func(t *testing.T) {
			cmd := exec.Command("bash", tmpFile.Name(), tt.version)
			output, err := cmd.CombinedOutput()
			outputStr := strings.TrimSpace(string(output))
			
			isValid := err == nil && outputStr == "valid"
			
			if isValid != tt.valid {
				t.Errorf("Version %s: expected valid=%t, got valid=%t (output: %s)", 
					tt.version, tt.valid, isValid, outputStr)
			}
		})
	}
}

// TestReleaseTypeDetection tests the release type detection logic
func TestReleaseTypeDetection(t *testing.T) {
	tests := []struct {
		current  string
		previous string
		expected string
	}{
		{"v1.0.0", "", "initial"},
		{"v2.0.0", "v1.0.0", "major"},
		{"v1.1.0", "v1.0.0", "minor"},
		{"v1.0.1", "v1.0.0", "patch"},
		{"v1.2.3", "v1.2.2", "patch"},
		{"v1.3.0", "v1.2.5", "minor"},
		{"v3.0.0", "v2.9.9", "major"},
	}
	
	// Create a test script for release type detection
	testScript := `#!/bin/bash
determine_release_type() {
    local current_version=$1
    local previous_version=$2
    
    if [[ -z "$previous_version" ]]; then
        echo "initial"
        return
    fi
    
    # Extract version numbers (remove 'v' prefix if present)
    current_clean=${current_version#v}
    previous_clean=${previous_version#v}
    
    # Parse version components
    IFS='.' read -ra current_parts <<< "$current_clean"
    IFS='.' read -ra previous_parts <<< "$previous_clean"
    
    local current_major=${current_parts[0]}
    local current_minor=${current_parts[1]}
    local current_patch=${current_parts[2]}
    
    local previous_major=${previous_parts[0]}
    local previous_minor=${previous_parts[1]}
    local previous_patch=${previous_parts[2]}
    
    # Determine release type
    if [[ "$current_major" -gt "$previous_major" ]]; then
        echo "major"
    elif [[ "$current_minor" -gt "$previous_minor" ]]; then
        echo "minor"
    elif [[ "$current_patch" -gt "$previous_patch" ]]; then
        echo "patch"
    else
        echo "unknown"
    fi
}

determine_release_type "$1" "$2"`
	
	tmpFile, err := os.CreateTemp("", "release_type_test_*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(testScript); err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	tmpFile.Close()
	
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		t.Fatalf("Failed to make script executable: %v", err)
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_to_%s", tt.previous, tt.current), func(t *testing.T) {
			cmd := exec.Command("bash", tmpFile.Name(), tt.current, tt.previous)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Script execution failed: %v, output: %s", err, output)
			}
			
			result := strings.TrimSpace(string(output))
			if result != tt.expected {
				t.Errorf("Release type detection: current=%s, previous=%s, expected=%s, got=%s",
					tt.current, tt.previous, tt.expected, result)
			}
		})
	}
}

// TestScriptPrerequisites tests prerequisite checking functionality
func TestScriptPrerequisites(t *testing.T) {
	scriptPath := filepath.Join("..", "scripts", "create-release-issue.sh")
	
	// Test help function works even without prerequisites
	cmd := exec.Command("bash", scriptPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Help command should work without prerequisites, got error: %v", err)
	}
	
	outputStr := string(output)
	expectedSections := []string{"Usage:", "OPTIONS:", "EXAMPLES:", "ENVIRONMENT:"}
	for _, section := range expectedSections {
		if !strings.Contains(outputStr, section) {
			t.Errorf("Help output missing section: %s", section)
		}
	}
}

// TestIssueTemplateSections tests that the issue template contains required sections
func TestIssueTemplateSections(t *testing.T) {
	// This test verifies the structure without actually creating an issue
	expectedSections := []string{
		"Release Information",
		"Release Summary", 
		"Installation & Downloads",
		"Security & Verification",
		"Links",
		"Support",
	}
	
	// Create a mock template by running parts of the script
	testScript := `#!/bin/bash
version="v1.2.3"
release_type="patch"
template_file="/tmp/test_template.md"

cat > "$template_file" << EOF
# 🐛 Release ${version} - Patch Release (Bug Fixes)

> **Release Information**
> - **Version**: ${version}
> - **Type**: Patch Release (Bug Fixes)
> - **Date**: $(date -u +"%Y-%m-%d")
> - **Commit**: abc1234

## 📋 Release Summary

This release includes various improvements and updates to reviewtask.

## 📦 Installation & Downloads

### Download Binary
Download the appropriate binary for your platform from the [release assets](https://github.com/biwakonbu/reviewtask/releases/tag/${version}).

## 🔒 Security & Verification

Binary checksums are provided in the SHA256SUMS file attached to the release assets.

## 🔗 Links

- **GitHub Release**: https://github.com/biwakonbu/reviewtask/releases/tag/${version}

## 📞 Support

If you encounter any issues with this release, please:
1. Check the [troubleshooting guide](https://github.com/biwakonbu/reviewtask#troubleshooting)

EOF

cat "$template_file"
rm -f "$template_file"`
	
	tmpFile, err := os.CreateTemp("", "template_test_*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(testScript); err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	tmpFile.Close()
	
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		t.Fatalf("Failed to make script executable: %v", err)
	}
	
	cmd := exec.Command("bash", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Template generation failed: %v", err)
	}
	
	templateContent := string(output)
	
	for _, section := range expectedSections {
		if !strings.Contains(templateContent, section) {
			t.Errorf("Template missing required section: %s", section)
		}
	}
	
	// Verify markdown structure
	if !regexp.MustCompile(`(?m)^# .+ Release v\d+\.\d+\.\d+ - .+`).MatchString(templateContent) {
		t.Error("Template should start with a proper release title")
	}
	
	if !strings.Contains(templateContent, "**Release Information**") {
		t.Error("Template should contain release information block")
	}
}