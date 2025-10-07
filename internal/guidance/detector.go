package guidance

import (
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
)

const (
	StatusTodo    = "todo"
	StatusDoing   = "doing"
	StatusDone    = "done"
	StatusPending = "pending"
	StatusHold    = "hold"
	StatusCancel  = "cancel"
)

// Detector analyzes current state to build guidance context.
type Detector struct {
	storage *storage.Manager
}

// NewDetector creates a new context detector.
func NewDetector(storage *storage.Manager) *Detector {
	return &Detector{
		storage: storage,
	}
}

// DetectContext analyzes the current state and returns a Context.
func (d *Detector) DetectContext() (*Context, error) {
	allTasks, err := d.storage.GetAllTasks()
	if err != nil {
		return nil, err
	}

	ctx := &Context{
		Language: "en", // Default, can be configured later
	}

	// Count tasks by status
	for _, task := range allTasks {
		switch task.Status {
		case StatusTodo:
			ctx.TodoCount++
		case StatusDoing:
			ctx.DoingCount++
		case StatusDone:
			ctx.DoneCount++
		case StatusPending:
			ctx.PendingCount++
		case StatusHold:
			ctx.HoldCount++
		case StatusCancel:
			// Cancelled tasks are treated as complete for guidance purposes
			ctx.DoneCount++
		}
	}

	// Set state flags
	ctx.HasPendingTasks = ctx.PendingCount > 0
	ctx.AllTasksComplete = (ctx.TodoCount == 0 && ctx.DoingCount == 0 && ctx.PendingCount == 0 && ctx.HoldCount == 0) && ctx.DoneCount > 0

	// Find next suggested task (highest priority TODO task)
	if ctx.TodoCount > 0 {
		// Collect all TODO tasks
		todoTasks := []storage.Task{}
		for _, task := range allTasks {
			if task.Status == StatusTodo {
				todoTasks = append(todoTasks, task)
			}
		}

		// Sort by priority (critical > high > medium > low)
		if len(todoTasks) > 0 {
			tasks.SortTasksByPriority(todoTasks)
			task := todoTasks[0]
			ctx.NextTaskID = task.ID
			ctx.NextTaskDesc = task.Description
		}
	}

	// Check for unresolved comments
	// This would require integration with threads package
	// For now, we'll leave it as false
	ctx.HasUnresolvedComments = false

	return ctx, nil
}

// DetectContextWithConfig allows custom configuration.
func (d *Detector) DetectContextWithConfig(language string) (*Context, error) {
	ctx, err := d.DetectContext()
	if err != nil {
		return nil, err
	}

	if language != "" {
		ctx.Language = language
	}

	return ctx, nil
}
