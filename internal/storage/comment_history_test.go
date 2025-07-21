package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCommentHistoryManager(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	defer func() {
		os.RemoveAll(filepath.Join(tempDir, ".pr-review"))
	}()

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	t.Run("LoadHistory_EmptyHistory", func(t *testing.T) {
		manager := NewCommentHistoryManager(1)
		history, err := manager.LoadHistory()
		if err != nil {
			t.Fatalf("Expected no error for non-existent file, got: %v", err)
		}
		if len(history) != 0 {
			t.Errorf("Expected empty history, got %d entries", len(history))
		}
	})

	t.Run("SaveAndLoadHistory", func(t *testing.T) {
		manager := NewCommentHistoryManager(1)

		// Create test history
		testHistory := map[int64]*CommentHistory{
			123: {
				CommentID:    123,
				OriginalText: "Test comment",
				CurrentText:  "Test comment",
				FirstSeen:    time.Now(),
				LastModified: time.Now(),
				IsDeleted:    false,
				TextHash:     CalculateTextHash("Test comment"),
			},
		}

		// Save history
		err := manager.SaveHistory(testHistory)
		if err != nil {
			t.Fatalf("Failed to save history: %v", err)
		}

		// Load history
		loadedHistory, err := manager.LoadHistory()
		if err != nil {
			t.Fatalf("Failed to load history: %v", err)
		}

		if len(loadedHistory) != 1 {
			t.Errorf("Expected 1 entry, got %d", len(loadedHistory))
		}

		if entry, exists := loadedHistory[123]; exists {
			if entry.CommentID != 123 {
				t.Errorf("Expected comment ID 123, got %d", entry.CommentID)
			}
			if entry.CurrentText != "Test comment" {
				t.Errorf("Expected text 'Test comment', got '%s'", entry.CurrentText)
			}
		} else {
			t.Error("Expected entry with ID 123 not found")
		}
	})

	t.Run("AnalyzeCommentChanges", func(t *testing.T) {
		manager := NewCommentHistoryManager(2)

		// Setup existing history
		history := map[int64]*CommentHistory{
			100: {
				CommentID:   100,
				CurrentText: "Original text",
				TextHash:    CalculateTextHash("Original text"),
				IsDeleted:   false,
			},
			200: {
				CommentID:   200,
				CurrentText: "To be deleted",
				TextHash:    CalculateTextHash("To be deleted"),
				IsDeleted:   false,
			},
			300: {
				CommentID:   300,
				CurrentText: "Previously deleted",
				TextHash:    CalculateTextHash("Previously deleted"),
				IsDeleted:   true,
			},
		}

		// Current comments
		currentComments := map[int64]string{
			100: "Modified text",      // Modified
			300: "Previously deleted", // Restored
			400: "New comment",        // New
			// 200 is missing (deleted)
		}

		changes := manager.AnalyzeCommentChanges(currentComments, history)

		// Verify changes
		changeMap := make(map[int64]string)
		for _, change := range changes {
			changeMap[change.CommentID] = change.Type
		}

		if changeMap[100] != "modified" {
			t.Errorf("Expected comment 100 to be 'modified', got '%s'", changeMap[100])
		}
		if changeMap[200] != "deleted" {
			t.Errorf("Expected comment 200 to be 'deleted', got '%s'", changeMap[200])
		}
		if changeMap[300] != "new" {
			t.Errorf("Expected comment 300 to be 'new' (restored), got '%s'", changeMap[300])
		}
		if changeMap[400] != "new" {
			t.Errorf("Expected comment 400 to be 'new', got '%s'", changeMap[400])
		}
	})

	t.Run("UpdateHistory", func(t *testing.T) {
		manager := NewCommentHistoryManager(3)
		history := make(map[int64]*CommentHistory)

		changes := []CommentChange{
			{Type: "new", CommentID: 1, CurrentText: "New comment"},
			{Type: "modified", CommentID: 2, CurrentText: "Modified text"},
			{Type: "deleted", CommentID: 3},
		}

		// Add existing entry for modification test
		history[2] = &CommentHistory{
			CommentID:         2,
			CurrentText:       "Original text",
			ModificationCount: 0,
		}

		// Add existing entry for deletion test
		history[3] = &CommentHistory{
			CommentID: 3,
			IsDeleted: false,
		}

		updatedHistory := manager.UpdateHistory(changes, history)

		// Verify new comment
		if entry, exists := updatedHistory[1]; exists {
			if entry.CurrentText != "New comment" {
				t.Errorf("Expected new comment text, got '%s'", entry.CurrentText)
			}
		} else {
			t.Error("New comment not added to history")
		}

		// Verify modified comment
		if entry, exists := updatedHistory[2]; exists {
			if entry.CurrentText != "Modified text" {
				t.Errorf("Expected modified text, got '%s'", entry.CurrentText)
			}
			if entry.ModificationCount != 1 {
				t.Errorf("Expected modification count 1, got %d", entry.ModificationCount)
			}
		} else {
			t.Error("Modified comment not found in history")
		}

		// Verify deleted comment
		if entry, exists := updatedHistory[3]; exists {
			if !entry.IsDeleted {
				t.Error("Expected comment to be marked as deleted")
			}
		} else {
			t.Error("Deleted comment not found in history")
		}
	})
}

func TestCalculateTextHash(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"Hello, world!", "315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3"},
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"Test comment", "7be45a95aef6b75bf37653507355d9b2408f4f7826f972699107a36e018cbdc2"},
	}

	for _, test := range tests {
		result := CalculateTextHash(test.text)
		if result != test.expected {
			t.Errorf("CalculateTextHash(%q) = %s; want %s", test.text, result, test.expected)
		}
	}
}
