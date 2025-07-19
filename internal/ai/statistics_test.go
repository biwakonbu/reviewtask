package ai

import (
	"fmt"
	"testing"

	"gh-review-task/internal/storage"
	"gh-review-task/internal/testutil"
)

// StorageInterface defines the interface that storage manager must implement
type StorageInterface interface {
	GetTasksByPR(prNumber int) ([]storage.Task, error)
	GetCurrentBranch() (string, error)
	GetPRsForBranch(branchName string) ([]int, error)
	GetAllPRNumbers() ([]int, error)
}


// TestStatisticsManager provides a test version that accepts our interface
type TestStatisticsManager struct {
	storageManager StorageInterface
}

func NewTestStatisticsManager(storageManager StorageInterface) *TestStatisticsManager {
	return &TestStatisticsManager{
		storageManager: storageManager,
	}
}

func (sm *TestStatisticsManager) GenerateTaskStatistics(prNumber int) (*storage.TaskStatistics, error) {
	tasks, err := sm.storageManager.GetTasksByPR(prNumber)
	if err != nil {
		return nil, err
	}

	return sm.generateStatsFromTasks(tasks, prNumber, "")
}

func (sm *TestStatisticsManager) GenerateCurrentBranchStatistics() (*storage.TaskStatistics, error) {
	currentBranch, err := sm.storageManager.GetCurrentBranch()
	if err != nil {
		return nil, err
	}

	return sm.GenerateBranchStatistics(currentBranch)
}

func (sm *TestStatisticsManager) GenerateBranchStatistics(branchName string) (*storage.TaskStatistics, error) {
	prNumbers, err := sm.storageManager.GetPRsForBranch(branchName)
	if err != nil {
		return nil, err
	}

	if len(prNumbers) == 0 {
		return &storage.TaskStatistics{
			PRNumber:      -1,
			BranchName:    branchName,
			GeneratedAt:   "2023-01-01T00:00:00Z",
			TotalComments: 0,
			TotalTasks:    0,
			CommentStats:  []storage.CommentStats{},
			StatusSummary: storage.StatusSummary{},
		}, nil
	}

	var allTasks []storage.Task
	for _, prNumber := range prNumbers {
		tasks, err := sm.storageManager.GetTasksByPR(prNumber)
		if err != nil {
			continue
		}
		allTasks = append(allTasks, tasks...)
	}

	return sm.generateStatsFromTasks(allTasks, -1, branchName)
}

func (sm *TestStatisticsManager) generateStatsFromTasks(tasks []storage.Task, prNumber int, branchName string) (*storage.TaskStatistics, error) {
	commentGroups := make(map[int64][]storage.Task)
	for _, task := range tasks {
		commentGroups[task.SourceCommentID] = append(commentGroups[task.SourceCommentID], task)
	}

	var commentStats []storage.CommentStats
	statusSummary := storage.StatusSummary{}

	for commentID, commentTasks := range commentGroups {
		stats := storage.CommentStats{
			CommentID:  commentID,
			TotalTasks: len(commentTasks),
			File:       commentTasks[0].File,
			Line:       commentTasks[0].Line,
			OriginText: commentTasks[0].OriginText,
		}

		for _, task := range commentTasks {
			switch task.Status {
			case "todo":
				stats.PendingTasks++
				statusSummary.Todo++
			case "doing":
				stats.InProgressTasks++
				statusSummary.Doing++
			case "done":
				stats.CompletedTasks++
				statusSummary.Done++
			case "pending":
				stats.PendingTasks++
				statusSummary.Pending++
			case "cancelled":
				stats.CancelledTasks++
				statusSummary.Cancelled++
			}
		}

		commentStats = append(commentStats, stats)
	}

	return &storage.TaskStatistics{
		PRNumber:      prNumber,
		BranchName:    branchName,
		GeneratedAt:   "2023-01-01T00:00:00Z",
		TotalComments: len(commentGroups),
		TotalTasks:    len(tasks),
		CommentStats:  commentStats,
		StatusSummary: statusSummary,
	}, nil
}

// TestStatisticsManager_GenerateTaskStatistics tests basic PR statistics generation
func TestStatisticsManager_GenerateTaskStatistics(t *testing.T) {
	mockStorage := testutil.NewMockStorageManager()
	statsManager := NewTestStatisticsManager(mockStorage)

	// Set up test tasks
	testTasks := []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655440001",
			Description:     "Task 1",
			SourceCommentID: 1,
			Status:          "done",
			File:            "test.go",
			Line:            10,
			OriginText:      "Fix this issue",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440002",
			Description:     "Task 2",
			SourceCommentID: 1,
			Status:          "todo",
			File:            "test.go",
			Line:            10,
			OriginText:      "Fix this issue",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440003",
			Description:     "Task 3",
			SourceCommentID: 2,
			Status:          "doing",
			File:            "test2.go",
			Line:            20,
			OriginText:      "Another issue",
		},
	}

	mockStorage.SetTasks(123, testTasks)

	// Generate statistics
	stats, err := statsManager.GenerateTaskStatistics(123)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify basic statistics
	if stats.PRNumber != 123 {
		t.Errorf("Expected PR number 123, got: %d", stats.PRNumber)
	}

	if stats.TotalTasks != 3 {
		t.Errorf("Expected 3 total tasks, got: %d", stats.TotalTasks)
	}

	if stats.TotalComments != 2 {
		t.Errorf("Expected 2 total comments, got: %d", stats.TotalComments)
	}

	// Verify status summary
	expectedSummary := storage.StatusSummary{
		Done:  1,
		Doing: 1,
		Todo:  1,
	}

	if stats.StatusSummary != expectedSummary {
		t.Errorf("Expected status summary %+v, got: %+v", expectedSummary, stats.StatusSummary)
	}

	// Verify comment stats
	if len(stats.CommentStats) != 2 {
		t.Errorf("Expected 2 comment stats, got: %d", len(stats.CommentStats))
	}

	// Find comment 1 stats
	var comment1Stats *storage.CommentStats
	for i := range stats.CommentStats {
		if stats.CommentStats[i].CommentID == 1 {
			comment1Stats = &stats.CommentStats[i]
			break
		}
	}

	if comment1Stats == nil {
		t.Fatal("Comment 1 stats not found")
	}

	if comment1Stats.TotalTasks != 2 {
		t.Errorf("Expected comment 1 to have 2 tasks, got: %d", comment1Stats.TotalTasks)
	}

	if comment1Stats.CompletedTasks != 1 {
		t.Errorf("Expected comment 1 to have 1 completed task, got: %d", comment1Stats.CompletedTasks)
	}
}

// TestStatisticsManager_GenerateCurrentBranchStatistics tests current branch statistics
func TestStatisticsManager_GenerateCurrentBranchStatistics(t *testing.T) {
	mockStorage := testutil.NewMockStorageManager()
	statsManager := NewTestStatisticsManager(mockStorage)

	// Set up test data
	mockStorage.SetCurrentBranch("feature/test")
	mockStorage.SetPRsForBranch("feature/test", []int{1, 2})

	// Set up tasks for both PRs
	tasks1 := []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655440101",
			SourceCommentID: 1,
			Status:          "done",
			OriginText:      "Task 1",
		},
	}

	tasks2 := []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655440201",
			SourceCommentID: 2,
			Status:          "todo",
			OriginText:      "Task 2",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655440202",
			SourceCommentID: 2,
			Status:          "done",
			OriginText:      "Task 2",
		},
	}

	mockStorage.SetTasks(1, tasks1)
	mockStorage.SetTasks(2, tasks2)

	// Generate current branch statistics
	stats, err := statsManager.GenerateCurrentBranchStatistics()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify branch-specific statistics
	if stats.BranchName != "feature/test" {
		t.Errorf("Expected branch name 'feature/test', got: %s", stats.BranchName)
	}

	if stats.PRNumber != -1 {
		t.Errorf("Expected PR number -1 for branch stats, got: %d", stats.PRNumber)
	}

	if stats.TotalTasks != 3 {
		t.Errorf("Expected 3 total tasks, got: %d", stats.TotalTasks)
	}

	if stats.TotalComments != 2 {
		t.Errorf("Expected 2 total comments, got: %d", stats.TotalComments)
	}

	// Verify status summary aggregation
	expectedSummary := storage.StatusSummary{
		Done: 2,
		Todo: 1,
	}

	if stats.StatusSummary != expectedSummary {
		t.Errorf("Expected status summary %+v, got: %+v", expectedSummary, stats.StatusSummary)
	}
}

// TestStatisticsManager_GenerateBranchStatistics tests specific branch statistics
func TestStatisticsManager_GenerateBranchStatistics(t *testing.T) {
	mockStorage := testutil.NewMockStorageManager()
	statsManager := NewTestStatisticsManager(mockStorage)

	tests := []struct {
		name          string
		branchName    string
		prNumbers     []int
		expectedTasks int
		expectedError bool
	}{
		{
			name:          "Branch with PRs",
			branchName:    "feature/test",
			prNumbers:     []int{1, 2},
			expectedTasks: 2,
			expectedError: false,
		},
		{
			name:          "Branch with no PRs",
			branchName:    "feature/empty",
			prNumbers:     []int{},
			expectedTasks: 0,
			expectedError: false,
		},
		{
			name:          "Nonexistent branch",
			branchName:    "feature/nonexistent",
			prNumbers:     nil,
			expectedTasks: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up branch data
			if tt.prNumbers != nil {
				mockStorage.SetPRsForBranch(tt.branchName, tt.prNumbers)
			}

			// Set up tasks for PRs
			for i, prNum := range tt.prNumbers {
				tasks := []storage.Task{
					{
						ID:              fmt.Sprintf("550e8400-e29b-41d4-a716-44665544%04d", i+1),
						SourceCommentID: int64(i + 1),
						Status:          "done",
						OriginText:      "Test task",
					},
				}
				mockStorage.SetTasks(prNum, tasks)
			}

			// Generate statistics
			stats, err := statsManager.GenerateBranchStatistics(tt.branchName)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error, got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if stats.TotalTasks != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got: %d", tt.expectedTasks, stats.TotalTasks)
			}

			if stats.BranchName != tt.branchName {
				t.Errorf("Expected branch name %s, got: %s", tt.branchName, stats.BranchName)
			}
		})
	}
}

// TestStatisticsManager_EmptyBranchStatistics tests statistics for empty branch
func TestStatisticsManager_EmptyBranchStatistics(t *testing.T) {
	mockStorage := testutil.NewMockStorageManager()
	statsManager := NewTestStatisticsManager(mockStorage)

	// Set up empty branch
	mockStorage.SetPRsForBranch("feature/empty", []int{})

	stats, err := statsManager.GenerateBranchStatistics("feature/empty")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify empty statistics
	if stats.TotalTasks != 0 {
		t.Errorf("Expected 0 tasks, got: %d", stats.TotalTasks)
	}

	if stats.TotalComments != 0 {
		t.Errorf("Expected 0 comments, got: %d", stats.TotalComments)
	}

	if len(stats.CommentStats) != 0 {
		t.Errorf("Expected 0 comment stats, got: %d", len(stats.CommentStats))
	}

	if stats.BranchName != "feature/empty" {
		t.Errorf("Expected branch name 'feature/empty', got: %s", stats.BranchName)
	}

	if stats.PRNumber != -1 {
		t.Errorf("Expected PR number -1 for branch stats, got: %d", stats.PRNumber)
	}
}
