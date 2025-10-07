package ai

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
)

// MockClaudeClient implements ClaudeClient for testing
type MockClaudeClient struct {
	// Responses maps input patterns to responses
	Responses map[string]string
	// Error to return if set
	Error error
	// CallCount tracks number of calls
	CallCount int
	// LastInput tracks the last input received
	LastInput string
	// mu protects concurrent access to CallCount and LastInput
	mu sync.Mutex
}

// NewMockClaudeClient creates a new mock Claude client
func NewMockClaudeClient() *MockClaudeClient {
	return &MockClaudeClient{
		Responses: make(map[string]string),
	}
}

// Execute returns a mocked response
func (m *MockClaudeClient) Execute(ctx context.Context, input string, outputFormat string) (string, error) {
	// Protect concurrent access
	m.mu.Lock()
	m.CallCount++
	m.LastInput = input
	m.mu.Unlock()

	if m.Error != nil {
		return "", m.Error
	}

	// Extract comment ID from the prompt (if needed for future use)
	// Currently we don't need to extract these since SimpleTaskRequest
	// doesn't include the IDs - they're added programmatically later

	// Look for a matching response pattern (excluding default)
	for pattern, response := range m.Responses {
		if pattern != "default" && strings.Contains(input, pattern) {
			// Special handling for nitpick pattern - check if it contains nitpick-related content
			if (pattern == "nitpick" || strings.Contains(pattern, "🧹")) &&
				strings.Contains(input, "Skip nitpick comments and suggestions") &&
				strings.Contains(input, "Actionable comments posted: 0") {
				// Return empty array for nitpick-only content when disabled
				return "[]", nil
			}
			return response, nil
		}
	}

	// Fall back to default response if no specific pattern matched
	if defaultResp, ok := m.Responses["default"]; ok {
		return defaultResp, nil
	}

	// Default response based on input content - using SimpleTaskRequest format
	var tasks []SimpleTaskRequest
	if strings.Contains(input, "nit:") || strings.Contains(input, "minor:") {
		// Generate a task with low priority
		// Leave InitialStatus empty to let pattern-based detection work
		tasks = []SimpleTaskRequest{
			{
				Description:   "Fix minor issue",
				Priority:      "low",
				InitialStatus: "", // Empty status triggers fallback pattern detection
			},
		}
	} else {
		// Default task generation
		// Leave InitialStatus empty to let pattern-based detection work
		tasks = []SimpleTaskRequest{
			{
				Description:   "Fix the issue mentioned in the comment",
				Priority:      "medium",
				InitialStatus: "", // Empty status triggers fallback pattern detection
			},
		}
	}

	// Always return JSON for SimpleTaskRequest processing
	data, _ := json.Marshal(tasks)
	return string(data), nil
}

// extractCommentText extracts the comment text from the prompt
func extractCommentText(input string) string {
	lines := strings.Split(input, "\n")
	inComment := false
	var commentText []string

	for _, line := range lines {
		if strings.HasPrefix(line, "- Comment Text: ") {
			text := strings.TrimPrefix(line, "- Comment Text: ")
			return text
		}
		// Handle multi-line comments
		if inComment {
			if strings.HasPrefix(line, "-") || line == "" {
				break
			}
			commentText = append(commentText, line)
		}
	}

	if len(commentText) > 0 {
		return strings.Join(commentText, "\n")
	}

	return ""
}

// MockCommandExecutor implements CommandExecutor for testing
type MockCommandExecutor struct {
	// Responses maps command patterns to responses
	Responses map[string][]byte
	// Error to return if set
	Error error
	// CallCount tracks number of calls
	CallCount int
	// LastCommand tracks the last command executed
	LastCommand string
	// LastArgs tracks the last args used
	LastArgs []string
	// mu protects concurrent access to CallCount, LastCommand, and LastArgs
	mu sync.Mutex
}

// NewMockCommandExecutor creates a new mock command executor
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		Responses: make(map[string][]byte),
	}
}

// Execute returns a mocked response
func (m *MockCommandExecutor) Execute(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, error) {
	// Protect concurrent access
	m.mu.Lock()
	m.CallCount++
	m.LastCommand = name
	m.LastArgs = args
	m.mu.Unlock()

	if m.Error != nil {
		return nil, m.Error
	}

	// Look for a matching response
	key := name + " " + strings.Join(args, " ")
	if response, ok := m.Responses[key]; ok {
		return response, nil
	}

	// Default response
	return []byte("mock output"), nil
}
