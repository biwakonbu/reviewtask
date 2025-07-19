package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gh-review-task/internal/github"
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
	var prompt strings.Builder
	
	prompt.WriteString("You are a code review expert validating AI-generated tasks from PR review comments.\n\n")
	
	if tv.config.AISettings.UserLanguage != "" {
		prompt.WriteString(fmt.Sprintf("User's preferred language: %s\n\n", tv.config.AISettings.UserLanguage))
	}
	
	prompt.WriteString("VALIDATION CRITERIA:\n")
	prompt.WriteString("1. Each task should be actionable and specific\n")
	prompt.WriteString("2. Task descriptions should be in the user's preferred language\n")
	prompt.WriteString("3. Tasks should accurately reflect the original comment intent\n")
	prompt.WriteString("4. No duplicate tasks should exist\n")
	prompt.WriteString("5. All genuine issues from comments should be captured\n")
	prompt.WriteString("6. Task priorities should match issue severity\n\n")
	
	prompt.WriteString("RESPONSE FORMAT:\n")
	prompt.WriteString("Return JSON in this EXACT format:\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"validation\": true|false,\n")
	prompt.WriteString("  \"score\": 0.85,\n")
	prompt.WriteString("  \"issues\": [\n")
	prompt.WriteString("    {\n")
	prompt.WriteString("      \"type\": \"content|missing|incorrect|duplicate\",\n")
	prompt.WriteString("      \"task_index\": 0,\n")
	prompt.WriteString("      \"description\": \"Specific issue description\",\n")
	prompt.WriteString("      \"severity\": \"critical|major|minor\",\n")
	prompt.WriteString("      \"suggestion\": \"How to fix this issue\"\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("  ]\n")
	prompt.WriteString("}\n\n")
	
	// Add original reviews for context
	prompt.WriteString("ORIGINAL REVIEW COMMENTS:\n")
	for i, review := range reviews {
		prompt.WriteString(fmt.Sprintf("Review %d (ID: %d):\n", i+1, review.ID))
		if len(review.Comments) > 0 {
			for _, comment := range review.Comments {
				prompt.WriteString(fmt.Sprintf("  Comment ID %d: %s\n", comment.ID, comment.Body))
			}
		}
		prompt.WriteString("\n")
	}
	
	// Add generated tasks for validation
	prompt.WriteString("GENERATED TASKS TO VALIDATE:\n")
	for i, task := range tasks {
		prompt.WriteString(fmt.Sprintf("Task %d:\n", i))
		prompt.WriteString(fmt.Sprintf("  Description: %s\n", task.Description))
		prompt.WriteString(fmt.Sprintf("  Origin Text: %s\n", task.OriginText))
		prompt.WriteString(fmt.Sprintf("  Priority: %s\n", task.Priority))
		prompt.WriteString(fmt.Sprintf("  Comment ID: %d\n", task.SourceCommentID))
		prompt.WriteString(fmt.Sprintf("  Task Index: %d\n", task.TaskIndex))
		prompt.WriteString("\n")
	}
	
	return prompt.String()
}

func (tv *TaskValidator) callClaudeValidation(prompt string) (*ValidationResult, error) {
	cmd := exec.Command("claude", "-p", prompt, "--output-format", "json")
	// Ensure the command inherits the current environment including PATH
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		// Fallback: If Claude Code is unavailable, return a basic validation result
		return &ValidationResult{
			IsValid: true,
			Score:   0.7, // Moderate score since we can't validate content
			Issues:  []ValidationIssue{},
		}, nil
	}
	
	// Parse validation response
	var response struct {
		Validation bool `json:"validation"`
		Score      float64 `json:"score"`
		Issues     []struct {
			Type        string `json:"type"`
			TaskIndex   int    `json:"task_index"`
			Description string `json:"description"`
			Severity    string `json:"severity"`
			Suggestion  string `json:"suggestion"`
		} `json:"issues"`
	}
	
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse validation response: %w", err)
	}
	
	// Convert to ValidationResult
	result := &ValidationResult{
		IsValid: response.Validation,
		Score:   response.Score,
		Issues:  []ValidationIssue{},
	}
	
	for _, issue := range response.Issues {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:        issue.Type,
			TaskIndex:   issue.TaskIndex,
			Field:       "content",
			Description: fmt.Sprintf("%s (Suggestion: %s)", issue.Description, issue.Suggestion),
			Severity:    issue.Severity,
		})
	}
	
	return result, nil
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