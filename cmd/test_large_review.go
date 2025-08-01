package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

var testLargeReviewCmd = &cobra.Command{
	Use:    "test-large-review",
	Hidden: true,
	Short:  "Test large review processing",
	RunE:   runTestLargeReview,
}

func init() {
	rootCmd.AddCommand(testLargeReviewCmd)
}

func runTestLargeReview(cmd *cobra.Command, args []string) error {
	// Load test data
	data, err := os.ReadFile("/tmp/test_large_review.json")
	if err != nil {
		return fmt.Errorf("failed to read test data: %w", err)
	}

	var testData struct {
		Reviews []github.Review `json:"reviews"`
	}
	if err := json.Unmarshal(data, &testData); err != nil {
		return fmt.Errorf("failed to parse test data: %w", err)
	}

	totalComments := 0
	for _, review := range testData.Reviews {
		totalComments += len(review.Comments)
	}
	fmt.Printf("Loaded %d reviews with total %d comments\n", len(testData.Reviews), totalComments)

	// Create configuration
	validationEnabled := false
	cfg := &config.Config{
		AISettings: config.AISettings{
			ValidationEnabled: &validationEnabled,
			MaxRetries:        3,
			UserLanguage:      "Japanese",
			VerboseMode:       true,
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}

	// Initialize AI analyzer
	analyzer := ai.NewAnalyzer(cfg)

	// Start task generation with progress tracking
	fmt.Println("\nStarting task generation...")
	startTime := time.Now()

	fmt.Printf("  Found %d comments to analyze\n", totalComments)

	// Use incremental processing with progress callback
	processedCount := 0
	opts := ai.IncrementalOptions{
		BatchSize:    10,
		Resume:       false,
		FastMode:     false,
		MaxTimeout:   10 * time.Minute,
		ShowProgress: true,
		OnProgress: func(processed, total int) {
			// Show progress for each comment
			if processed > processedCount {
				processedCount = processed
				fmt.Printf("\r  Processing: %d/%d comments (%.1f%%)                    ",
					processed, total, float64(processed)/float64(total)*100)
			}
		},
		OnBatchComplete: func(batchTasks []storage.Task) {
			fmt.Printf("\n  Generated %d tasks from batch\n", len(batchTasks))
		},
	}

	// Generate tasks with incremental processing
	storageManager := storage.NewManager()
	tasks, err := analyzer.GenerateTasksIncremental(testData.Reviews, 999, storageManager, opts)
	if err != nil {
		return fmt.Errorf("failed to generate tasks: %w", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("\nCompleted in %v\n", duration)
	fmt.Printf("Generated %d tasks from %d comments\n", len(tasks), totalComments)

	// Save tasks to verify they were generated correctly
	if err := storageManager.SaveTasks(999, tasks); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	fmt.Println("\nTasks saved successfully!")
	return nil
}
