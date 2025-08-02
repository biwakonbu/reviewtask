package ai

import (
	"fmt"
	"strings"
	"time"
)

// RetryStrategy handles intelligent retry logic for incomplete/failed API responses
type RetryStrategy struct {
	config       *RetryConfig
	verboseMode  bool
	truncationPattern *TruncationPatternDetector
}

// RetryConfig contains configuration for retry behavior
type RetryConfig struct {
	EnableSmartRetry      bool          `json:"enable_smart_retry"`
	MaxRetries           int           `json:"max_retries"`
	BaseDelay            time.Duration `json:"base_delay"`
	MaxDelay             time.Duration `json:"max_delay"`
	BackoffMultiplier    float64       `json:"backoff_multiplier"`
	TruncationThreshold  int           `json:"truncation_threshold"`
	PromptSizeReduction  float64       `json:"prompt_size_reduction"`
}

// RetryAttempt contains information about a retry attempt
type RetryAttempt struct {
	AttemptNumber    int           `json:"attempt_number"`
	Strategy         string        `json:"strategy"`
	PromptSize       int           `json:"prompt_size"`
	ResponseSize     int           `json:"response_size"`
	Error            string        `json:"error"`
	TruncationScore  float64       `json:"truncation_score"`
	Delay            time.Duration `json:"delay"`
	AdjustedPrompt   bool          `json:"adjusted_prompt"`
}

// TruncationPatternDetector analyzes response patterns to detect truncation issues
type TruncationPatternDetector struct {
	responseSizes     []int
	truncationEvents  []TruncationEvent
	verboseMode       bool
}

// TruncationEvent records a truncation occurrence
type TruncationEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	PromptSize   int       `json:"prompt_size"`
	ResponseSize int       `json:"response_size"`
	ErrorType    string    `json:"error_type"`
}

// NewRetryStrategy creates a new retry strategy handler
func NewRetryStrategy(verboseMode bool) *RetryStrategy {
	return &RetryStrategy{
		config: &RetryConfig{
			EnableSmartRetry:     true,
			MaxRetries:          3,
			BaseDelay:           time.Second * 2,
			MaxDelay:            time.Second * 30,
			BackoffMultiplier:   2.0,
			TruncationThreshold: 20000, // 20KB threshold for prompt size adjustments
			PromptSizeReduction: 0.7,   // Reduce prompt size by 30%
		},
		verboseMode:       verboseMode,
		truncationPattern: NewTruncationPatternDetector(verboseMode),
	}
}

// NewTruncationPatternDetector creates a new truncation pattern detector
func NewTruncationPatternDetector(verboseMode bool) *TruncationPatternDetector {
	return &TruncationPatternDetector{
		responseSizes:    make([]int, 0),
		truncationEvents: make([]TruncationEvent, 0),
		verboseMode:      verboseMode,
	}
}

// ShouldRetry determines if a retry should be attempted based on error analysis
func (rs *RetryStrategy) ShouldRetry(attempt int, err error, promptSize int, responseSize int) (*RetryAttempt, bool) {
	if !rs.config.EnableSmartRetry || attempt >= rs.config.MaxRetries {
		return nil, false
	}

	// Analyze the error to determine retry strategy
	errorType := rs.categorizeRetryError(err)
	truncationScore := rs.truncationPattern.AnalyzeResponse(promptSize, responseSize, errorType)
	
	retryAttempt := &RetryAttempt{
		AttemptNumber:   attempt + 1,
		Strategy:        rs.determineRetryStrategy(errorType, truncationScore, promptSize),
		PromptSize:      promptSize,
		ResponseSize:    responseSize,
		Error:           err.Error(),
		TruncationScore: truncationScore,
		Delay:           rs.calculateDelay(attempt),
		AdjustedPrompt:  false,
	}

	// Record truncation event for pattern analysis
	rs.truncationPattern.RecordTruncation(promptSize, responseSize, errorType)

	if rs.verboseMode {
		fmt.Printf("  🔄 Retry analysis (attempt %d):\n", retryAttempt.AttemptNumber)
		fmt.Printf("    - Error type: %s\n", errorType)
		fmt.Printf("    - Truncation score: %.2f\n", truncationScore)
		fmt.Printf("    - Strategy: %s\n", retryAttempt.Strategy)
		fmt.Printf("    - Delay: %v\n", retryAttempt.Delay)
	}

	return retryAttempt, true
}

// categorizeRetryError categorizes errors for retry strategy selection
func (rs *RetryStrategy) categorizeRetryError(err error) string {
	errMsg := strings.ToLower(err.Error())
	
	if strings.Contains(errMsg, "unexpected end of json input") ||
	   strings.Contains(errMsg, "unexpected end of input") {
		return "json_truncation"
	}
	
	if strings.Contains(errMsg, "prompt size") && strings.Contains(errMsg, "exceeds") {
		return "prompt_size_limit"
	}
	
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "too many requests") {
		return "rate_limit"
	}
	
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded") {
		return "timeout"
	}
	
	if strings.Contains(errMsg, "network") || strings.Contains(errMsg, "connection") {
		return "network_error"
	}
	
	if strings.Contains(errMsg, "invalid character") {
		return "malformed_response"
	}
	
	return "unknown_error"
}

// determineRetryStrategy selects the appropriate retry strategy
func (rs *RetryStrategy) determineRetryStrategy(errorType string, truncationScore float64, promptSize int) string {
	switch errorType {
	case "json_truncation":
		if truncationScore > 0.7 {
			return "reduce_prompt_aggressive"
		} else if promptSize > rs.config.TruncationThreshold {
			return "reduce_prompt_moderate"
		}
		return "simple_retry"
		
	case "prompt_size_limit":
		return "reduce_prompt_aggressive"
		
	case "rate_limit":
		return "exponential_backoff"
		
	case "timeout":
		if promptSize > rs.config.TruncationThreshold {
			return "reduce_prompt_moderate"
		}
		return "exponential_backoff"
		
	case "network_error":
		return "exponential_backoff"
		
	case "malformed_response":
		return "simple_retry"
		
	default:
		return "simple_retry"
	}
}

// calculateDelay calculates the retry delay using exponential backoff
func (rs *RetryStrategy) calculateDelay(attempt int) time.Duration {
	delay := rs.config.BaseDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rs.config.BackoffMultiplier)
	}
	
	if delay > rs.config.MaxDelay {
		delay = rs.config.MaxDelay
	}
	
	return delay
}

// AdjustPromptForRetry modifies the prompt based on retry strategy
func (rs *RetryStrategy) AdjustPromptForRetry(originalPrompt string, retryAttempt *RetryAttempt) string {
	switch retryAttempt.Strategy {
	case "reduce_prompt_aggressive":
		return rs.reducePromptSize(originalPrompt, 0.5) // 50% reduction
	case "reduce_prompt_moderate":
		return rs.reducePromptSize(originalPrompt, rs.config.PromptSizeReduction)
	default:
		return originalPrompt
	}
}

// reducePromptSize reduces prompt size by the specified factor
func (rs *RetryStrategy) reducePromptSize(prompt string, reductionFactor float64) string {
	if reductionFactor >= 1.0 || reductionFactor <= 0.0 {
		return prompt
	}
	
	// Strategy: Reduce the review data section while preserving system prompts
	lines := strings.Split(prompt, "\n")
	
	// Find the start of review data (usually after "PR Reviews to analyze:")
	reviewStartIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "PR Reviews to analyze:") ||
		   strings.Contains(line, "Review Context:") ||
		   strings.Contains(line, "Comment Details:") {
			reviewStartIndex = i
			break
		}
	}
	
	if reviewStartIndex == -1 {
		// If we can't find review data section, reduce overall length
		targetSize := int(float64(len(prompt)) * reductionFactor)
		if targetSize < len(prompt) {
			return prompt[:targetSize] + "\n\n[Content truncated for retry]"
		}
		return prompt
	}
	
	// Calculate how much review data to keep
	systemPromptSize := 0
	for i := 0; i < reviewStartIndex; i++ {
		systemPromptSize += len(lines[i]) + 1 // +1 for newline
	}
	
	availableSize := int(float64(len(prompt)) * reductionFactor)
	reviewDataBudget := availableSize - systemPromptSize
	
	if reviewDataBudget <= 0 {
		// If system prompt is too large, return minimal version
		return strings.Join(lines[:reviewStartIndex], "\n") + "\n\n[Review data omitted for retry]"
	}
	
	// Truncate review data to fit budget
	reviewData := strings.Join(lines[reviewStartIndex:], "\n")
	if len(reviewData) > reviewDataBudget {
		reviewData = reviewData[:reviewDataBudget] + "\n\n[Content truncated for retry]"
	}
	
	return strings.Join(lines[:reviewStartIndex], "\n") + "\n" + reviewData
}

// ExecuteDelay waits for the calculated retry delay
func (rs *RetryStrategy) ExecuteDelay(retryAttempt *RetryAttempt) {
	if retryAttempt.Delay > 0 {
		if rs.verboseMode {
			fmt.Printf("  ⏱️  Waiting %v before retry %d...\n", retryAttempt.Delay, retryAttempt.AttemptNumber)
		}
		time.Sleep(retryAttempt.Delay)
	}
}

// AnalyzeResponse analyzes response patterns for truncation detection
func (tpd *TruncationPatternDetector) AnalyzeResponse(promptSize, responseSize int, errorType string) float64 {
	tpd.responseSizes = append(tpd.responseSizes, responseSize)
	
	// Keep only recent response sizes (last 10)
	if len(tpd.responseSizes) > 10 {
		tpd.responseSizes = tpd.responseSizes[len(tpd.responseSizes)-10:]
	}
	
	// Calculate truncation likelihood score
	score := 0.0
	
	// Factor 1: Error type indicates truncation
	if errorType == "json_truncation" {
		score += 0.4
	}
	
	// Factor 2: Prompt size vs typical limits
	if promptSize > 30000 { // 30KB
		score += 0.3
	} else if promptSize > 20000 { // 20KB
		score += 0.2
	}
	
	// Factor 3: Response size patterns
	if len(tpd.responseSizes) >= 3 {
		// Check if recent responses are getting smaller
		recent := tpd.responseSizes[len(tpd.responseSizes)-3:]
		if recent[2] < recent[1] && recent[1] < recent[0] {
			score += 0.2
		}
	}
	
	// Factor 4: Absolute response size
	if responseSize > 0 && responseSize < 1000 {
		score += 0.1
	}
	
	return score
}

// RecordTruncation records a truncation event for pattern analysis
func (tpd *TruncationPatternDetector) RecordTruncation(promptSize, responseSize int, errorType string) {
	event := TruncationEvent{
		Timestamp:    time.Now(),
		PromptSize:   promptSize,
		ResponseSize: responseSize,
		ErrorType:    errorType,
	}
	
	tpd.truncationEvents = append(tpd.truncationEvents, event)
	
	// Keep only recent events (last 20)
	if len(tpd.truncationEvents) > 20 {
		tpd.truncationEvents = tpd.truncationEvents[len(tpd.truncationEvents)-20:]
	}
	
	if tpd.verboseMode {
		fmt.Printf("    📊 Truncation event recorded: prompt=%d, response=%d, type=%s\n", 
			promptSize, responseSize, errorType)
	}
}

// GetTruncationStats returns statistics about truncation patterns
func (tpd *TruncationPatternDetector) GetTruncationStats() map[string]interface{} {
	if len(tpd.truncationEvents) == 0 {
		return map[string]interface{}{
			"total_events": 0,
			"average_prompt_size": 0,
			"average_response_size": 0,
		}
	}
	
	totalPromptSize := 0
	totalResponseSize := 0
	errorTypeCounts := make(map[string]int)
	
	for _, event := range tpd.truncationEvents {
		totalPromptSize += event.PromptSize
		totalResponseSize += event.ResponseSize
		errorTypeCounts[event.ErrorType]++
	}
	
	return map[string]interface{}{
		"total_events": len(tpd.truncationEvents),
		"average_prompt_size": totalPromptSize / len(tpd.truncationEvents),
		"average_response_size": totalResponseSize / len(tpd.truncationEvents),
		"error_type_distribution": errorTypeCounts,
	}
}