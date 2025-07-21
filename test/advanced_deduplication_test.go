package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

func TestAdvancedDeduplication(t *testing.T) {
	// Skip if no Claude access
	if _, err := ai.FindClaudeCommand(""); err != nil {
		t.Skip("Claude not available, skipping advanced deduplication tests")
	}

	// Create test configuration
	testConfig := &config.Config{
		AISettings: config.AISettings{
			UserLanguage:         "English",
			DeduplicationEnabled: true,
			MaxTasksPerComment:   2, // Should be ignored in AI mode
			SimilarityThreshold:  0.8,
			DebugMode:           true,
		},
	}

	analyzer := ai.NewAnalyzer(testConfig)

	t.Run("CommentEditTracking", func(t *testing.T) {
		// Simulate a PR with comment history
		prNumber := 100
		tempDir := t.TempDir()
		os.Chdir(tempDir)

		// Create initial comment
		reviews := []github.Review{
			{
				ID:       1,
				Reviewer: "reviewer1",
				State:    "COMMENTED",
				Comments: []github.Comment{
					{
						ID:     1001,
						Author: "reviewer1",
						Body:   "Please add input validation for the username field",
						File:   "auth.go",
						Line:   42,
					},
				},
			},
		}

		// Generate initial tasks
		storageManager := storage.NewManager()
		tasks1, err := analyzer.GenerateTasksWithCache(reviews, prNumber, storageManager)
		if err != nil {
			t.Fatalf("Failed to generate initial tasks: %v", err)
		}

		if len(tasks1) == 0 {
			t.Error("Expected at least one task from initial comment")
		}

		// Simulate comment edit (cosmetic change)
		reviews[0].Comments[0].Body = "Please add input validation for the username field."
		
		// Generate tasks again
		tasks2, err := analyzer.GenerateTasksWithCache(reviews, prNumber, storageManager)
		if err != nil {
			t.Fatalf("Failed to generate tasks after edit: %v", err)
		}

		// Should not generate new tasks for cosmetic change
		if len(tasks2) != len(tasks1) {
			t.Errorf("Cosmetic edit created different number of tasks: %d vs %d", len(tasks2), len(tasks1))
		}
	})

	t.Run("SemanticChangeDetection", func(t *testing.T) {
		// Test semantic changes in comments
		prNumber := 101
		tempDir := t.TempDir()
		os.Chdir(tempDir)

		// Create initial comment
		reviews := []github.Review{
			{
				ID:       2,
				Reviewer: "reviewer2",
				State:    "CHANGES_REQUESTED",
				Comments: []github.Comment{
					{
						ID:     2001,
						Author: "reviewer2",
						Body:   "Add error handling for database connection",
						File:   "db.go",
						Line:   15,
					},
				},
			},
		}

		storageManager := storage.NewManager()
		tasks1, err := analyzer.GenerateTasksWithCache(reviews, prNumber, storageManager)
		if err != nil {
			t.Fatalf("Failed to generate initial tasks: %v", err)
		}

		// Simulate semantic change
		reviews[0].Comments[0].Body = "Add retry logic with exponential backoff for database connection"
		
		// Sleep briefly to ensure different timestamps
		time.Sleep(100 * time.Millisecond)
		
		// Generate tasks again
		tasks2, err := analyzer.GenerateTasksWithCache(reviews, prNumber, storageManager)
		if err != nil {
			t.Fatalf("Failed to generate tasks after semantic change: %v", err)
		}

		// Should generate new tasks for semantic change
		// Note: This depends on AI detecting the semantic difference
		t.Logf("Tasks before: %d, Tasks after: %d", len(tasks1), len(tasks2))
	})

	t.Run("DeletedCommentHandling", func(t *testing.T) {
		prNumber := 102
		tempDir := t.TempDir()
		os.Chdir(tempDir)

		// Create comments
		reviews := []github.Review{
			{
				ID:       3,
				Reviewer: "reviewer3",
				State:    "COMMENTED",
				Comments: []github.Comment{
					{
						ID:     3001,
						Author: "reviewer3",
						Body:   "Fix memory leak in cache implementation",
						File:   "cache.go",
						Line:   78,
					},
					{
						ID:     3002,
						Author: "reviewer3",
						Body:   "Add unit tests for cache",
						File:   "cache_test.go",
						Line:   1,
					},
				},
			},
		}

		storageManager := storage.NewManager()
		tasks1, err := analyzer.GenerateTasksWithCache(reviews, prNumber, storageManager)
		if err != nil {
			t.Fatalf("Failed to generate initial tasks: %v", err)
		}

		// Save tasks
		if err := storageManager.SaveTasks(prNumber, tasks1); err != nil {
			t.Fatalf("Failed to save tasks: %v", err)
		}

		// Simulate comment deletion (remove second comment)
		reviews[0].Comments = reviews[0].Comments[:1]
		
		// Generate tasks again
		tasks2, err := analyzer.GenerateTasksWithCache(reviews, prNumber, storageManager)
		if err != nil {
			t.Fatalf("Failed to generate tasks after deletion: %v", err)
		}

		// Check if tasks from deleted comment are marked as cancelled
		var cancelledCount int
		for _, task := range tasks2 {
			if task.Status == "cancel" && task.SourceCommentID == 3002 {
				cancelledCount++
			}
		}

		if cancelledCount == 0 {
			t.Error("Expected tasks from deleted comment to be marked as cancelled")
		}
	})

	t.Run("AIDeduplicationWithoutLimits", func(t *testing.T) {
		// Test that AI deduplication works without task count limits
		reviews := []github.Review{
			{
				ID:       4,
				Reviewer: "reviewer4",
				State:    "CHANGES_REQUESTED",
				Comments: []github.Comment{
					{
						ID:     4001,
						Author: "reviewer4",
						Body: `This function has several issues:
						1. No input validation
						2. Missing error handling
						3. Potential SQL injection vulnerability
						4. No logging for debugging
						5. Performance could be improved with caching`,
						File: "api.go",
						Line: 125,
					},
				},
			},
		}

		// Generate tasks - should create multiple tasks despite max_tasks_per_comment=2
		tasks, err := analyzer.GenerateTasks(reviews)
		if err != nil {
			t.Fatalf("Failed to generate tasks: %v", err)
		}

		// With AI deduplication, we should get appropriate number of tasks
		// not limited by max_tasks_per_comment
		if len(tasks) <= 2 {
			t.Logf("Warning: Only %d tasks generated. AI might have combined issues.", len(tasks))
		} else {
			t.Logf("Success: Generated %d tasks without artificial limit", len(tasks))
		}

		// Verify no duplicate tasks
		taskDescriptions := make(map[string]bool)
		for _, task := range tasks {
			if taskDescriptions[task.Description] {
				t.Errorf("Found duplicate task description: %s", task.Description)
			}
			taskDescriptions[task.Description] = true
		}
	})
}

func TestCommentHistoryPersistence(t *testing.T) {
	tempDir := t.TempDir()
	os.Chdir(tempDir)

	prNumber := 200
	manager := storage.NewCommentHistoryManager(prNumber)

	// Test saving and loading history
	history := map[int64]*storage.CommentHistory{
		5001: {
			CommentID:    5001,
			OriginalText: "Original review comment",
			CurrentText:  "Edited review comment",
			FirstSeen:    time.Now().Add(-time.Hour),
			LastModified: time.Now(),
			IsDeleted:    false,
			TextHash:     storage.CalculateTextHash("Edited review comment"),
			ModificationCount: 1,
		},
	}

	// Save history
	if err := manager.SaveHistory(history); err != nil {
		t.Fatalf("Failed to save history: %v", err)
	}

	// Verify file exists
	historyFile := filepath.Join(".pr-review", "PR-200", "comment_history.json")
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Error("History file was not created")
	}

	// Load history
	loadedHistory, err := manager.LoadHistory()
	if err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	if len(loadedHistory) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(loadedHistory))
	}

	if entry, exists := loadedHistory[5001]; exists {
		if entry.ModificationCount != 1 {
			t.Errorf("Expected modification count 1, got %d", entry.ModificationCount)
		}
		if entry.CurrentText != "Edited review comment" {
			t.Errorf("Expected current text 'Edited review comment', got '%s'", entry.CurrentText)
		}
	} else {
		t.Error("History entry 5001 not found")
	}
}