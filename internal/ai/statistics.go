package ai

import (
	"time"

	"reviewtask/internal/storage"
)

type StatisticsManager struct {
	storageManager *storage.Manager
}

func NewStatisticsManager(storageManager *storage.Manager) *StatisticsManager {
	return &StatisticsManager{
		storageManager: storageManager,
	}
}

func (sm *StatisticsManager) GenerateTaskStatistics(prNumber int) (*storage.TaskStatistics, error) {
	tasks, err := sm.storageManager.GetTasksByPR(prNumber)
	if err != nil {
		return nil, err
	}

	return sm.generateStatsFromTasks(tasks, prNumber, "")
}

// GenerateCurrentBranchStatistics generates statistics for the current branch
func (sm *StatisticsManager) GenerateCurrentBranchStatistics() (*storage.TaskStatistics, error) {
	currentBranch, err := sm.storageManager.GetCurrentBranch()
	if err != nil {
		return nil, err
	}

	return sm.GenerateBranchStatistics(currentBranch)
}

// GenerateBranchStatistics generates statistics for a specific branch
func (sm *StatisticsManager) GenerateBranchStatistics(branchName string) (*storage.TaskStatistics, error) {
	prNumbers, err := sm.storageManager.GetPRsForBranch(branchName)
	if err != nil {
		return nil, err
	}

	if len(prNumbers) == 0 {
		return &storage.TaskStatistics{
			PRNumber:      -1, // Indicate this is branch stats, not single PR
			BranchName:    branchName,
			GeneratedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
			TotalComments: 0,
			TotalTasks:    0,
			CommentStats:  []storage.CommentStats{},
			StatusSummary: storage.StatusSummary{},
		}, nil
	}

	// Collect all tasks from all PRs for this branch
	var allTasks []storage.Task
	for _, prNumber := range prNumbers {
		tasks, err := sm.storageManager.GetTasksByPR(prNumber)
		if err != nil {
			continue // Skip PRs that can't be read
		}
		allTasks = append(allTasks, tasks...)
	}

	return sm.generateStatsFromTasks(allTasks, -1, branchName)
}

// generateStatsFromTasks is a helper function to generate statistics from a list of tasks
func (sm *StatisticsManager) generateStatsFromTasks(tasks []storage.Task, prNumber int, branchName string) (*storage.TaskStatistics, error) {
	// Group tasks by comment ID
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

		// Count by status
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
		GeneratedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
		TotalComments: len(commentGroups),
		TotalTasks:    len(tasks),
		CommentStats:  commentStats,
		StatusSummary: statusSummary,
	}, nil
}
