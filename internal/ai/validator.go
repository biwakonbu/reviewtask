package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reviewtask/internal/github"
)

// Stage 1: Format Validation (Mechanical)
func (tv *TaskValidator) validateFormat(tasks []TaskRequest) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid: false,
		Score:   0.0,
		Issues:  []ValidationIssue{},
	}

	// Validate each task structure
	validTasks := []TaskRequest{}
	for i, task := range tasks {
		taskIssues := tv.validateTaskFields(task, i)
		result.Issues = append(result.Issues, taskIssues...)

		// Only include tasks with no critical issues
		if !tv.hasCriticalIssues(taskIssues) {
			validTasks = append(validTasks, task)
		}
	}

	result.Tasks = validTasks
	result.Score = tv.calculateFormatScore(result.Issues, len(tasks))
	result.IsValid = len(result.Issues) == 0 || !tv.hasCriticalIssues(result.Issues)

	return result, nil
}

func (tv *TaskValidator) validateTaskFields(task TaskRequest, index int) []ValidationIssue {
	var issues []ValidationIssue

	// Required field validation
	if task.Description == "" {
		issues = append(issues, ValidationIssue{
			Type:        "missing",
			TaskIndex:   index,
			Field:       "description",
			Description: "Task description is empty",
			Severity:    "critical",
		})
	}

	if task.OriginText == "" {
		issues = append(issues, ValidationIssue{
			Type:        "missing",
			TaskIndex:   index,
			Field:       "origin_text",
			Description: "Origin text is missing",
			Severity:    "critical",
		})
	}

	if task.SourceCommentID == 0 {
		issues = append(issues, ValidationIssue{
			Type:        "missing",
			TaskIndex:   index,
			Field:       "source_comment_id",
			Description: "Source comment ID is missing",
			Severity:    "critical",
		})
	}

	// Priority validation
	if !tv.isValidPriority(task.Priority) {
		issues = append(issues, ValidationIssue{
			Type:        "incorrect",
			TaskIndex:   index,
			Field:       "priority",
			Description: fmt.Sprintf("Invalid priority '%s', must be critical|high|medium|low", task.Priority),
			Severity:    "major",
		})
	}

	// Task index validation
	if task.TaskIndex < 0 {
		issues = append(issues, ValidationIssue{
			Type:        "incorrect",
			TaskIndex:   index,
			Field:       "task_index",
			Description: "Task index must be >= 0",
			Severity:    "major",
		})
	}

	return issues
}

// Stage 2: Content Validation (AI-Powered)
func (tv *TaskValidator) validateContent(tasks []TaskRequest, originalReviews []github.Review) (*ValidationResult, error) {
	if len(tasks) == 0 {
		return &ValidationResult{
			IsValid: false,
			Score:   0.0,
			Issues: []ValidationIssue{{
				Type:        "content",
				TaskIndex:   -1,
				Description: "No tasks generated",
				Severity:    "critical",
			}},
		}, nil
	}

	// Create validation prompt
	prompt := tv.buildValidationPrompt(tasks, originalReviews)

	// Call Claude Code for content validation
	validationResponse, err := tv.callClaudeValidation(prompt)
	if err != nil {
		return nil, fmt.Errorf("validation call failed: %w", err)
	}

	return validationResponse, nil
}

func (tv *TaskValidator) buildValidationPrompt(tasks []TaskRequest, reviews []github.Review) string {
	var userLanguage string
	if tv.config.AISettings.UserLanguage != "" {
		userLanguage = fmt.Sprintf("User's preferred language: %s\n\n", tv.config.AISettings.UserLanguage)
	}

	// Build original reviews data
	var reviewsData strings.Builder
	reviewsData.WriteString("ORIGINAL REVIEW COMMENTS:\n")
	for i, review := range reviews {
		reviewsData.WriteString(fmt.Sprintf("Review %d (ID: %d):\n", i+1, review.ID))
		if len(review.Comments) > 0 {
			for _, comment := range review.Comments {
				reviewsData.WriteString(fmt.Sprintf("  Comment ID %d: %s\n", comment.ID, comment.Body))
			}
		}
		reviewsData.WriteString("\n")
	}

	// Build generated tasks data
	var tasksData strings.Builder
	tasksData.WriteString("GENERATED TASKS TO VALIDATE:\n")
	for i, task := range tasks {
		tasksData.WriteString(fmt.Sprintf("Task %d:\n", i))
		tasksData.WriteString(fmt.Sprintf("  Description: %s\n", task.Description))
		tasksData.WriteString(fmt.Sprintf("  Origin Text: %s\n", task.OriginText))
		tasksData.WriteString(fmt.Sprintf("  Priority: %s\n", task.Priority))
		tasksData.WriteString(fmt.Sprintf("  Comment ID: %d\n", task.SourceCommentID))
		tasksData.WriteString(fmt.Sprintf("  Task Index: %d\n", task.TaskIndex))
		tasksData.WriteString("\n")
	}

	prompt := fmt.Sprintf(`You are a code review expert validating AI-generated tasks from PR review comments.

%s
VALIDATION CRITERIA:
1. Each task should be actionable and specific
2. Task descriptions should be in the user's preferred language
3. Tasks should accurately reflect the original comment intent
4. No duplicate tasks should exist
5. All genuine issues from comments should be captured
6. Task priorities should match issue severity

RESPONSE FORMAT:
Return JSON in this EXACT format:
{
  "validation": true|false,
  "score": 0.85,
  "issues": [
    {
      "type": "content|missing|incorrect|duplicate",
      "task_index": 0,
      "description": "Specific issue description",
      "severity": "critical|major|minor",
      "suggestion": "How to fix this issue"
    }
  ]
}

%s
%s`, userLanguage, reviewsData.String(), tasksData.String())

	return prompt
}

func (tv *TaskValidator) callClaudeValidation(prompt string) (*ValidationResult, error) {
	// Use the injected AI provider instead of calling Claude directly
	if tv.aiProvider == nil {
		return nil, fmt.Errorf("AI provider not initialized for validation")
	}

	// Check for very large prompts that might exceed system limits
	const maxPromptSize = 32 * 1024 // 32KB limit for safety
	if len(prompt) > maxPromptSize {
		return nil, fmt.Errorf("prompt size (%d bytes) exceeds maximum limit (%d bytes). Please shorten or chunk the prompt content", len(prompt), maxPromptSize)
	}

	// Call AI provider (supports both Claude and Cursor)
	ctx := context.Background()
	output, err := tv.aiProvider.Execute(ctx, prompt, "json")
	if err != nil {
		return nil, fmt.Errorf("AI validation execution failed: %w", err)
	}

	// Extract JSON from result (may be wrapped in markdown code block)
	result := output
	result = strings.TrimSpace(result)
	if strings.HasPrefix(result, "```json") && strings.HasSuffix(result, "```") {
		// Remove markdown code block wrapper
		lines := strings.Split(result, "\n")
		if len(lines) >= 3 {
			result = strings.Join(lines[1:len(lines)-1], "\n")
		}
	} else if strings.HasPrefix(result, "```") && strings.HasSuffix(result, "```") {
		// Remove generic code block wrapper
		lines := strings.Split(result, "\n")
		if len(lines) >= 3 {
			result = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Parse validation response
	var response struct {
		Validation bool    `json:"validation"`
		Score      float64 `json:"score"`
		Issues     []struct {
			Type        string `json:"type"`
			TaskIndex   int    `json:"task_index"`
			Description string `json:"description"`
			Severity    string `json:"severity"`
			Suggestion  string `json:"suggestion"`
		} `json:"issues"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return nil, fmt.Errorf("failed to parse validation response")
	}

	// Convert to ValidationResult
	validationResult := &ValidationResult{
		IsValid: response.Validation,
		Score:   response.Score,
		Issues:  []ValidationIssue{},
	}

	for _, issue := range response.Issues {
		validationResult.Issues = append(validationResult.Issues, ValidationIssue{
			Type:        issue.Type,
			TaskIndex:   issue.TaskIndex,
			Field:       "content",
			Description: fmt.Sprintf("%s (Suggestion: %s)", issue.Description, issue.Suggestion),
			Severity:    issue.Severity,
		})
	}

	return validationResult, nil
}

// Helper functions for validation system
func (tv *TaskValidator) hasCriticalIssues(issues []ValidationIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "critical" {
			return true
		}
	}
	return false
}

func (tv *TaskValidator) calculateFormatScore(issues []ValidationIssue, totalTasks int) float64 {
	if totalTasks == 0 {
		return 0.0
	}

	score := 1.0
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			score -= 0.3
		case "major":
			score -= 0.2
		case "minor":
			score -= 0.1
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (tv *TaskValidator) isValidPriority(priority string) bool {
	validPriorities := []string{"critical", "high", "medium", "low"}
	for _, valid := range validPriorities {
		if priority == valid {
			return true
		}
	}
	return false
}
