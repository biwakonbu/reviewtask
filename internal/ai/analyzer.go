package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gh-review-task/internal/config"
	"gh-review-task/internal/github"
	"gh-review-task/internal/storage"
)

type Analyzer struct {
	config             *config.Config
	validationFeedback []ValidationIssue
}

func NewAnalyzer(cfg *config.Config) *Analyzer {
	return &Analyzer{
		config: cfg,
	}
}

type TaskRequest struct {
	Description     string `json:"description"`        // AI-generated task description (user language)
	OriginText      string `json:"origin_text"`        // Original review comment text
	Priority        string `json:"priority"`
	SourceReviewID  int64  `json:"source_review_id"`
	SourceCommentID int64  `json:"source_comment_id"`  // Required: specific comment ID
	File            string `json:"file"`
	Line            int    `json:"line"`
	Status          string `json:"status"`
	TaskIndex       int    `json:"task_index"`         // New: index within comment (0, 1, 2...)
}

type ValidationResult struct {
	IsValid bool                `json:"is_valid"`
	Score   float64             `json:"score"`        // 0.0-1.0 quality score
	Issues  []ValidationIssue   `json:"issues"`
	Tasks   []TaskRequest       `json:"tasks"`
}

type ValidationIssue struct {
	Type        string `json:"type"`        // "format", "content", "missing", "incorrect"
	TaskIndex   int    `json:"task_index"`  // -1 for general issues
	Field       string `json:"field"`       // specific field with issue
	Description string `json:"description"` // human-readable issue description
	Severity    string `json:"severity"`    // "critical", "major", "minor"
}

type TaskValidator struct {
	config     *config.Config
	maxRetries int
}

func NewTaskValidator(cfg *config.Config) *TaskValidator {
	return &TaskValidator{
		config:     cfg,
		maxRetries: cfg.AISettings.MaxRetries,
	}
}

func (a *Analyzer) GenerateTasks(reviews []github.Review) ([]storage.Task, error) {
	if len(reviews) == 0 {
		return []storage.Task{}, nil
	}

	if a.config.AISettings.ValidationEnabled {
		return a.GenerateTasksWithValidation(reviews)
	} else {
		return a.generateTasksLegacy(reviews)
	}
}

func (a *Analyzer) GenerateTasksWithValidation(reviews []github.Review) ([]storage.Task, error) {
	validator := NewTaskValidator(a.config)
	var bestResult *ValidationResult
	var bestTasks []TaskRequest
	maxScore := 0.0
	
	for attempt := 1; attempt <= validator.maxRetries; attempt++ {
		fmt.Printf("🔄 Task generation attempt %d/%d...\n", attempt, validator.maxRetries)
		
		// Generate tasks
		tasks, err := a.callClaudeCodeWithRetry(reviews, attempt)
		if err != nil {
			fmt.Printf("  ❌ Generation failed: %v\n", err)
			continue
		}
		
		// Stage 1: Format validation
		formatResult, err := validator.validateFormat(tasks)
		if err != nil {
			fmt.Printf("  ❌ Format validation failed: %v\n", err)
			continue
		}
		
		if !formatResult.IsValid {
			fmt.Printf("  ⚠️  Format issues found (score: %.2f)\n", formatResult.Score)
			if formatResult.Score > maxScore {
				bestResult = formatResult
				bestTasks = formatResult.Tasks
				maxScore = formatResult.Score
			}
			continue
		}
		
		// Stage 2: Content validation
		contentResult, err := validator.validateContent(formatResult.Tasks, reviews)
		if err != nil {
			fmt.Printf("  ❌ Content validation failed: %v\n", err)
			continue
		}
		
		fmt.Printf("  📊 Validation score: %.2f\n", contentResult.Score)
		
		// Track best result
		if contentResult.Score > maxScore {
			bestResult = contentResult
			bestTasks = formatResult.Tasks
			maxScore = contentResult.Score
		}
		
		// Check if validation passed
		if contentResult.IsValid && contentResult.Score >= a.config.AISettings.QualityThreshold {
			fmt.Printf("  ✅ Validation passed!\n")
			return a.convertToStorageTasks(formatResult.Tasks), nil
		}
		
		// If not valid, add validation feedback for next iteration
		if attempt < validator.maxRetries {
			fmt.Printf("  🔧 Preparing improved prompt for next attempt...\n")
			a.addValidationFeedback(contentResult.Issues)
		}
	}
	
	// Use best result if no perfect validation achieved
	if bestResult != nil && len(bestTasks) > 0 {
		fmt.Printf("⚠️  Using best result (score: %.2f) after %d attempts\n", maxScore, validator.maxRetries)
		return a.convertToStorageTasks(bestTasks), nil
	}
	
	return nil, fmt.Errorf("failed to generate valid tasks after %d attempts", validator.maxRetries)
}

func (a *Analyzer) generateTasksLegacy(reviews []github.Review) ([]storage.Task, error) {
	// Legacy implementation without validation
	prompt := a.buildAnalysisPrompt(reviews)
	tasks, err := a.callClaudeCode(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Claude Code: %w", err)
	}
	
	return a.convertToStorageTasks(tasks), nil
}

func (a *Analyzer) buildAnalysisPrompt(reviews []github.Review) string {
	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" {
		languageInstruction = fmt.Sprintf("IMPORTANT: Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage)
	}
	
	priorityPrompt := a.config.GetPriorityPrompt()
	
	// Build review data
	var reviewsData strings.Builder
	reviewsData.WriteString("PR Reviews to analyze:\n\n")
	
	for i, review := range reviews {
		reviewsData.WriteString(fmt.Sprintf("Review %d (ID: %d):\n", i+1, review.ID))
		reviewsData.WriteString(fmt.Sprintf("Reviewer: %s\n", review.Reviewer))
		reviewsData.WriteString(fmt.Sprintf("State: %s\n", review.State))
		
		if review.Body != "" {
			reviewsData.WriteString(fmt.Sprintf("Review Body: %s\n", review.Body))
		}
		
		if len(review.Comments) > 0 {
			reviewsData.WriteString("Comments:\n")
			for _, comment := range review.Comments {
				reviewsData.WriteString(fmt.Sprintf("  Comment ID: %d\n", comment.ID))
				reviewsData.WriteString(fmt.Sprintf("  File: %s:%d\n", comment.File, comment.Line))
				reviewsData.WriteString(fmt.Sprintf("  Author: %s\n", comment.Author))
				reviewsData.WriteString(fmt.Sprintf("  Text: %s\n", comment.Body))
				
				if len(comment.Replies) > 0 {
					reviewsData.WriteString("  Replies:\n")
					for _, reply := range comment.Replies {
						reviewsData.WriteString(fmt.Sprintf("    - %s: %s\n", reply.Author, reply.Body))
					}
				}
				reviewsData.WriteString("\n")
			}
		}
		reviewsData.WriteString("\n")
	}
	
	prompt := fmt.Sprintf(`You are an AI assistant helping to analyze GitHub PR reviews and generate actionable tasks.

%s
%s

CRITICAL: Return response as JSON array with this EXACT format:
[
  {
    "description": "Actionable task description in specified language",
    "origin_text": "Original review comment text (preserve exactly)",
    "priority": "critical|high|medium|low",
    "source_review_id": 12345,
    "source_comment_id": 67890,
    "file": "path/to/file.go",
    "line": 42,
    "task_index": 0
  }
]

Requirements:
1. PRESERVE original comment text in 'origin_text' field exactly as written
2. Generate clear, actionable 'description' in the specified user language
3. SPLIT multiple issues in a single comment into separate tasks
4. Assign task_index starting from 0 for multiple tasks from same comment
5. Only create tasks for comments requiring developer action
6. Consider comment chains - don't create tasks for resolved issues

Task Splitting Guidelines:
- One comment may contain multiple distinct issues or suggestions
- Each issue should become a separate task with its own priority
- All tasks from same comment share the same origin_text and source_comment_id
- Use task_index to distinguish tasks: 0, 1, 2, etc.

%s`, languageInstruction, priorityPrompt, reviewsData.String())
	
	return prompt
}

func (a *Analyzer) callClaudeCode(prompt string) ([]TaskRequest, error) {
	claudePath, err := a.findClaudeCommand()
	if err != nil {
		return nil, fmt.Errorf("claude command not found: %w", err)
	}
	
	// Use Claude Code CLI with stdin to avoid command line length limits
	cmd := exec.Command(claudePath, "--output-format", "json")
	cmd.Stdin = strings.NewReader(prompt)
	// Ensure the command inherits the current environment including PATH
	cmd.Env = os.Environ()
	
	// Debug information if enabled
	if a.config.AISettings.DebugMode {
		fmt.Printf("  🐛 Using Claude at: %s\n", claudePath)
		fmt.Printf("  🐛 PATH: %s\n", os.Getenv("PATH"))
		fmt.Printf("  🐛 Prompt size: %d characters\n", len(prompt))
	}
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("claude code execution failed: %w", err)
	}
	
	// Parse Claude Code CLI response wrapper
	var claudeResponse struct {
		Type     string `json:"type"`
		Subtype  string `json:"subtype"`
		IsError  bool   `json:"is_error"`
		Result   string `json:"result"`
	}
	
	if err := json.Unmarshal(output, &claudeResponse); err != nil {
		return nil, fmt.Errorf("failed to parse claude wrapper response: %w", err)
	}
	
	if claudeResponse.IsError {
		return nil, fmt.Errorf("claude returned error: %s", claudeResponse.Result)
	}
	
	// Extract JSON from result (may be wrapped in markdown code block or text)
	result := claudeResponse.Result
	result = strings.TrimSpace(result)
	
	// Find JSON array in the response
	jsonStart := strings.Index(result, "[")
	jsonEnd := strings.LastIndex(result, "]")
	
	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		return nil, fmt.Errorf("no valid JSON array found in Claude response")
	}
	
	result = result[jsonStart : jsonEnd+1]
	
	// Parse the actual task array
	var tasks []TaskRequest
	if err := json.Unmarshal([]byte(result), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse task array from result: %w\nResult was: %s", err, result)
	}
	
	return tasks, nil
}


func (a *Analyzer) convertToStorageTasks(tasks []TaskRequest) []storage.Task {
	var result []storage.Task
	now := time.Now().Format("2006-01-02T15:04:05Z")
	
	for _, task := range tasks {
		storageTask := storage.Task{
			ID:               fmt.Sprintf("comment-%d-task-%d", task.SourceCommentID, task.TaskIndex),
			Description:      task.Description,
			OriginText:       task.OriginText,
			Priority:         task.Priority,
			SourceReviewID:   task.SourceReviewID,
			SourceCommentID:  task.SourceCommentID,
			TaskIndex:        task.TaskIndex,
			File:             task.File,
			Line:             task.Line,
			Status:           a.config.TaskSettings.DefaultStatus,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		result = append(result, storageTask)
	}
	
	return result
}

func (a *Analyzer) callClaudeCodeWithRetry(reviews []github.Review, attempt int) ([]TaskRequest, error) {
	var prompt string
	if attempt == 1 {
		prompt = a.buildAnalysisPrompt(reviews)
	} else {
		prompt = a.buildAnalysisPromptWithFeedback(reviews)
	}
	
	return a.callClaudeCode(prompt)
}

func (a *Analyzer) buildAnalysisPromptWithFeedback(reviews []github.Review) string {
	basePrompt := a.buildAnalysisPrompt(reviews)
	
	// Add validation feedback if available
	if len(a.validationFeedback) > 0 {
		var feedback strings.Builder
		feedback.WriteString("\n\nIMPROVEMENT FEEDBACK from previous attempt:\n")
		feedback.WriteString("Please address these issues in your task generation:\n\n")
		
		for i, issue := range a.validationFeedback {
			feedback.WriteString(fmt.Sprintf("%d. %s (Severity: %s)\n", i+1, issue.Description, issue.Severity))
		}
		
		feedback.WriteString("\nEnsure your response addresses all these concerns.\n")
		basePrompt += feedback.String()
	}
	
	return basePrompt
}

func (a *Analyzer) addValidationFeedback(issues []ValidationIssue) {
	a.validationFeedback = issues
}

// findClaudeCommand searches for Claude CLI in order of preference:
// 1. Custom path from config (claude_path)
// 2. Environment variable CLAUDE_PATH
// 3. PATH environment variable (exec.LookPath)
// 4. Common installation locations
func (a *Analyzer) findClaudeCommand() (string, error) {
	// 1. Check custom path in config
	if a.config.AISettings.ClaudePath != "" {
		if _, err := os.Stat(a.config.AISettings.ClaudePath); err == nil {
			return a.config.AISettings.ClaudePath, nil
		}
		return "", fmt.Errorf("custom claude path not found: %s", a.config.AISettings.ClaudePath)
	}
	
	// 2. Check environment variable
	if envPath := os.Getenv("CLAUDE_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("CLAUDE_PATH environment variable points to non-existent file: %s", envPath)
	}
	
	// 3. Check PATH
	if claudePath, err := exec.LookPath("claude"); err == nil {
		return claudePath, nil
	}
	
	// 4. Check common installation locations
	homeDir := os.Getenv("HOME")
	commonPaths := []string{
		filepath.Join(homeDir, ".claude/local/claude"),           // Local installation
		filepath.Join(homeDir, ".local/bin/claude"),             // User local bin
		filepath.Join(homeDir, ".npm-global/bin/claude"),        // npm global with custom prefix
		"/usr/local/bin/claude",                                 // System-wide installation
		"/opt/claude/bin/claude",                                // Alternative system location
	}
	
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	return "", fmt.Errorf("claude command not found in any search location")
}