package ai

import (
	"fmt"
	"reviewtask/internal/ui"
	"strings"
)

var globalProgressTracker interface {
	AddError(message string)
}

// SetProgressTracker allows the AI package to send errors to the progress display
func SetProgressTracker(tracker interface{ AddError(message string) }) {
	globalProgressTracker = tracker
}

// printf is a wrapper around the synchronized console output
// This replaces direct fmt.Printf calls in AI processing to prevent display corruption
func printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	// If this looks like an error message and we have a progress tracker,
	// send it through the progress system for proper queuing
	if isErrorMessage(msg) && globalProgressTracker != nil {
		// Strip newlines for clean display in progress queue
		cleanMsg := strings.TrimSpace(strings.ReplaceAll(msg, "\n", " "))
		globalProgressTracker.AddError(cleanMsg)
	} else {
		ui.Printf(format, args...)
	}
}

// println is a wrapper around the synchronized console output
func println(msg string) {
	if isErrorMessage(msg) && globalProgressTracker != nil {
		globalProgressTracker.AddError(msg)
	} else {
		ui.Println(msg)
	}
}

// print is a wrapper around the synchronized console output
func print(msg string) {
	if isErrorMessage(msg) && globalProgressTracker != nil {
		globalProgressTracker.AddError(msg)
	} else {
		ui.Print(msg)
	}
}

// isErrorMessage detects if a message is an error/warning that should be queued
func isErrorMessage(msg string) bool {
	// Check for emoji indicators first
	if strings.Contains(msg, "⚠️") || strings.Contains(msg, "❌") {
		return true
	}
	
	// Check for error keywords (case-insensitive)
	msgLower := strings.ToLower(msg)
	errorKeywords := []string{"error", "failed", "warning", "exception"}
	
	for _, keyword := range errorKeywords {
		if strings.Contains(msgLower, keyword) {
			return true
		}
	}
	
	return false
}
