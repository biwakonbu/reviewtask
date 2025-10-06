package tasks

import (
	"reviewtask/internal/storage"
)

// NextTaskRecommendation contains information about the next recommended task
type NextTaskRecommendation struct {
	HasNext        bool
	NextTask       *storage.Task
	RemainingTodo  int
	RemainingDoing int
	AllComplete    bool
	Message        string
}

// TaskRecommender provides next task recommendations
type TaskRecommender struct {
	storage StorageInterface
}

// StorageInterface defines storage operations needed by recommender
type StorageInterface interface {
	GetAllTasks() ([]storage.Task, error)
}

// NewTaskRecommender creates a new task recommender instance
func NewTaskRecommender(storageManager StorageInterface) *TaskRecommender {
	return &TaskRecommender{
		storage: storageManager,
	}
}

// GetNextRecommendedTask returns the next recommended task to work on
// Priority order:
// 1. Tasks with status "todo" (highest priority first)
// 2. Tasks with status "pending" (highest priority first)
// 3. No more tasks (all complete)
func (r *TaskRecommender) GetNextRecommendedTask() (*NextTaskRecommendation, error) {
	allTasks, err := r.storage.GetAllTasks()
	if err != nil {
		return nil, err
	}

	// Filter and categorize tasks
	var todoTasks []storage.Task
	var doingTasks []storage.Task
	var pendingTasks []storage.Task

	for _, task := range allTasks {
		switch task.Status {
		case "todo":
			todoTasks = append(todoTasks, task)
		case "doing":
			doingTasks = append(doingTasks, task)
		case "pending":
			pendingTasks = append(pendingTasks, task)
		}
	}

	// Sort by priority
	SortTasksByPriority(todoTasks)
	SortTasksByPriority(pendingTasks)

	recommendation := &NextTaskRecommendation{
		RemainingTodo:  len(todoTasks),
		RemainingDoing: len(doingTasks),
	}

	// Recommend highest priority todo task first
	if len(todoTasks) > 0 {
		recommendation.HasNext = true
		recommendation.NextTask = &todoTasks[0]
		recommendation.Message = "Next recommended task from TODO"
		return recommendation, nil
	}

	// If no todo tasks, recommend pending tasks
	if len(pendingTasks) > 0 {
		recommendation.HasNext = true
		recommendation.NextTask = &pendingTasks[0]
		recommendation.Message = "No TODO tasks remaining, showing highest priority PENDING task"
		return recommendation, nil
	}

	// All tasks complete
	recommendation.AllComplete = true
	recommendation.Message = "All tasks complete!"
	return recommendation, nil
}

// GetRecommendationAfterCompletion returns recommendation after completing a task
func (r *TaskRecommender) GetRecommendationAfterCompletion(completedTaskID string) (*NextTaskRecommendation, error) {
	return r.GetNextRecommendedTask()
}
