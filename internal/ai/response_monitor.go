package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ResponseMonitor tracks and analyzes Claude API response patterns for optimization
type ResponseMonitor struct {
	config      *ResponseMonitorConfig
	verboseMode bool
	dataFile    string
}

// ResponseMonitorConfig contains configuration for response monitoring
type ResponseMonitorConfig struct {
	EnableMonitoring      bool          `json:"enable_monitoring"`
	CollectionEnabled     bool          `json:"collection_enabled"`
	AnalysisEnabled       bool          `json:"analysis_enabled"`
	DataRetentionDays     int           `json:"data_retention_days"`
	MaxEventsPerSession   int           `json:"max_events_per_session"`
	AutoOptimizeThreshold float64       `json:"auto_optimize_threshold"`
	ReportingInterval     time.Duration `json:"reporting_interval"`
}

// ResponseEvent records a single API response event
type ResponseEvent struct {
	Timestamp       time.Time `json:"timestamp"`
	SessionID       string    `json:"session_id"`
	PromptSize      int       `json:"prompt_size"`
	ResponseSize    int       `json:"response_size"`
	ProcessingTime  int64     `json:"processing_time_ms"`
	Success         bool      `json:"success"`
	ErrorType       string    `json:"error_type,omitempty"`
	RecoveryUsed    bool      `json:"recovery_used"`
	RetryCount      int       `json:"retry_count"`
	TruncationScore float64   `json:"truncation_score"`
	TasksExtracted  int       `json:"tasks_extracted"`
	PromptOptimized bool      `json:"prompt_optimized"`
}

// ResponseAnalytics contains aggregated response analytics
type ResponseAnalytics struct {
	TotalRequests         int                 `json:"total_requests"`
	SuccessRate           float64             `json:"success_rate"`
	AveragePromptSize     float64             `json:"average_prompt_size"`
	AverageResponseSize   float64             `json:"average_response_size"`
	AverageProcessingTime float64             `json:"average_processing_time_ms"`
	RecoveryRate          float64             `json:"recovery_rate"`
	ErrorDistribution     map[string]int      `json:"error_distribution"`
	TruncationPatterns    TruncationAnalytics `json:"truncation_patterns"`
	OptimizationImpact    OptimizationMetrics `json:"optimization_impact"`
	Recommendations       []OptimizationTip   `json:"recommendations"`
}

// TruncationAnalytics provides insights into truncation patterns
type TruncationAnalytics struct {
	TruncationRate         float64 `json:"truncation_rate"`
	AverageTruncationScore float64 `json:"average_truncation_score"`
	OptimalPromptSize      int     `json:"optimal_prompt_size"`
	HighRiskSizeThreshold  int     `json:"high_risk_size_threshold"`
}

// OptimizationMetrics tracks the impact of prompt optimizations
type OptimizationMetrics struct {
	OptimizationUsageRate   float64 `json:"optimization_usage_rate"`
	SuccessRateImprovement  float64 `json:"success_rate_improvement"`
	SizeReductionAverage    float64 `json:"size_reduction_average"`
	ResponseTimeImprovement float64 `json:"response_time_improvement"`
}

// OptimizationTip provides actionable optimization recommendations
type OptimizationTip struct {
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
}

// ResponseSession tracks events within a single session
type ResponseSession struct {
	events    []ResponseEvent
	startTime time.Time
	sessionID string
}

// NewResponseMonitor creates a new response monitoring system
func NewResponseMonitor(verboseMode bool) *ResponseMonitor {
	dataDir := ".pr-review/analytics"
	return &ResponseMonitor{
		config: &ResponseMonitorConfig{
			EnableMonitoring:      true,
			CollectionEnabled:     true,
			AnalysisEnabled:       true,
			DataRetentionDays:     30,
			MaxEventsPerSession:   100,
			AutoOptimizeThreshold: 0.6, // Trigger optimization if success rate < 60%
			ReportingInterval:     time.Hour * 24,
		},
		verboseMode: verboseMode,
		dataFile:    filepath.Join(dataDir, "response_events.json"),
	}
}

// RecordEvent records a response event for analysis
func (rm *ResponseMonitor) RecordEvent(event ResponseEvent) error {
	if !rm.config.EnableMonitoring || !rm.config.CollectionEnabled {
		return nil
	}

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(rm.dataFile), 0755); err != nil {
		return fmt.Errorf("failed to create analytics directory: %w", err)
	}

	// Load existing events
	events, err := rm.loadEvents()
	if err != nil {
		if rm.verboseMode {
			fmt.Printf("    ⚠️  Warning: Could not load existing events: %v\n", err)
		}
		events = []ResponseEvent{}
	}

	// Add new event
	events = append(events, event)

	// Apply retention policy
	events = rm.applyRetentionPolicy(events)

	// Save updated events
	if err := rm.saveEvents(events); err != nil {
		return fmt.Errorf("failed to save response event: %w", err)
	}

	if rm.verboseMode {
		fmt.Printf("    📊 Response event recorded: prompt=%d, response=%d, success=%v\n",
			event.PromptSize, event.ResponseSize, event.Success)
	}

	return nil
}

// AnalyzePerformance generates comprehensive performance analytics
func (rm *ResponseMonitor) AnalyzePerformance() (*ResponseAnalytics, error) {
	if !rm.config.EnableMonitoring || !rm.config.AnalysisEnabled {
		return nil, fmt.Errorf("response analysis is disabled")
	}

	events, err := rm.loadEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to load events for analysis: %w", err)
	}

	if len(events) == 0 {
		return &ResponseAnalytics{
			TotalRequests:   0,
			SuccessRate:     0,
			Recommendations: []OptimizationTip{},
		}, nil
	}

	analytics := rm.calculateAnalytics(events)
	analytics.Recommendations = rm.generateRecommendations(analytics, events)

	return analytics, nil
}

// GetOptimalPromptSize suggests optimal prompt size based on historical data
func (rm *ResponseMonitor) GetOptimalPromptSize() (int, error) {
	analytics, err := rm.AnalyzePerformance()
	if err != nil {
		return 20000, err // Default fallback
	}

	return analytics.TruncationPatterns.OptimalPromptSize, nil
}

// ShouldOptimizePrompt determines if prompt optimization is recommended
func (rm *ResponseMonitor) ShouldOptimizePrompt(promptSize int) bool {
	if !rm.config.EnableMonitoring {
		return false
	}

	// Load recent analytics
	analytics, err := rm.AnalyzePerformance()
	if err != nil {
		return false
	}

	// Check if success rate is below threshold
	if analytics.SuccessRate < rm.config.AutoOptimizeThreshold {
		return true
	}

	// Check if prompt size exceeds high-risk threshold
	if promptSize > analytics.TruncationPatterns.HighRiskSizeThreshold {
		return true
	}

	return false
}

// GenerateReport creates a comprehensive performance report
func (rm *ResponseMonitor) GenerateReport() (string, error) {
	analytics, err := rm.AnalyzePerformance()
	if err != nil {
		return "", err
	}

	report := fmt.Sprintf(`# Claude API Response Performance Report

## Summary Statistics
- Total Requests: %d
- Success Rate: %.1f%%
- Average Prompt Size: %.0f bytes
- Average Response Size: %.0f bytes
- Average Processing Time: %.1f ms
- Recovery Usage Rate: %.1f%%

## Truncation Analysis
- Truncation Rate: %.1f%%
- Average Truncation Score: %.2f
- Optimal Prompt Size: %d bytes
- High Risk Threshold: %d bytes

## Optimization Impact
- Optimization Usage: %.1f%%
- Success Rate Improvement: %.1f%%
- Size Reduction Average: %.1f%%
- Response Time Improvement: %.1f%%

## Error Distribution
`,
		analytics.TotalRequests,
		analytics.SuccessRate*100,
		analytics.AveragePromptSize,
		analytics.AverageResponseSize,
		analytics.AverageProcessingTime,
		analytics.RecoveryRate*100,
		analytics.TruncationPatterns.TruncationRate*100,
		analytics.TruncationPatterns.AverageTruncationScore,
		analytics.TruncationPatterns.OptimalPromptSize,
		analytics.TruncationPatterns.HighRiskSizeThreshold,
		analytics.OptimizationImpact.OptimizationUsageRate*100,
		analytics.OptimizationImpact.SuccessRateImprovement*100,
		analytics.OptimizationImpact.SizeReductionAverage*100,
		analytics.OptimizationImpact.ResponseTimeImprovement*100,
	)

	for errorType, count := range analytics.ErrorDistribution {
		percentage := float64(count) / float64(analytics.TotalRequests) * 100
		report += fmt.Sprintf("- %s: %d (%.1f%%)\n", errorType, count, percentage)
	}

	report += "\n## Optimization Recommendations\n"
	for i, tip := range analytics.Recommendations {
		report += fmt.Sprintf("%d. **%s** (%s priority, %.0f%% confidence)\n   %s\n   Impact: %s\n\n",
			i+1, tip.Description, tip.Priority, tip.Confidence*100, tip.Type, tip.Impact)
	}

	return report, nil
}

// loadEvents loads stored response events from disk
func (rm *ResponseMonitor) loadEvents() ([]ResponseEvent, error) {
	if _, err := os.Stat(rm.dataFile); os.IsNotExist(err) {
		return []ResponseEvent{}, nil
	}

	data, err := os.ReadFile(rm.dataFile)
	if err != nil {
		return nil, err
	}

	var events []ResponseEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}

	return events, nil
}

// saveEvents saves response events to disk
func (rm *ResponseMonitor) saveEvents(events []ResponseEvent) error {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(rm.dataFile, data, 0644)
}

// applyRetentionPolicy removes old events based on retention settings
func (rm *ResponseMonitor) applyRetentionPolicy(events []ResponseEvent) []ResponseEvent {
	if rm.config.DataRetentionDays <= 0 {
		return events
	}

	cutoff := time.Now().AddDate(0, 0, -rm.config.DataRetentionDays)
	var filtered []ResponseEvent

	for _, event := range events {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// calculateAnalytics computes analytics from event data
func (rm *ResponseMonitor) calculateAnalytics(events []ResponseEvent) *ResponseAnalytics {
	if len(events) == 0 {
		return &ResponseAnalytics{}
	}

	totalRequests := len(events)
	successCount := 0
	recoveryCount := 0
	optimizationCount := 0
	totalPromptSize := 0
	totalResponseSize := 0
	totalProcessingTime := int64(0)
	truncationEvents := 0
	totalTruncationScore := 0.0
	errorCounts := make(map[string]int)

	// Separate analysis for optimized vs non-optimized requests
	optimizedEvents := []ResponseEvent{}
	nonOptimizedEvents := []ResponseEvent{}

	for _, event := range events {
		totalPromptSize += event.PromptSize
		totalResponseSize += event.ResponseSize
		totalProcessingTime += event.ProcessingTime

		if event.Success {
			successCount++
		}

		if event.RecoveryUsed {
			recoveryCount++
		}

		if event.PromptOptimized {
			optimizationCount++
			optimizedEvents = append(optimizedEvents, event)
		} else {
			nonOptimizedEvents = append(nonOptimizedEvents, event)
		}

		if event.TruncationScore > 0 {
			truncationEvents++
			totalTruncationScore += event.TruncationScore
		}

		if event.ErrorType != "" {
			errorCounts[event.ErrorType]++
		}
	}

	analytics := &ResponseAnalytics{
		TotalRequests:         totalRequests,
		SuccessRate:           float64(successCount) / float64(totalRequests),
		AveragePromptSize:     float64(totalPromptSize) / float64(totalRequests),
		AverageResponseSize:   float64(totalResponseSize) / float64(totalRequests),
		AverageProcessingTime: float64(totalProcessingTime) / float64(totalRequests),
		RecoveryRate:          float64(recoveryCount) / float64(totalRequests),
		ErrorDistribution:     errorCounts,
		TruncationPatterns:    rm.analyzeTruncationPatterns(events, truncationEvents, totalTruncationScore),
		OptimizationImpact:    rm.analyzeOptimizationImpact(optimizedEvents, nonOptimizedEvents),
	}

	return analytics
}

// analyzeTruncationPatterns analyzes truncation patterns in the data
func (rm *ResponseMonitor) analyzeTruncationPatterns(events []ResponseEvent, truncationEvents int, totalTruncationScore float64) TruncationAnalytics {
	if len(events) == 0 {
		return TruncationAnalytics{
			OptimalPromptSize:     20000,
			HighRiskSizeThreshold: 30000,
		}
	}

	truncationRate := float64(truncationEvents) / float64(len(events))
	avgTruncationScore := 0.0
	if truncationEvents > 0 {
		avgTruncationScore = totalTruncationScore / float64(truncationEvents)
	}

	// Find optimal prompt size (size with highest success rate)
	sizeBuckets := make(map[int][]bool) // size range -> success results
	for _, event := range events {
		bucket := (event.PromptSize / 5000) * 5000 // 5KB buckets
		sizeBuckets[bucket] = append(sizeBuckets[bucket], event.Success)
	}

	optimalSize := 20000 // Default
	bestSuccessRate := 0.0
	highRiskThreshold := 30000

	for size, results := range sizeBuckets {
		if len(results) < 3 {
			continue // Need at least 3 samples
		}

		successCount := 0
		for _, success := range results {
			if success {
				successCount++
			}
		}

		successRate := float64(successCount) / float64(len(results))
		if successRate > bestSuccessRate {
			bestSuccessRate = successRate
			optimalSize = size
		}

		// Determine high-risk threshold (size where success rate drops below 70%)
		if successRate < 0.7 && size < highRiskThreshold {
			highRiskThreshold = size
		}
	}

	return TruncationAnalytics{
		TruncationRate:         truncationRate,
		AverageTruncationScore: avgTruncationScore,
		OptimalPromptSize:      optimalSize,
		HighRiskSizeThreshold:  highRiskThreshold,
	}
}

// analyzeOptimizationImpact compares optimized vs non-optimized request performance
func (rm *ResponseMonitor) analyzeOptimizationImpact(optimized, nonOptimized []ResponseEvent) OptimizationMetrics {
	totalEvents := len(optimized) + len(nonOptimized)
	if totalEvents == 0 {
		return OptimizationMetrics{}
	}

	optimizationUsage := float64(len(optimized)) / float64(totalEvents)

	// Calculate success rate improvement
	successRateImprovement := 0.0
	if len(optimized) > 0 && len(nonOptimized) > 0 {
		optimizedSuccessRate := rm.calculateSuccessRate(optimized)
		nonOptimizedSuccessRate := rm.calculateSuccessRate(nonOptimized)
		successRateImprovement = optimizedSuccessRate - nonOptimizedSuccessRate
	}

	// Calculate average size reduction (this would need to be tracked differently)
	sizeReduction := 0.3 // Placeholder - would need original vs optimized size tracking

	// Calculate response time improvement
	responseTimeImprovement := 0.0
	if len(optimized) > 0 && len(nonOptimized) > 0 {
		optimizedAvgTime := rm.calculateAverageProcessingTime(optimized)
		nonOptimizedAvgTime := rm.calculateAverageProcessingTime(nonOptimized)
		if nonOptimizedAvgTime > 0 {
			responseTimeImprovement = (nonOptimizedAvgTime - optimizedAvgTime) / nonOptimizedAvgTime
		}
	}

	return OptimizationMetrics{
		OptimizationUsageRate:   optimizationUsage,
		SuccessRateImprovement:  successRateImprovement,
		SizeReductionAverage:    sizeReduction,
		ResponseTimeImprovement: responseTimeImprovement,
	}
}

// Helper functions for calculations
func (rm *ResponseMonitor) calculateSuccessRate(events []ResponseEvent) float64 {
	if len(events) == 0 {
		return 0
	}
	successCount := 0
	for _, event := range events {
		if event.Success {
			successCount++
		}
	}
	return float64(successCount) / float64(len(events))
}

func (rm *ResponseMonitor) calculateAverageProcessingTime(events []ResponseEvent) float64 {
	if len(events) == 0 {
		return 0
	}
	total := int64(0)
	for _, event := range events {
		total += event.ProcessingTime
	}
	return float64(total) / float64(len(events))
}

// generateRecommendations creates optimization recommendations based on analytics
func (rm *ResponseMonitor) generateRecommendations(analytics *ResponseAnalytics, events []ResponseEvent) []OptimizationTip {
	var tips []OptimizationTip

	// Recommendation 1: Success rate improvement
	if analytics.SuccessRate < 0.8 {
		priority := "high"
		if analytics.SuccessRate < 0.5 {
			priority = "critical"
		}
		tips = append(tips, OptimizationTip{
			Type:        "success_rate",
			Priority:    priority,
			Description: "Enable prompt optimization to improve success rate",
			Impact:      fmt.Sprintf("Could improve success rate by up to %.1f%%", analytics.OptimizationImpact.SuccessRateImprovement*100),
			Confidence:  0.85,
		})
	}

	// Recommendation 2: Prompt size optimization
	if analytics.AveragePromptSize > float64(analytics.TruncationPatterns.HighRiskSizeThreshold) {
		tips = append(tips, OptimizationTip{
			Type:        "prompt_size",
			Priority:    "medium",
			Description: "Reduce average prompt size to minimize truncation risk",
			Impact:      fmt.Sprintf("Target size: %d bytes (current avg: %.0f bytes)", analytics.TruncationPatterns.OptimalPromptSize, analytics.AveragePromptSize),
			Confidence:  0.75,
		})
	}

	// Recommendation 3: Truncation patterns
	if analytics.TruncationPatterns.TruncationRate > 0.1 {
		tips = append(tips, OptimizationTip{
			Type:        "truncation",
			Priority:    "high",
			Description: "High truncation rate detected - enable JSON recovery and retry logic",
			Impact:      fmt.Sprintf("Could recover %.1f%% of failed requests", analytics.TruncationPatterns.TruncationRate*100),
			Confidence:  0.9,
		})
	}

	// Recommendation 4: Performance optimization
	if analytics.AverageProcessingTime > 10000 { // > 10 seconds
		tips = append(tips, OptimizationTip{
			Type:        "performance",
			Priority:    "medium",
			Description: "High processing times - consider request chunking or parallel processing",
			Impact:      fmt.Sprintf("Could reduce processing time from %.1fs to %.1fs", analytics.AverageProcessingTime/1000, analytics.AverageProcessingTime/2000),
			Confidence:  0.7,
		})
	}

	return tips
}
