package tasks

import (
	"fmt"
	"sort"

	"reviewtask/internal/storage"
)

// TaskStats holds statistics about tasks
type TaskStats struct {
	StatusCounts   map[string]int
	PriorityCounts map[string]int
	PRCounts       map[int]int
}

// FilterTasksByStatus filters tasks by their status
func FilterTasksByStatus(tasks []storage.Task, status string) []storage.Task {
	var filtered []storage.Task
	for _, task := range tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// SortTasksByPriority sorts tasks by priority (critical > high > medium > low)
func SortTasksByPriority(tasks []storage.Task) {
	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}
	sort.Slice(tasks, func(i, j int) bool {
		return priorityOrder[tasks[i].Priority] < priorityOrder[tasks[j].Priority]
	})
}

// GenerateTaskID generates a task ID in TSK-XXX format
func GenerateTaskID(task storage.Task) string {
	return fmt.Sprintf("TSK-%03d", task.PRNumber)
}

// CalculateTaskStats calculates statistics for a slice of tasks
func CalculateTaskStats(tasks []storage.Task) TaskStats {
	stats := TaskStats{
		StatusCounts:   make(map[string]int),
		PriorityCounts: make(map[string]int),
		PRCounts:       make(map[int]int),
	}
	for _, task := range tasks {
		// Normalize "cancelled" to "cancel" for backward compatibility
		status := task.Status
		if status == "cancelled" {
			status = "cancel"
		}
		stats.StatusCounts[status]++
		stats.PriorityCounts[task.Priority]++
		stats.PRCounts[task.PRNumber]++
	}
	return stats
}
