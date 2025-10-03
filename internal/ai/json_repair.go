package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RepairJSONResponse attempts to fix common JSON formatting issues from AI providers
// This is especially useful for cursor-agent(grok) which sometimes returns malformed JSON
func RepairJSONResponse(input string) (string, error) {
	// Try strategies in order of likelihood
	strategies := []func(string) string{
		removeMarkdownWrapper,
		escapeControlCharacters,
		fixTruncatedJSON,
	}

	current := strings.TrimSpace(input)

	// First, try parsing as-is
	var test interface{}
	if err := json.Unmarshal([]byte(current), &test); err == nil {
		return current, nil // Already valid
	}

	// Apply each repair strategy
	for _, strategy := range strategies {
		current = strategy(current)

		// Test if valid after this strategy
		if err := json.Unmarshal([]byte(current), &test); err == nil {
			return current, nil
		}
	}

	// All strategies failed, return error
	var finalErr error
	finalErr = json.Unmarshal([]byte(current), &test) // Get the error
	if finalErr == nil {
		finalErr = fmt.Errorf("unknown JSON parsing error")
	}

	return "", fmt.Errorf("JSON repair failed after all strategies: %w", finalErr)
}

// removeMarkdownWrapper removes ```json or ``` code block wrappers
func removeMarkdownWrapper(input string) string {
	input = strings.TrimSpace(input)

	// Remove ```json\n...\n``` wrapper
	if strings.HasPrefix(input, "```json") && strings.HasSuffix(input, "```") {
		lines := strings.Split(input, "\n")
		if len(lines) >= 3 {
			return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}

	// Remove ```\n...\n``` wrapper
	if strings.HasPrefix(input, "```") && strings.HasSuffix(input, "```") {
		lines := strings.Split(input, "\n")
		if len(lines) >= 3 {
			return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}

	return input
}

// escapeControlCharacters escapes unescaped control characters in JSON strings
// This fixes the most common cursor-agent(grok) issue: literal newlines/tabs in strings
func escapeControlCharacters(input string) string {
	var result strings.Builder
	result.Grow(len(input) * 11 / 10) // Preallocate slightly more space

	inString := false
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		// Handle escape sequences
		if escaped {
			result.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			result.WriteByte(ch)
			escaped = true
			continue
		}

		// Track if we're inside a string
		if ch == '"' {
			result.WriteByte(ch)
			inString = !inString
			continue
		}

		// Escape control characters when inside strings
		if inString && ch < 0x20 {
			switch ch {
			case '\n':
				result.WriteString("\\n")
			case '\r':
				result.WriteString("\\r")
			case '\t':
				result.WriteString("\\t")
			case '\b':
				result.WriteString("\\b")
			case '\f':
				result.WriteString("\\f")
			default:
				// Unicode escape for other control characters
				result.WriteString(fmt.Sprintf("\\u%04x", ch))
			}
			continue
		}

		result.WriteByte(ch)
	}

	return result.String()
}

// fixTruncatedJSON attempts to close unclosed brackets/braces
func fixTruncatedJSON(input string) string {
	input = strings.TrimSpace(input)

	// Count brackets
	openArray := strings.Count(input, "[")
	closeArray := strings.Count(input, "]")
	openObject := strings.Count(input, "{")
	closeObject := strings.Count(input, "}")

	// Add missing closing brackets (arrays first, then objects)
	for i := 0; i < openArray-closeArray; i++ {
		input += "]"
	}
	for i := 0; i < openObject-closeObject; i++ {
		input += "}"
	}

	return input
}
