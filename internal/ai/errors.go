package ai

import (
	"errors"
	"fmt"
	"strings"
)

// Critical error types that should stop processing
var (
	// ErrClaudeAPI indicates a Claude API related error
	ErrClaudeAPI = errors.New("claude API error")

	// ErrAuthentication indicates an authentication failure
	ErrAuthentication = errors.New("authentication error")

	// ErrCritical is a generic critical error
	ErrCritical = errors.New("critical error")
)

// ClaudeAPIError represents a Claude API specific error
type ClaudeAPIError struct {
	Message string
	Err     error
}

func (e *ClaudeAPIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("claude API error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("claude API error: %s", e.Message)
}

func (e *ClaudeAPIError) Unwrap() error {
	return e.Err
}

// AuthenticationError represents an authentication failure
type AuthenticationError struct {
	Source  string
	Message string
	Err     error
}

func (e *AuthenticationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("authentication error (%s): %s: %v", e.Source, e.Message, e.Err)
	}
	return fmt.Sprintf("authentication error (%s): %s", e.Source, e.Message)
}

func (e *AuthenticationError) Unwrap() error {
	return e.Err
}

// NewClaudeAPIError creates a new Claude API error
func NewClaudeAPIError(message string, err error) error {
	return &ClaudeAPIError{
		Message: message,
		Err:     err,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(source, message string, err error) error {
	return &AuthenticationError{
		Source:  source,
		Message: message,
		Err:     err,
	}
}

// isCriticalError determines if an error is critical and should stop processing
func isCriticalError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	var claudeErr *ClaudeAPIError
	var authErr *AuthenticationError

	if errors.As(err, &claudeErr) {
		return true
	}
	if errors.As(err, &authErr) {
		return true
	}

	// Check for wrapped sentinel errors
	if errors.Is(err, ErrClaudeAPI) || errors.Is(err, ErrAuthentication) || errors.Is(err, ErrCritical) {
		return true
	}

	// Fallback to string matching for legacy errors
	// This can be removed once all errors are properly typed
	errStr := err.Error()
	return strings.Contains(errStr, "claude") || strings.Contains(errStr, "authentication")
}
