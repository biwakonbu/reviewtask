package ai

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/github"

	"github.com/google/uuid"
)

// TestMain sets up test environment for all tests in this package
func TestMain(m *testing.M) {
	// Skip authentication checks in tests to avoid hanging on CI/test environments
	os.Setenv("SKIP_CLAUDE_AUTH_CHECK", "true")
	os.Setenv("SKIP_CURSOR_AUTH_CHECK", "true")

	// Run tests
	os.Exit(m.Run())
}

func TestConvertToStorageTasksUUIDGeneration(t *testing.T) {
	// Create analyzer with default config
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Create test TaskRequest data
	testTasks := []TaskRequest{
		{
			Description:     "Test task 1",
			OriginText:      "Original comment text 1",
			Priority:        "high",
			SourceReviewID:  12345,
			SourceCommentID: 67890,
			File:            "test.go",
			Line:            42,
			Status:          "todo",
			TaskIndex:       0,
		},
		{
			Description:     "Test task 2",
			OriginText:      "Original comment text 2",
			Priority:        "medium",
			SourceReviewID:  12345,
			SourceCommentID: 67890,
			File:            "test.go",
			Line:            45,
			Status:          "todo",
			TaskIndex:       1,
		},
	}

	// Convert to storage tasks
	storageTasks := analyzer.convertToStorageTasks(testTasks)

	// Verify task count
	if len(storageTasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(storageTasks))
	}

	// Test UUID generation and uniqueness
	seenIDs := make(map[string]bool)
	for i, task := range storageTasks {
		// Verify ID is a valid UUID
		_, err := uuid.Parse(task.ID)
		if err != nil {
			t.Errorf("Task %d ID '%s' is not a valid UUID: %v", i, task.ID, err)
		}

		// Verify ID uniqueness
		if seenIDs[task.ID] {
			t.Errorf("Task %d has duplicate ID '%s'", i, task.ID)
		}
		seenIDs[task.ID] = true

		// Verify ID is not the old comment-based format
		if len(task.ID) < 36 { // UUID v4 is 36 characters with hyphens
			t.Errorf("Task %d ID '%s' appears to be using old comment-based format", i, task.ID)
		}

		// Verify other fields are preserved correctly
		expectedTask := testTasks[i]
		if task.Description != expectedTask.Description {
			t.Errorf("Task %d description mismatch: expected '%s', got '%s'",
				i, expectedTask.Description, task.Description)
		}
		if task.OriginText != expectedTask.OriginText {
			t.Errorf("Task %d origin text mismatch: expected '%s', got '%s'",
				i, expectedTask.OriginText, task.OriginText)
		}
		if task.Priority != expectedTask.Priority {
			t.Errorf("Task %d priority mismatch: expected '%s', got '%s'",
				i, expectedTask.Priority, task.Priority)
		}
		if task.SourceCommentID != expectedTask.SourceCommentID {
			t.Errorf("Task %d source comment ID mismatch: expected %d, got %d",
				i, expectedTask.SourceCommentID, task.SourceCommentID)
		}
	}
}

func TestConvertToStorageTasksUUIDUniqueness(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Create large number of tasks to test uniqueness at scale
	const numTasks = 1000
	testTasks := make([]TaskRequest, numTasks)
	for i := 0; i < numTasks; i++ {
		testTasks[i] = TaskRequest{
			Description:     "Test task",
			OriginText:      "Original comment text",
			Priority:        "medium",
			SourceReviewID:  12345,
			SourceCommentID: 67890,
			File:            "test.go",
			Line:            42,
			Status:          "todo",
			TaskIndex:       i,
		}
	}

	// Convert to storage tasks
	storageTasks := analyzer.convertToStorageTasks(testTasks)

	// Verify all IDs are unique
	seenIDs := make(map[string]bool)
	for i, task := range storageTasks {
		if seenIDs[task.ID] {
			t.Errorf("Task %d has duplicate ID '%s'", i, task.ID)
		}
		seenIDs[task.ID] = true

		// Verify each ID is a valid UUID
		_, err := uuid.Parse(task.ID)
		if err != nil {
			t.Errorf("Task %d ID '%s' is not a valid UUID: %v", i, task.ID, err)
		}
	}

	// Verify we have exactly the expected number of unique IDs
	if len(seenIDs) != numTasks {
		t.Errorf("Expected %d unique IDs, got %d", numTasks, len(seenIDs))
	}
}

func TestConvertToStorageTasksTimestamps(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	testTask := TaskRequest{
		Description:     "Test task",
		OriginText:      "Original comment text",
		Priority:        "high",
		SourceReviewID:  12345,
		SourceCommentID: 67890,
		File:            "test.go",
		Line:            42,
		Status:          "todo",
		TaskIndex:       0,
	}

	// Convert to storage task
	storageTasks := analyzer.convertToStorageTasks([]TaskRequest{testTask})

	if len(storageTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(storageTasks))
	}

	task := storageTasks[0]

	// Verify timestamp format is correct
	_, err := time.Parse("2006-01-02T15:04:05Z", task.CreatedAt)
	if err != nil {
		t.Errorf("Invalid CreatedAt timestamp format: %v", err)
	}

	_, err = time.Parse("2006-01-02T15:04:05Z", task.UpdatedAt)
	if err != nil {
		t.Errorf("Invalid UpdatedAt timestamp format: %v", err)
	}

	// Verify CreatedAt and UpdatedAt are the same for new tasks
	if task.CreatedAt != task.UpdatedAt {
		t.Errorf("CreatedAt (%s) and UpdatedAt (%s) should be the same for new tasks",
			task.CreatedAt, task.UpdatedAt)
	}

	// Verify timestamps are not empty
	if task.CreatedAt == "" {
		t.Errorf("CreatedAt should not be empty")
	}

	if task.UpdatedAt == "" {
		t.Errorf("UpdatedAt should not be empty")
	}
}

func TestConvertToStorageTasksEmptyInput(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Test empty input
	storageTasks := analyzer.convertToStorageTasks([]TaskRequest{})

	if len(storageTasks) != 0 {
		t.Errorf("Expected 0 tasks for empty input, got %d", len(storageTasks))
	}
}

// TestConvertToStorageTasksPreservesAllFields verifies that all fields from TaskRequest
// are correctly preserved when converting to storage.Task. This includes critical fields
// like SourceCommentID which is used to map tasks back to their original GitHub comments.
// This test addresses the code review concern about ID mapping assumptions.
func TestConvertToStorageTasksPreservesAllFields(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "pending",
		},
	}
	analyzer := NewAnalyzer(cfg)

	testTask := TaskRequest{
		Description:     "Fix memory leak in parser",
		OriginText:      "There seems to be a memory leak in the parser when processing large files",
		Priority:        "critical",
		SourceReviewID:  98765,
		SourceCommentID: 11111,
		File:            "internal/parser/lexer.go",
		Line:            127,
		Status:          "todo",
		TaskIndex:       3,
	}

	storageTasks := analyzer.convertToStorageTasks([]TaskRequest{testTask})

	if len(storageTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(storageTasks))
	}

	task := storageTasks[0]

	// Verify all fields are preserved correctly
	if task.Description != testTask.Description {
		t.Errorf("Description mismatch: expected '%s', got '%s'", testTask.Description, task.Description)
	}
	if task.OriginText != testTask.OriginText {
		t.Errorf("OriginText mismatch: expected '%s', got '%s'", testTask.OriginText, task.OriginText)
	}
	if task.Priority != testTask.Priority {
		t.Errorf("Priority mismatch: expected '%s', got '%s'", testTask.Priority, task.Priority)
	}
	if task.SourceReviewID != testTask.SourceReviewID {
		t.Errorf("SourceReviewID mismatch: expected %d, got %d", testTask.SourceReviewID, task.SourceReviewID)
	}
	// Critical: Verify SourceCommentID is preserved for mapping tasks to GitHub comments
	if task.SourceCommentID != testTask.SourceCommentID {
		t.Errorf("SourceCommentID mismatch: expected %d, got %d", testTask.SourceCommentID, task.SourceCommentID)
	}
	if task.TaskIndex != testTask.TaskIndex {
		t.Errorf("TaskIndex mismatch: expected %d, got %d", testTask.TaskIndex, task.TaskIndex)
	}
	if task.File != testTask.File {
		t.Errorf("File mismatch: expected '%s', got '%s'", testTask.File, task.File)
	}
	if task.Line != testTask.Line {
		t.Errorf("Line mismatch: expected %d, got %d", testTask.Line, task.Line)
	}
	// Verify Status is preserved from TaskRequest, not overridden by DefaultStatus
	if task.Status != testTask.Status {
		t.Errorf("Status mismatch: expected '%s', got '%s'", testTask.Status, task.Status)
	}

	// Verify ID is a valid UUID
	_, err := uuid.Parse(task.ID)
	if err != nil {
		t.Errorf("Task ID '%s' is not a valid UUID: %v", task.ID, err)
	}
}

func TestTaskIDFormatSpecification(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	testTask := TaskRequest{
		Description:     "Test task for ID format validation",
		OriginText:      "Original comment text",
		Priority:        "high",
		SourceReviewID:  12345,
		SourceCommentID: 67890,
		File:            "test.go",
		Line:            42,
		Status:          "todo",
		TaskIndex:       0,
	}

	storageTasks := analyzer.convertToStorageTasks([]TaskRequest{testTask})

	if len(storageTasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(storageTasks))
	}

	task := storageTasks[0]

	// Test 1: UUID format verification
	parsedUUID, err := uuid.Parse(task.ID)
	if err != nil {
		t.Errorf("Task ID '%s' is not a valid UUID: %v", task.ID, err)
	}

	// Test 2: UUID version verification (should be version 4)
	if parsedUUID.Version() != 4 {
		t.Errorf("Task ID '%s' is not UUID version 4, got version %d", task.ID, parsedUUID.Version())
	}

	// Test 3: UUID length verification (36 characters with hyphens)
	if len(task.ID) != 36 {
		t.Errorf("Task ID '%s' length is %d, expected 36 characters", task.ID, len(task.ID))
	}

	// Test 4: UUID format pattern verification (8-4-4-4-12)
	// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	parts := strings.Split(task.ID, "-")
	if len(parts) != 5 {
		t.Errorf("Task ID '%s' should have 5 parts separated by hyphens, got %d parts", task.ID, len(parts))
	} else {
		expectedLengths := []int{8, 4, 4, 4, 12}
		for i, part := range parts {
			if len(part) != expectedLengths[i] {
				t.Errorf("Task ID '%s' part %d has length %d, expected %d", task.ID, i+1, len(part), expectedLengths[i])
			}
		}
	}

	// Test 5: Verify ID contains only valid hexadecimal characters and hyphens
	validChars := "0123456789abcdefABCDEF-"
	for i, char := range task.ID {
		if !strings.ContainsRune(validChars, char) {
			t.Errorf("Task ID '%s' contains invalid character '%c' at position %d", task.ID, char, i)
		}
	}

	// Test 6: Verify ID is not old comment-based format
	if strings.Contains(task.ID, "comment-") || strings.Contains(task.ID, "task-") {
		t.Errorf("Task ID '%s' appears to use old comment-based format", task.ID)
	}

	// Test 7: Verify ID is not predictable (no sequential patterns)
	if strings.Contains(task.ID, "00000000") || strings.Contains(task.ID, "11111111") {
		t.Errorf("Task ID '%s' contains predictable patterns", task.ID)
	}
}

func TestTaskIDUniquenessAcrossMultipleGenerations(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Generate multiple batches of task IDs to test uniqueness across time
	const numBatches = 10
	const tasksPerBatch = 100
	allIDs := make(map[string]bool)
	totalTasks := 0

	for batch := 0; batch < numBatches; batch++ {
		// Create identical tasks for each batch
		testTasks := make([]TaskRequest, tasksPerBatch)
		for i := 0; i < tasksPerBatch; i++ {
			testTasks[i] = TaskRequest{
				Description:     "Uniqueness test task",
				OriginText:      "Original comment for uniqueness testing",
				Priority:        "medium",
				SourceReviewID:  12345,
				SourceCommentID: 67890,
				File:            "test.go",
				Line:            42,
				Status:          "todo",
				TaskIndex:       i,
			}
		}

		// Convert to storage tasks
		storageTasks := analyzer.convertToStorageTasks(testTasks)

		// Verify all IDs in this batch are unique
		batchIDs := make(map[string]bool)
		for _, task := range storageTasks {
			// Check uniqueness within batch
			if batchIDs[task.ID] {
				t.Errorf("Batch %d: Duplicate ID within batch: %s", batch, task.ID)
			}
			batchIDs[task.ID] = true

			// Check uniqueness across all batches
			if allIDs[task.ID] {
				t.Errorf("Batch %d: ID collision across batches: %s", batch, task.ID)
			}
			allIDs[task.ID] = true
			totalTasks++

			// Verify ID format for each task
			_, err := uuid.Parse(task.ID)
			if err != nil {
				t.Errorf("Batch %d: Invalid UUID format: %s", batch, task.ID)
			}
		}
	}

	expectedTotal := numBatches * tasksPerBatch
	if totalTasks != expectedTotal {
		t.Errorf("Expected %d total tasks, got %d", expectedTotal, totalTasks)
	}

	if len(allIDs) != totalTasks {
		t.Errorf("UUID uniqueness failed: expected %d unique IDs, got %d", totalTasks, len(allIDs))
	}

	t.Logf("Successfully generated %d unique UUID task IDs across %d batches", totalTasks, numBatches)
}

func TestIsLowPriorityComment(t *testing.T) {
	// Create analyzer with low-priority patterns
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"},
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		comment  string
		expected bool
	}{
		{
			name:     "Comment starting with nit:",
			comment:  "nit: Consider using const instead of let",
			expected: true,
		},
		{
			name:     "Comment starting with NITS: (uppercase)",
			comment:  "NITS: Fix indentation",
			expected: true,
		},
		{
			name:     "Comment starting with minor:",
			comment:  "minor: Could improve variable naming",
			expected: true,
		},
		{
			name:     "Comment with nit: after newline",
			comment:  "Here's a review comment.\nnit: Fix spacing",
			expected: true,
		},
		{
			name:     "Comment with pattern in middle (not at start or after newline)",
			comment:  "This is a nit: but not at start",
			expected: false,
		},
		{
			name:     "Comment without any patterns",
			comment:  "This is a critical security issue that needs fixing",
			expected: false,
		},
		{
			name:     "Comment with style: pattern",
			comment:  "style: Use consistent naming convention",
			expected: true,
		},
		{
			name:     "Empty comment",
			comment:  "",
			expected: false,
		},
		{
			name:     "Comment with mixed case SUGGESTION:",
			comment:  "SuGgEsTiOn: Consider refactoring this method",
			expected: true,
		},
		{
			name:     "Multi-line comment with pattern on second line",
			comment:  "Good implementation overall.\noptional: You could add more tests",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isLowPriorityComment(tt.comment)
			if result != tt.expected {
				t.Errorf("isLowPriorityComment(%q) = %v, expected %v", tt.comment, result, tt.expected)
			}
		})
	}
}

func TestIsLowPriorityCommentNoPatterns(t *testing.T) {
	// Create analyzer with empty patterns
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{},
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Should always return false when no patterns configured
	comments := []string{
		"nit: This should not be detected",
		"minor: No patterns configured",
		"Any comment at all",
	}

	for _, comment := range comments {
		result := analyzer.isLowPriorityComment(comment)
		if result {
			t.Errorf("Expected false for comment %q when no patterns configured, got true", comment)
		}
	}
}

func TestIsCodeRabbitNitpickComment(t *testing.T) {
	// Create analyzer with default config
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{"nit:", "nits:"},
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		comment  string
		expected bool
	}{
		{
			name: "CodeRabbit nitpick comment with emoji",
			comment: `<details>
<summary>🧹 Nitpick comments (3)</summary>
<blockquote>
Some nitpick content here
</blockquote>
</details>`,
			expected: true,
		},
		{
			name: "CodeRabbit nitpick comment without emoji",
			comment: `<details>
<summary>Nitpick comments (2)</summary>
<blockquote>
Some nitpick content here
</blockquote>
</details>`,
			expected: true,
		},
		{
			name:     "Simple nitpick pattern",
			comment:  `🧹 Nitpick: Fix this minor issue`,
			expected: true,
		},
		{
			name:     "Direct summary tag with emoji",
			comment:  `<summary>🧹 nitpick comments (1)</summary>`,
			expected: true,
		},
		{
			name:     "Mixed case nitpick",
			comment:  `<summary>NITPICK Comments (5)</summary>`,
			expected: true,
		},
		{
			name:     "Regular comment without nitpick",
			comment:  `This is a regular review comment about functionality`,
			expected: false,
		},
		{
			name: "Details without nitpick summary",
			comment: `<details>
<summary>Review comments (3)</summary>
<blockquote>
Some review content here
</blockquote>
</details>`,
			expected: false,
		},
		{
			name: "Summary with style indicator",
			comment: `<details>
<summary>Style suggestions (2)</summary>
<blockquote>
Style related comments
</blockquote>
</details>`,
			expected: true,
		},
		{
			name: "Summary with minor indicator",
			comment: `<details>
<summary>Minor improvements</summary>
<blockquote>
Minor improvement suggestions
</blockquote>
</details>`,
			expected: true,
		},
		{
			name:     "Actual PR #120 CodeRabbit review format",
			comment:  `**Actionable comments posted: 0**\n\n<details>\n<summary>🧹 Nitpick comments (3)</summary><blockquote>\n\nSome actual nitpick content here\n\n</blockquote></details>`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isLowPriorityComment(tt.comment)
			if result != tt.expected {
				t.Errorf("isLowPriorityComment(%q) = %v, expected %v", tt.comment, result, tt.expected)
			}
		})
	}
}

func TestHasStructuredNitpickContent(t *testing.T) {
	// Create analyzer with default config
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			LowPriorityPatterns: []string{},
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		comment  string
		expected bool
	}{
		{
			name: "Complete details block with nitpick",
			comment: `<details>
<summary>🧹 Nitpick comments (3)</summary>
<blockquote>
Content here
</blockquote>
</details>`,
			expected: true,
		},
		{
			name:     "Details block without summary",
			comment:  `<details><blockquote>Content</blockquote></details>`,
			expected: false,
		},
		{
			name:     "Summary without details",
			comment:  `<summary>Some summary</summary>`,
			expected: false,
		},
		{
			name: "Summary with nit indicator",
			comment: `<details>
<summary>Just nit comments</summary>
</details>`,
			expected: true,
		},
		{
			name: "Summary with suggestion indicator",
			comment: `<details>
<summary>Suggestion for improvement</summary>
</details>`,
			expected: true,
		},
		{
			name: "Multiple indicators in summary",
			comment: `<details>
<summary>Style and minor suggestions (5)</summary>
</details>`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.hasStructuredNitpickContent(strings.ToLower(tt.comment))
			if result != tt.expected {
				t.Errorf("hasStructuredNitpickContent(%q) = %v, expected %v", tt.comment, result, tt.expected)
			}
		})
	}
}

func TestConvertToStorageTasksWithLowPriorityStatus(t *testing.T) {
	// Create analyzer with low-priority configuration
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus:       "todo",
			LowPriorityPatterns: []string{"nit:", "minor:"},
			LowPriorityStatus:   "pending",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// Create test tasks with various comment patterns
	testTasks := []TaskRequest{
		{
			Description:     "Fix indentation",
			OriginText:      "nit: Please fix the indentation here",
			Priority:        "low",
			SourceReviewID:  12345,
			SourceCommentID: 67890,
			File:            "test.go",
			Line:            42,
			TaskIndex:       0,
		},
		{
			Description:     "Add error handling",
			OriginText:      "This function needs proper error handling",
			Priority:        "high",
			SourceReviewID:  12345,
			SourceCommentID: 67891,
			File:            "test.go",
			Line:            50,
			TaskIndex:       0,
		},
		{
			Description:     "Improve naming",
			OriginText:      "MINOR: Variable names could be more descriptive",
			Priority:        "low",
			SourceReviewID:  12345,
			SourceCommentID: 67892,
			File:            "test.go",
			Line:            60,
			TaskIndex:       0,
		},
	}

	// Convert to storage tasks
	storageTasks := analyzer.convertToStorageTasks(testTasks)

	// Verify task count
	if len(storageTasks) != len(testTasks) {
		t.Fatalf("Expected %d tasks, got %d", len(testTasks), len(storageTasks))
	}

	// Test expectations
	expectedStatuses := []string{"pending", "todo", "pending"}

	for i, task := range storageTasks {
		// Verify status based on pattern detection
		if task.Status != expectedStatuses[i] {
			t.Errorf("Task %d: Expected status %q, got %q (origin: %q)",
				i, expectedStatuses[i], task.Status, task.OriginText)
		}

		// Verify other fields are preserved
		if task.Description != testTasks[i].Description {
			t.Errorf("Task %d: Description mismatch", i)
		}
		if task.Priority != testTasks[i].Priority {
			t.Errorf("Task %d: Priority mismatch", i)
		}
	}
}

// TestGenerateTasksValidationModeUsesParallelProcessing tests that validation mode
// now uses parallel processing instead of batch processing to handle large PRs.
// This addresses Issue #116: Claude Code CLI prompt size limit exceeded.
func TestGenerateTasksValidationModeUsesParallelProcessing(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{false}[0], // Disable validation for simpler test
			UserLanguage:      "English",
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}

	// Create a mock Claude client that tracks call patterns
	mockClient := NewMockClaudeClient()

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create test reviews with multiple comments to trigger parallel processing
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "PENDING",
			Comments: []github.Comment{
				{
					ID:     1,
					Author: "reviewer1",
					Body:   "Comment 1 - this needs fixing",
					File:   "test.go",
					Line:   1,
				},
				{
					ID:     2,
					Author: "reviewer1",
					Body:   "Comment 2 - another issue",
					File:   "test.go",
					Line:   2,
				},
			},
		},
	}

	// Generate tasks using parallel processing
	tasks, err := analyzer.GenerateTasks(reviews)

	// Verify no errors and tasks were generated
	if err != nil {
		t.Fatalf("GenerateTasks failed: %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Expected tasks to be generated, got 0")
	}

	// Verify the mock client was called (parallel processing means multiple individual calls)
	if mockClient.CallCount == 0 {
		t.Error("Expected Claude client to be called for parallel processing")
	}
}

// TestGenerateTasksHandlesPromptSizeLimitGracefully tests that prompt size errors
// are detected early and retries are avoided to prevent wasted API calls.
// This addresses Issue #116: Elimination of wasteful 5 retry attempts.
func TestGenerateTasksHandlesPromptSizeLimitGracefully(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{true}[0],
			MaxRetries:        5, // Should not retry 5 times for size errors
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}

	// Create a mock client that returns size limit error
	mockClient := NewMockClaudeClient()
	mockClient.Error = fmt.Errorf("prompt size (39982 bytes) exceeds maximum limit (32768 bytes). Please shorten or chunk the prompt content")

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create large review that would trigger size error in batch mode
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "PENDING",
			Comments: []github.Comment{
				{
					ID:     1,
					Author: "reviewer1",
					Body:   "This is a comment that would cause size issues in batch mode",
					File:   "test.go",
					Line:   1,
				},
			},
		},
	}

	// Generate tasks - should handle size error gracefully
	tasks, err := analyzer.GenerateTasks(reviews)

	// With parallel processing, size errors should be rare/handled per comment
	// But if they occur, we should get a meaningful error without excessive retries
	if err != nil && strings.Contains(err.Error(), "prompt size") {
		// This is expected for this test case
		t.Logf("Size error handled gracefully: %v", err)
	}

	// Verify we didn't make excessive retry attempts
	// With parallel processing and early detection, call count should be minimal
	if mockClient.CallCount > 2 {
		t.Errorf("Expected minimal retry attempts for size errors, got %d calls", mockClient.CallCount)
	}

	// If no error occurred, tasks should be generated
	if err == nil && len(tasks) == 0 {
		t.Error("Expected either tasks to be generated or a meaningful error")
	}
}

// TestValidationModeParallelProcessingPerformance tests that parallel processing
// improves performance for large PRs compared to batch processing.
func TestValidationModeParallelProcessingPerformance(t *testing.T) {
	cfg := &config.Config{
		AISettings: config.AISettings{
			ValidationEnabled: &[]bool{false}[0], // Disable validation for simpler test
			VerboseMode:       true,              // Enable debug for visibility
		},
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}

	// Create mock client - it will use default response since patterns don't need to match exactly
	mockClient := NewMockClaudeClient()

	analyzer := NewAnalyzerWithClient(cfg, mockClient)

	// Create review with multiple comments to test parallel processing
	reviews := []github.Review{
		{
			ID:       1,
			Reviewer: "test-reviewer",
			State:    "PENDING",
			Comments: []github.Comment{
				{ID: 1, Author: "reviewer1", Body: "Comment 1", File: "test.go", Line: 1},
				{ID: 2, Author: "reviewer1", Body: "Comment 2", File: "test.go", Line: 2},
				{ID: 3, Author: "reviewer1", Body: "Comment 3", File: "test.go", Line: 3},
			},
		},
	}

	// Measure execution time
	start := time.Now()
	tasks, err := analyzer.GenerateTasks(reviews)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateTasks failed: %v", err)
	}

	// Verify tasks were generated
	expectedTaskCount := 3 // One task per comment
	if len(tasks) != expectedTaskCount {
		t.Errorf("Expected %d tasks, got %d", expectedTaskCount, len(tasks))
	}

	// Verify parallel processing occurred (multiple calls to Claude)
	if mockClient.CallCount < 3 {
		t.Errorf("Expected at least 3 Claude calls for parallel processing, got %d", mockClient.CallCount)
	}

	// Performance should be reasonable (parallel processing should not be significantly slower)
	maxExpectedDuration := time.Second * 5 // Generous limit for test environment
	if duration > maxExpectedDuration {
		t.Errorf("Parallel processing took too long: %v (max expected: %v)", duration, maxExpectedDuration)
	}

	t.Logf("Parallel processing completed in %v with %d Claude calls", duration, mockClient.CallCount)
}

// TestIsCommentResolved_GitHubThreadResolvedField tests that isCommentResolved
// properly checks the GitHubThreadResolved field before falling back to text markers.
// This addresses Issue #233: isCommentResolved not checking GitHubThreadResolved field.
func TestIsCommentResolved_GitHubThreadResolvedField(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		comment  github.Comment
		expected bool
	}{
		{
			name: "GitHubThreadResolved is true - should return true immediately",
			comment: github.Comment{
				ID:                   1,
				Body:                 "This is a comment without any resolution markers",
				GitHubThreadResolved: true,
			},
			expected: true,
		},
		{
			name: "GitHubThreadResolved is false with no markers - should return false",
			comment: github.Comment{
				ID:                   2,
				Body:                 "This is an unresolved comment",
				GitHubThreadResolved: false,
			},
			expected: false,
		},
		{
			name: "GitHubThreadResolved is false but has resolution marker - should return true",
			comment: github.Comment{
				ID:                   3,
				Body:                 "✅ Addressed in commit abc123",
				GitHubThreadResolved: false,
			},
			expected: true,
		},
		{
			name: "GitHubThreadResolved is true with resolution marker - should return true",
			comment: github.Comment{
				ID:                   4,
				Body:                 "✅ Fixed in commit def456",
				GitHubThreadResolved: true,
			},
			expected: true,
		},
		{
			name: "Empty body with GitHubThreadResolved true - should return true",
			comment: github.Comment{
				ID:                   5,
				Body:                 "",
				GitHubThreadResolved: true,
			},
			expected: true,
		},
		{
			name: "Empty body with GitHubThreadResolved false - should return false",
			comment: github.Comment{
				ID:                   6,
				Body:                 "",
				GitHubThreadResolved: false,
			},
			expected: false,
		},
		{
			name: "GitHubThreadResolved false with reply containing resolution marker",
			comment: github.Comment{
				ID:                   7,
				Body:                 "This needs fixing",
				GitHubThreadResolved: false,
				Replies: []github.Reply{
					{
						ID:   701,
						Body: "Fixed in commit xyz789",
					},
				},
			},
			expected: true,
		},
		{
			name: "GitHubThreadResolved true even without text markers",
			comment: github.Comment{
				ID:                   8,
				Body:                 "This comment is resolved on GitHub but has no text marker",
				GitHubThreadResolved: true,
				Replies:              []github.Reply{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isCommentResolved(tt.comment)
			if result != tt.expected {
				t.Errorf("isCommentResolved(%+v) = %v, expected %v", tt.comment, result, tt.expected)
			}
		})
	}
}

// TestIsCommentResolved_TextMarkerFallback tests that text marker detection
// still works as a fallback when GitHubThreadResolved is false.
func TestIsCommentResolved_TextMarkerFallback(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name     string
		comment  github.Comment
		expected bool
	}{
		{
			name: "Comment with ✅ Addressed marker",
			comment: github.Comment{
				ID:                   1,
				Body:                 "✅ Addressed in commit abc123",
				GitHubThreadResolved: false,
			},
			expected: true,
		},
		{
			name: "Comment with ✅ Fixed marker",
			comment: github.Comment{
				ID:                   2,
				Body:                 "✅ Fixed in commit def456",
				GitHubThreadResolved: false,
			},
			expected: true,
		},
		{
			name: "Comment with ✅ Resolved marker",
			comment: github.Comment{
				ID:                   3,
				Body:                 "✅ Resolved in commit ghi789",
				GitHubThreadResolved: false,
			},
			expected: true,
		},
		{
			name: "Comment with plain Addressed marker (no emoji)",
			comment: github.Comment{
				ID:                   4,
				Body:                 "Addressed in commit jkl012",
				GitHubThreadResolved: false,
			},
			expected: true,
		},
		{
			name: "Comment with plain Fixed marker (no emoji)",
			comment: github.Comment{
				ID:                   5,
				Body:                 "Fixed in commit mno345",
				GitHubThreadResolved: false,
			},
			expected: true,
		},
		{
			name: "Comment with plain Resolved marker (no emoji)",
			comment: github.Comment{
				ID:                   6,
				Body:                 "Resolved in commit pqr678",
				GitHubThreadResolved: false,
			},
			expected: true,
		},
		{
			name: "Reply with resolution marker",
			comment: github.Comment{
				ID:                   7,
				Body:                 "This needs to be fixed",
				GitHubThreadResolved: false,
				Replies: []github.Reply{
					{
						ID:   701,
						Body: "✅ Fixed in commit stu901",
					},
				},
			},
			expected: true,
		},
		{
			name: "Multiple replies with one containing marker",
			comment: github.Comment{
				ID:                   8,
				Body:                 "Multiple issues here",
				GitHubThreadResolved: false,
				Replies: []github.Reply{
					{
						ID:   801,
						Body: "Working on it",
					},
					{
						ID:   802,
						Body: "Addressed in commit vwx234",
					},
				},
			},
			expected: true,
		},
		{
			name: "Comment and reply without markers",
			comment: github.Comment{
				ID:                   9,
				Body:                 "This is an issue",
				GitHubThreadResolved: false,
				Replies: []github.Reply{
					{
						ID:   901,
						Body: "I'm looking into this",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isCommentResolved(tt.comment)
			if result != tt.expected {
				t.Errorf("isCommentResolved() = %v, expected %v\nComment: %+v", result, tt.expected, tt.comment)
			}
		})
	}
}

// TestIsCommentResolved_PriorityOrder tests that GitHubThreadResolved
// is checked before text markers for performance optimization.
func TestIsCommentResolved_PriorityOrder(t *testing.T) {
	cfg := &config.Config{
		TaskSettings: config.TaskSettings{
			DefaultStatus: "todo",
		},
	}
	analyzer := NewAnalyzer(cfg)

	// This comment has GitHubThreadResolved=true, so it should return true
	// immediately without needing to check the body or replies
	comment := github.Comment{
		ID:                   1,
		Body:                 "This is a very long comment body that would take time to parse",
		GitHubThreadResolved: true,
		Replies: []github.Reply{
			{ID: 101, Body: "Reply 1"},
			{ID: 102, Body: "Reply 2"},
			{ID: 103, Body: "Reply 3"},
		},
	}

	result := analyzer.isCommentResolved(comment)
	if !result {
		t.Errorf("isCommentResolved() should return true for GitHubThreadResolved=true, got false")
	}

	// This comment has GitHubThreadResolved=false but has a resolution marker
	// It should still return true by checking the fallback text markers
	comment2 := github.Comment{
		ID:                   2,
		Body:                 "✅ Fixed in commit abc123",
		GitHubThreadResolved: false,
	}

	result2 := analyzer.isCommentResolved(comment2)
	if !result2 {
		t.Errorf("isCommentResolved() should return true for resolution marker, got false")
	}
}
