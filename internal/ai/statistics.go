package ai

import (
	"time"

	"gh-review-task/internal/storage"
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
		PRNumber:       prNumber,
		GeneratedAt:    time.Now().Format("2006-01-02T15:04:05Z"),
		TotalComments:  len(commentGroups),
		TotalTasks:     len(tasks),
		CommentStats:   commentStats,
		StatusSummary:  statusSummary,
	}, nil
}