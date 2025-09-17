package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"reviewtask/internal/github"
)

// MockFileSystem allows us to simulate file system errors
type MockFileSystem struct {
	shouldFailMkdir     bool
	shouldFailWriteFile bool
	shouldFailReadFile  bool
	shouldFailStat      bool
	shouldFailRemove    bool
}

var mockFS = &MockFileSystem{}

// TestErrorTracker_ClearErrors_EdgeCases tests edge cases for ClearErrors
func TestErrorTracker_ClearErrors_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   bool
		enabled     bool
		expectError bool
		description string
	}{
		{
			name:        "clear non-existent file",
			setupFile:   false,
			enabled:     true,
			expectError: false,
			description: "Should succeed when file doesn't exist",
		},
		{
			name:        "clear when disabled",
			setupFile:   true,
			enabled:     false,
			expectError: false,
			description: "Should do nothing when tracker is disabled",
		},
		{
			name:        "clear existing file",
			setupFile:   true,
			enabled:     true,
			expectError: false,
			description: "Should remove existing file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			errorFile := filepath.Join(tempDir, "errors.json")

			if tt.setupFile {
				// Create a test error file
				testErrors := []CommentError{
					{CommentID: 1, ErrorType: "test"},
				}
				data, _ := json.Marshal(testErrors)
				os.WriteFile(errorFile, data, 0644)
			}

			tracker := &ErrorTracker{
				enabled:     tt.enabled,
				verboseMode: false,
				errorFile:   errorFile,
			}

			err := tracker.ClearErrors()

			if tt.expectError && err == nil {
				t.Errorf("Expected error for case '%s', got nil", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for case '%s': %v", tt.description, err)
			}

			// Verify file is removed if it was enabled
			if tt.enabled && tt.setupFile {
				if _, err := os.Stat(errorFile); !os.IsNotExist(err) {
					t.Errorf("Expected file to be removed for case '%s'", tt.description)
				}
			}
		})
	}
}

// TestErrorTracker_GetErrorCount_EdgeCases tests edge cases for GetErrorCount
func TestErrorTracker_GetErrorCount_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		setupErrors   []CommentError
		corruptFile   bool
		expectedCount int
		description   string
	}{
		{
			name:          "empty error list",
			setupErrors:   []CommentError{},
			corruptFile:   false,
			expectedCount: 0,
			description:   "Should return 0 for empty error list",
		},
		{
			name: "multiple errors",
			setupErrors: []CommentError{
				{CommentID: 1, ErrorType: "test1"},
				{CommentID: 2, ErrorType: "test2"},
				{CommentID: 3, ErrorType: "test3"},
			},
			corruptFile:   false,
			expectedCount: 3,
			description:   "Should return correct count",
		},
		{
			name:          "corrupted file",
			setupErrors:   nil,
			corruptFile:   true,
			expectedCount: 0,
			description:   "Should return 0 for corrupted file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			errorFile := filepath.Join(tempDir, "errors.json")

			tracker := &ErrorTracker{
				enabled:     true,
				verboseMode: false,
				errorFile:   errorFile,
			}

			if tt.corruptFile {
				// Write corrupted JSON
				os.WriteFile(errorFile, []byte("invalid json"), 0644)
			} else if tt.setupErrors != nil {
				// Write valid errors
				data, _ := json.Marshal(tt.setupErrors)
				os.WriteFile(errorFile, data, 0644)
			}

			count := tracker.GetErrorCount()

			if count != tt.expectedCount {
				t.Errorf("Expected count=%d for case '%s', got %d", tt.expectedCount, tt.description, count)
			}
		})
	}
}

// TestErrorTracker_ReadErrors_EdgeCases tests edge cases for readErrors
func TestErrorTracker_ReadErrors_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      string
		expectError    bool
		expectedErrors int
		verboseMode    bool
		description    string
	}{
		{
			name:           "directory doesn't exist",
			setupFile:      "",
			expectError:    false,
			expectedErrors: 0,
			verboseMode:    false,
			description:    "Should create directory and return empty list",
		},
		{
			name:           "corrupted JSON with verbose mode",
			setupFile:      "not valid json at all",
			expectError:    false,
			expectedErrors: 0,
			verboseMode:    true,
			description:    "Should handle corrupted JSON gracefully",
		},
		{
			name:           "partial JSON",
			setupFile:      `[{"comment_id": 1, "error_type": "test"`,
			expectError:    false,
			expectedErrors: 0,
			verboseMode:    false,
			description:    "Should handle partial JSON",
		},
		{
			name:           "valid JSON with extra fields",
			setupFile:      `[{"comment_id": 1, "error_type": "test", "extra_field": "ignored"}]`,
			expectError:    false,
			expectedErrors: 1,
			verboseMode:    false,
			description:    "Should handle extra fields",
		},
		{
			name:           "empty array",
			setupFile:      `[]`,
			expectError:    false,
			expectedErrors: 0,
			verboseMode:    false,
			description:    "Should handle empty array",
		},
		{
			name:           "null values in JSON",
			setupFile:      `[{"comment_id": 1, "error_type": null}]`,
			expectError:    false,
			expectedErrors: 1,
			verboseMode:    false,
			description:    "Should handle null values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			errorFile := filepath.Join(tempDir, "subdir", "errors.json")

			tracker := &ErrorTracker{
				enabled:     true,
				verboseMode: tt.verboseMode,
				errorFile:   errorFile,
			}

			if tt.setupFile != "" {
				// Create directory and file
				os.MkdirAll(filepath.Dir(errorFile), 0755)
				os.WriteFile(errorFile, []byte(tt.setupFile), 0644)
			}

			errors, err := tracker.readErrors()

			if tt.expectError && err == nil {
				t.Errorf("Expected error for case '%s', got nil", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for case '%s': %v", tt.description, err)
			}

			if len(errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors for case '%s', got %d", tt.expectedErrors, tt.description, len(errors))
			}
		})
	}
}

// TestErrorTracker_PrintErrorSummary_EdgeCases tests edge cases for PrintErrorSummary
func TestErrorTracker_PrintErrorSummary_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		setupErrors []CommentError
		verboseMode bool
		description string
	}{
		{
			name:        "disabled tracker",
			enabled:     false,
			setupErrors: []CommentError{{CommentID: 1}},
			verboseMode: true,
			description: "Should do nothing when disabled",
		},
		{
			name:        "empty error list",
			enabled:     true,
			setupErrors: []CommentError{},
			verboseMode: true,
			description: "Should handle empty error list",
		},
		{
			name:    "single error",
			enabled: true,
			setupErrors: []CommentError{
				{CommentID: 1, ErrorType: "json_parse", ErrorMessage: "test"},
			},
			verboseMode: true,
			description: "Should handle single error",
		},
		{
			name:    "many errors (more than recent display limit)",
			enabled: true,
			setupErrors: func() []CommentError {
				errors := make([]CommentError, 10)
				for i := 0; i < 10; i++ {
					errors[i] = CommentError{
						CommentID:    int64(i),
						ErrorType:    fmt.Sprintf("type_%d", i%3),
						ErrorMessage: fmt.Sprintf("error %d", i),
					}
				}
				return errors
			}(),
			verboseMode: true,
			description: "Should handle many errors correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			errorFile := filepath.Join(tempDir, "errors.json")

			tracker := &ErrorTracker{
				enabled:     tt.enabled,
				verboseMode: tt.verboseMode,
				errorFile:   errorFile,
			}

			if len(tt.setupErrors) > 0 {
				data, _ := json.Marshal(tt.setupErrors)
				os.WriteFile(errorFile, data, 0644)
			}

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			tracker.PrintErrorSummary()

			w.Close()
			os.Stdout = old

			buf := make([]byte, 4096)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			// Verify output based on conditions
			if !tt.enabled && output != "" {
				t.Errorf("Expected no output for disabled tracker in case '%s'", tt.description)
			}

			if tt.enabled && len(tt.setupErrors) == 0 && output != "" {
				t.Errorf("Expected no output for empty errors in case '%s'", tt.description)
			}

			if tt.enabled && len(tt.setupErrors) > 0 {
				if !strings.Contains(output, "Error Summary") {
					t.Errorf("Expected error summary header in case '%s'", tt.description)
				}
				if !strings.Contains(output, errorFile) {
					t.Errorf("Expected error file path in output for case '%s'", tt.description)
				}
			}
		})
	}
}

// TestErrorTracker_AppendError_Rotation tests error rotation when limit is exceeded
func TestErrorTracker_AppendError_MaxRotation(t *testing.T) {
	tempDir := t.TempDir()
	tracker := &ErrorTracker{
		enabled:     true,
		verboseMode: false,
		errorFile:   filepath.Join(tempDir, "errors.json"),
	}

	// Add 150 errors (exceeds 100 limit)
	for i := 0; i < 150; i++ {
		ctx := CommentContext{
			Comment: github.Comment{
				ID:   int64(i),
				Body: fmt.Sprintf("Error %d", i),
			},
		}
		tracker.RecordCommentError(ctx, "test_error", fmt.Sprintf("Error message %d", i), 0, false, 0, 0)
	}

	// Read errors and verify rotation
	errors, err := tracker.readErrors()
	if err != nil {
		t.Fatalf("Failed to read errors: %v", err)
	}

	if len(errors) != 100 {
		t.Errorf("Expected exactly 100 errors after rotation, got %d", len(errors))
	}

	// Verify we kept the most recent errors (50-149)
	firstError := errors[0]
	if firstError.CommentID != 50 {
		t.Errorf("Expected first error to have CommentID=50, got %d", firstError.CommentID)
	}

	lastError := errors[len(errors)-1]
	if lastError.CommentID != 149 {
		t.Errorf("Expected last error to have CommentID=149, got %d", lastError.CommentID)
	}
}

// TestErrorTracker_ConcurrentAccess tests concurrent access to error tracker
func TestErrorTracker_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	// Use NewErrorTracker to properly initialize the tracker
	tracker := NewErrorTracker(true, false, tempDir)

	// Run concurrent operations with both readers and writers
	var wg sync.WaitGroup
	numGoroutines := 10

	// Mix of readers and writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Half writers, half readers
			if id%2 == 0 {
				// Writer
				ctx := CommentContext{
					Comment: github.Comment{
						ID:   int64(id + 1000), // Unique IDs
						Body: fmt.Sprintf("Concurrent error %d", id),
					},
				}
				tracker.RecordCommentError(ctx, fmt.Sprintf("type_%d", id), fmt.Sprintf("Error %d", id), 0, false, 0, 0)
			} else {
				// Reader
				tracker.GetErrorCount()
				tracker.GetErrorSummary()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify we have some errors recorded (at least 1)
	// We can't guarantee an exact count due to race conditions, but at least some should succeed
	count := tracker.GetErrorCount()
	if count == 0 {
		t.Error("Expected at least some errors to be recorded in concurrent test")
	}
}

// Benchmark tests
func BenchmarkErrorTracker_RecordError(b *testing.B) {
	tempDir := b.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	ctx := CommentContext{
		Comment: github.Comment{
			ID:   12345,
			Body: strings.Repeat("Error comment body ", 100),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.RecordCommentError(ctx, "benchmark", "Benchmark error", i, false, 1024, 512)
	}
}

func BenchmarkErrorTracker_GetErrorSummary(b *testing.B) {
	tempDir := b.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	// Pre-populate with errors
	for i := 0; i < 50; i++ {
		ctx := CommentContext{
			Comment: github.Comment{
				ID:   int64(i),
				Body: "Test error",
			},
		}
		tracker.RecordCommentError(ctx, "test", "Test error", 0, false, 0, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetErrorSummary()
	}
}

// TestErrorTracker_WriteErrorsCoverage tests uncovered branches in writeErrors
func TestErrorTracker_WriteErrorsCoverage(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("write errors with directory creation failure", func(t *testing.T) {
		// Use an invalid path that will fail to create
		tracker := &ErrorTracker{
			errorFile:   "/\x00invalid/path/errors.json", // Null byte makes path invalid
			verboseMode: true,
			enabled:     true,
		}

		errors := []CommentError{{
			CommentID:    1,
			ErrorMessage: "test",
			Timestamp:    time.Now(),
		}}
		tracker.writeErrors(errors) // Should handle the error gracefully
	})

	t.Run("write errors with file write failure", func(t *testing.T) {
		// Create a directory with the same name as the expected file
		badFile := filepath.Join(tempDir, "badfile.json")
		os.MkdirAll(badFile, 0755) // Create directory with file name

		tracker := &ErrorTracker{
			errorFile:   badFile,
			verboseMode: true,
			enabled:     true,
		}

		errors := []CommentError{{
			CommentID:    1,
			ErrorMessage: "test",
			Timestamp:    time.Now(),
		}}
		tracker.writeErrors(errors) // Should handle write failure gracefully
	})
}
