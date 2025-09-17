package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"reviewtask/internal/github"
)

func TestNewErrorTracker(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		enabled     bool
		verboseMode bool
		storageDir  string
		expectedDir string
	}{
		{
			name:        "enabled with custom directory",
			enabled:     true,
			verboseMode: true,
			storageDir:  tempDir,
			expectedDir: tempDir,
		},
		{
			name:        "disabled with default directory",
			enabled:     false,
			verboseMode: false,
			storageDir:  "",
			expectedDir: ".pr-review",
		},
		{
			name:        "enabled with default directory",
			enabled:     true,
			verboseMode: false,
			storageDir:  "",
			expectedDir: ".pr-review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewErrorTracker(tt.enabled, tt.verboseMode, tt.storageDir)

			if tracker.enabled != tt.enabled {
				t.Errorf("Expected enabled=%v, got %v", tt.enabled, tracker.enabled)
			}

			if tracker.verboseMode != tt.verboseMode {
				t.Errorf("Expected verboseMode=%v, got %v", tt.verboseMode, tracker.verboseMode)
			}

			expectedErrorFile := filepath.Join(tt.expectedDir, "errors.json")
			if tracker.errorFile != expectedErrorFile {
				t.Errorf("Expected errorFile=%s, got %s", expectedErrorFile, tracker.errorFile)
			}
		})
	}
}

func TestErrorTracker_RecordCommentError(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	// Create test comment context
	comment := github.Comment{
		ID:     12345,
		Body:   "This is a test comment with some content that caused an error",
		Author: "testuser",
		File:   "test.go",
		Line:   42,
	}

	review := github.Review{
		ID: 67890,
	}

	ctx := CommentContext{
		Comment:      comment,
		SourceReview: review,
	}

	// Record an error
	tracker.RecordCommentError(ctx, "json_parse", "failed to parse JSON", 1, true, 1024, 512)

	// Verify error was recorded
	errors, err := tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get error summary: %v", err)
	}

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	recordedError := errors[0]
	if recordedError.CommentID != comment.ID {
		t.Errorf("Expected CommentID=%d, got %d", comment.ID, recordedError.CommentID)
	}

	if recordedError.ErrorType != "json_parse" {
		t.Errorf("Expected ErrorType='json_parse', got '%s'", recordedError.ErrorType)
	}

	if recordedError.ErrorMessage != "failed to parse JSON" {
		t.Errorf("Expected ErrorMessage='failed to parse JSON', got '%s'", recordedError.ErrorMessage)
	}

	if recordedError.RetryCount != 1 {
		t.Errorf("Expected RetryCount=1, got %d", recordedError.RetryCount)
	}

	if !recordedError.RecoveryUsed {
		t.Error("Expected RecoveryUsed=true, got false")
	}

	if recordedError.PromptSize != 1024 {
		t.Errorf("Expected PromptSize=1024, got %d", recordedError.PromptSize)
	}

	if recordedError.ResponseSize != 512 {
		t.Errorf("Expected ResponseSize=512, got %d", recordedError.ResponseSize)
	}
}

func TestErrorTracker_RecordCommentError_LongBody(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	// Create comment with very long body
	longBody := ""
	for i := 0; i < 100; i++ {
		longBody += "This is a very long comment body that exceeds 500 characters. "
	}

	comment := github.Comment{
		ID:   12345,
		Body: longBody,
	}

	ctx := CommentContext{
		Comment: comment,
	}

	tracker.RecordCommentError(ctx, "api_failure", "timeout error", 0, false, 0, 0)

	errors, err := tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get error summary: %v", err)
	}

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	// Check that body was truncated
	recordedError := errors[0]
	if len(recordedError.CommentBody) > 503 { // 500 + "..."
		t.Errorf("Expected comment body to be truncated to ~503 chars, got %d", len(recordedError.CommentBody))
	}

	if len(recordedError.CommentBody) <= 500 {
		t.Error("Expected truncated body to include '...' suffix")
	}

	if recordedError.CommentBody[len(recordedError.CommentBody)-3:] != "..." {
		t.Error("Expected truncated body to end with '...'")
	}
}

func TestErrorTracker_DisabledTracker(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(false, false, tempDir) // Disabled

	comment := github.Comment{
		ID:   12345,
		Body: "test comment",
	}

	ctx := CommentContext{
		Comment: comment,
	}

	// Record an error - should be ignored
	tracker.RecordCommentError(ctx, "json_parse", "test error", 0, false, 0, 0)

	// Verify no errors were recorded
	errors, err := tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get error summary: %v", err)
	}

	if errors != nil {
		t.Errorf("Expected nil errors for disabled tracker, got %v", errors)
	}
}

func TestErrorTracker_ClearErrors(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	// Record some errors
	comment := github.Comment{
		ID:   12345,
		Body: "test comment",
	}

	ctx := CommentContext{
		Comment: comment,
	}

	tracker.RecordCommentError(ctx, "json_parse", "error 1", 0, false, 0, 0)
	tracker.RecordCommentError(ctx, "api_failure", "error 2", 0, false, 0, 0)

	// Verify errors exist
	errors, err := tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get error summary: %v", err)
	}

	if len(errors) != 2 {
		t.Fatalf("Expected 2 errors before clearing, got %d", len(errors))
	}

	// Clear errors
	err = tracker.ClearErrors()
	if err != nil {
		t.Fatalf("Failed to clear errors: %v", err)
	}

	// Verify errors are gone
	errors, err = tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get error summary after clearing: %v", err)
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors after clearing, got %d", len(errors))
	}
}

func TestErrorTracker_ErrorRotation(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	comment := github.Comment{
		ID:   12345,
		Body: "test comment",
	}

	ctx := CommentContext{
		Comment: comment,
	}

	// Record more than 100 errors
	for i := 0; i < 120; i++ {
		tracker.RecordCommentError(ctx, "json_parse", "test error", i, false, 0, 0)
	}

	// Verify only last 100 errors are kept
	errors, err := tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get error summary: %v", err)
	}

	if len(errors) != 100 {
		t.Errorf("Expected exactly 100 errors after rotation, got %d", len(errors))
	}

	// Verify we kept the most recent errors (retry counts 20-119)
	firstError := errors[0]
	if firstError.RetryCount != 20 {
		t.Errorf("Expected first error to have RetryCount=20, got %d", firstError.RetryCount)
	}

	lastError := errors[len(errors)-1]
	if lastError.RetryCount != 119 {
		t.Errorf("Expected last error to have RetryCount=119, got %d", lastError.RetryCount)
	}
}

func TestErrorTracker_GetErrorCount(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	// Initially should have 0 errors
	count := tracker.GetErrorCount()
	if count != 0 {
		t.Errorf("Expected initial error count=0, got %d", count)
	}

	// Record some errors
	comment := github.Comment{
		ID:   12345,
		Body: "test comment",
	}

	ctx := CommentContext{
		Comment: comment,
	}

	tracker.RecordCommentError(ctx, "json_parse", "error 1", 0, false, 0, 0)
	tracker.RecordCommentError(ctx, "api_failure", "error 2", 0, false, 0, 0)
	tracker.RecordCommentError(ctx, "timeout", "error 3", 0, false, 0, 0)

	// Should have 3 errors
	count = tracker.GetErrorCount()
	if count != 3 {
		t.Errorf("Expected error count=3, got %d", count)
	}
}

func TestErrorTracker_CorruptedErrorFile(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, true, tempDir) // Verbose mode for corruption handling

	// Create corrupted error file
	errorFile := filepath.Join(tempDir, "errors.json")
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Write invalid JSON
	err = os.WriteFile(errorFile, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// Should handle corruption gracefully
	errors, err := tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Expected no error when reading corrupted file, got: %v", err)
	}

	if len(errors) != 0 {
		t.Errorf("Expected empty error list for corrupted file, got %d errors", len(errors))
	}

	// Should be able to record new errors after corruption
	comment := github.Comment{
		ID:   12345,
		Body: "test comment",
	}

	ctx := CommentContext{
		Comment: comment,
	}

	tracker.RecordCommentError(ctx, "json_parse", "new error", 0, false, 0, 0)

	errors, err = tracker.GetErrorSummary()
	if err != nil {
		t.Fatalf("Failed to get errors after recording new error: %v", err)
	}

	if len(errors) != 1 {
		t.Errorf("Expected 1 error after recording, got %d", len(errors))
	}
}

func TestErrorTracker_JSONPersistence(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, false, tempDir)

	// Record an error
	comment := github.Comment{
		ID:     12345,
		Body:   "test comment",
		Author: "testuser",
		File:   "test.go",
		Line:   10,
	}

	review := github.Review{
		ID: 67890,
	}

	ctx := CommentContext{
		Comment:      comment,
		SourceReview: review,
	}

	beforeTime := time.Now()
	tracker.RecordCommentError(ctx, "json_parse", "test error message", 2, true, 1024, 512)
	afterTime := time.Now()

	// Read the JSON file directly
	errorFile := filepath.Join(tempDir, "errors.json")
	data, err := os.ReadFile(errorFile)
	if err != nil {
		t.Fatalf("Failed to read error file: %v", err)
	}

	var errors []CommentError
	err = json.Unmarshal(data, &errors)
	if err != nil {
		t.Fatalf("Failed to unmarshal error file: %v", err)
	}

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error in file, got %d", len(errors))
	}

	storedError := errors[0]

	// Verify all fields are stored correctly
	if storedError.CommentID != comment.ID {
		t.Errorf("Expected CommentID=%d, got %d", comment.ID, storedError.CommentID)
	}

	if storedError.CommentBody != comment.Body {
		t.Errorf("Expected CommentBody='%s', got '%s'", comment.Body, storedError.CommentBody)
	}

	if storedError.SourceReview != review.ID {
		t.Errorf("Expected SourceReview=%d, got %d", review.ID, storedError.SourceReview)
	}

	if storedError.File != comment.File {
		t.Errorf("Expected File='%s', got '%s'", comment.File, storedError.File)
	}

	if storedError.Line != comment.Line {
		t.Errorf("Expected Line=%d, got %d", comment.Line, storedError.Line)
	}

	if storedError.Author != comment.Author {
		t.Errorf("Expected Author='%s', got '%s'", comment.Author, storedError.Author)
	}

	if storedError.ErrorType != "json_parse" {
		t.Errorf("Expected ErrorType='json_parse', got '%s'", storedError.ErrorType)
	}

	if storedError.ErrorMessage != "test error message" {
		t.Errorf("Expected ErrorMessage='test error message', got '%s'", storedError.ErrorMessage)
	}

	if storedError.RetryCount != 2 {
		t.Errorf("Expected RetryCount=2, got %d", storedError.RetryCount)
	}

	if !storedError.RecoveryUsed {
		t.Error("Expected RecoveryUsed=true, got false")
	}

	if storedError.PromptSize != 1024 {
		t.Errorf("Expected PromptSize=1024, got %d", storedError.PromptSize)
	}

	if storedError.ResponseSize != 512 {
		t.Errorf("Expected ResponseSize=512, got %d", storedError.ResponseSize)
	}

	// Verify timestamp is reasonable
	if storedError.Timestamp.Before(beforeTime) || storedError.Timestamp.After(afterTime) {
		t.Errorf("Expected timestamp between %v and %v, got %v", beforeTime, afterTime, storedError.Timestamp)
	}
}

func TestErrorTracker_PrintErrorSummary(t *testing.T) {
	tempDir := t.TempDir()
	tracker := NewErrorTracker(true, true, tempDir) // Verbose mode

	// This test mainly verifies that PrintErrorSummary doesn't crash
	// We can't easily test the actual output without capturing stdout

	// Test with no errors
	tracker.PrintErrorSummary()

	// Test with some errors
	comment := github.Comment{
		ID:   12345,
		Body: "test comment",
	}

	ctx := CommentContext{
		Comment: comment,
	}

	tracker.RecordCommentError(ctx, "json_parse", "error 1", 0, false, 0, 0)
	tracker.RecordCommentError(ctx, "api_failure", "error 2", 1, true, 1024, 512)
	tracker.RecordCommentError(ctx, "json_parse", "error 3", 0, false, 0, 0)

	// Should not crash
	tracker.PrintErrorSummary()

	// Test disabled tracker
	disabledTracker := NewErrorTracker(false, true, tempDir)
	disabledTracker.PrintErrorSummary() // Should do nothing
}
