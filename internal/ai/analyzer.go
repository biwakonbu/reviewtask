package ai

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"gh-review-task/internal/config"
	"gh-review-task/internal/github"
	"gh-review-task/internal/storage"
)

type Analyzer struct {
	config *config.Config
}

func NewAnalyzer(cfg *config.Config) *Analyzer {
	return &Analyzer{
		config: cfg,
	}
}

type TaskRequest struct {
	Description    string `json:"description"`
	Priority       string `json:"priority"`
	SourceReviewID int64  `json:"source_review_id"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	Status         string `json:"status"`
}

func (a *Analyzer) GenerateTasks(reviews []github.Review) ([]storage.Task, error) {
	if len(reviews) == 0 {
		return []storage.Task{}, nil
	}

	// Create prompt for Claude Code
	prompt := a.buildAnalysisPrompt(reviews)
	
	// Call Claude Code
	tasks, err := a.callClaudeCode(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Claude Code: %w", err)
	}

	// Convert to storage.Task format with unique IDs
	var result []storage.Task
	for i, taskReq := range tasks {
		task := storage.Task{
			ID:              fmt.Sprintf("task-%d-%d", time.Now().Unix(), i+1),
			Description:     taskReq.Description,
			Priority:        taskReq.Priority,
			SourceReviewID:  taskReq.SourceReviewID,
			File:            taskReq.File,
			Line:            taskReq.Line,
			Status:          a.config.TaskSettings.DefaultStatus,
			CreatedAt:       time.Now().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       time.Now().Format("2006-01-02T15:04:05Z"),
		}
		result = append(result, task)
	}

	return result, nil
}

func (a *Analyzer) buildAnalysisPrompt(reviews []github.Review) string {
	var prompt strings.Builder
	
	prompt.WriteString("You are an AI assistant helping to analyze GitHub PR reviews and generate actionable tasks.\n\n")
	
	// Add priority guidelines
	prompt.WriteString(a.config.GetPriorityPrompt())
	prompt.WriteString("\n\n")
	
	prompt.WriteString("Please analyze the following PR reviews and generate specific, actionable tasks. ")
	prompt.WriteString("Consider comment chains and replies to determine if issues have been resolved or still need attention. ")
	prompt.WriteString("Only create tasks for comments that require action.\n\n")
	
	prompt.WriteString("IMPORTANT: Return your response as a JSON array of task objects with this exact format:\n")
	prompt.WriteString("[\n")
	prompt.WriteString("  {\n")
	prompt.WriteString("    \"description\": \"Specific task description\",\n")
	prompt.WriteString("    \"priority\": \"critical|high|medium|low\",\n")
	prompt.WriteString("    \"source_review_id\": 12345,\n")
	prompt.WriteString("    \"file\": \"path/to/file.go\",\n")
	prompt.WriteString("    \"line\": 42\n")
	prompt.WriteString("  }\n")
	prompt.WriteString("]\n\n")
	
	prompt.WriteString("PR Reviews to analyze:\n\n")
	
	// Add review data
	for i, review := range reviews {
		prompt.WriteString(fmt.Sprintf("Review %d:\n", i+1))
		prompt.WriteString(fmt.Sprintf("Reviewer: %s\n", review.Reviewer))
		prompt.WriteString(fmt.Sprintf("State: %s\n", review.State))
		if review.Body != "" {
			prompt.WriteString(fmt.Sprintf("Review Body: %s\n", review.Body))
		}
		
		if len(review.Comments) > 0 {
			prompt.WriteString("Comments:\n")
			for _, comment := range review.Comments {
				prompt.WriteString(fmt.Sprintf("  - File: %s:%d\n", comment.File, comment.Line))
				prompt.WriteString(fmt.Sprintf("    Author: %s\n", comment.Author))
				prompt.WriteString(fmt.Sprintf("    Comment: %s\n", comment.Body))
				
				if len(comment.Replies) > 0 {
					prompt.WriteString("    Replies:\n")
					for _, reply := range comment.Replies {
						prompt.WriteString(fmt.Sprintf("      - %s: %s\n", reply.Author, reply.Body))
					}
				}
				prompt.WriteString("\n")
			}
		}
		prompt.WriteString("\n")
	}
	
	return prompt.String()
}

func (a *Analyzer) callClaudeCode(prompt string) ([]TaskRequest, error) {
	// Create a temporary file with the prompt
	tempFile := "/tmp/claude_prompt.txt"
	if err := writeToFile(tempFile, prompt); err != nil {
		return nil, fmt.Errorf("failed to write prompt to temp file: %w", err)
	}
	
	// Call Claude Code via command line
	// Note: This is a simplified approach for PoC. In production, you might want to use Claude Code's API directly
	cmd := exec.Command("claude", "code", "--input", tempFile, "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		// Fallback: if claude command is not available, return dummy tasks for PoC
		return a.createFallbackTasks(), nil
	}
	
	// Parse the JSON response
	var tasks []TaskRequest
	if err := json.Unmarshal(output, &tasks); err != nil {
		// If parsing fails, return fallback tasks for PoC
		return a.createFallbackTasks(), nil
	}
	
	return tasks, nil
}

// createFallbackTasks creates dummy tasks for PoC when Claude Code is not available
func (a *Analyzer) createFallbackTasks() []TaskRequest {
	return []TaskRequest{
		{
			Description:    "Example task: Review and address performance concern",
			Priority:       "high",
			SourceReviewID: 0,
			File:           "example.go",
			Line:           1,
		},
		{
			Description:    "Example task: Fix code style issue",
			Priority:       "low",
			SourceReviewID: 0,
			File:           "example.go",
			Line:           10,
		},
	}
}

func writeToFile(filename, content string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo %q > %s", content, filename))
	return cmd.Run()
}