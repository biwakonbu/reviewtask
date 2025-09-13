package test

import (
	"testing"

	"reviewtask/internal/storage"
	"reviewtask/internal/testutil/mocks"
)

// TestBranchStatisticsWorkflow tests the complete workflow with mocks
func TestBranchStatisticsWorkflow(t *testing.T) {
	// This test demonstrates the complete workflow using mocks
	// instead of real file system operations

	// Setup: Create mock storage with test data
	mockStorage := mocks.NewMockStorageManager()
	mockStorage.SetCurrentBranch("feature/auth")
	mockStorage.SetPRsForBranch("feature/auth", []int{1, 3})
	mockStorage.SetPRsForBranch("feature/db", []int{2})
	mockStorage.SetPRsForBranch("main", []int{4})

	// Setup tasks for each PR
	mockStorage.SetTasks(1, []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655441001",
			Description:     "Add authentication middleware",
			SourceCommentID: 1,
			Status:          "done",
			File:            "auth.go",
			Line:            10,
			OriginText:      "Add auth middleware",
			Priority:        "high",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655441002",
			Description:     "Add tests for auth",
			SourceCommentID: 1,
			Status:          "todo",
			File:            "auth.go",
			Line:            10,
			OriginText:      "Add auth middleware",
			Priority:        "medium",
		},
	})

	mockStorage.SetTasks(2, []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655442001",
			Description:     "Optimize database queries",
			SourceCommentID: 2,
			Status:          "doing",
			File:            "db.go",
			Line:            20,
			OriginText:      "Optimize queries",
			Priority:        "critical",
		},
	})

	mockStorage.SetTasks(3, []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655443001",
			Description:     "Add OAuth support",
			SourceCommentID: 3,
			Status:          "todo",
			File:            "oauth.go",
			Line:            5,
			OriginText:      "Add OAuth",
			Priority:        "low",
		},
	})

	mockStorage.SetTasks(4, []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655444001",
			Description:     "Update documentation",
			SourceCommentID: 4,
			Status:          "done",
			File:            "README.md",
			Line:            1,
			OriginText:      "Update docs",
			Priority:        "low",
		},
	})

	statsManager := NewTestStatisticsManager(mockStorage)

	// Test 1: Current branch statistics (feature/auth)
	t.Run("CurrentBranchStatistics", func(t *testing.T) {
		stats, err := statsManager.GenerateCurrentBranchStatistics()
		if err != nil {
			t.Fatalf("Failed to generate current branch stats: %v", err)
		}

		if stats.BranchName != "feature/auth" {
			t.Errorf("Expected branch 'feature/auth', got: %s", stats.BranchName)
		}

		if stats.TotalTasks != 3 {
			t.Errorf("Expected 3 total tasks, got: %d", stats.TotalTasks)
		}

		if stats.TotalComments != 2 {
			t.Errorf("Expected 2 comments, got: %d", stats.TotalComments)
		}

		// Verify status summary
		expected := storage.StatusSummary{Done: 1, Todo: 2}
		if stats.StatusSummary.Done != expected.Done || stats.StatusSummary.Todo != expected.Todo {
			t.Errorf("Expected summary %+v, got %+v", expected, stats.StatusSummary)
		}
	})

	// Test 2: Specific branch statistics (feature/db)
	t.Run("SpecificBranchStatistics", func(t *testing.T) {
		stats, err := statsManager.GenerateBranchStatistics("feature/db")
		if err != nil {
			t.Fatalf("Failed to generate branch stats: %v", err)
		}

		if stats.BranchName != "feature/db" {
			t.Errorf("Expected branch 'feature/db', got: %s", stats.BranchName)
		}

		if stats.TotalTasks != 1 {
			t.Errorf("Expected 1 task, got: %d", stats.TotalTasks)
		}

		if stats.StatusSummary.Doing != 1 {
			t.Errorf("Expected 1 doing task, got: %d", stats.StatusSummary.Doing)
		}
	})

	// Test 3: Empty branch statistics
	t.Run("EmptyBranchStatistics", func(t *testing.T) {
		stats, err := statsManager.GenerateBranchStatistics("feature/nonexistent")
		if err != nil {
			t.Fatalf("Failed to generate empty branch stats: %v", err)
		}

		if stats.TotalTasks != 0 {
			t.Errorf("Expected 0 tasks, got: %d", stats.TotalTasks)
		}

		if stats.BranchName != "feature/nonexistent" {
			t.Errorf("Expected branch 'feature/nonexistent', got: %s", stats.BranchName)
		}
	})

	// Test 4: PR filtering functionality
	t.Run("PRFiltering", func(t *testing.T) {
		// Test feature/auth branch PRs
		authPRs, err := mockStorage.GetPRsForBranch("feature/auth")
		if err != nil {
			t.Fatalf("Failed to get PRs for feature/auth: %v", err)
		}

		expectedAuthPRs := []int{1, 3}
		if len(authPRs) != len(expectedAuthPRs) {
			t.Errorf("Expected %d PRs, got %d", len(expectedAuthPRs), len(authPRs))
		}

		for _, expectedPR := range expectedAuthPRs {
			found := false
			for _, actualPR := range authPRs {
				if actualPR == expectedPR {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected PR %d not found in results", expectedPR)
			}
		}

		// Test feature/db branch PRs
		dbPRs, err := mockStorage.GetPRsForBranch("feature/db")
		if err != nil {
			t.Fatalf("Failed to get PRs for feature/db: %v", err)
		}

		if len(dbPRs) != 1 || dbPRs[0] != 2 {
			t.Errorf("Expected PR [2], got %v", dbPRs)
		}
	})

	// Test 5: Cross-branch task aggregation
	t.Run("CrossBranchAggregation", func(t *testing.T) {
		// Get all PRs numbers
		allPRs, err := mockStorage.GetAllPRNumbers()
		if err != nil {
			t.Fatalf("Failed to get all PR numbers: %v", err)
		}

		expectedTotal := 4 // PRs 1, 2, 3, 4
		if len(allPRs) != expectedTotal {
			t.Errorf("Expected %d total PRs, got %d", expectedTotal, len(allPRs))
		}

		// Verify all expected PRs are present
		expectedPRs := map[int]bool{1: true, 2: true, 3: true, 4: true}
		for _, pr := range allPRs {
			if !expectedPRs[pr] {
				t.Errorf("Unexpected PR %d found", pr)
			}
			delete(expectedPRs, pr)
		}

		if len(expectedPRs) > 0 {
			t.Errorf("Missing PRs: %v", expectedPRs)
		}
	})
}

// TestCommandLineWorkflow simulates the command-line usage patterns
func TestCommandLineWorkflow(t *testing.T) {
	// This test simulates how users would interact with the CLI commands
	// using the new branch-specific functionality

	mockStorage := mocks.NewMockStorageManager()
	mockStorage.SetCurrentBranch("feature/new-feature")
	mockStorage.SetPRsForBranch("feature/new-feature", []int{5})
	mockStorage.SetPRsForBranch("main", []int{6})

	mockStorage.SetTasks(5, []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655445001",
			SourceCommentID: 5,
			Status:          "todo",
			Priority:        "high",
			OriginText:      "Implement feature",
		},
		{
			ID:              "550e8400-e29b-41d4-a716-446655445002",
			SourceCommentID: 5,
			Status:          "doing",
			Priority:        "medium",
			OriginText:      "Implement feature",
		},
	})

	mockStorage.SetTasks(6, []storage.Task{
		{
			ID:              "550e8400-e29b-41d4-a716-446655446001",
			SourceCommentID: 6,
			Status:          "done",
			Priority:        "low",
			OriginText:      "Fix typo",
		},
	})

	statsManager := NewTestStatisticsManager(mockStorage)

	// Simulate: reviewtask stats (default: current branch)
	t.Run("DefaultCurrentBranchStats", func(t *testing.T) {
		stats, err := statsManager.GenerateCurrentBranchStatistics()
		if err != nil {
			t.Fatalf("Default stats command failed: %v", err)
		}

		if stats.BranchName != "feature/new-feature" {
			t.Errorf("Expected current branch stats, got branch: %s", stats.BranchName)
		}

		if stats.TotalTasks != 2 {
			t.Errorf("Expected 2 tasks for current branch, got: %d", stats.TotalTasks)
		}
	})

	// Simulate: reviewtask stats --branch main
	t.Run("SpecificBranchStats", func(t *testing.T) {
		stats, err := statsManager.GenerateBranchStatistics("main")
		if err != nil {
			t.Fatalf("Branch-specific stats command failed: %v", err)
		}

		if stats.BranchName != "main" {
			t.Errorf("Expected main branch stats, got branch: %s", stats.BranchName)
		}

		if stats.TotalTasks != 1 {
			t.Errorf("Expected 1 task for main branch, got: %d", stats.TotalTasks)
		}

		if stats.StatusSummary.Done != 1 {
			t.Errorf("Expected 1 done task, got: %d", stats.StatusSummary.Done)
		}
	})

	// Simulate: reviewtask stats --branch nonexistent
	t.Run("NonexistentBranchStats", func(t *testing.T) {
		stats, err := statsManager.GenerateBranchStatistics("feature/does-not-exist")
		if err != nil {
			t.Fatalf("Nonexistent branch stats should not error: %v", err)
		}

		if stats.TotalTasks != 0 {
			t.Errorf("Expected 0 tasks for nonexistent branch, got: %d", stats.TotalTasks)
		}

		if stats.BranchName != "feature/does-not-exist" {
			t.Errorf("Expected branch name preserved, got: %s", stats.BranchName)
		}
	})
}
