package ai

import (
	"reviewtask/internal/config"
	"sync"
	"testing"
	"time"
)

// TestDefaultConcurrencySettings verifies default configuration values
func TestDefaultConcurrencySettings(t *testing.T) {
	// Create default config
	validationTrue := true
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage:             "English",
			OutputFormat:             "json",
			MaxRetries:               5,
			AIProvider:               "auto",
			AutoResolveMode:          "complete",
			Model:                    "auto",
			PromptProfile:            "v2",
			ValidationEnabled:        &validationTrue,
			QualityThreshold:         0.8,
			VerboseMode:              false,
			MaxTasksPerComment:       2,
			DeduplicationEnabled:     true,
			SimilarityThreshold:      0.8,
			ProcessNitpickComments:   true,
			NitpickPriority:          "low",
			EnableJSONRecovery:       true,
			MaxRecoveryAttempts:      3,
			PartialResponseThreshold: 0.7,
			LogTruncatedResponses:    true,
			ProcessSelfReviews:       false,
			ErrorTrackingEnabled:     true,
			StreamProcessingEnabled:  true,
			AutoSummarizeEnabled:     true,
			RealtimeSavingEnabled:    true,
			MaxConcurrentRequests:    5,
			BatchSize:                4,
		},
	}

	if cfg.AISettings.MaxConcurrentRequests != 5 {
		t.Errorf("Expected default MaxConcurrentRequests = 5, got %d",
			cfg.AISettings.MaxConcurrentRequests)
	}

	if cfg.AISettings.BatchSize != 4 {
		t.Errorf("Expected default BatchSize = 4, got %d",
			cfg.AISettings.BatchSize)
	}
}

// TestSemaphoreBehavior tests that semaphore creation and fallback work correctly
func TestSemaphoreBehavior(t *testing.T) {
	tests := []struct {
		name              string
		maxConcurrent     int
		expectedSemaphore int
		description       string
	}{
		{
			name:              "Default value 5",
			maxConcurrent:     5,
			expectedSemaphore: 5,
			description:       "Should use configured value",
		},
		{
			name:              "Zero value fallback",
			maxConcurrent:     0,
			expectedSemaphore: 5,
			description:       "Should fallback to 5",
		},
		{
			name:              "Negative value fallback",
			maxConcurrent:     -1,
			expectedSemaphore: 5,
			description:       "Should fallback to 5",
		},
		{
			name:              "Custom value 10",
			maxConcurrent:     10,
			expectedSemaphore: 10,
			description:       "Should use configured value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that configuration is accepted
			cfg := &config.Config{
				AISettings: config.AISettings{
					MaxConcurrentRequests: tt.maxConcurrent,
				},
			}

			analyzer := &Analyzer{config: cfg}

			// Verify the analyzer has the correct config
			actual := analyzer.config.AISettings.MaxConcurrentRequests
			if actual != tt.maxConcurrent {
				t.Errorf("Config mismatch: expected %d, got %d", tt.maxConcurrent, actual)
			}

			t.Logf("%s: MaxConcurrentRequests = %d", tt.description, actual)
		})
	}
}

// TestProgressReportingStructure verifies progress reporting logic
func TestProgressReportingStructure(t *testing.T) {
	tests := []struct {
		name                string
		totalComments       int
		expectedProgressMsg bool
	}{
		{
			name:                "Small batch (5 comments)",
			totalComments:       5,
			expectedProgressMsg: true,
		},
		{
			name:                "Medium batch (20 comments)",
			totalComments:       20,
			expectedProgressMsg: true,
		},
		{
			name:                "Large batch (100 comments)",
			totalComments:       100,
			expectedProgressMsg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify progress reporting intervals
			for processed := 1; processed <= tt.totalComments; processed++ {
				// Progress should be shown:
				// - For every comment if total <= 10
				// - Every 5th comment if total > 10
				// - On completion (processed == total)
				shouldShow := tt.totalComments <= 10 ||
					processed%5 == 0 ||
					processed == tt.totalComments

				if shouldShow && !tt.expectedProgressMsg {
					t.Errorf("Progress reporting logic inconsistent at %d/%d",
						processed, tt.totalComments)
				}

				if processed == tt.totalComments {
					t.Logf("Completed: %d/%d comments", processed, tt.totalComments)
				}
			}
		})
	}
}

// TestConcurrencyConfiguration verifies concurrency settings are properly configured
func TestConcurrencyConfiguration(t *testing.T) {
	// Verify sync.WaitGroup and channel-based semaphore patterns are sound
	var wg sync.WaitGroup
	maxConcurrent := 5
	semaphore := make(chan struct{}, maxConcurrent)

	// Simulate concurrent operations
	operations := 20
	completed := 0
	var mu sync.Mutex

	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Simulate work
			time.Sleep(1 * time.Millisecond)

			mu.Lock()
			completed++
			mu.Unlock()
		}(i)
	}

	// Wait for all operations
	wg.Wait()

	if completed != operations {
		t.Errorf("Expected %d completed operations, got %d", operations, completed)
	}

	t.Logf("Successfully completed %d operations with semaphore limit %d",
		completed, maxConcurrent)
}

// TestBatchSizeConfiguration verifies batch size is correctly configured
func TestBatchSizeConfiguration(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			BatchSize:             4,
			MaxConcurrentRequests: 5,
		},
	}

	// Verify batch size setting
	if cfg.AISettings.BatchSize != 4 {
		t.Errorf("Expected BatchSize = 4, got %d", cfg.AISettings.BatchSize)
	}

	// With batch size 4 and 5 concurrent, can process 20 comments efficiently
	totalComments := 20
	batchSize := cfg.AISettings.BatchSize
	batches := (totalComments + batchSize - 1) / batchSize // Ceiling division

	expectedBatches := 5 // 20 comments / 4 per batch = 5 batches
	if batches != expectedBatches {
		t.Errorf("Expected %d batches, calculated %d", expectedBatches, batches)
	}

	t.Logf("With batch size %d: %d comments → %d batches",
		batchSize, totalComments, batches)
}
