package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gh-review-task/internal/github"
)

const (
	StorageDir = ".pr-review"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

type Manager struct {
	baseDir string
}

type Task struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	Priority        string `json:"priority"`
	SourceReviewID  int64  `json:"source_review_id"`
	File            string `json:"file"`
	Line            int    `json:"line"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	PRNumber        int    `json:"pr_number"` // Added for tracking which PR this task belongs to
}

type TasksFile struct {
	GeneratedAt string `json:"generated_at"`
	Tasks       []Task `json:"tasks"`
}

type ReviewsFile struct {
	Reviews []github.Review `json:"reviews"`
}

func NewManager() *Manager {
	return &Manager{
		baseDir: StorageDir,
	}
}

func (m *Manager) ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (m *Manager) getPRDir(prNumber int) string {
	return filepath.Join(m.baseDir, fmt.Sprintf("PR-%d", prNumber))
}

func (m *Manager) SavePRInfo(prNumber int, info *github.PRInfo) error {
	prDir := m.getPRDir(prNumber)
	if err := m.ensureDir(prDir); err != nil {
		return err
	}

	filePath := filepath.Join(prDir, "info.json")
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (m *Manager) SaveReviews(prNumber int, reviews []github.Review) error {
	prDir := m.getPRDir(prNumber)
	if err := m.ensureDir(prDir); err != nil {
		return err
	}

	reviewsFile := ReviewsFile{Reviews: reviews}
	filePath := filepath.Join(prDir, "reviews.json")
	data, err := json.MarshalIndent(reviewsFile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (m *Manager) SaveTasks(prNumber int, tasks []Task) error {
	prDir := m.getPRDir(prNumber)
	if err := m.ensureDir(prDir); err != nil {
		return err
	}

	// Add PR number to each task and set timestamps
	now := time.Now().Format("2006-01-02T15:04:05Z")
	for i := range tasks {
		tasks[i].PRNumber = prNumber
		if tasks[i].CreatedAt == "" {
			tasks[i].CreatedAt = now
		}
		if tasks[i].UpdatedAt == "" {
			tasks[i].UpdatedAt = now
		}
	}

	tasksFile := TasksFile{
		GeneratedAt: now,
		Tasks:       tasks,
	}

	filePath := filepath.Join(prDir, "tasks.json")
	data, err := json.MarshalIndent(tasksFile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (m *Manager) GetAllTasks() ([]Task, error) {
	var allTasks []Task

	// Check if storage directory exists
	if _, err := os.Stat(m.baseDir); os.IsNotExist(err) {
		return allTasks, nil // Return empty slice if directory doesn't exist
	}

	// Walk through all PR directories
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip config.json and other non-PR files
		if !isPRDir(entry.Name()) {
			continue
		}

		tasksPath := filepath.Join(m.baseDir, entry.Name(), "tasks.json")
		tasks, err := m.loadTasksFromFile(tasksPath)
		if err != nil {
			// Skip if tasks.json doesn't exist or can't be read
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to load tasks from %s: %w", tasksPath, err)
		}

		allTasks = append(allTasks, tasks...)
	}

	return allTasks, nil
}

func (m *Manager) UpdateTaskStatus(taskID, newStatus string) error {
	// Find the task across all PRs
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() || !isPRDir(entry.Name()) {
			continue
		}

		tasksPath := filepath.Join(m.baseDir, entry.Name(), "tasks.json")
		tasksFile, err := m.loadTasksFile(tasksPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		// Check if task exists in this file
		taskFound := false
		for i := range tasksFile.Tasks {
			if tasksFile.Tasks[i].ID == taskID {
				tasksFile.Tasks[i].Status = newStatus
				tasksFile.Tasks[i].UpdatedAt = time.Now().Format("2006-01-02T15:04:05Z")
				taskFound = true
				break
			}
		}

		if taskFound {
			// Save updated tasks file
			data, err := json.MarshalIndent(tasksFile, "", "  ")
			if err != nil {
				return err
			}
			return os.WriteFile(tasksPath, data, 0644)
		}
	}

	return ErrTaskNotFound
}

func (m *Manager) loadTasksFromFile(filePath string) ([]Task, error) {
	tasksFile, err := m.loadTasksFile(filePath)
	if err != nil {
		return nil, err
	}
	return tasksFile.Tasks, nil
}

func (m *Manager) loadTasksFile(filePath string) (*TasksFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var tasksFile TasksFile
	if err := json.Unmarshal(data, &tasksFile); err != nil {
		return nil, err
	}

	return &tasksFile, nil
}

// isPRDir checks if directory name matches PR-{number} pattern
func isPRDir(name string) bool {
	return len(name) > 3 && name[:3] == "PR-"
}