package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// EnhancedJSONRecovery provides advanced JSON repair capabilities
type EnhancedJSONRecovery struct {
	verboseMode bool
	config      *JSONRecoveryConfig
}

// NewEnhancedJSONRecovery creates an enhanced JSON recovery system
func NewEnhancedJSONRecovery(enableRecovery bool, verboseMode bool) *EnhancedJSONRecovery {
	return &EnhancedJSONRecovery{
		verboseMode: verboseMode,
		config: &JSONRecoveryConfig{
			EnableRecovery:        enableRecovery,
			MaxRecoveryAttempts:   5,   // Increased for enhanced recovery
			PartialThreshold:      0.6, // Lower threshold for more aggressive recovery
			LogTruncatedResponses: true,
		},
	}
}

// RepairAndRecover attempts to repair JSON structure and recover tasks
func (ejr *EnhancedJSONRecovery) RepairAndRecover(rawResponse string, originalError error) *JSONRecoveryResult {
	result := &JSONRecoveryResult{
		IsRecovered:   false,
		Tasks:         []TaskRequest{},
		ErrorType:     ejr.categorizeError(originalError),
		Message:       originalError.Error(),
		OriginalSize:  len(rawResponse),
		RecoveredSize: 0,
	}

	if !ejr.config.EnableRecovery {
		result.Message = "Enhanced JSON recovery disabled"
		return result
	}

	if ejr.verboseMode {
		fmt.Printf("  🔧 Enhanced JSON recovery for %s error (size: %d bytes)\n",
			result.ErrorType, result.OriginalSize)
	}

	// Enhanced recovery strategies
	strategies := []func(string, *JSONRecoveryResult) *JSONRecoveryResult{
		ejr.repairStructuralIssues,
		ejr.completeTruncatedJSON,
		ejr.extractPartialStructures,
		ejr.reconstructFromFragments,
		ejr.intelligentFieldCompletion,
	}

	for i, strategy := range strategies {
		if ejr.verboseMode {
			fmt.Printf("    🎯 Trying strategy %d...\n", i+1)
		}

		attemptResult := strategy(rawResponse, result)
		if attemptResult.IsRecovered && len(attemptResult.Tasks) > 0 {
			if ejr.verboseMode {
				fmt.Printf("    ✅ Strategy %d successful: recovered %d tasks\n", i+1, len(attemptResult.Tasks))
			}
			return attemptResult
		}
	}

	if ejr.verboseMode {
		fmt.Printf("    ❌ All enhanced recovery strategies failed\n")
	}

	return result
}

// repairStructuralIssues fixes common JSON structural problems
func (ejr *EnhancedJSONRecovery) repairStructuralIssues(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	repaired := response

	// Fix common structural issues
	fixes := []struct {
		pattern     string
		replacement string
		description string
	}{
		{`}\s*{`, `}, {`, "Fix missing comma between objects"},
		{`"\s*:\s*"([^"]*)"([^,}\]])`, `": "$1"$2`, "Fix unterminated string values"},
		{`]\s*\[`, `], [`, "Fix missing comma between arrays"},
		{`([^,\[\{])\s*"`, `$1, "`, "Add missing comma before quote"},
		{`([^,\[\{])\s*\{`, `$1, {`, "Add missing comma before object"},
		{`}\s*]$`, `}]`, "Remove trailing whitespace before array end"},
		{`^\s*\[?\s*`, `[`, "Ensure array starts properly"},
		{`\s*\]?\s*$`, `]`, "Ensure array ends properly"},
	}

	for _, fix := range fixes {
		re := regexp.MustCompile(fix.pattern)
		if re.MatchString(repaired) {
			oldRepaired := repaired
			repaired = re.ReplaceAllString(repaired, fix.replacement)
			if ejr.verboseMode && repaired != oldRepaired {
				fmt.Printf("      🔧 Applied fix: %s\n", fix.description)
			}
		}
	}

	// Try to parse repaired JSON
	if tasks := ejr.tryParseJSON(repaired); tasks != nil {
		result.IsRecovered = true
		result.Tasks = tasks
		result.RecoveredSize = len(repaired)
		result.Message = "Repaired structural JSON issues"
		return result
	}

	return result
}

// completeTruncatedJSON attempts to complete truncated JSON
func (ejr *EnhancedJSONRecovery) completeTruncatedJSON(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	// Find the last complete object or partial object
	lastBraceIndex := strings.LastIndex(response, "{")
	if lastBraceIndex == -1 {
		return result
	}

	// Try to complete the truncated object
	truncatedPart := response[lastBraceIndex:]

	// Add minimal completion to make it parseable
	completionStrategies := []string{
		truncatedPart + `}]`,                                          // Simple object end
		truncatedPart + `", "priority": "medium"}]`,                   // Complete common field
		truncatedPart + `", "priority": "medium", "status": "todo"}]`, // Complete with defaults
	}

	for _, completed := range completionStrategies {
		fullResponse := response[:lastBraceIndex] + completed
		if tasks := ejr.tryParseJSON(fullResponse); tasks != nil {
			result.IsRecovered = true
			result.Tasks = tasks
			result.RecoveredSize = len(fullResponse)
			result.Message = "Completed truncated JSON structure"
			if ejr.verboseMode {
				fmt.Printf("      ✅ Completed with strategy: %s\n", completed[len(truncatedPart):])
			}
			return result
		}
	}

	return result
}

// extractPartialStructures extracts valid structures from malformed JSON
func (ejr *EnhancedJSONRecovery) extractPartialStructures(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	// Use regex to find task-like structures
	taskPattern := regexp.MustCompile(`\{\s*"[^"]*description[^"]*"\s*:\s*"([^"]*)"[^}]*\}`)
	matches := taskPattern.FindAllString(response, -1)

	var recoveredTasks []TaskRequest

	for _, match := range matches {
		// Try to extract and complete partial task structures
		if task := ejr.parsePartialTask(match); task != nil {
			recoveredTasks = append(recoveredTasks, *task)
		}
	}

	if len(recoveredTasks) > 0 {
		result.IsRecovered = true
		result.Tasks = recoveredTasks
		result.RecoveredSize = len(response)
		result.Message = fmt.Sprintf("Extracted %d partial task structures", len(recoveredTasks))
		return result
	}

	return result
}

// reconstructFromFragments attempts to reconstruct JSON from fragments
func (ejr *EnhancedJSONRecovery) reconstructFromFragments(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	// Find all string values that look like descriptions
	descPattern := regexp.MustCompile(`"([^"]{20,})"`)
	descriptions := descPattern.FindAllStringSubmatch(response, -1)

	if len(descriptions) == 0 {
		return result
	}

	var reconstructedTasks []TaskRequest
	for _, desc := range descriptions {
		if len(desc) > 1 && ejr.looksLikeTaskDescription(desc[1]) {
			task := TaskRequest{
				Description: desc[1],
				Priority:    "medium", // Default priority
				Status:      "todo",   // Default status
			}
			reconstructedTasks = append(reconstructedTasks, task)
		}
	}

	if len(reconstructedTasks) > 0 {
		result.IsRecovered = true
		result.Tasks = reconstructedTasks
		result.RecoveredSize = len(response)
		result.Message = fmt.Sprintf("Reconstructed %d tasks from fragments", len(reconstructedTasks))
		return result
	}

	return result
}

// intelligentFieldCompletion completes missing fields with intelligent defaults
func (ejr *EnhancedJSONRecovery) intelligentFieldCompletion(response string, result *JSONRecoveryResult) *JSONRecoveryResult {
	// Try to find incomplete JSON objects and complete them
	objPattern := regexp.MustCompile(`\{[^}]*"description"\s*:\s*"([^"]*)"[^}]*`)
	matches := objPattern.FindAllString(response, -1)

	var completedObjects []string
	for _, match := range matches {
		completed := ejr.completeTaskObject(match)
		if completed != "" {
			completedObjects = append(completedObjects, completed)
		}
	}

	if len(completedObjects) > 0 {
		// Construct a valid JSON array
		jsonArray := "[" + strings.Join(completedObjects, ", ") + "]"
		if tasks := ejr.tryParseJSON(jsonArray); tasks != nil {
			result.IsRecovered = true
			result.Tasks = tasks
			result.RecoveredSize = len(jsonArray)
			result.Message = fmt.Sprintf("Completed %d incomplete objects", len(tasks))
			return result
		}
	}

	return result
}

// Helper methods

func (ejr *EnhancedJSONRecovery) tryParseJSON(jsonStr string) []TaskRequest {
	var tasks []TaskRequest
	if err := json.Unmarshal([]byte(jsonStr), &tasks); err == nil {
		return tasks
	}
	return nil
}

func (ejr *EnhancedJSONRecovery) parsePartialTask(jsonStr string) *TaskRequest {
	// Try to parse as a complete task first
	var task TaskRequest
	if err := json.Unmarshal([]byte(jsonStr), &task); err == nil {
		// Fill in missing required fields
		if task.Priority == "" {
			task.Priority = "medium"
		}
		if task.Status == "" {
			task.Status = "todo"
		}
		return &task
	}

	// Try to extract description manually and create minimal task
	descPattern := regexp.MustCompile(`"description"\s*:\s*"([^"]*)"`)
	if match := descPattern.FindStringSubmatch(jsonStr); len(match) > 1 {
		return &TaskRequest{
			Description: match[1],
			Priority:    "medium",
			Status:      "todo",
		}
	}

	return nil
}

func (ejr *EnhancedJSONRecovery) looksLikeTaskDescription(text string) bool {
	// Simple heuristics to identify task descriptions
	taskKeywords := []string{"fix", "add", "update", "remove", "change", "implement", "check"}
	lowText := strings.ToLower(text)

	for _, keyword := range taskKeywords {
		if strings.Contains(lowText, keyword) {
			return true
		}
	}

	// Also accept if it's a reasonable length and contains common patterns
	return len(text) > 10 && len(text) < 500
}

func (ejr *EnhancedJSONRecovery) completeTaskObject(partialObj string) string {
	// Ensure the object has all required fields
	required := []string{"description", "priority", "status"}
	defaults := map[string]string{
		"priority": "medium",
		"status":   "todo",
	}

	// Start with the partial object, removing incomplete braces
	obj := strings.TrimSpace(partialObj)
	if !strings.HasPrefix(obj, "{") {
		obj = "{" + obj
	}
	if !strings.HasSuffix(obj, "}") {
		obj = strings.TrimSuffix(obj, ",") + "}"
	}

	// Try to parse to see what's missing
	var taskMap map[string]interface{}
	if err := json.Unmarshal([]byte(obj), &taskMap); err == nil {
		// Add missing fields
		for _, field := range required {
			if _, exists := taskMap[field]; !exists {
				if defaultVal, hasDefault := defaults[field]; hasDefault {
					taskMap[field] = defaultVal
				}
			}
		}

		// Convert back to JSON
		if completed, err := json.Marshal(taskMap); err == nil {
			return string(completed)
		}
	}

	return ""
}

func (ejr *EnhancedJSONRecovery) categorizeError(err error) string {
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
