package ai

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"

	"github.com/google/uuid"
)

type Analyzer struct {
	config             *config.Config
	validationFeedback []ValidationIssue
	claudeClient       ClaudeClient
	promptSizeTracker  *PromptSizeTracker
	responseMonitor    *ResponseMonitor
}

func NewAnalyzer(cfg *config.Config) *Analyzer {
	// Default to real Claude client
	client, err := NewRealClaudeClient()
	if err != nil {
		// If Claude is not available, return analyzer without client
		// This allows tests to inject their own mock
		return &Analyzer{
			config:          cfg,
			responseMonitor: NewResponseMonitor(cfg.AISettings.VerboseMode),
		}
	}
	return &Analyzer{
		config:          cfg,
		claudeClient:    client,
		responseMonitor: NewResponseMonitor(cfg.AISettings.VerboseMode),
	}
}

// NewAnalyzerWithClient creates an analyzer with a specific Claude client (for testing)
func NewAnalyzerWithClient(cfg *config.Config, client ClaudeClient) *Analyzer {
	return &Analyzer{
		config:          cfg,
		claudeClient:    client,
		responseMonitor: NewResponseMonitor(cfg.AISettings.VerboseMode),
	}
}

type TaskRequest struct {
	Description     string `json:"description"` // AI-generated task description (user language)
	OriginText      string `json:"origin_text"` // Original review comment text
	Priority        string `json:"priority"`
	SourceReviewID  int64  `json:"source_review_id"`
	SourceCommentID int64  `json:"source_comment_id"` // Required: specific comment ID
	File            string `json:"file"`
	Line            int    `json:"line"`
	Status          string `json:"status"`
	TaskIndex       int    `json:"task_index"` // New: index within comment (0, 1, 2...)
}

type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Score   float64           `json:"score"` // 0.0-1.0 quality score
	Issues  []ValidationIssue `json:"issues"`
	Tasks   []TaskRequest     `json:"tasks"`
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
	// Clear validation feedback to ensure clean state for each PR analysis
	a.clearValidationFeedback()

	if len(reviews) == 0 {
		return []storage.Task{}, nil
	}

	// Extract all comments from all reviews, filtering out resolved comments
	var allComments []CommentContext
	resolvedCommentCount := 0

	for _, review := range reviews {
		// Process review body as a comment if it exists and contains content
		if review.Body != "" {
			// Create a pseudo-comment from the review body
			reviewBodyComment := github.Comment{
				ID:        review.ID, // Use review ID
				File:      "",        // Review body is not file-specific
				Line:      0,         // Review body is not line-specific
				Body:      review.Body,
				Author:    review.Reviewer,
				CreatedAt: review.SubmittedAt,
			}

			// Skip if this review body comment has been marked as resolved
			if !a.isCommentResolved(reviewBodyComment) {
				allComments = append(allComments, CommentContext{
					Comment:      reviewBodyComment,
					SourceReview: review,
				})
			} else {
				resolvedCommentCount++
				if a.config.AISettings.VerboseMode {
					fmt.Printf("✅ Skipping resolved review body %d: %.50s...\n", review.ID, review.Body)
				}
			}
		}

		// Process individual inline comments
		for _, comment := range review.Comments {
			// Skip comments that have been marked as addressed/resolved
			if a.isCommentResolved(comment) {
				resolvedCommentCount++
				if a.config.AISettings.VerboseMode {
					fmt.Printf("✅ Skipping resolved comment %d: %.50s...\n", comment.ID, comment.Body)
				}
				continue
			}

			allComments = append(allComments, CommentContext{
				Comment:      comment,
				SourceReview: review,
			})
		}
	}

	if resolvedCommentCount > 0 {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("📝 Filtered out %d resolved comments\n", resolvedCommentCount)
		}
	}

	if len(allComments) == 0 {
		return []storage.Task{}, nil
	}

	// Check if validation is enabled in config
	if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  🐛 Using validation-enabled path with parallel processing\n")
		}
		// Use parallel processing for validation mode to handle large PRs
		return a.generateTasksParallelWithValidation(allComments)
	}

	return a.generateTasksParallel(allComments)
}

// GenerateTasksWithCache generates tasks with MD5-based change detection using existing data
func (a *Analyzer) GenerateTasksWithCache(reviews []github.Review, prNumber int, storageManager *storage.Manager) ([]storage.Task, error) {
	// Clear validation feedback to ensure clean state for each PR analysis
	a.clearValidationFeedback()

	if len(reviews) == 0 {
		return []storage.Task{}, nil
	}

	// Extract all comments and create content hash map
	var allComments []github.Comment
	var allCommentsCtx []CommentContext
	commentHashMap := make(map[int64]string)
	resolvedCommentCount := 0

	for _, review := range reviews {
		// Process review body as a comment if it exists and contains content
		if review.Body != "" {
			// Create a pseudo-comment from the review body
			reviewBodyComment := github.Comment{
				ID:        review.ID, // Use review ID
				File:      "",        // Review body is not file-specific
				Line:      0,         // Review body is not line-specific
				Body:      review.Body,
				Author:    review.Reviewer,
				CreatedAt: review.SubmittedAt,
			}

			// Skip if this review body comment has been marked as resolved
			if !a.isCommentResolved(reviewBodyComment) {
				allComments = append(allComments, reviewBodyComment)
				allCommentsCtx = append(allCommentsCtx, CommentContext{
					Comment:      reviewBodyComment,
					SourceReview: review,
				})
				// Calculate MD5 hash of review body
				commentHashMap[review.ID] = a.calculateCommentHash(reviewBodyComment)
			} else {
				resolvedCommentCount++
				if a.config.AISettings.VerboseMode {
					fmt.Printf("✅ Skipping resolved review body %d: %.50s...\n", review.ID, review.Body)
				}
			}
		}

		// Process individual inline comments
		for _, comment := range review.Comments {
			// Skip comments that have been marked as addressed/resolved
			if a.isCommentResolved(comment) {
				resolvedCommentCount++
				if a.config.AISettings.VerboseMode {
					fmt.Printf("✅ Skipping resolved comment %d: %.50s...\n", comment.ID, comment.Body)
				}
				continue
			}

			allComments = append(allComments, comment)
			allCommentsCtx = append(allCommentsCtx, CommentContext{
				Comment:      comment,
				SourceReview: review,
			})
			// Calculate MD5 hash of entire comment thread (main comment + all replies)
			commentHashMap[comment.ID] = a.calculateCommentHash(comment)
		}
	}

	if resolvedCommentCount > 0 {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("📝 Filtered out %d resolved comments\n", resolvedCommentCount)
		}
	}

	if len(allComments) == 0 {
		return []storage.Task{}, nil
	}

	// Load existing tasks to compare hashes
	existingTasks, err := storageManager.GetTasksByPR(prNumber)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load existing tasks: %w", err)
	}

	// Create map of existing task hashes by comment ID
	existingHashMap := make(map[int64]string)
	existingTasksByComment := make(map[int64][]storage.Task)
	for _, task := range existingTasks {
		if task.CommentHash != "" {
			existingHashMap[task.SourceCommentID] = task.CommentHash
		}
		existingTasksByComment[task.SourceCommentID] = append(existingTasksByComment[task.SourceCommentID], task)
	}

	// Detect changes by comparing current hashes with existing hashes
	var changedCommentsCtx []CommentContext
	var unchangedTasks []storage.Task

	for _, commentCtx := range allCommentsCtx {
		comment := commentCtx.Comment
		currentHash := commentHashMap[comment.ID]
		existingHash, hasExistingTasks := existingHashMap[comment.ID]

		if !hasExistingTasks || existingHash != currentHash {
			// Comment is new or has changed - needs reprocessing
			changedCommentsCtx = append(changedCommentsCtx, commentCtx)
		} else {
			// Comment is unchanged - keep existing tasks
			unchangedTasks = append(unchangedTasks, existingTasksByComment[comment.ID]...)
		}
	}

	fmt.Printf("📊 Change analysis: %d unchanged, %d changed/new comments\n",
		len(allCommentsCtx)-len(changedCommentsCtx), len(changedCommentsCtx))

	var newTasks []storage.Task
	if len(changedCommentsCtx) > 0 {
		fmt.Printf("🤖 Generating tasks for %d changed/new comments...\n", len(changedCommentsCtx))

		// Generate tasks only for changed comments
		if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
			newTasks, err = a.generateTasksParallelWithValidation(changedCommentsCtx)
		} else {
			newTasks, err = a.generateTasksParallel(changedCommentsCtx)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to generate tasks: %w", err)
		}

		// Set comment hash for new tasks
		for i := range newTasks {
			newTasks[i].CommentHash = commentHashMap[newTasks[i].SourceCommentID]
		}
	} else {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("✅ All comments are unchanged - no AI processing needed\n")
		}
	}

	// Combine unchanged tasks with newly generated tasks
	allTasks := append(unchangedTasks, newTasks...)

	fmt.Printf("📋 Task summary: %d unchanged + %d newly generated = %d total\n",
		len(unchangedTasks), len(newTasks), len(allTasks))

	return allTasks, nil
}

// calculateCommentHash generates MD5 hash of entire comment thread including replies
func (a *Analyzer) calculateCommentHash(comment github.Comment) string {
	content := comment.Body
	for _, reply := range comment.Replies {
		content += reply.Body
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(content)))
}

type CommentContext struct {
	Comment      github.Comment
	SourceReview github.Review
}

func (a *Analyzer) GenerateTasksWithValidation(reviews []github.Review) ([]storage.Task, error) {
	validator := NewTaskValidator(a.config)
	var bestResult *ValidationResult
	var bestTasks []TaskRequest
	maxScore := 0.0

	for attempt := 1; attempt <= validator.maxRetries; attempt++ {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("🔄 Task generation attempt %d/%d...\n", attempt, validator.maxRetries)
		}

		// Generate tasks
		tasks, err := a.callClaudeCodeWithRetry(reviews, attempt)
		if err != nil {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  ❌ Generation failed: %v\n", err)
			}
			// If it's a prompt size error, no point in retrying
			if strings.Contains(err.Error(), "prompt size") && strings.Contains(err.Error(), "exceeds maximum limit") {
				if a.config.AISettings.VerboseMode {
					fmt.Printf("  ⚠️  Prompt size limit exceeded - stopping retries (use parallel processing instead)\n")
				}
				break
			}
			continue
		}

		// Stage 1: Format validation
		formatResult, err := validator.validateFormat(tasks)
		if err != nil {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  ❌ Format validation failed: %v\n", err)
			}
			continue
		}

		if !formatResult.IsValid {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  ⚠️  Format issues found (score: %.2f)\n", formatResult.Score)
			}
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
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  ❌ Content validation failed: %v\n", err)
			}
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
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  ✅ Validation passed!\n")
			}
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

// generateTasksLegacy is kept for historical reference; not used.
// Leaving the function commented to avoid unused warnings.
// func (a *Analyzer) generateTasksLegacy(reviews []github.Review) ([]storage.Task, error) {
//     prompt := a.buildAnalysisPrompt(reviews)
//     tasks, err := a.callClaudeCode(prompt)
//     if err != nil {
//         return nil, fmt.Errorf("failed to call Claude Code: %w", err)
//     }
//     return a.convertToStorageTasks(tasks), nil
// }

func (a *Analyzer) buildAnalysisPrompt(reviews []github.Review) string {
	// Initialize prompt size tracker
	a.promptSizeTracker = NewPromptSizeTracker()

	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" {
		languageInstruction = fmt.Sprintf("IMPORTANT: Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage)
	}
	a.promptSizeTracker.TrackLanguagePrompt(languageInstruction)

	priorityPrompt := a.config.GetPriorityPrompt()
	a.promptSizeTracker.TrackPriorityPrompt(priorityPrompt)

	// Add nitpick handling instructions
	nitpickInstruction := a.buildNitpickInstruction()
	a.promptSizeTracker.TrackNitpickPrompt(nitpickInstruction)

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

	// Track review data
	reviewsDataStr := reviewsData.String()
	a.promptSizeTracker.TrackReviewsData(reviewsDataStr, reviews)

	// Build system prompt
	systemPrompt := `You are an AI assistant helping to analyze GitHub PR reviews and generate actionable tasks.

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
3. Create appropriate number of tasks based on the comment's content
4. Each distinct actionable item should be a separate task
5. Assign task_index starting from 0 for multiple tasks
6. Only create tasks for comments requiring developer action
7. Consider comment chains - don't create tasks for resolved issues

Task Generation Guidelines:
- Create separate tasks for logically distinct actions
- If a comment mentions multiple unrelated issues, create separate tasks
- Ensure each task is self-contained and actionable
- Don't artificially combine unrelated items
- AI deduplication will handle any redundancy later`
	a.promptSizeTracker.TrackSystemPrompt(systemPrompt)

	prompt := fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n%s", systemPrompt, languageInstruction, priorityPrompt, nitpickInstruction, reviewsDataStr)

	return prompt
}

func (a *Analyzer) callClaudeCode(prompt string) ([]TaskRequest, error) {
	return a.callClaudeCodeWithRetryStrategy(prompt, 0)
}

// callClaudeCodeWithRetryStrategy executes Claude API call with intelligent retry logic
func (a *Analyzer) callClaudeCodeWithRetryStrategy(originalPrompt string, attemptNumber int) ([]TaskRequest, error) {
	prompt := originalPrompt
	startTime := time.Now()
	sessionID := uuid.New().String()

	// Initialize retry strategy if this is the first attempt
	var retryStrategy *RetryStrategy
	if attemptNumber == 0 && a.config.AISettings.EnableJSONRecovery {
		retryStrategy = NewRetryStrategy(a.config.AISettings.VerboseMode)
	}

	// Check for very large prompts that might exceed system limits
	const maxPromptSize = 32 * 1024 // 32KB limit for safety
	if len(prompt) > maxPromptSize {
		// Generate detailed error message if tracker is available
		if a.promptSizeTracker != nil && a.promptSizeTracker.IsExceeded() {
			if a.config.AISettings.VerboseMode {
				return nil, fmt.Errorf("%s", a.promptSizeTracker.GenerateErrorMessage())
			} else {
				// In non-debug mode, show simplified error with key info
				largestComponent, largestSize := a.promptSizeTracker.GetLargestComponent()
				return nil, fmt.Errorf("prompt size (%d bytes) exceeds maximum limit (%d bytes). %s is too large (%d bytes). Use --verbose for detailed breakdown",
					len(prompt), maxPromptSize, largestComponent, largestSize)
			}
		}
		return nil, fmt.Errorf("prompt size (%d bytes) exceeds maximum limit (%d bytes). Please shorten or chunk the prompt content", len(prompt), maxPromptSize)
	}

	// Use injected client if available, otherwise create a real one
	client := a.claudeClient
	if client == nil {
		var err error
		client, err = NewRealClaudeClient()
		if err != nil {
			return nil, NewClaudeAPIError("client initialization failed", err)
		}
	}

	// Debug information if enabled
	if a.config.AISettings.VerboseMode {
		fmt.Printf("  🐛 Prompt size: %d characters (attempt %d)\n", len(prompt), attemptNumber+1)
	}

	ctx := context.Background()
	output, err := client.Execute(ctx, prompt, "json")

	// Track response size for retry strategy analysis
	responseSize := len(output)

	if err != nil {
		// Check if we should retry with enhanced strategy
		if retryStrategy != nil {
			retryAttempt, shouldRetry := retryStrategy.ShouldRetry(attemptNumber, err, len(prompt), responseSize)
			if shouldRetry {
				// Execute retry delay
				retryStrategy.ExecuteDelay(retryAttempt)

				// Adjust prompt if needed
				adjustedPrompt := retryStrategy.AdjustPromptForRetry(originalPrompt, retryAttempt)
				if adjustedPrompt != originalPrompt {
					retryAttempt.AdjustedPrompt = true
					if a.config.AISettings.VerboseMode {
						fmt.Printf("  🔧 Adjusted prompt size: %d -> %d bytes\n", len(originalPrompt), len(adjustedPrompt))
					}
				}

				// Recursive retry with adjusted prompt
				return a.callClaudeCodeWithRetryStrategy(adjustedPrompt, attemptNumber+1)
			}
		}
		// Record API failure event for monitoring
		if a.responseMonitor != nil {
			processingTime := time.Since(startTime).Milliseconds()
			event := ResponseEvent{
				Timestamp:       time.Now(),
				SessionID:       sessionID,
				PromptSize:      len(prompt),
				ResponseSize:    responseSize,
				ProcessingTime:  processingTime,
				Success:         false,
				ErrorType:       "api_execution_failed",
				RecoveryUsed:    false,
				RetryCount:      attemptNumber,
				TasksExtracted:  0,
				PromptOptimized: len(prompt) < len(originalPrompt),
			}
			_ = a.responseMonitor.RecordEvent(event)
		}

		return nil, NewClaudeAPIError("execution failed", err)
	}

	// Parse Claude Code CLI response wrapper
	var claudeResponse struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}

	if err := json.Unmarshal([]byte(output), &claudeResponse); err != nil {
		return nil, fmt.Errorf("failed to parse claude wrapper response: %w", err)
	}

	if claudeResponse.IsError {
		return nil, fmt.Errorf("claude returned error: %s", claudeResponse.Result)
	}

	// Extract JSON from result (may be wrapped in markdown code block or text)
	result := claudeResponse.Result
	result = strings.TrimSpace(result)

	// Debug: log first part of response if debug mode is enabled
	if a.config.AISettings.VerboseMode {
		preview := result
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("  🐛 Claude response preview: %s\n", preview)
	}

	// Enhanced JSON extraction for better CodeRabbit compatibility
	result = a.extractJSON(result)
	if result == "" {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  🐛 Full Claude response: %s\n", claudeResponse.Result)
		}
		// For CodeRabbit nitpick comments, return empty array instead of error
		if a.config.AISettings.ProcessNitpickComments && a.isCodeRabbitNitpickResponse(claudeResponse.Result) {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  🔄 CodeRabbit nitpick detected with no actionable tasks - returning empty array\n")
			}
			return []TaskRequest{}, nil
		}
		return nil, fmt.Errorf("no valid JSON found in Claude response")
	}

	// Parse the actual task array with recovery mechanism
	var tasks []TaskRequest
	if err := json.Unmarshal([]byte(result), &tasks); err != nil {
		// First attempt JSON recovery for incomplete/malformed responses
		recoverer := NewJSONRecoverer(
			a.config.AISettings.EnableJSONRecovery,
			a.config.AISettings.VerboseMode,
		)

		recoveryResult := recoverer.RecoverJSON(result, err)
		recoverer.LogRecoveryAttempt(recoveryResult)

		if recoveryResult.IsRecovered && len(recoveryResult.Tasks) > 0 {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  ✅ JSON recovery successful: %s\n", recoveryResult.Message)
			}
			tasks = recoveryResult.Tasks

			// Record successful recovery event for monitoring
			if a.responseMonitor != nil {
				processingTime := time.Since(startTime).Milliseconds()
				event := ResponseEvent{
					Timestamp:       time.Now(),
					SessionID:       sessionID,
					PromptSize:      len(prompt),
					ResponseSize:    responseSize,
					ProcessingTime:  processingTime,
					Success:         true,
					RecoveryUsed:    true,
					RetryCount:      attemptNumber,
					TasksExtracted:  len(recoveryResult.Tasks),
					PromptOptimized: len(prompt) < len(originalPrompt),
				}
				_ = a.responseMonitor.RecordEvent(event)
			}

			return tasks, nil
		} else {
			// JSON recovery failed, check if we should retry the entire request
			if retryStrategy != nil && attemptNumber < retryStrategy.config.MaxRetries {
				retryAttempt, shouldRetry := retryStrategy.ShouldRetry(attemptNumber, err, len(prompt), responseSize)
				if shouldRetry {
					if a.config.AISettings.VerboseMode {
						fmt.Printf("  🔄 JSON parsing failed, attempting full retry with strategy: %s\n", retryAttempt.Strategy)
					}

					// Execute retry delay
					retryStrategy.ExecuteDelay(retryAttempt)

					// Adjust prompt if needed
					adjustedPrompt := retryStrategy.AdjustPromptForRetry(originalPrompt, retryAttempt)
					if adjustedPrompt != originalPrompt {
						retryAttempt.AdjustedPrompt = true
						if a.config.AISettings.VerboseMode {
							fmt.Printf("  🔧 Adjusted prompt size for retry: %d -> %d bytes\n", len(originalPrompt), len(adjustedPrompt))
						}
					}

					// Recursive retry with adjusted prompt
					return a.callClaudeCodeWithRetryStrategy(adjustedPrompt, attemptNumber+1)
				}
			}

			// Record failure event for monitoring
			if a.responseMonitor != nil {
				processingTime := time.Since(startTime).Milliseconds()
				event := ResponseEvent{
					Timestamp:       time.Now(),
					SessionID:       sessionID,
					PromptSize:      len(prompt),
					ResponseSize:    responseSize,
					ProcessingTime:  processingTime,
					Success:         false,
					ErrorType:       "json_parsing_failed",
					RecoveryUsed:    true,
					RetryCount:      attemptNumber,
					TasksExtracted:  0,
					PromptOptimized: len(prompt) < len(originalPrompt),
				}
				_ = a.responseMonitor.RecordEvent(event)
			}

			// Recovery and retry both failed, return original error with recovery info
			return nil, fmt.Errorf("failed to parse task array from result: %w (recovery attempted: %s)\nResult was: %s",
				err, recoveryResult.Message, result)
		}
	}

	// Record successful response event for monitoring
	if a.responseMonitor != nil {
		processingTime := time.Since(startTime).Milliseconds()
		event := ResponseEvent{
			Timestamp:       time.Now(),
			SessionID:       sessionID,
			PromptSize:      len(prompt),
			ResponseSize:    responseSize,
			ProcessingTime:  processingTime,
			Success:         true,
			RecoveryUsed:    false, // Will be updated if recovery was used
			RetryCount:      attemptNumber,
			TasksExtracted:  len(tasks),
			PromptOptimized: len(prompt) < len(originalPrompt),
		}

		_ = a.responseMonitor.RecordEvent(event)
	}

	return tasks, nil
}

// convertToStorageTasks converts AI-generated TaskRequest objects to storage.Task objects.
//
// SPECIFICATION: UUID-based Task ID Generation
// Task IDs are generated using UUIDs (via uuid.New().String()) to ensure:
// 1. Global uniqueness guarantee - no collisions possible
// 2. Unpredictability for security - cannot be guessed
// 3. No dependency on other field values - future-proof design
// 4. Standards compliance - follows RFC 4122
//
// WARNING: DO NOT revert to comment-based ID formats like "comment-%d-task-%d".
// Such approaches are fundamentally flawed and create collision risks.
func (a *Analyzer) convertToStorageTasks(tasks []TaskRequest) []storage.Task {
	var result []storage.Task
	now := time.Now().Format("2006-01-02T15:04:05Z")

	for _, task := range tasks {
		// Determine initial status based on low-priority patterns
		status := a.config.TaskSettings.DefaultStatus
		if a.isLowPriorityComment(task.OriginText) {
			status = a.config.TaskSettings.LowPriorityStatus
		}

		// Override priority for nitpick comments if configured
		priority := task.Priority
		if a.config.AISettings.ProcessNitpickComments && a.isLowPriorityComment(task.OriginText) {
			priority = a.config.AISettings.NitpickPriority
		}

		storageTask := storage.Task{
			// UUID-based ID generation ensures global uniqueness and security
			ID:              uuid.New().String(),
			Description:     task.Description,
			OriginText:      task.OriginText,
			Priority:        priority,
			SourceReviewID:  task.SourceReviewID,
			SourceCommentID: task.SourceCommentID,
			TaskIndex:       task.TaskIndex,
			File:            task.File,
			Line:            task.Line,
			Status:          status,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		result = append(result, storageTask)
	}

	return result
}

// IsLowPriorityComment checks if a comment body contains any low-priority patterns (public for testing)
func (a *Analyzer) IsLowPriorityComment(commentBody string) bool {
	return a.isLowPriorityComment(commentBody)
}

// isLowPriorityComment checks if a comment body contains any low-priority patterns
func (a *Analyzer) isLowPriorityComment(commentBody string) bool {
	if len(a.config.TaskSettings.LowPriorityPatterns) == 0 {
		return false
	}

	// Convert to lowercase for case-insensitive matching
	lowerBody := strings.ToLower(commentBody)

	// Check traditional patterns first
	for _, pattern := range a.config.TaskSettings.LowPriorityPatterns {
		// Check if the comment starts with the pattern (case-insensitive)
		if strings.HasPrefix(lowerBody, strings.ToLower(pattern)) {
			return true
		}
		// Also check if the pattern appears after newline (for multi-line comments)
		if strings.Contains(lowerBody, "\n"+strings.ToLower(pattern)) {
			return true
		}
	}

	// Check for CodeRabbit structured patterns
	if a.isCodeRabbitNitpickComment(lowerBody) {
		return true
	}

	return false
}

// isCodeRabbitNitpickComment detects CodeRabbit nitpick comments in structured format
func (a *Analyzer) isCodeRabbitNitpickComment(lowerBody string) bool {
	// CodeRabbit patterns to detect
	coderabbitPatterns := []string{
		"🧹 nitpick",
		"nitpick comments",
		"nitpick comment",
		"<summary>🧹 nitpick",
		"<summary>nitpick",
		"nitpick comments (",
		"nitpick comment (",
	}

	for _, pattern := range coderabbitPatterns {
		if strings.Contains(lowerBody, pattern) {
			return true
		}
	}

	// Check for structured HTML content that might contain nitpicks
	if a.hasStructuredNitpickContent(lowerBody) {
		return true
	}

	return false
}

// hasStructuredNitpickContent checks for structured HTML content with nitpick indicators
func (a *Analyzer) hasStructuredNitpickContent(lowerBody string) bool {
	// Look for <details> blocks with summary containing nitpick-related content
	if strings.Contains(lowerBody, "<details>") && strings.Contains(lowerBody, "<summary>") {
		// Extract content between <summary> tags
		summaryStart := strings.Index(lowerBody, "<summary>")
		if summaryStart == -1 {
			return false
		}

		summaryEnd := strings.Index(lowerBody[summaryStart:], "</summary>")
		if summaryEnd == -1 {
			// Look for closing pattern without explicit tag
			summaryEnd = strings.Index(lowerBody[summaryStart:], ">")
			if summaryEnd == -1 {
				return false
			}
		}

		// Validate that summaryStart+summaryEnd doesn't exceed bounds before applying buffer
		baseEndPos := summaryStart + summaryEnd
		if baseEndPos > len(lowerBody) {
			baseEndPos = len(lowerBody)
		}

		// Apply buffer with bounds checking
		endPos := baseEndPos + 20
		if endPos > len(lowerBody) {
			endPos = len(lowerBody)
		}
		summaryContent := lowerBody[summaryStart:endPos]

		// Check if summary contains nitpick indicators
		nitpickIndicators := []string{
			"nitpick",
			"nit",
			"🧹",
			"minor",
			"style",
			"suggestion",
		}

		for _, indicator := range nitpickIndicators {
			if strings.Contains(summaryContent, indicator) {
				return true
			}
		}
	}

	return false
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

// clearValidationFeedback clears validation feedback when starting new analysis
func (a *Analyzer) clearValidationFeedback() {
	a.validationFeedback = nil
}

// processCommentsParallel handles common parallel processing logic
func (a *Analyzer) processCommentsParallel(comments []CommentContext, processor func(CommentContext) ([]TaskRequest, error)) ([]storage.Task, error) {
	type commentResult struct {
		tasks []TaskRequest
		err   error
		index int
	}

	results := make(chan commentResult, len(comments))
	var wg sync.WaitGroup

	// Process each comment in parallel
	for i, commentCtx := range comments {
		wg.Add(1)
		go func(index int, ctx CommentContext) {
			defer wg.Done()

			tasks, err := processor(ctx)
			results <- commentResult{
				tasks: tasks,
				err:   err,
				index: index,
			}
		}(i, commentCtx)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allTasks []TaskRequest
	var errors []error

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Errorf("comment %d: %w", result.index, result.err))
		} else {
			allTasks = append(allTasks, result.tasks...)
		}
	}

	// Report errors but continue if we have some successful results
	if len(errors) > 0 {
		if a.config.AISettings.VerboseMode {
			for _, err := range errors {
				fmt.Printf("  ⚠️  %v\n", err)
			}
		}
		if len(allTasks) == 0 {
			return nil, fmt.Errorf("all comment processing failed")
		}
	}

	// Convert to storage tasks
	storageTasks := a.convertToStorageTasks(allTasks)

	// Apply deduplication
	dedupedTasks := a.deduplicateTasks(storageTasks)

	if a.config.AISettings.DeduplicationEnabled && len(dedupedTasks) < len(storageTasks) && a.config.AISettings.VerboseMode {
		fmt.Printf("  🔄 Deduplication: %d tasks → %d tasks (removed %d duplicates)\n",
			len(storageTasks), len(dedupedTasks), len(storageTasks)-len(dedupedTasks))
	}

	return dedupedTasks, nil
}

// generateTasksParallel processes comments in parallel using goroutines
func (a *Analyzer) generateTasksParallel(comments []CommentContext) ([]storage.Task, error) {
	if a.config.AISettings.VerboseMode {
		fmt.Printf("Processing %d comments in parallel...\n", len(comments))
	}
	tasks, err := a.processCommentsParallel(comments, a.processComment)
	if err == nil && a.config.AISettings.VerboseMode {
		fmt.Printf("✓ Generated %d tasks from %d comments\n", len(tasks), len(comments))
	}
	return tasks, err
}

// processComment handles a single comment and returns tasks for it
func (a *Analyzer) processComment(ctx CommentContext) ([]TaskRequest, error) {
	// Check if comment needs chunking
	chunker := NewCommentChunker(20000) // 20KB chunks to leave room for prompt template
	if chunker.ShouldChunkComment(ctx.Comment) {
		return a.processLargeComment(ctx, chunker)
	}

	if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
		return a.processCommentWithValidation(ctx)
	}

	prompt := a.buildCommentPrompt(ctx)
	return a.callClaudeCode(prompt)
}

// processLargeComment handles comments that exceed size limits by chunking
func (a *Analyzer) processLargeComment(ctx CommentContext, chunker *CommentChunker) ([]TaskRequest, error) {
	if a.config.AISettings.VerboseMode {
		fmt.Printf("  📄 Large comment detected (ID: %d, size: %d bytes), chunking...\n",
			ctx.Comment.ID, len(ctx.Comment.Body))
	}

	chunks := chunker.ChunkComment(ctx.Comment)
	var allTasks []TaskRequest

	for i, chunk := range chunks {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("    Processing chunk %d/%d (size: %d bytes)\n", i+1, len(chunks), len(chunk.Body))
		}

		// Create a new context with the chunked comment
		chunkCtx := CommentContext{
			Comment:      chunk,
			SourceReview: ctx.SourceReview,
		}

		// Process the chunk
		var tasks []TaskRequest
		var err error

		if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
			tasks, err = a.processCommentWithValidation(chunkCtx)
		} else {
			prompt := a.buildCommentPrompt(chunkCtx)
			tasks, err = a.callClaudeCode(prompt)
		}

		if err != nil {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("    ❌ Failed to process chunk %d: %v\n", i+1, err)
			}
			// Continue with other chunks even if one fails
			continue
		}

		allTasks = append(allTasks, tasks...)
	}

	if len(allTasks) == 0 && len(chunks) > 0 {
		return nil, fmt.Errorf("failed to process any chunks of large comment %d", ctx.Comment.ID)
	}

	return allTasks, nil
}

// processCommentWithValidation validates individual comment JSON responses
func (a *Analyzer) processCommentWithValidation(ctx CommentContext) ([]TaskRequest, error) {
	// Pre-check: Calculate actual prompt size to avoid validation failures
	testPrompt := a.buildCommentPrompt(ctx)
	const maxPromptSize = 32 * 1024 // 32KB limit (same as validator)

	if len(testPrompt) > maxPromptSize {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  📄 Comment %d prompt too large (%d bytes), using chunking instead of validation\n",
				ctx.Comment.ID, len(testPrompt))
		}
		// Use chunking without validation for oversized prompts
		chunker := NewCommentChunker(20000)
		return a.processLargeComment(ctx, chunker)
	}

	// Check if comment needs chunking based on size
	chunker := NewCommentChunker(20000) // 20KB chunks to leave room for prompt template
	if chunker.ShouldChunkComment(ctx.Comment) {
		// Process large comment with chunking (no validation for chunks)
		return a.processLargeComment(ctx, chunker)
	}

	validator := NewTaskValidator(a.config)

	for attempt := 1; attempt <= validator.maxRetries; attempt++ {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("    🔄 Comment %d validation attempt %d/%d\n", ctx.Comment.ID, attempt, validator.maxRetries)
		}

		prompt := a.buildCommentPrompt(ctx)
		tasks, err := a.callClaudeCode(prompt)
		if err != nil {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("    ❌ Comment %d generation failed: %v\n", ctx.Comment.ID, err)
			}
			// If it's a prompt size error, no point in retrying individual comments
			if strings.Contains(err.Error(), "prompt size") && strings.Contains(err.Error(), "exceeds maximum limit") {
				if a.config.AISettings.VerboseMode {
					fmt.Printf("    ⚠️  Comment %d prompt size limit exceeded - stopping retries\n", ctx.Comment.ID)
				}
				break
			}
			continue
		}

		// Stage 1: Format validation for this comment's tasks
		formatResult, err := validator.validateFormat(tasks)
		if err != nil {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("    ❌ Comment %d format validation failed: %v\n", ctx.Comment.ID, err)
			}
			continue
		}

		if !formatResult.IsValid {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("    ⚠️  Comment %d format issues (score: %.2f)\n", ctx.Comment.ID, formatResult.Score)
			}
			if attempt == validator.maxRetries {
				// Use best attempt on final try
				return formatResult.Tasks, nil
			}
			continue
		}

		// Stage 2: Content validation for this comment's tasks
		// Create a mini-review with only the current comment for validation context
		miniReview := github.Review{
			ID:       ctx.SourceReview.ID,
			Body:     ctx.SourceReview.Body,
			Comments: []github.Comment{ctx.Comment}, // Only include the current comment
		}
		miniReviews := []github.Review{miniReview}
		contentResult, err := validator.validateContent(formatResult.Tasks, miniReviews)
		if err != nil {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("    ❌ Comment %d content validation failed: %v\n", ctx.Comment.ID, err)
			}
			continue
		}

		if a.config.AISettings.VerboseMode {
			fmt.Printf("    ✅ Comment %d validation passed (score: %.2f)\n", ctx.Comment.ID, contentResult.Score)
		}

		// Return validated tasks
		return formatResult.Tasks, nil
	}

	return nil, fmt.Errorf("comment %d validation failed after %d attempts", ctx.Comment.ID, validator.maxRetries)
}

// buildCommentPrompt creates a focused prompt for analyzing a single comment
func (a *Analyzer) buildCommentPrompt(ctx CommentContext) string {
	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" {
		languageInstruction = fmt.Sprintf("IMPORTANT: Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage)
	}

	priorityPrompt := a.config.GetPriorityPrompt()

	// Add nitpick handling instructions
	nitpickInstruction := a.buildNitpickInstruction()

	// Build example task using proper JSON marshaling
	exampleTask := map[string]interface{}{
		"description":       "Actionable task description in specified language",
		"origin_text":       "Original review comment text (preserve exactly)",
		"priority":          "critical|high|medium|low",
		"source_review_id":  ctx.SourceReview.ID,
		"source_comment_id": ctx.Comment.ID,
		"file":              ctx.Comment.File,
		"line":              ctx.Comment.Line,
		"task_index":        0,
	}

	exampleJSON, err := json.MarshalIndent([]interface{}{exampleTask}, "", "  ")
	if err != nil {
		// Fallback to simple format if marshaling fails
		exampleJSON = []byte(fmt.Sprintf(`[
  {
    "description": "Actionable task description in specified language",
    "origin_text": "Original review comment text (preserve exactly)",
    "priority": "critical|high|medium|low",
    "source_review_id": %d,
    "source_comment_id": %d,
    "file": "%s",
    "line": %d,
    "task_index": 0
  }
]`, ctx.SourceReview.ID, ctx.Comment.ID, ctx.Comment.File, ctx.Comment.Line))
	}

	prompt := fmt.Sprintf(`You are an AI assistant helping to analyze GitHub PR review comments and generate actionable tasks.

%s
%s
%s

Analyze this single comment and create actionable tasks if needed:

Review Context:
- Review ID: %d
- Reviewer: %s
- Review State: %s

Comment Details:
- Comment ID: %d
- Author: %s
- File: %s:%d
- Comment Text: %s

%s

CRITICAL: Return response as JSON array with this EXACT format:
%s

Requirements:
1. PRESERVE original comment text in 'origin_text' field exactly as written
2. Generate clear, actionable 'description' in the specified user language
3. Create appropriate number of tasks based on the comment's content
4. Each distinct actionable item should be a separate task
5. Assign task_index starting from 0 for multiple tasks
6. Only create tasks for comments requiring developer action
7. Consider if this comment has already been resolved in discussion chains
8. Return empty array [] if no actionable tasks are needed

Task Generation Guidelines:
- Create separate tasks for logically distinct actions
- If a comment mentions multiple unrelated issues, create separate tasks
- Ensure each task is self-contained and actionable
- Don't artificially combine unrelated items
- AI deduplication will handle any redundancy later`,
		languageInstruction,
		priorityPrompt,
		nitpickInstruction,
		ctx.SourceReview.ID,
		ctx.SourceReview.Reviewer,
		ctx.SourceReview.State,
		ctx.Comment.ID,
		ctx.Comment.Author,
		ctx.Comment.File,
		ctx.Comment.Line,
		ctx.Comment.Body,
		a.buildRepliesContext(ctx.Comment),
		string(exampleJSON))

	return prompt
}

// buildRepliesContext formats reply chain for context
func (a *Analyzer) buildRepliesContext(comment github.Comment) string {
	if len(comment.Replies) == 0 {
		return ""
	}

	var repliesContext strings.Builder
	repliesContext.WriteString("\nReply Chain (for context):\n")
	for _, reply := range comment.Replies {
		repliesContext.WriteString(fmt.Sprintf("  - %s: %s\n", reply.Author, reply.Body))
	}

	return repliesContext.String()
}

// findClaudeCommand searches for Claude CLI using the shared utility function
// func (a *Analyzer) findClaudeCommand() (string, error) {
//     return FindClaudeCommand(a.config.AISettings.ClaudePath)
// }

// generateTasksParallelWithValidation processes comments in parallel with validation enabled
func (a *Analyzer) generateTasksParallelWithValidation(comments []CommentContext) ([]storage.Task, error) {
	tasks, err := a.processCommentsParallel(comments, a.processCommentWithValidation)
	if err == nil {
		// Tasks are already deduplicated in processCommentsParallel
		if a.config.AISettings.VerboseMode {
			fmt.Printf("✓ Generated %d tasks from %d comments with validation\n", len(tasks), len(comments))
		}
	}
	return tasks, err
}

// isCommentResolved checks if a comment has been marked as resolved/addressed
func (a *Analyzer) isCommentResolved(comment github.Comment) bool {
	// Check for common resolution markers in the comment body or replies
	resolvedMarkers := []string{
		"✅ Addressed in commit",
		"✅ Fixed in commit",
		"✅ Resolved in commit",
		"Addressed in commit",
		"Fixed in commit",
		"Resolved in commit",
	}

	// Check comment body
	commentText := strings.ToLower(comment.Body)
	for _, marker := range resolvedMarkers {
		if strings.Contains(commentText, strings.ToLower(marker)) {
			return true
		}
	}

	// Check replies for resolution markers
	for _, reply := range comment.Replies {
		replyText := strings.ToLower(reply.Body)
		for _, marker := range resolvedMarkers {
			if strings.Contains(replyText, strings.ToLower(marker)) {
				return true
			}
		}
	}

	return false
}

// extractJSON extracts JSON content from Claude response with improved robustness
func (a *Analyzer) extractJSON(response string) string {
	// First, check if the response contains a markdown code block
	if strings.Contains(response, "```json") {
		// Extract content between ```json and ```
		start := strings.Index(response, "```json")
		if start != -1 {
			start += 7 // Skip past "```json"
			end := strings.Index(response[start:], "```")
			if end != -1 {
				return strings.TrimSpace(response[start : start+end])
			}
		}
	} else if strings.Contains(response, "```") {
		// Extract content between ``` and ```
		start := strings.Index(response, "```")
		if start != -1 {
			start += 3 // Skip past "```"
			// Skip language identifier if present
			if newlineIdx := strings.Index(response[start:], "\n"); newlineIdx != -1 && newlineIdx < 20 {
				start += newlineIdx + 1
			}
			end := strings.Index(response[start:], "```")
			if end != -1 {
				return strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// If no code blocks, try to find JSON array or object
	response = strings.TrimSpace(response)

	// Look for JSON array first
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")

	if jsonStart != -1 && jsonEnd != -1 && jsonStart < jsonEnd {
		// Validate that this is likely the main JSON content
		// by checking if there's significant text before the JSON
		textBefore := strings.TrimSpace(response[:jsonStart])
		if len(textBefore) > 100 {
			// There's a lot of text before the JSON, try to find the actual JSON
			// Look for the last occurrence of a newline before the JSON array
			lastNewline := strings.LastIndex(response[:jsonStart], "\n")
			if lastNewline != -1 {
				jsonStart = strings.Index(response[lastNewline:], "[") + lastNewline
			}
		}
		return response[jsonStart : jsonEnd+1]
	}

	// Try to find JSON object and wrap it in array
	objStart := strings.Index(response, "{")
	objEnd := strings.LastIndex(response, "}")
	if objStart != -1 && objEnd != -1 && objStart < objEnd {
		objContent := response[objStart : objEnd+1]
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  🐛 Found JSON object instead of array: %s\n", objContent)
		}
		// Wrap single object in array
		return "[" + objContent + "]"
	}

	// Check for common non-JSON responses that should return empty array
	lowerResponse := strings.ToLower(response)
	emptyResponsePatterns := []string{
		"no actionable tasks",
		"no tasks needed",
		"[]",
		"empty array",
		"no action required",
		"already resolved",
		"no implementation needed",
	}

	for _, pattern := range emptyResponsePatterns {
		if strings.Contains(lowerResponse, pattern) {
			return "[]"
		}
	}

	return ""
}

// isCodeRabbitNitpickResponse checks if the response is about CodeRabbit nitpicks with no actionable tasks
func (a *Analyzer) isCodeRabbitNitpickResponse(response string) bool {
	lowerResponse := strings.ToLower(response)

	// Check for CodeRabbit-style responses about nitpicks
	codeRabbitPatterns := []string{
		"actionable comments posted: 0",
		"nitpick comments",
		"need to analyze if it contains any actionable tasks",
		"no actionable tasks in the nitpick",
		"nitpick suggestions don't require",
	}

	nitpickCount := 0
	for _, pattern := range codeRabbitPatterns {
		if strings.Contains(lowerResponse, pattern) {
			nitpickCount++
		}
	}

	// If we find multiple indicators of CodeRabbit nitpick responses, it's likely a nitpick-only comment
	return nitpickCount >= 1
}

// buildNitpickInstruction generates nitpick processing instructions based on configuration
func (a *Analyzer) buildNitpickInstruction() string {
	if a.config.AISettings.ProcessNitpickComments {
		return fmt.Sprintf(`
IMPORTANT: Nitpick Comment Processing Instructions:
- Process nitpick comments from review bots (like CodeRabbit) even when marked with "Actionable comments posted: 0"
- Ignore "Actionable comments posted: 0" headers when nitpick content is present
- Extract actionable tasks from nitpick sections and collapsible details
- Set priority to "%s" for tasks generated from nitpick comments
- Look for nitpick content in <details> blocks, summaries, and structured formats
- Do not skip comments containing valuable improvement suggestions just because they're labeled as nitpicks

`, a.config.AISettings.NitpickPriority)
	} else {
		return `
IMPORTANT: Nitpick Comment Processing:
- Skip nitpick comments and suggestions
- Ignore CodeRabbit nitpick sections
- Focus only on actionable review feedback requiring implementation

`
	}
}

// deduplicateTasks removes duplicate tasks based on comment ID and similarity
func (a *Analyzer) deduplicateTasks(tasks []storage.Task) []storage.Task {
	if !a.config.AISettings.DeduplicationEnabled {
		return tasks
	}

	// Use AI-powered deduplication if available
	deduplicator := NewTaskDeduplicator(a.config)

	// First, perform AI-based deduplication across all tasks
	deduplicatedTasks, err := deduplicator.DeduplicateTasks(tasks)
	if err != nil {
		fmt.Printf("⚠️  AI deduplication failed, falling back to rule-based: %v\n", err)
		// Fall back to the original similarity-based deduplication
		return a.deduplicateTasksRuleBased(tasks)
	}

	// Group tasks by comment ID for per-comment deduplication
	tasksByComment := make(map[int64][]storage.Task)
	for _, task := range deduplicatedTasks {
		tasksByComment[task.SourceCommentID] = append(tasksByComment[task.SourceCommentID], task)
	}

	var result []storage.Task
	for commentID, commentTasks := range tasksByComment {
		// Skip max_tasks_per_comment limit when using AI deduplication
		// The AI will handle determining the appropriate number of tasks
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  ✨ Comment %d: %d unique tasks identified by AI\n", commentID, len(commentTasks))
		}

		result = append(result, commentTasks...)
	}

	return result
}

// deduplicateTasksRuleBased is the fallback rule-based deduplication
func (a *Analyzer) deduplicateTasksRuleBased(tasks []storage.Task) []storage.Task {
	// Group tasks by comment ID
	tasksByComment := make(map[int64][]storage.Task)
	for _, task := range tasks {
		tasksByComment[task.SourceCommentID] = append(tasksByComment[task.SourceCommentID], task)
	}

	var result []storage.Task

	for commentID, commentTasks := range tasksByComment {
		// Apply max tasks per comment limit (only in rule-based mode)
		if len(commentTasks) > a.config.AISettings.MaxTasksPerComment {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  🔄 Comment %d: Limiting from %d to %d tasks (rule-based)\n",
					commentID, len(commentTasks), a.config.AISettings.MaxTasksPerComment)
			}
			// Sort by priority to keep the most important tasks
			sortedTasks := a.sortTasksByPriority(commentTasks)
			commentTasks = sortedTasks[:a.config.AISettings.MaxTasksPerComment]
		}

		// Apply similarity deduplication within the comment's tasks
		deduped := a.deduplicateSimilarTasks(commentTasks)
		result = append(result, deduped...)
	}

	return result
}

// sortTasksByPriority sorts tasks by priority (critical > high > medium > low)
func (a *Analyzer) sortTasksByPriority(tasks []storage.Task) []storage.Task {
	// Create a copy to avoid modifying the original
	sorted := make([]storage.Task, len(tasks))
	copy(sorted, tasks)

	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	// Sort by priority, then by task index using Go's built-in sort.Slice
	sort.Slice(sorted, func(i, j int) bool {
		pi := a.getPriorityValue(sorted[i].Priority)
		pj := a.getPriorityValue(sorted[j].Priority)

		if pi != pj {
			return pi < pj
		}
		return sorted[i].TaskIndex < sorted[j].TaskIndex
	})

	return sorted
}

// deduplicateSimilarTasks removes tasks with similar descriptions
func (a *Analyzer) deduplicateSimilarTasks(tasks []storage.Task) []storage.Task {
	if len(tasks) <= 1 {
		return tasks
	}

	// First, sort tasks by priority to ensure we process higher priority tasks first
	sortedTasks := a.sortTasksByPriority(tasks)

	var result []storage.Task
	seen := make(map[int]bool)

	for i, task1 := range sortedTasks {
		if seen[i] {
			continue
		}

		// Check similarity with remaining tasks
		for j := i + 1; j < len(sortedTasks); j++ {
			if seen[j] {
				continue
			}

			similarity := a.calculateSimilarity(task1.Description, sortedTasks[j].Description)
			if similarity >= a.config.AISettings.SimilarityThreshold {
				// Since we're sorted by priority, task1 has higher or equal priority
				// Always mark the later task (lower or equal priority) as duplicate
				seen[j] = true
				if a.config.AISettings.VerboseMode {
					fmt.Printf("  🔄 Deduplicating task: '%s' (similar to '%s', similarity: %.2f)\n",
						sortedTasks[j].Description, task1.Description, similarity)
				}
			}
		}

		if !seen[i] {
			result = append(result, task1)
		}
	}

	return result
}

// calculateSimilarity calculates the similarity between two strings (0.0 to 1.0)
func (a *Analyzer) calculateSimilarity(s1, s2 string) float64 {
	// Simple Jaccard similarity based on words
	words1 := strings.Fields(strings.ToLower(s1))
	words2 := strings.Fields(strings.ToLower(s2))

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Create word sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, w := range words1 {
		set1[w] = true
	}
	for _, w := range words2 {
		set2[w] = true
	}

	// Calculate intersection and union
	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection

	return float64(intersection) / float64(union)
}

// getPriorityValue returns numeric value for priority comparison
func (a *Analyzer) getPriorityValue(priority string) int {
	switch priority {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}
