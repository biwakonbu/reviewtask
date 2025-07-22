package ai

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
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

	// Check if validation is enabled in config
	if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
		fmt.Printf("  🐛 Using validation-enabled path\n")
		return a.GenerateTasksWithValidation(reviews)
	}

	// Extract all comments from all reviews, filtering out resolved comments
	var allComments []CommentContext
	resolvedCommentCount := 0

	for _, review := range reviews {
		for _, comment := range review.Comments {
			// Skip comments that have been marked as addressed/resolved
			if a.isCommentResolved(comment) {
				resolvedCommentCount++
				fmt.Printf("✅ Skipping resolved comment %d: %.50s...\n", comment.ID, comment.Body)
				continue
			}

			allComments = append(allComments, CommentContext{
				Comment:      comment,
				SourceReview: review,
			})
		}
	}

	if resolvedCommentCount > 0 {
		fmt.Printf("📝 Filtered out %d resolved comments\n", resolvedCommentCount)
	}

	if len(allComments) == 0 {
		return []storage.Task{}, nil
	}

	fmt.Printf("Processing %d comments in parallel...\n", len(allComments))
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
		for _, comment := range review.Comments {
			// Skip comments that have been marked as addressed/resolved
			if a.isCommentResolved(comment) {
				resolvedCommentCount++
				fmt.Printf("✅ Skipping resolved comment %d: %.50s...\n", comment.ID, comment.Body)
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
		fmt.Printf("📝 Filtered out %d resolved comments\n", resolvedCommentCount)
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
		fmt.Printf("🤖 Processing %d changed/new comments with AI...\n", len(changedCommentsCtx))

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
		fmt.Printf("✅ All comments are unchanged - no AI processing needed\n")
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
- AI deduplication will handle any redundancy later

%s`, languageInstruction, priorityPrompt, reviewsData.String())

	return prompt
}

func (a *Analyzer) callClaudeCode(prompt string) ([]TaskRequest, error) {
	// Check for very large prompts that might exceed system limits
	const maxPromptSize = 32 * 1024 // 32KB limit for safety
	if len(prompt) > maxPromptSize {
		return nil, fmt.Errorf("prompt size (%d bytes) exceeds maximum limit (%d bytes). Please shorten or chunk the prompt content", len(prompt), maxPromptSize)
	}

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
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
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

	// Debug: log first part of response if debug mode is enabled
	if a.config.AISettings.DebugMode {
		preview := result
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("  🐛 Claude response preview: %s\n", preview)
	}

	// Find JSON array in the response
	jsonStart := strings.Index(result, "[")
	jsonEnd := strings.LastIndex(result, "]")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		// Try to find JSON object instead of array
		objStart := strings.Index(result, "{")
		objEnd := strings.LastIndex(result, "}")
		if objStart != -1 && objEnd != -1 && objStart < objEnd {
			// Check if it's a single task wrapped in object
			objContent := result[objStart : objEnd+1]
			if a.config.AISettings.DebugMode {
				fmt.Printf("  🐛 Found JSON object instead of array: %s\n", objContent)
			}
			// Wrap single object in array
			result = "[" + objContent + "]"
		} else {
			if a.config.AISettings.DebugMode {
				fmt.Printf("  🐛 Full Claude response: %s\n", result)
			}
			return nil, fmt.Errorf("no valid JSON array found in Claude response")
		}
	} else {
		result = result[jsonStart : jsonEnd+1]
	}

	// Parse the actual task array
	var tasks []TaskRequest
	if err := json.Unmarshal([]byte(result), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse task array from result: %w\nResult was: %s", err, result)
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
		storageTask := storage.Task{
			// UUID-based ID generation ensures global uniqueness and security
			ID:              uuid.New().String(),
			Description:     task.Description,
			OriginText:      task.OriginText,
			Priority:        task.Priority,
			SourceReviewID:  task.SourceReviewID,
			SourceCommentID: task.SourceCommentID,
			TaskIndex:       task.TaskIndex,
			File:            task.File,
			Line:            task.Line,
			Status:          a.config.TaskSettings.DefaultStatus,
			CreatedAt:       now,
			UpdatedAt:       now,
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
		for _, err := range errors {
			fmt.Printf("  ⚠️  %v\n", err)
		}
		if len(allTasks) == 0 {
			return nil, fmt.Errorf("all comment processing failed")
		}
	}

	// Convert to storage tasks
	storageTasks := a.convertToStorageTasks(allTasks)

	// Apply deduplication
	dedupedTasks := a.deduplicateTasks(storageTasks)

	if a.config.AISettings.DeduplicationEnabled && len(dedupedTasks) < len(storageTasks) {
		fmt.Printf("  🔄 Deduplication: %d tasks → %d tasks (removed %d duplicates)\n",
			len(storageTasks), len(dedupedTasks), len(storageTasks)-len(dedupedTasks))
	}

	return dedupedTasks, nil
}

// generateTasksParallel processes comments in parallel using goroutines
func (a *Analyzer) generateTasksParallel(comments []CommentContext) ([]storage.Task, error) {
	fmt.Printf("Processing %d comments in parallel...\n", len(comments))
	tasks, err := a.processCommentsParallel(comments, a.processComment)
	if err == nil {
		fmt.Printf("✓ Generated %d tasks from %d comments\n", len(tasks), len(comments))
	}
	return tasks, err
}

// processComment handles a single comment and returns tasks for it
func (a *Analyzer) processComment(ctx CommentContext) ([]TaskRequest, error) {
	if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
		return a.processCommentWithValidation(ctx)
	}

	prompt := a.buildCommentPrompt(ctx)
	return a.callClaudeCode(prompt)
}

// processCommentWithValidation validates individual comment JSON responses
func (a *Analyzer) processCommentWithValidation(ctx CommentContext) ([]TaskRequest, error) {
	validator := NewTaskValidator(a.config)

	for attempt := 1; attempt <= validator.maxRetries; attempt++ {
		if a.config.AISettings.DebugMode {
			fmt.Printf("    🔄 Comment %d validation attempt %d/%d\n", ctx.Comment.ID, attempt, validator.maxRetries)
		}

		prompt := a.buildCommentPrompt(ctx)
		tasks, err := a.callClaudeCode(prompt)
		if err != nil {
			if a.config.AISettings.DebugMode {
				fmt.Printf("    ❌ Comment %d generation failed: %v\n", ctx.Comment.ID, err)
			}
			continue
		}

		// Stage 1: Format validation for this comment's tasks
		formatResult, err := validator.validateFormat(tasks)
		if err != nil {
			if a.config.AISettings.DebugMode {
				fmt.Printf("    ❌ Comment %d format validation failed: %v\n", ctx.Comment.ID, err)
			}
			continue
		}

		if !formatResult.IsValid {
			if a.config.AISettings.DebugMode {
				fmt.Printf("    ⚠️  Comment %d format issues (score: %.2f)\n", ctx.Comment.ID, formatResult.Score)
			}
			if attempt == validator.maxRetries {
				// Use best attempt on final try
				return formatResult.Tasks, nil
			}
			continue
		}

		// Stage 2: Content validation for this comment's tasks
		// Create a mini-review slice for validation context
		miniReviews := []github.Review{ctx.SourceReview}
		contentResult, err := validator.validateContent(formatResult.Tasks, miniReviews)
		if err != nil {
			if a.config.AISettings.DebugMode {
				fmt.Printf("    ❌ Comment %d content validation failed: %v\n", ctx.Comment.ID, err)
			}
			continue
		}

		if a.config.AISettings.DebugMode {
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

	prompt := fmt.Sprintf(`You are an AI assistant helping to analyze GitHub PR review comments and generate actionable tasks.

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
[
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
]

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
		ctx.SourceReview.ID,
		ctx.SourceReview.Reviewer,
		ctx.SourceReview.State,
		ctx.Comment.ID,
		ctx.Comment.Author,
		ctx.Comment.File,
		ctx.Comment.Line,
		ctx.Comment.Body,
		a.buildRepliesContext(ctx.Comment),
		ctx.SourceReview.ID,
		ctx.Comment.ID,
		ctx.Comment.File,
		ctx.Comment.Line)

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
func (a *Analyzer) findClaudeCommand() (string, error) {
	return FindClaudeCommand(a.config.AISettings.ClaudePath)
}

// generateTasksParallelWithValidation processes comments in parallel with validation enabled
func (a *Analyzer) generateTasksParallelWithValidation(comments []CommentContext) ([]storage.Task, error) {
	tasks, err := a.processCommentsParallel(comments, a.processCommentWithValidation)
	if err == nil {
		// Tasks are already deduplicated in processCommentsParallel
		fmt.Printf("✓ Generated %d tasks from %d comments with validation\n", len(tasks), len(comments))
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
		if a.config.AISettings.DebugMode {
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
			if a.config.AISettings.DebugMode {
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
		pi, _ := priorityOrder[sorted[i].Priority]
		pj, _ := priorityOrder[sorted[j].Priority]

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
				if a.config.AISettings.DebugMode {
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
