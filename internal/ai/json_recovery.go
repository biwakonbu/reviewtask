package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONRecoveryResult contains the result of JSON recovery attempt
type JSONRecoveryResult struct {
	IsRecovered  bool          `json:"is_recovered"`
	Tasks        []TaskRequest `json:"tasks"`
	ErrorType    string        `json:"error_type"`
	Message      string        `json:"message"`
	OriginalSize int           `json:"original_size"`
	RecoveredSize int          `json:"recovered_size"`
}

// JSONRecoveryConfig contains configuration for JSON recovery
type JSONRecoveryConfig struct {
	EnableRecovery         bool    `json:"enable_recovery"`
	MaxRecoveryAttempts    int     `json:"max_recovery_attempts"`
	PartialThreshold       float64 `json:"partial_threshold"`
	LogTruncatedResponses  bool    `json:"log_truncated_responses"`
}

// JSONRecoverer handles recovery of incomplete JSON responses
type JSONRecoverer struct {
	config       *JSONRecoveryConfig
	verboseMode  bool
}

// NewJSONRecoverer creates a new JSON recovery handler
func NewJSONRecoverer(enableRecovery bool, verboseMode bool) *JSONRecoverer {
	return &JSONRecoverer{
		config: &JSONRecoveryConfig{
			EnableRecovery:         enableRecovery,
			MaxRecoveryAttempts:    3,
			PartialThreshold:       0.7,
			LogTruncatedResponses:  true,
		},
		verboseMode: verboseMode,
	}
}

// RecoverJSON attempts to recover valid tasks from incomplete JSON
func (jr *JSONRecoverer) RecoverJSON(rawResponse string, originalError error) *JSONRecoveryResult {
	result := &JSONRecoveryResult{
		IsRecovered:   false,
		Tasks:         []TaskRequest{},
		ErrorType:     jr.categorizeError(originalError),
		Message:       originalError.Error(),
		OriginalSize:  len(rawResponse),
		RecoveredSize: 0,
	}

	if !jr.config.EnableRecovery {
		result.Message = "JSON recovery disabled"
		return result
	}

	if jr.verboseMode {
		fmt.Printf("  🔧 Attempting JSON recovery for %s error (size: %d bytes)\n", 
			result.ErrorType, result.OriginalSize)
	}

	// Try different recovery strategies based on error type
	switch result.ErrorType {
	case "truncation":
		return jr.recoverTruncatedJSON(rawResponse, result)
	case "malformed":
		return jr.recoverMalformedJSON(rawResponse, result)
	case "incomplete_array":
		return jr.recoverIncompleteArray(rawResponse, result)
	default:
		return jr.attemptGenericRecovery(rawResponse, result)
	}
}

// categorizeError determines the type of JSON parsing error
func (jr *JSONRecoverer) categorizeError(err error) string {
	errMsg := strings.ToLower(err.Error())
	
	if strings.Contains(errMsg, "unexpected end of json input") ||
	   strings.Contains(errMsg, "unexpected end of input") {
		return "truncation"
	}
	
	if strings.Contains(errMsg, "invalid character") ||
	   strings.Contains(errMsg, "looking for beginning of object") {
		return "malformed"
	}
	
	if strings.Contains(errMsg, "cannot unmarshal") {
		return "type_mismatch"
	}
	
	return "unknown"
}

// recoverTruncatedJSON handles truncated JSON responses
func (jr *JSONRecoverer) recoverTruncatedJSON(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	if jr.verboseMode {
		fmt.Printf("    🔍 Analyzing truncated JSON response\n")
	}

	// Strategy 1: Try to recover from partial array
	if recovered := jr.tryRecoverPartialArray(response); recovered != nil {
		result.IsRecovered = true
		result.Tasks = recovered
		result.RecoveredSize = len(response)
		result.Message = fmt.Sprintf("Recovered %d tasks from truncated array", len(recovered))
		
		if jr.verboseMode {
			fmt.Printf("    ✅ Recovered %d tasks from partial array\n", len(recovered))
		}
		return result
	}

	// Strategy 2: Try to find complete JSON objects within the response
	if recovered := jr.extractCompleteObjects(response); len(recovered) > 0 {
		result.IsRecovered = true
		result.Tasks = recovered
		result.RecoveredSize = len(response)
		result.Message = fmt.Sprintf("Extracted %d complete objects", len(recovered))
		
		if jr.verboseMode {
			fmt.Printf("    ✅ Extracted %d complete JSON objects\n", len(recovered))
		}
		return result
	}

	result.Message = "No recoverable content found in truncated response"
	return result
}

// recoverMalformedJSON handles malformed JSON responses
func (jr *JSONRecoverer) recoverMalformedJSON(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	if jr.verboseMode {
		fmt.Printf("    🔍 Attempting to fix malformed JSON\n")
	}

	// Try to clean up common malformation issues
	cleaned := jr.cleanMalformedJSON(response)
	
	var tasks []TaskRequest
	if err := json.Unmarshal([]byte(cleaned), &tasks); err == nil {
		result.IsRecovered = true
		result.Tasks = tasks
		result.RecoveredSize = len(cleaned)
		result.Message = fmt.Sprintf("Fixed malformed JSON, recovered %d tasks", len(tasks))
		
		if jr.verboseMode {
			fmt.Printf("    ✅ Fixed malformed JSON, recovered %d tasks\n", len(tasks))
		}
		return result
	}

	// If cleaning didn't work, try extracting objects
	if recovered := jr.extractCompleteObjects(response); len(recovered) > 0 {
		result.IsRecovered = true
		result.Tasks = recovered
		result.RecoveredSize = len(response)
		result.Message = fmt.Sprintf("Extracted %d objects from malformed JSON", len(recovered))
		return result
	}

	result.Message = "Could not repair malformed JSON"
	return result
}

// recoverIncompleteArray handles incomplete array structures
func (jr *JSONRecoverer) recoverIncompleteArray(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	if jr.verboseMode {
		fmt.Printf("    🔍 Attempting to complete incomplete array\n")
	}

	// Try to add missing closing bracket and parse
	if recovered := jr.tryRecoverPartialArray(response); recovered != nil {
		result.IsRecovered = true
		result.Tasks = recovered
		result.RecoveredSize = len(response)
		result.Message = fmt.Sprintf("Completed incomplete array, recovered %d tasks", len(recovered))
		
		if jr.verboseMode {
			fmt.Printf("    ✅ Completed array structure, recovered %d tasks\n", len(recovered))
		}
		return result
	}

	result.Message = "Could not complete incomplete array structure"
	return result
}

// attemptGenericRecovery tries generic recovery strategies
func (jr *JSONRecoverer) attemptGenericRecovery(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	if jr.verboseMode {
		fmt.Printf("    🔍 Attempting generic JSON recovery\n")
	}

	// Try all recovery strategies
	strategies := []func(string) []TaskRequest{
		jr.tryRecoverPartialArray,
		jr.extractCompleteObjects,
	}

	for i, strategy := range strategies {
		if recovered := strategy(response); len(recovered) > 0 {
			result.IsRecovered = true
			result.Tasks = recovered
			result.RecoveredSize = len(response)
			result.Message = fmt.Sprintf("Generic recovery strategy %d succeeded, recovered %d tasks", i+1, len(recovered))
			
			if jr.verboseMode {
				fmt.Printf("    ✅ Generic strategy %d recovered %d tasks\n", i+1, len(recovered))
			}
			return result
		}
	}

	result.Message = "All recovery strategies failed"
	return result
}

// tryRecoverPartialArray attempts to recover from a partial JSON array
func (jr *JSONRecoverer) tryRecoverPartialArray(response string) []TaskRequest {
	// Find the start of the JSON array
	arrayStart := strings.Index(response, "[")
	if arrayStart == -1 {
		return nil
	}

	// Look for the last complete object before truncation
	arrayContent := response[arrayStart+1:]
	
	// Try to find complete JSON objects within the array
	objects := jr.findCompleteJSONObjects(arrayContent)
	if len(objects) == 0 {
		return nil
	}

	// Reconstruct array with complete objects
	reconstructed := "[" + strings.Join(objects, ",") + "]"
	
	var tasks []TaskRequest
	if err := json.Unmarshal([]byte(reconstructed), &tasks); err == nil {
		return tasks
	}

	return nil
}

// extractCompleteObjects finds and extracts complete JSON objects from response
func (jr *JSONRecoverer) extractCompleteObjects(response string) []TaskRequest {
	objects := jr.findCompleteJSONObjects(response)
	if len(objects) == 0 {
		return nil
	}

	var allTasks []TaskRequest
	for _, objStr := range objects {
		var task TaskRequest
		if err := json.Unmarshal([]byte(objStr), &task); err == nil {
			// Validate that this looks like a valid task
			if jr.isValidTaskRequest(task) {
				allTasks = append(allTasks, task)
			}
		}
	}

	return allTasks
}

// findCompleteJSONObjects locates complete JSON objects in a string
func (jr *JSONRecoverer) findCompleteJSONObjects(text string) []string {
	var objects []string
	braceCount := 0
	start := -1
	inString := false
	escaped := false

	for i, char := range text {
		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch char {
		case '{':
			if braceCount == 0 {
				start = i
			}
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 && start != -1 {
				// Found complete object
				objStr := text[start : i+1]
				objects = append(objects, objStr)
				start = -1
			}
		}
	}

	return objects
}

// cleanMalformedJSON attempts to fix common JSON malformation issues
func (jr *JSONRecoverer) cleanMalformedJSON(response string) string {
	// Remove common prefixes/suffixes that aren't JSON
	cleaned := strings.TrimSpace(response)
	
	// Remove markdown code block markers
	cleaned = regexp.MustCompile("```(?:json)?\n?").ReplaceAllString(cleaned, "")
	cleaned = regexp.MustCompile("\n?```").ReplaceAllString(cleaned, "")
	
	// Fix common trailing comma issues
	cleaned = regexp.MustCompile(",\\s*]").ReplaceAllString(cleaned, "]")
	cleaned = regexp.MustCompile(",\\s*}").ReplaceAllString(cleaned, "}")
	
	// Fix missing quotes around field names
	cleaned = regexp.MustCompile(`(\w+):`).ReplaceAllString(cleaned, `"$1":`)
	
	return cleaned
}

// isValidTaskRequest validates that a task request has required fields
func (jr *JSONRecoverer) isValidTaskRequest(task TaskRequest) bool {
	return task.Description != "" &&
		   task.OriginText != "" &&
		   task.Priority != "" &&
		   task.SourceCommentID != 0
}

// LogRecoveryAttempt logs the recovery attempt if configured
func (jr *JSONRecoverer) LogRecoveryAttempt(result *JSONRecoveryResult) {
	if !jr.config.LogTruncatedResponses {
		return
	}

	if jr.verboseMode {
		fmt.Printf("  📊 JSON Recovery Summary:\n")
		fmt.Printf("    - Error Type: %s\n", result.ErrorType)
		fmt.Printf("    - Recovery Success: %v\n", result.IsRecovered)
		fmt.Printf("    - Tasks Recovered: %d\n", len(result.Tasks))
		fmt.Printf("    - Original Size: %d bytes\n", result.OriginalSize)
		fmt.Printf("    - Message: %s\n", result.Message)
	}
}