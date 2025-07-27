package ai

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reviewtask/internal/config"
	"reviewtask/internal/storage"
	"strings"
)

// TaskDeduplicator handles AI-powered task deduplication
type TaskDeduplicator struct {
	config *config.Config
}

// NewTaskDeduplicator creates a new task deduplicator
func NewTaskDeduplicator(config *config.Config) *TaskDeduplicator {
	return &TaskDeduplicator{
		config: config,
	}
}

// DeduplicationRequest represents a request to check for duplicate tasks
type DeduplicationRequest struct {
	Tasks []TaskSummary `json:"tasks"`
}

// TaskSummary represents a simplified task for deduplication
type TaskSummary struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	CommentID   int64  `json:"comment_id"`
	Priority    string `json:"priority"`
}

// DeduplicationResponse represents the AI's response about task duplicates
type DeduplicationResponse struct {
	UniqueTaskIDs   []string         `json:"unique_task_ids"`
	DuplicateGroups []DuplicateGroup `json:"duplicate_groups"`
	Reasoning       string           `json:"reasoning"`
}

// DuplicateGroup represents a group of duplicate tasks
type DuplicateGroup struct {
	PrimaryTaskID    string   `json:"primary_task_id"`
	DuplicateTaskIDs []string `json:"duplicate_task_ids"`
	Reason           string   `json:"reason"`
}

// DeduplicateTasks uses AI to identify and remove duplicate tasks
func (d *TaskDeduplicator) DeduplicateTasks(tasks []storage.Task) ([]storage.Task, error) {
	if !d.config.AISettings.DeduplicationEnabled {
		return tasks, nil
	}

	if len(tasks) <= 1 {
		return tasks, nil
	}

	// Prepare task summaries for AI
	var taskSummaries []TaskSummary
	taskMap := make(map[string]storage.Task)

	for _, task := range tasks {
		summary := TaskSummary{
			ID:          task.ID,
			Description: task.Description,
			CommentID:   task.SourceCommentID,
			Priority:    task.Priority,
		}
		taskSummaries = append(taskSummaries, summary)
		taskMap[task.ID] = task
	}

	// Call AI to identify duplicates
	response, err := d.identifyDuplicates(taskSummaries)
	if err != nil {
		if d.config.AISettings.DebugMode {
			fmt.Printf("⚠️  AI deduplication failed, keeping all tasks: %v\n", err)
		}
		return tasks, nil
	}

	// Build result based on AI response
	var deduplicatedTasks []storage.Task
	processedIDs := make(map[string]bool)

	// Add all unique tasks
	for _, taskID := range response.UniqueTaskIDs {
		if task, exists := taskMap[taskID]; exists && !processedIDs[taskID] {
			deduplicatedTasks = append(deduplicatedTasks, task)
			processedIDs[taskID] = true
		}
	}

	// Add primary tasks from duplicate groups
	for _, group := range response.DuplicateGroups {
		if task, exists := taskMap[group.PrimaryTaskID]; exists && !processedIDs[group.PrimaryTaskID] {
			deduplicatedTasks = append(deduplicatedTasks, task)
			processedIDs[group.PrimaryTaskID] = true
		}

		// Log duplicate removals
		if d.config.AISettings.DebugMode {
			for _, dupID := range group.DuplicateTaskIDs {
				if _, exists := taskMap[dupID]; exists {
					fmt.Printf("🔄 Removed duplicate task %s: %s\n", dupID, group.Reason)
				}
			}
		}
	}

	if d.config.AISettings.DebugMode {
		fmt.Printf("✨ AI deduplication: %d tasks → %d unique tasks\n", len(tasks), len(deduplicatedTasks))
		if response.Reasoning != "" {
			fmt.Printf("   Reasoning: %s\n", response.Reasoning)
		}
	}

	return deduplicatedTasks, nil
}

// identifyDuplicates calls AI to identify duplicate tasks
func (d *TaskDeduplicator) identifyDuplicates(tasks []TaskSummary) (*DeduplicationResponse, error) {
	prompt := fmt.Sprintf(`Analyze these tasks and identify duplicates based on their semantic meaning.

Tasks to analyze:
%s

Instructions:
1. Identify tasks that are semantically duplicate (same work, different wording)
2. Group duplicates together, selecting the most comprehensive one as primary
3. Consider tasks duplicate if they:
   - Request the same code change
   - Fix the same issue
   - Implement the same feature
   - Address the same problem (even if worded differently)
4. Do NOT consider tasks duplicate if they:
   - Target different files or components
   - Have different scopes (e.g., one is broader)
   - Address different aspects of a problem
   - Come from different review contexts that require separate attention

Respond in JSON format:
{
  "unique_task_ids": ["task_id1", "task_id2"],
  "duplicate_groups": [
    {
      "primary_task_id": "task_id3",
      "duplicate_task_ids": ["task_id4", "task_id5"],
      "reason": "All tasks request the same validation fix"
    }
  ],
  "reasoning": "Brief explanation of deduplication decisions"
}

Ensure every task ID appears exactly once in either unique_task_ids or as a primary/duplicate.`,
		d.formatTasksForPrompt(tasks))

	claudeCmd, err := FindClaudeCommand(d.config.AISettings.ClaudePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Claude: %w", err)
	}

	cmd := exec.Command(claudeCmd,
		"--output-format", "json",
		prompt)

	// Set environment for Claude command
	cmd.Env = append(cmd.Environ(), "TERM=dumb", "NO_COLOR=1")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude command failed: %w\nOutput: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run claude: %w", err)
	}

	// Parse JSON response
	var response DeduplicationResponse
	responseStr := strings.TrimSpace(string(output))

	if err := json.Unmarshal([]byte(responseStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w\nResponse: %s", err, responseStr)
	}

	return &response, nil
}

// formatTasksForPrompt formats tasks for the AI prompt
func (d *TaskDeduplicator) formatTasksForPrompt(tasks []TaskSummary) string {
	data, _ := json.MarshalIndent(tasks, "", "  ")
	return string(data)
}

// DeduplicateWithinComment removes duplicate tasks within a single comment
func (d *TaskDeduplicator) DeduplicateWithinComment(tasks []storage.Task, commentID int64) ([]storage.Task, error) {
	// Filter tasks for this comment
	var commentTasks []storage.Task
	var otherTasks []storage.Task

	for _, task := range tasks {
		if task.SourceCommentID == commentID {
			commentTasks = append(commentTasks, task)
		} else {
			otherTasks = append(otherTasks, task)
		}
	}

	if len(commentTasks) <= 1 {
		return tasks, nil
	}

	// Deduplicate tasks for this comment
	deduplicatedCommentTasks, err := d.DeduplicateTasks(commentTasks)
	if err != nil {
		return tasks, err
	}

	// Combine with other tasks
	result := append(otherTasks, deduplicatedCommentTasks...)
	return result, nil
}
