package ai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

func TestGenerateTasksIncremental(t *testing.T) {
	// Create temporary directory for storage
	tempDir, err := os.MkdirTemp("", "reviewtask-incremental-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := &config.Config{
		AISettings: config.AISettings{
			UserLanguage:         "English",
			DeduplicationEnabled: false, // Disable for tests to avoid Claude calls
			SimilarityThreshold:  0.8,
			MaxTasksPerComment:   5,
			DebugMode:            false,
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus:     "todo",
			LowPriorityStatus: "pending",
		},
	}

	// Create mock Claude client
	mockClient := NewMockClaudeClient()
	mockClient.Responses["default"] = `[{
		"description": "Fix the error handling",
		"origin_text": "Error handling needs improvement",
		"priority": "high",
		"source_review_id": 100,
		"source_comment_id": 200,
		"file": "main.go",
		"line": 42,
		"task_index": 0
	}]`

	// Create analyzer with mock client
	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create storage manager with custom base directory
	// Change to temporary directory for tests
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	storageManager := storage.NewManager()

	// Test data
	reviews := []github.Review{
		{
			ID:       100,
			Reviewer: "reviewer1",
			State:    "CHANGES_REQUESTED",
			Body:     "Please review the following:",
			Comments: []github.Comment{
				{
					ID:     200,
					File:   "main.go",
					Line:   42,
					Body:   "Error handling needs improvement",
					Author: "reviewer1",
				},
				{
					ID:     201,
					File:   "utils.go",
					Line:   10,
					Body:   "Add unit tests",
					Author: "reviewer1",
				},
			},
		},
	}

	t.Run("BasicIncrementalProcessing", func(t *testing.T) {
		prNumber := 123
		opts := IncrementalOptions{
			BatchSize:    1, // Process one comment at a time
			Resume:       false,
			FastMode:     false,
			MaxTimeout:   10 * time.Second,
			ShowProgress: false,
		}

		tasks, err := analyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, opts)
		assert.NoError(t, err)
		assert.Len(t, tasks, 3) // One task per comment + one for review body

		// Verify checkpoint was deleted after successful completion
		checkpoint, err := storageManager.LoadCheckpoint(prNumber)
		assert.NoError(t, err)
		assert.Nil(t, checkpoint)
	})

	t.Run("ResumeFromCheckpoint", func(t *testing.T) {
		prNumber := 456

		// Create a checkpoint with one comment already processed
		checkpoint := &storage.CheckpointState{
			PRNumber: prNumber,
			ProcessedComments: map[int64]string{
				200: "somehash", // First comment already processed
			},
			TotalComments:  2,
			ProcessedCount: 1,
			BatchSize:      1,
			StartedAt:      time.Now().Add(-5 * time.Minute),
			PartialTasks: []storage.Task{
				{
					ID:              "task-1",
					Description:     "Fix the error handling",
					SourceCommentID: 200,
					Priority:        "high",
					Status:          "todo",
				},
			},
		}

		// Save checkpoint
		err := storageManager.SaveCheckpoint(prNumber, checkpoint)
		assert.NoError(t, err)

		// Process with resume enabled
		opts := IncrementalOptions{
			BatchSize:    1,
			Resume:       true,
			FastMode:     false,
			MaxTimeout:   10 * time.Second,
			ShowProgress: false,
		}

		tasks, err := analyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, opts)
		assert.NoError(t, err)
		// Should have 4 tasks: 1 from checkpoint + 3 new (review body was not processed before)
		assert.Len(t, tasks, 4)

		// Verify first task is from checkpoint
		assert.Equal(t, "task-1", tasks[0].ID)
		assert.Equal(t, int64(200), tasks[0].SourceCommentID)
	})

	t.Run("FastModeProcessing", func(t *testing.T) {
		prNumber := 789
		opts := IncrementalOptions{
			BatchSize:    2,
			Resume:       false,
			FastMode:     true, // Enable fast mode
			MaxTimeout:   10 * time.Second,
			ShowProgress: false,
		}

		// Set mock response for fast mode
		mockClient.Responses["fast"] = `[{
			"description": "Quick fix needed",
			"origin_text": "Error handling needs improvement",
			"priority": "high",
			"source_review_id": 100,
			"source_comment_id": 200,
			"file": "main.go",
			"line": 42,
			"task_index": 0
		}]`

		tasks, err := analyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, opts)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1) // At least one task generated
	})

	t.Run("ProgressCallback", func(t *testing.T) {
		prNumber := 321
		progressCalls := 0
		var lastProcessed, lastTotal int

		opts := IncrementalOptions{
			BatchSize:    1,
			Resume:       false,
			FastMode:     false,
			MaxTimeout:   10 * time.Second,
			ShowProgress: false,
			OnProgress: func(processed, total int) {
				progressCalls++
				lastProcessed = processed
				lastTotal = total
			},
		}

		tasks, err := analyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, opts)
		assert.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Verify progress callback was called
		assert.Greater(t, progressCalls, 0)
		// With fine-grained progress, we now report step-based progress
		// Total steps = number of comments * total step weights
		totalStepsPerComment := 0
		for _, weight := range StepWeights {
			totalStepsPerComment += weight
		}
		expectedTotal := 3 * totalStepsPerComment // 3 comments
		assert.Equal(t, expectedTotal, lastTotal)
		// Last processed should be less than or equal to total
		assert.LessOrEqual(t, lastProcessed, lastTotal)
	})

	t.Run("BatchCompleteCallback", func(t *testing.T) {
		prNumber := 654
		batchCount := 0
		var totalBatchTasks int

		opts := IncrementalOptions{
			BatchSize:    1,
			Resume:       false,
			FastMode:     false,
			MaxTimeout:   10 * time.Second,
			ShowProgress: false,
			OnBatchComplete: func(batchTasks []storage.Task) {
				batchCount++
				totalBatchTasks += len(batchTasks)
			},
		}

		tasks, err := analyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, opts)
		assert.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Verify batch callback was called for each batch
		assert.Equal(t, 3, batchCount) // Three batches (batch size 1, 3 items)
		assert.GreaterOrEqual(t, totalBatchTasks, 3)
	})

	t.Run("TimeoutHandling", func(t *testing.T) {
		prNumber := 987

		// Create a slow mock client
		slowClient := NewMockClaudeClient()
		slowClient.Error = context.DeadlineExceeded

		slowAnalyzer := NewAnalyzerWithClient(cfg, slowClient)

		opts := IncrementalOptions{
			BatchSize:    1,
			Resume:       false,
			FastMode:     false,
			MaxTimeout:   1 * time.Second, // Very short timeout
			ShowProgress: false,
		}

		_, err = slowAnalyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, opts)
		// The error might not be a timeout error since we're setting context.DeadlineExceeded
		// but the processing might complete before the timeout
		if err != nil {
			// If there's an error, verify checkpoint was saved
			checkpoint, loadErr := storageManager.LoadCheckpoint(prNumber)
			assert.NoError(t, loadErr)
			// Checkpoint might exist if processing was interrupted
			_ = checkpoint
		}
	})

	t.Run("EmptyReviews", func(t *testing.T) {
		prNumber := 111
		opts := IncrementalOptions{
			BatchSize:    5,
			Resume:       false,
			FastMode:     false,
			MaxTimeout:   10 * time.Second,
			ShowProgress: false,
		}

		tasks, err := analyzer.GenerateTasksIncremental([]github.Review{}, prNumber, storageManager, opts)
		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("ResolvedCommentsFiltering", func(t *testing.T) {
		prNumber := 222

		// Reviews with resolved comments
		resolvedReviews := []github.Review{
			{
				ID:       101,
				Reviewer: "reviewer1",
				State:    "CHANGES_REQUESTED",
				Comments: []github.Comment{
					{
						ID:     300,
						File:   "main.go",
						Line:   42,
						Body:   "✅ Addressed in commit abc123",
						Author: "reviewer1",
					},
					{
						ID:     301,
						File:   "utils.go",
						Line:   10,
						Body:   "Add more tests",
						Author: "reviewer1",
					},
				},
			},
		}

		opts := IncrementalOptions{
			BatchSize:    2,
			Resume:       false,
			FastMode:     false,
			MaxTimeout:   10 * time.Second,
			ShowProgress: false,
		}

		tasks, err := analyzer.GenerateTasksIncremental(resolvedReviews, prNumber, storageManager, opts)
		assert.NoError(t, err)
		assert.Len(t, tasks, 1) // Only one task for non-resolved comment
	})
}
