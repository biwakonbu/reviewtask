package ai

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Extract comment ID, file, and line from the prompt
	var commentID int64
	var reviewID int64
	var file string
	var lineNum int
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "- Comment ID: ") {
			_, _ = fmt.Sscanf(line, "- Comment ID: %d", &commentID)
		} else if strings.HasPrefix(line, "- Review ID: ") {
			_, _ = fmt.Sscanf(line, "- Review ID: %d", &reviewID)
		} else if strings.HasPrefix(line, "- File: ") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				file = strings.TrimPrefix(parts[0], "- File: ")
				_, _ = fmt.Sscanf(parts[1], "%d", &lineNum)
			}
		}
	}

	// Look for a matching response pattern
	for pattern, response := range m.Responses {
		if strings.Contains(input, pattern) {
			// Replace dynamic values in response
			if commentID > 0 {
				response = strings.ReplaceAll(response, `"source_comment_id": 0`, fmt.Sprintf(`"source_comment_id": %d`, commentID))
			}
			if reviewID > 0 {
				response = strings.ReplaceAll(response, `"source_review_id": 0`, fmt.Sprintf(`"source_review_id": %d`, reviewID))
			}
			if file != "" {
				response = strings.ReplaceAll(response, `"file": ""`, fmt.Sprintf(`"file": "%s"`, file))
			}
			if lineNum > 0 {
				response = strings.ReplaceAll(response, `"line": 0`, fmt.Sprintf(`"line": %d`, lineNum))
			}

			// Wrap response in Claude CLI format
			if outputFormat == "json" {
				wrapped := map[string]interface{}{
					"type":     "text",
					"subtype":  "assistant_response",
					"is_error": false,
					"result":   response,
				}
				data, _ := json.Marshal(wrapped)
				return string(data), nil
			}
			return response, nil
		}
	}

	// Default response based on input content
	var task TaskRequest
	if strings.Contains(input, "nit:") || strings.Contains(input, "minor:") {
		// Generate a task with low priority
		task = TaskRequest{
			Description:     "Fix minor issue",
			OriginText:      extractCommentText(input),
			Priority:        "low",
			SourceReviewID:  reviewID,
			SourceCommentID: commentID,
			File:            file,
			Line:            lineNum,
			Status:          "pending",
			TaskIndex:       0,
		}
	} else {
		// Default task generation
		task = TaskRequest{
			Description:     "Fix the issue mentioned in the comment",
			OriginText:      extractCommentText(input),
			Priority:        "medium",
			SourceReviewID:  reviewID,
			SourceCommentID: commentID,
			File:            file,
			Line:            lineNum,
			Status:          "todo",
			TaskIndex:       0,
		}
	}

	if outputFormat == "json" {
		data, _ := json.Marshal([]TaskRequest{task})
		wrapped := map[string]interface{}{
			"type":     "text",
			"subtype":  "assistant_response",
			"is_error": false,
			"result":   string(data),
		}
		wrapData, _ := json.Marshal(wrapped)
		return string(wrapData), nil
	}

	return "Generated 1 task", nil
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
