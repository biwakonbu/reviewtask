package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CommentError represents an error that occurred while processing a comment
type CommentError struct {
	CommentID     int64     `json:"comment_id"`
	CommentBody   string    `json:"comment_body"`          // First 500 chars for context
	SourceReview  int64     `json:"source_review_id"`
	File          string    `json:"file"`
	Line          int       `json:"line"`
	Author        string    `json:"author"`
	ErrorType     string    `json:"error_type"`            // "json_parse", "api_failure", "context_overflow", "timeout"
	ErrorMessage  string    `json:"error_message"`
	RetryCount    int       `json:"retry_count"`
	Timestamp     time.Time `json:"timestamp"`
	RecoveryUsed  bool      `json:"recovery_used"`         // Whether JSON recovery was attempted
	PromptSize    int       `json:"prompt_size,omitempty"` // Size of prompt that caused the error
	ResponseSize  int       `json:"response_size,omitempty"` // Size of response received
}

// ErrorTracker manages error logging for comment processing failures
type ErrorTracker struct {
	enabled   bool
	verboseMode bool
	errorFile   string
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker(enabled, verboseMode bool, storageDir string) *ErrorTracker {
	if storageDir == "" {
		storageDir = ".pr-review"
	}

	return &ErrorTracker{
		enabled:     enabled,
		verboseMode: verboseMode,
		errorFile:   filepath.Join(storageDir, "errors.json"),
	}
}

// RecordCommentError records an error that occurred during comment processing
func (et *ErrorTracker) RecordCommentError(commentCtx CommentContext, errorType, errorMessage string, retryCount int, recoveryUsed bool, promptSize, responseSize int) {
	if !et.enabled {
		return
	}

	// Truncate comment body for storage (keep first 500 chars)
	commentBody := commentCtx.Comment.Body
	if len(commentBody) > 500 {
		commentBody = commentBody[:500] + "..."
	}

	commentError := CommentError{
		CommentID:    commentCtx.Comment.ID,
		CommentBody:  commentBody,
		SourceReview: commentCtx.SourceReview.ID,
		File:         commentCtx.Comment.File,
		Line:         commentCtx.Comment.Line,
		Author:       commentCtx.Comment.Author,
		ErrorType:    errorType,
		ErrorMessage: errorMessage,
		RetryCount:   retryCount,
		Timestamp:    time.Now(),
		RecoveryUsed: recoveryUsed,
		PromptSize:   promptSize,
		ResponseSize: responseSize,
	}

	et.appendError(commentError)

	if et.verboseMode {
		fmt.Printf("  ❌ Error recorded for comment %d: %s (%s)\n",
			commentError.CommentID, errorType, errorMessage)
	}
}

// GetErrorSummary returns a summary of recent errors
func (et *ErrorTracker) GetErrorSummary() ([]CommentError, error) {
	if !et.enabled {
		return nil, nil
	}

	return et.readErrors()
}

// ClearErrors removes all recorded errors (typically after successful resolution)
func (et *ErrorTracker) ClearErrors() error {
	if !et.enabled {
		return nil
	}

	// Remove the error file
	if _, err := os.Stat(et.errorFile); err == nil {
		return os.Remove(et.errorFile)
	}
	return nil
}

// GetErrorCount returns the number of errors recorded
func (et *ErrorTracker) GetErrorCount() int {
	errors, err := et.GetErrorSummary()
	if err != nil {
		return 0
	}
	return len(errors)
}

// appendError adds an error to the error file
func (et *ErrorTracker) appendError(commentError CommentError) {
	// Read existing errors
	errors, _ := et.readErrors() // Ignore read errors, start fresh if needed

	// Add new error
	errors = append(errors, commentError)

	// Keep only recent errors (last 100) to prevent file from growing too large
	if len(errors) > 100 {
		errors = errors[len(errors)-100:]
	}

	// Write back to file
	et.writeErrors(errors)
}

// readErrors reads all errors from the error file
func (et *ErrorTracker) readErrors() ([]CommentError, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(et.errorFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create error directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(et.errorFile); os.IsNotExist(err) {
		return []CommentError{}, nil
	}

	// Read file
	data, err := os.ReadFile(et.errorFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read error file: %w", err)
	}

	var errors []CommentError
	if err := json.Unmarshal(data, &errors); err != nil {
		// If JSON is corrupted, start fresh
		if et.verboseMode {
			fmt.Printf("  ⚠️  Error file corrupted, starting fresh: %v\n", err)
		}
		return []CommentError{}, nil
	}

	return errors, nil
}

// writeErrors writes errors to the error file
func (et *ErrorTracker) writeErrors(errors []CommentError) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(et.errorFile), 0755); err != nil {
		if et.verboseMode {
			fmt.Printf("  ⚠️  Failed to create error directory: %v\n", err)
		}
		return
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(errors, "", "  ")
	if err != nil {
		if et.verboseMode {
			fmt.Printf("  ⚠️  Failed to marshal errors: %v\n", err)
		}
		return
	}

	// Write to file
	if err := os.WriteFile(et.errorFile, data, 0644); err != nil {
		if et.verboseMode {
			fmt.Printf("  ⚠️  Failed to write error file: %v\n", err)
		}
	}
}

// PrintErrorSummary prints a summary of errors to console
func (et *ErrorTracker) PrintErrorSummary() {
	if !et.enabled {
		return
	}

	errors, err := et.GetErrorSummary()
	if err != nil {
		if et.verboseMode {
			fmt.Printf("  ⚠️  Failed to read error summary: %v\n", err)
		}
		return
	}

	if len(errors) == 0 {
		return
	}

	fmt.Printf("\n📊 Error Summary (%d errors recorded):\n", len(errors))

	// Count by error type
	errorTypes := make(map[string]int)
	for _, e := range errors {
		errorTypes[e.ErrorType]++
	}

	for errorType, count := range errorTypes {
		fmt.Printf("  • %s: %d errors\n", errorType, count)
	}

	// Show most recent errors
	recentCount := 3
	if len(errors) < recentCount {
		recentCount = len(errors)
	}

	fmt.Printf("\nMost recent errors:\n")
	for i := len(errors) - recentCount; i < len(errors); i++ {
		e := errors[i]
		fmt.Printf("  • Comment %d (%s): %s\n", e.CommentID, e.ErrorType, e.ErrorMessage)
	}

	fmt.Printf("\nFor full details, see: %s\n", et.errorFile)
}