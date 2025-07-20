package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	ID              string `json:"id"`          // Format: "comment-{commentID}-task-{index}"
	Description     string `json:"description"` // AI-generated task description (user language)
	OriginText      string `json:"origin_text"` // Original review comment text
	Priority        string `json:"priority"`
	SourceReviewID  int64  `json:"source_review_id"`
	SourceCommentID int64  `json:"source_comment_id"` // Required: comment this task belongs to
	TaskIndex       int    `json:"task_index"`        // Index within the comment (for multiple tasks per comment)
	File            string `json:"file"`
	Line            int    `json:"line"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	PRNumber        int    `json:"pr_number"`
}

type TasksFile struct {
	GeneratedAt string `json:"generated_at"`
	Tasks       []Task `json:"tasks"`
}

type ReviewsFile struct {
	Reviews []github.Review `json:"reviews"`
}

type CommentStats struct {
	CommentID       int64  `json:"comment_id"`
	TotalTasks      int    `json:"total_tasks"`
	CompletedTasks  int    `json:"completed_tasks"`
	PendingTasks    int    `json:"pending_tasks"`
	InProgressTasks int    `json:"in_progress_tasks"`
	CancelledTasks  int    `json:"cancelled_tasks"`
	File            string `json:"file"`
	Line            int    `json:"line"`
	Author          string `json:"author"`
	OriginText      string `json:"origin_text"`
}

type TaskStatistics struct {
	PRNumber      int            `json:"pr_number"`
	BranchName    string         `json:"branch_name,omitempty"` // Only set for branch-specific stats
	GeneratedAt   string         `json:"generated_at"`
	TotalComments int            `json:"total_comments"`
	TotalTasks    int            `json:"total_tasks"`
	CommentStats  []CommentStats `json:"comment_stats"`
	StatusSummary StatusSummary  `json:"status_summary"`
}

type StatusSummary struct {
	Todo      int `json:"todo"`
	Doing     int `json:"doing"`
	Done      int `json:"done"`
	Pending   int `json:"pending"`
	Cancelled int `json:"cancelled"`
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

func (m *Manager) GetTasksByPR(prNumber int) ([]Task, error) {
	tasksPath := filepath.Join(m.getPRDir(prNumber), "tasks.json")
	return m.loadTasksFromFile(tasksPath)
}

func (m *Manager) GetTasksByComment(prNumber int, commentID int64) ([]Task, error) {
	allTasks, err := m.GetTasksByPR(prNumber)
	if err != nil {
		return nil, err
	}

	var commentTasks []Task
	for _, task := range allTasks {
		if task.SourceCommentID == commentID {
			commentTasks = append(commentTasks, task)
		}
	}

	return commentTasks, nil
}

func (m *Manager) UpdateTaskStatusByCommentAndIndex(prNumber int, commentID int64, taskIndex int, newStatus string) error {
	taskID := fmt.Sprintf("comment-%d-task-%d", commentID, taskIndex)
	return m.UpdateTaskStatus(taskID, newStatus)
}

// MergeTasks combines new tasks with existing ones, preserving existing task statuses
func (m *Manager) MergeTasks(prNumber int, newTasks []Task) error {
	existingTasks, err := m.GetTasksByPR(prNumber)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing tasks: %w", err)
	}

	// Create map of existing tasks by source comment ID for quick lookup
	existingByComment := make(map[int64][]Task)
	for _, task := range existingTasks {
		existingByComment[task.SourceCommentID] = append(existingByComment[task.SourceCommentID], task)
	}

	var mergedTasks []Task
	newTasksByComment := make(map[int64][]Task)

	// Group new tasks by comment ID
	for _, task := range newTasks {
		newTasksByComment[task.SourceCommentID] = append(newTasksByComment[task.SourceCommentID], task)
	}

	// Process each comment ID
	allCommentIDs := make(map[int64]bool)
	for commentID := range existingByComment {
		allCommentIDs[commentID] = true
	}
	for commentID := range newTasksByComment {
		allCommentIDs[commentID] = true
	}

	for commentID := range allCommentIDs {
		existingForComment := existingByComment[commentID]
		newForComment := newTasksByComment[commentID]

		mergedForComment := m.mergeTasksForComment(commentID, existingForComment, newForComment)
		mergedTasks = append(mergedTasks, mergedForComment...)
	}

	return m.SaveTasks(prNumber, mergedTasks)
}

// mergeTasksForComment handles task merging for a specific comment
func (m *Manager) mergeTasksForComment(commentID int64, existing, new []Task) []Task {
	var result []Task

	if len(existing) == 0 {
		// No existing tasks, use all new tasks
		return new
	}

	if len(new) == 0 {
		// No new tasks, mark existing tasks as cancelled if they're not already done
		for _, task := range existing {
			if task.Status != "done" && task.Status != "cancelled" {
				task.Status = "cancelled"
				task.UpdatedAt = time.Now().Format("2006-01-02T15:04:05Z")
			}
			result = append(result, task)
		}
		return result
	}

	// Compare origin text to detect content changes
	existingOriginText := ""
	if len(existing) > 0 {
		existingOriginText = existing[0].OriginText
	}

	newOriginText := ""
	if len(new) > 0 {
		newOriginText = new[0].OriginText
	}

	// If origin text changed significantly, cancel old tasks and add new ones
	if m.hasSignificantTextChange(existingOriginText, newOriginText) {
		// Cancel existing tasks that aren't done
		for _, task := range existing {
			if task.Status != "done" && task.Status != "cancelled" {
				task.Status = "cancelled"
				task.UpdatedAt = time.Now().Format("2006-01-02T15:04:05Z")
			}
			result = append(result, task)
		}

		// Add new tasks
		result = append(result, new...)
		return result
	}

	// Content is similar, preserve existing task statuses
	// Preserve existing tasks and their statuses
	for _, existingTask := range existing {
		result = append(result, existingTask)
	}

	// Add any genuinely new tasks (beyond existing count)
	if len(new) > len(existing) {
		for i := len(existing); i < len(new); i++ {
			result = append(result, new[i])
		}
	}

	return result
}

// hasSignificantTextChange determines if comment content changed significantly
func (m *Manager) hasSignificantTextChange(old, new string) bool {
	// Simple comparison - consider it significant if strings are notably different
	// This is a basic implementation; could be enhanced with fuzzy matching
	if old == "" || new == "" {
		return old != new
	}

	// Remove common markdown and whitespace for comparison
	cleanOld := strings.TrimSpace(strings.ReplaceAll(old, "\n", " "))
	cleanNew := strings.TrimSpace(strings.ReplaceAll(new, "\n", " "))

	// If texts are very different in length, consider it significant
	if len(cleanOld) > 0 && len(cleanNew) > 0 {
		ratio := float64(len(cleanNew)) / float64(len(cleanOld))
		if ratio < 0.5 || ratio > 2.0 {
			return true
		}
	}

	// If they're completely different, it's significant
	return cleanOld != cleanNew
}

// GetCurrentBranch returns the current git branch name
func (m *Manager) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("no current branch found")
	}

	return branch, nil
}

// GetPRsForBranch returns all PR numbers that match the given branch name
func (m *Manager) GetPRsForBranch(branchName string) ([]int, error) {
	var prNumbers []int

	// Read all PR directories
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "PR-") {
			continue
		}

		// Extract PR number from directory name
		var prNumber int
		if _, err := fmt.Sscanf(entry.Name(), "PR-%d", &prNumber); err != nil {
			continue
		}

		// Check if this PR's branch matches
		infoPath := filepath.Join(m.baseDir, entry.Name(), "info.json")
		data, err := os.ReadFile(infoPath)
		if err != nil {
			continue
		}

		var prInfo github.PRInfo
		if err := json.Unmarshal(data, &prInfo); err != nil {
			continue
		}

		if prInfo.Branch == branchName {
			prNumbers = append(prNumbers, prNumber)
		}
	}

	return prNumbers, nil
}

// GetAllPRNumbers returns all PR numbers that have data stored
func (m *Manager) GetAllPRNumbers() ([]int, error) {
	var prNumbers []int

	// Read all PR directories
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "PR-") {
			continue
		}

		// Extract PR number from directory name
		var prNumber int
		if _, err := fmt.Sscanf(entry.Name(), "PR-%d", &prNumber); err != nil {
			continue
		}

		prNumbers = append(prNumbers, prNumber)
	}

	return prNumbers, nil
}
