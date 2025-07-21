package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNamingUnificationSpecification tests that Issue #31 requirements are met
func TestNamingUnificationSpecification(t *testing.T) {
	t.Skip("Skipping naming unification tests - infrastructure files not yet implemented (see issue #40)")
	
	t.Run("Binary name is reviewtask", testBinaryName)
	t.Run("Module name is reviewtask", testModuleName)
	t.Run("Command references use reviewtask", testCommandReferences)
	t.Run("Documentation uses reviewtask", testDocumentationNaming)
	t.Run("Build system uses reviewtask", testBuildSystemNaming)
}

func testBinaryName(t *testing.T) {
	// Test that build scripts use reviewtask as binary name
	buildScript, err := os.ReadFile("../scripts/build.sh")
	if err != nil {
		t.Fatalf("Failed to read build script: %v", err)
	}

	if !strings.Contains(string(buildScript), `BINARY_NAME="reviewtask"`) {
		t.Error("Build script should use reviewtask as binary name")
	}

	// Test Makefile
	makefile, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatalf("Failed to read Makefile: %v", err)
	}

	if !strings.Contains(string(makefile), "BINARY_NAME=reviewtask") {
		t.Error("Makefile should use reviewtask as binary name")
	}
}

func testModuleName(t *testing.T) {
	// Test go.mod has correct module name
	goMod, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	if !strings.HasPrefix(string(goMod), "module reviewtask") {
		t.Error("go.mod should start with 'module reviewtask'")
	}
}

func testCommandReferences(t *testing.T) {
	// Test that CLI help text uses reviewtask
	cmdFiles := []string{
		"../cmd/root.go",
		"../cmd/version.go",
		"../cmd/auth.go",
		"../cmd/show.go",
		"../cmd/update.go",
		"../cmd/status.go",
		"../cmd/stats.go",
		"../cmd/init.go",
	}

	for _, file := range cmdFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", file, err)
		}

		// Should not contain old command name
		if strings.Contains(string(content), "gh-review-task") {
			t.Errorf("%s should not contain 'gh-review-task' references", file)
		}

		// Should use reviewtask module imports
		if strings.Contains(string(content), `"gh-review-task/`) {
			t.Errorf("%s should not contain 'gh-review-task/' import paths", file)
		}
	}
}

func testDocumentationNaming(t *testing.T) {
	// Test documentation files use reviewtask
	docFiles := map[string]string{
		"../README.md":          "reviewtask - AI-Powered PR Review Management Tool",
		"../PRD.md":             "reviewtask: AI-Powered PR Review Management Tool",
		"../docs/VERSIONING.md": "`reviewtask` follows",
	}

	for file, expectedContent := range docFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue // Skip if file doesn't exist
		}

		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", file, err)
		}

		if !strings.Contains(string(content), expectedContent) {
			t.Errorf("%s should contain '%s'", file, expectedContent)
		}

		// Should not contain old names
		if strings.Contains(string(content), "gh-review-task") {
			t.Errorf("%s should not contain 'gh-review-task' references", file)
		}
	}
}

func testBuildSystemNaming(t *testing.T) {
	// Test GitHub Actions workflow uses reviewtask
	workflowFile := "../.github/workflows/release.yml"
	if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
		t.Skip("Release workflow file not found")
	}

	content, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", workflowFile, err)
	}

	// Should use new binary names in asset paths
	if strings.Contains(string(content), "gh-review-task-${VERSION_CLEAN}") {
		t.Error("GitHub Actions should use reviewtask binary naming")
	}

	// Test version and release scripts
	scriptFiles := []string{
		"../scripts/version.sh",
		"../scripts/release.sh",
	}

	for _, file := range scriptFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", file, err)
		}

		// Should reference reviewtask in comments and documentation
		if strings.Contains(string(content), "gh-review-task") &&
			!strings.Contains(string(content), "reviewtask") {
			t.Errorf("%s should reference reviewtask instead of gh-review-task", file)
		}
	}
}

// TestRepositoryURLConsistency tests that all repository URLs point to new location
func TestRepositoryURLConsistency(t *testing.T) {
	t.Skip("Skipping repository URL consistency tests - infrastructure files not yet implemented (see issue #40)")
	
	// Find all files that might contain repository URLs
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip binary files, directories, test files, git logs, and PR data files
		if info.IsDir() || strings.HasSuffix(path, ".git") || strings.HasSuffix(path, "_test.go") ||
			strings.Contains(path, ".git/") || strings.Contains(path, ".pr-review/") {
			return nil
		}

		// Skip non-text files
		ext := filepath.Ext(path)
		if ext == ".png" || ext == ".jpg" || ext == ".gif" || ext == ".pdf" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Check for old repository URLs
		if strings.Contains(string(content), "github.com/biwakonbu/ai-pr-review-checker") {
			t.Errorf("File %s contains old repository URL", path)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}
}

// TestImportPathConsistency tests that all Go files use correct import paths
func TestImportPathConsistency(t *testing.T) {
	t.Skip("Skipping import path consistency tests - infrastructure files not yet implemented (see issue #40)")
	
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only check Go files, but skip test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Check for old import paths
		if strings.Contains(string(content), `"gh-review-task/`) {
			t.Errorf("Go file %s contains old import path 'gh-review-task/'", path)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}
}
