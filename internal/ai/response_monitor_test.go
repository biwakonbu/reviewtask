package ai

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewResponseMonitor(t *testing.T) {
	tests := []struct {
		name        string
		verboseMode bool
	}{
		{
			name:        "verbose mode enabled",
			verboseMode: true,
		},
		{
			name:        "verbose mode disabled",
			verboseMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewResponseMonitor(tt.verboseMode)

			if monitor.verboseMode != tt.verboseMode {
				t.Errorf("Expected verboseMode=%v, got %v", tt.verboseMode, monitor.verboseMode)
			}

			// Check default configuration
			if !monitor.config.EnableMonitoring {
				t.Error("Expected EnableMonitoring to be true")
			}

			if !monitor.config.CollectionEnabled {
				t.Error("Expected CollectionEnabled to be true")
			}

			if monitor.config.DataRetentionDays != 30 {
				t.Errorf("Expected DataRetentionDays=30, got %d", monitor.config.DataRetentionDays)
			}

			if monitor.config.AutoOptimizeThreshold != 0.6 {
				t.Errorf("Expected AutoOptimizeThreshold=0.6, got %f", monitor.config.AutoOptimizeThreshold)
			}

			expectedDataFile := filepath.Join(".pr-review/analytics", "response_events.json")
			if monitor.dataFile != expectedDataFile {
				t.Errorf("Expected dataFile=%s, got %s", expectedDataFile, monitor.dataFile)
			}
		})
	}
}

func TestResponseMonitor_RecordEvent(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	monitor := NewResponseMonitor(false)
	monitor.dataFile = filepath.Join(tempDir, "test_events.json")

	event := ResponseEvent{
		Timestamp:       time.Now(),
		SessionID:       "test-session",
		PromptSize:      1000,
		ResponseSize:    500,
		ProcessingTime:  2000,
		Success:         true,
		RecoveryUsed:    false,
		RetryCount:      0,
		TasksExtracted:  2,
		PromptOptimized: false,
	}

	// Test successful recording
	err := monitor.RecordEvent(event)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(monitor.dataFile); os.IsNotExist(err) {
		t.Error("Expected data file to be created")
	}

	// Test with monitoring disabled
	monitor.config.EnableMonitoring = false
	err = monitor.RecordEvent(event)
	if err != nil {
		t.Errorf("Expected no error when monitoring disabled, got %v", err)
	}
}

func TestResponseMonitor_AnalyzePerformance(t *testing.T) {
	tempDir := t.TempDir()

	monitor := NewResponseMonitor(false)
	monitor.dataFile = filepath.Join(tempDir, "test_events.json")

	// Test with no events
	analytics, err := monitor.AnalyzePerformance()
	if err != nil {
		t.Errorf("Expected no error with empty data, got %v", err)
	}

	if analytics.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", analytics.TotalRequests)
	}

	// Add some test events
	events := []ResponseEvent{
		{
			Timestamp:       time.Now().Add(-1 * time.Hour),
			SessionID:       "session-1",
			PromptSize:      10000,
			ResponseSize:    5000,
			ProcessingTime:  2000,
			Success:         true,
			RecoveryUsed:    false,
			RetryCount:      0,
			TasksExtracted:  3,
			PromptOptimized: false,
		},
		{
			Timestamp:       time.Now().Add(-30 * time.Minute),
			SessionID:       "session-2",
			PromptSize:      20000,
			ResponseSize:    3000,
			ProcessingTime:  5000,
			Success:         false,
			ErrorType:       "json_truncation",
			RecoveryUsed:    true,
			RetryCount:      2,
			TasksExtracted:  1,
			PromptOptimized: true,
		},
		{
			Timestamp:       time.Now().Add(-10 * time.Minute),
			SessionID:       "session-3",
			PromptSize:      15000,
			ResponseSize:    4000,
			ProcessingTime:  3000,
			Success:         true,
			RecoveryUsed:    true,
			RetryCount:      1,
			TasksExtracted:  2,
			PromptOptimized: false,
		},
	}

	for _, event := range events {
		monitor.RecordEvent(event)
	}

	// Analyze performance
	analytics, err = monitor.AnalyzePerformance()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check basic metrics
	if analytics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", analytics.TotalRequests)
	}

	expectedSuccessRate := 2.0 / 3.0 // 2 successful out of 3
	if analytics.SuccessRate < expectedSuccessRate-0.01 || analytics.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("Expected success rate ~%f, got %f", expectedSuccessRate, analytics.SuccessRate)
	}

	expectedAvgPromptSize := 15000.0 // (10000 + 20000 + 15000) / 3
	if analytics.AveragePromptSize != expectedAvgPromptSize {
		t.Errorf("Expected average prompt size %f, got %f", expectedAvgPromptSize, analytics.AveragePromptSize)
	}

	expectedRecoveryRate := 2.0 / 3.0 // 2 used recovery out of 3
	if analytics.RecoveryRate < expectedRecoveryRate-0.01 || analytics.RecoveryRate > expectedRecoveryRate+0.01 {
		t.Errorf("Expected recovery rate ~%f, got %f", expectedRecoveryRate, analytics.RecoveryRate)
	}

	// Check error distribution
	if analytics.ErrorDistribution["json_truncation"] != 1 {
		t.Errorf("Expected 1 json_truncation error, got %d", analytics.ErrorDistribution["json_truncation"])
	}
}

func TestResponseMonitor_GetOptimalPromptSize(t *testing.T) {
	tempDir := t.TempDir()

	monitor := NewResponseMonitor(false)
	monitor.dataFile = filepath.Join(tempDir, "test_events.json")

	// Test with no data (should return default)
	optimalSize, err := monitor.GetOptimalPromptSize()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if optimalSize != 20000 { // Default fallback
		t.Errorf("Expected default size 20000, got %d", optimalSize)
	}

	// Add events with different prompt sizes and success rates
	events := []ResponseEvent{
		// Small prompts (10KB) - high success rate
		{PromptSize: 10000, Success: true},
		{PromptSize: 10000, Success: true},
		{PromptSize: 10000, Success: true},
		// Medium prompts (15KB) - medium success rate
		{PromptSize: 15000, Success: true},
		{PromptSize: 15000, Success: true},
		{PromptSize: 15000, Success: false},
		// Large prompts (25KB) - low success rate
		{PromptSize: 25000, Success: true},
		{PromptSize: 25000, Success: false},
		{PromptSize: 25000, Success: false},
	}

	for _, event := range events {
		event.Timestamp = time.Now()
		event.SessionID = "test"
		monitor.RecordEvent(event)
	}

	optimalSize, err = monitor.GetOptimalPromptSize()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should prefer the size with highest success rate (10KB bucket)
	if optimalSize != 10000 {
		t.Errorf("Expected optimal size 10000, got %d", optimalSize)
	}
}

func TestResponseMonitor_ShouldOptimizePrompt(t *testing.T) {
	tempDir := t.TempDir()

	monitor := NewResponseMonitor(false)
	monitor.dataFile = filepath.Join(tempDir, "test_events.json")

	// Test with monitoring disabled
	monitor.config.EnableMonitoring = false
	if monitor.ShouldOptimizePrompt(25000) {
		t.Error("Expected no optimization when monitoring disabled")
	}

	// Re-enable monitoring
	monitor.config.EnableMonitoring = true

	// Add events that result in low success rate
	events := []ResponseEvent{
		{Timestamp: time.Now(), SessionID: "test", Success: false},
		{Timestamp: time.Now(), SessionID: "test", Success: false},
		{Timestamp: time.Now(), SessionID: "test", Success: true},
	}

	for _, event := range events {
		monitor.RecordEvent(event)
	}

	// Should recommend optimization due to low success rate (33% < 60% threshold)
	if !monitor.ShouldOptimizePrompt(15000) {
		t.Error("Expected optimization recommendation due to low success rate")
	}
}

func TestResponseMonitor_GenerateReport(t *testing.T) {
	tempDir := t.TempDir()

	monitor := NewResponseMonitor(false)
	monitor.dataFile = filepath.Join(tempDir, "test_events.json")

	// Add some test events
	events := []ResponseEvent{
		{
			Timestamp:       time.Now(),
			SessionID:       "test-1",
			PromptSize:      10000,
			ResponseSize:    5000,
			ProcessingTime:  2000,
			Success:         true,
			RecoveryUsed:    false,
			RetryCount:      0,
			TasksExtracted:  2,
			PromptOptimized: false,
		},
		{
			Timestamp:       time.Now(),
			SessionID:       "test-2",
			PromptSize:      25000,
			ResponseSize:    1000,
			ProcessingTime:  8000,
			Success:         false,
			ErrorType:       "json_truncation",
			RecoveryUsed:    true,
			RetryCount:      2,
			TasksExtracted:  0,
			PromptOptimized: true,
		},
	}

	for _, event := range events {
		monitor.RecordEvent(event)
	}

	report, err := monitor.GenerateReport()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that report contains expected sections
	expectedSections := []string{
		"# Claude API Response Performance Report",
		"## Summary Statistics",
		"## Truncation Analysis",
		"## Optimization Impact",
		"## Error Distribution",
		"## Optimization Recommendations",
	}

	for _, section := range expectedSections {
		if len(report) == 0 {
			t.Errorf("Report is empty")
			break
		}
		// Simple check that section exists in report
		found := false
		for i := 0; i <= len(report)-len(section); i++ {
			if len(report) >= i+len(section) && report[i:i+len(section)] == section {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected report to contain section: %s", section)
		}
	}

	// Check for some key metrics in the report
	expectedMetrics := []string{
		"Total Requests: 2",
		"Success Rate: 50.0%",
		"json_truncation:",
	}

	for _, metric := range expectedMetrics {
		found := false
		for i := 0; i <= len(report)-len(metric); i++ {
			if len(report) >= i+len(metric) && report[i:i+len(metric)] == metric {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected report to contain metric: %s", metric)
		}
	}
}

func TestResponseMonitor_ApplyRetentionPolicy(t *testing.T) {
	monitor := NewResponseMonitor(false)
	monitor.config.DataRetentionDays = 7 // 7 days retention

	now := time.Now()
	events := []ResponseEvent{
		{Timestamp: now.AddDate(0, 0, -10), SessionID: "old-1"},   // 10 days old - should be removed
		{Timestamp: now.AddDate(0, 0, -5), SessionID: "recent-1"}, // 5 days old - should be kept
		{Timestamp: now.AddDate(0, 0, -3), SessionID: "recent-2"}, // 3 days old - should be kept
		{Timestamp: now.AddDate(0, 0, -1), SessionID: "recent-3"}, // 1 day old - should be kept
	}

	filtered := monitor.applyRetentionPolicy(events)

	if len(filtered) != 3 {
		t.Errorf("Expected 3 events after retention policy, got %d", len(filtered))
	}

	// Check that old event was removed
	for _, event := range filtered {
		if event.SessionID == "old-1" {
			t.Error("Expected old event to be filtered out")
		}
	}

	// Test with retention disabled (0 days)
	monitor.config.DataRetentionDays = 0
	filtered = monitor.applyRetentionPolicy(events)

	if len(filtered) != len(events) {
		t.Errorf("Expected all events when retention disabled, got %d", len(filtered))
	}
}

func TestResponseMonitor_CalculateAnalytics(t *testing.T) {
	monitor := NewResponseMonitor(false)

	// Test with empty events
	analytics := monitor.calculateAnalytics([]ResponseEvent{})
	if analytics.TotalRequests != 0 {
		t.Error("Expected 0 requests for empty events")
	}

	// Test with sample events
	events := []ResponseEvent{
		{
			PromptSize:      10000,
			ResponseSize:    5000,
			ProcessingTime:  2000,
			Success:         true,
			RecoveryUsed:    false,
			PromptOptimized: false,
			TruncationScore: 0.0,
		},
		{
			PromptSize:      20000,
			ResponseSize:    3000,
			ProcessingTime:  5000,
			Success:         false,
			ErrorType:       "json_truncation",
			RecoveryUsed:    true,
			PromptOptimized: true,
			TruncationScore: 0.8,
		},
		{
			PromptSize:      15000,
			ResponseSize:    4000,
			ProcessingTime:  3000,
			Success:         true,
			RecoveryUsed:    false,
			PromptOptimized: false,
			TruncationScore: 0.0,
		},
	}

	analytics = monitor.calculateAnalytics(events)

	// Check basic calculations
	if analytics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", analytics.TotalRequests)
	}

	if analytics.SuccessRate != 2.0/3.0 {
		t.Errorf("Expected success rate 0.667, got %f", analytics.SuccessRate)
	}

	if analytics.AveragePromptSize != 15000.0 {
		t.Errorf("Expected average prompt size 15000, got %f", analytics.AveragePromptSize)
	}

	if analytics.RecoveryRate != 1.0/3.0 {
		t.Errorf("Expected recovery rate 0.333, got %f", analytics.RecoveryRate)
	}

	// Check error distribution
	if analytics.ErrorDistribution["json_truncation"] != 1 {
		t.Errorf("Expected 1 json_truncation error, got %d", analytics.ErrorDistribution["json_truncation"])
	}
}

func TestResponseMonitor_AnalyzeTruncationPatterns(t *testing.T) {
	monitor := NewResponseMonitor(false)

	// Test with empty events
	patterns := monitor.analyzeTruncationPatterns([]ResponseEvent{}, 0, 0.0)
	if patterns.OptimalPromptSize != 20000 {
		t.Errorf("Expected default optimal size 20000, got %d", patterns.OptimalPromptSize)
	}

	// Test with sample events
	events := []ResponseEvent{
		{PromptSize: 10000, Success: true, TruncationScore: 0.0},
		{PromptSize: 10000, Success: true, TruncationScore: 0.0},
		{PromptSize: 10000, Success: true, TruncationScore: 0.0},
		{PromptSize: 15000, Success: true, TruncationScore: 0.3},
		{PromptSize: 15000, Success: false, TruncationScore: 0.7},
		{PromptSize: 25000, Success: false, TruncationScore: 0.9},
		{PromptSize: 25000, Success: false, TruncationScore: 0.8},
	}

	patterns = monitor.analyzeTruncationPatterns(events, 4, 2.7)

	// Check truncation rate calculation
	expectedTruncationRate := 4.0 / 7.0
	if patterns.TruncationRate < expectedTruncationRate-0.01 || patterns.TruncationRate > expectedTruncationRate+0.01 {
		t.Errorf("Expected truncation rate ~%f, got %f", expectedTruncationRate, patterns.TruncationRate)
	}

	// Check average truncation score
	expectedAvgScore := 2.7 / 4.0
	if patterns.AverageTruncationScore < expectedAvgScore-0.01 || patterns.AverageTruncationScore > expectedAvgScore+0.01 {
		t.Errorf("Expected avg truncation score ~%f, got %f", expectedAvgScore, patterns.AverageTruncationScore)
	}

	// Should prefer 10KB bucket with 100% success rate
	if patterns.OptimalPromptSize != 10000 {
		t.Errorf("Expected optimal size 10000, got %d", patterns.OptimalPromptSize)
	}
}

func TestResponseMonitor_LoadSaveEvents(t *testing.T) {
	tempDir := t.TempDir()

	monitor := NewResponseMonitor(false)
	monitor.dataFile = filepath.Join(tempDir, "test_events.json")

	// Test loading from non-existent file
	events, err := monitor.loadEvents()
	if err != nil {
		t.Errorf("Expected no error loading from non-existent file, got %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected empty events, got %d", len(events))
	}

	// Test saving and loading events
	testEvents := []ResponseEvent{
		{
			Timestamp:  time.Now(),
			SessionID:  "test-1",
			PromptSize: 1000,
			Success:    true,
		},
		{
			Timestamp:  time.Now(),
			SessionID:  "test-2",
			PromptSize: 2000,
			Success:    false,
		},
	}

	err = monitor.saveEvents(testEvents)
	if err != nil {
		t.Errorf("Expected no error saving events, got %v", err)
	}

	// Load events back
	loadedEvents, err := monitor.loadEvents()
	if err != nil {
		t.Errorf("Expected no error loading events, got %v", err)
	}

	if len(loadedEvents) != len(testEvents) {
		t.Errorf("Expected %d events, got %d", len(testEvents), len(loadedEvents))
	}

	// Check that data was preserved
	for i, event := range loadedEvents {
		if event.SessionID != testEvents[i].SessionID {
			t.Errorf("Expected session ID %s, got %s", testEvents[i].SessionID, event.SessionID)
		}
		if event.PromptSize != testEvents[i].PromptSize {
			t.Errorf("Expected prompt size %d, got %d", testEvents[i].PromptSize, event.PromptSize)
		}
		if event.Success != testEvents[i].Success {
			t.Errorf("Expected success %v, got %v", testEvents[i].Success, event.Success)
		}
	}
}
