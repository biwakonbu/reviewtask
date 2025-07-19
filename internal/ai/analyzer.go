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
	var prompt strings.Builder
	
	prompt.WriteString("You are an AI assistant helping to analyze GitHub PR reviews and generate actionable tasks.\n\n")
	
	// User language configuration
	if a.config.AISettings.UserLanguage != "" {
		prompt.WriteString(fmt.Sprintf("IMPORTANT: Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage))
	}
	
	// Add priority guidelines
	prompt.WriteString(a.config.GetPriorityPrompt())
	prompt.WriteString("\n\n")
	
	// Format specification with origin text preservation
	prompt.WriteString("CRITICAL: Return response as JSON array with this EXACT format:\n")
	prompt.WriteString("[\n")
	prompt.WriteString("  {\n")
	prompt.WriteString("    \"description\": \"Actionable task description in specified language\",\n")
	prompt.WriteString("    \"origin_text\": \"Original review comment text (preserve exactly)\",\n")
	prompt.WriteString("    \"priority\": \"critical|high|medium|low\",\n")
	prompt.WriteString("    \"source_review_id\": 12345,\n")
	prompt.WriteString("    \"source_comment_id\": 67890,\n")
	prompt.WriteString("    \"file\": \"path/to/file.go\",\n")
	prompt.WriteString("    \"line\": 42,\n")
	prompt.WriteString("    \"task_index\": 0\n")
	prompt.WriteString("  }\n")
	prompt.WriteString("]\n\n")
	
	prompt.WriteString("Requirements:\n")
	prompt.WriteString("1. PRESERVE original comment text in 'origin_text' field exactly as written\n")
	prompt.WriteString("2. Generate clear, actionable 'description' in the specified user language\n")
	prompt.WriteString("3. SPLIT multiple issues in a single comment into separate tasks\n")
	prompt.WriteString("4. Assign task_index starting from 0 for multiple tasks from same comment\n")
	prompt.WriteString("5. Only create tasks for comments requiring developer action\n")
	prompt.WriteString("6. Consider comment chains - don't create tasks for resolved issues\n\n")
	
	prompt.WriteString("Task Splitting Guidelines:\n")
	prompt.WriteString("- One comment may contain multiple distinct issues or suggestions\n")
	prompt.WriteString("- Each issue should become a separate task with its own priority\n")
	prompt.WriteString("- All tasks from same comment share the same origin_text and source_comment_id\n")
	prompt.WriteString("- Use task_index to distinguish tasks: 0, 1, 2, etc.\n\n")
	
	// Add review data with enhanced metadata
	prompt.WriteString("PR Reviews to analyze:\n\n")
	
	for i, review := range reviews {
		prompt.WriteString(fmt.Sprintf("Review %d (ID: %d):\n", i+1, review.ID))
		prompt.WriteString(fmt.Sprintf("Reviewer: %s\n", review.Reviewer))
		prompt.WriteString(fmt.Sprintf("State: %s\n", review.State))
		
		if review.Body != "" {
			prompt.WriteString(fmt.Sprintf("Review Body: %s\n", review.Body))
		}
		
		if len(review.Comments) > 0 {
			prompt.WriteString("Comments:\n")
			for _, comment := range review.Comments {
				prompt.WriteString(fmt.Sprintf("  Comment ID: %d\n", comment.ID))
				prompt.WriteString(fmt.Sprintf("  File: %s:%d\n", comment.File, comment.Line))
				prompt.WriteString(fmt.Sprintf("  Author: %s\n", comment.Author))
				prompt.WriteString(fmt.Sprintf("  Text: %s\n", comment.Body))
				
				if len(comment.Replies) > 0 {
					prompt.WriteString("  Replies:\n")
					for _, reply := range comment.Replies {
						prompt.WriteString(fmt.Sprintf("    - %s: %s\n", reply.Author, reply.Body))
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
	// Use proper Claude Code one-shot syntax
	cmd := exec.Command("claude", "-p", prompt, "--output-format", "json")
	output, err := cmd.Output()
	
	if err != nil {
		if a.config.AISettings.FallbackEnabled {
			fmt.Printf("  ⚠️  Claude Code unavailable, using fallback tasks\n")
			return a.createFallbackTasks(), nil
		}
		return nil, fmt.Errorf("claude code execution failed: %w", err)
	}
	
	// Parse the JSON response
	var tasks []TaskRequest
	if err := json.Unmarshal(output, &tasks); err != nil {
		if a.config.AISettings.FallbackEnabled {
			fmt.Printf("  ⚠️  Failed to parse Claude response, using fallback tasks\n")
			return a.createFallbackTasks(), nil
		}
		return nil, fmt.Errorf("failed to parse claude response: %w", err)
	}
	
	return tasks, nil
}

// createFallbackTasks creates dummy tasks for PoC when Claude Code is not available
func (a *Analyzer) createFallbackTasks() []TaskRequest {
	return []TaskRequest{
		{
			Description:     "Example task: Review and address performance concern",
			OriginText:      "This looks like it could cause performance issues",
			Priority:        "high",
			SourceReviewID:  0,
			SourceCommentID: 1,
			File:            "example.go",
			Line:            1,
			TaskIndex:       0,
		},
		{
			Description:     "Example task: Fix code style issue",
			OriginText:      "Please fix the formatting and naming conventions",
			Priority:        "low",
			SourceReviewID:  0,
			SourceCommentID: 2,
			File:            "example.go",
			Line:            10,
			TaskIndex:       0,
		},
	}
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