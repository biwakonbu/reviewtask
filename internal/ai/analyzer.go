package ai

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
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
	errorTracker       *ErrorTracker
}

func NewAnalyzer(cfg *config.Config) *Analyzer {
	// Use factory to create AI provider
	client, err := NewAIProvider(cfg)

	if err != nil {
		// If no AI client is available, return analyzer without client
		// This allows tests to inject their own mock
		if cfg.AISettings.VerboseMode {
			fmt.Printf("⚠️  AI provider initialization failed: %v\n", err)
		}
		return &Analyzer{
			config:          cfg,
			responseMonitor: NewResponseMonitor(cfg.AISettings.VerboseMode),
			errorTracker:    NewErrorTracker(cfg.AISettings.ErrorTrackingEnabled, cfg.AISettings.VerboseMode, ".pr-review"),
		}
	}

	return &Analyzer{
		config:          cfg,
		claudeClient:    client,
		responseMonitor: NewResponseMonitor(cfg.AISettings.VerboseMode),
		errorTracker:    NewErrorTracker(cfg.AISettings.ErrorTrackingEnabled, cfg.AISettings.VerboseMode, ".pr-review"),
	}
}

// NewAnalyzerWithClient creates an analyzer with a specific Claude client (for testing)
func NewAnalyzerWithClient(cfg *config.Config, client ClaudeClient) *Analyzer {
	return &Analyzer{
		config:          cfg,
		claudeClient:    client,
		responseMonitor: NewResponseMonitor(cfg.AISettings.VerboseMode),
		errorTracker:    NewErrorTracker(cfg.AISettings.ErrorTrackingEnabled, cfg.AISettings.VerboseMode, ".pr-review"),
	}
}

// SimpleTaskRequest is what AI generates - minimal fields only
type SimpleTaskRequest struct {
	Description   string `json:"description"`    // AI-generated task description (user language)
	Priority      string `json:"priority"`       // critical|high|medium|low
	InitialStatus string `json:"initial_status"` // todo|pending (AI-assigned based on impact)
}

// TaskRequest is the full task structure with all fields
type TaskRequest struct {
	Description     string `json:"description"` // AI-generated task description (user language)
	OriginText      string `json:"origin_text"` // Original review comment text
	Priority        string `json:"priority"`
	SourceReviewID  int64  `json:"source_review_id"`
	SourceCommentID int64  `json:"source_comment_id"` // Required: specific comment ID
	File            string `json:"file"`
	Line            int    `json:"line"`
	Status          string `json:"status"`         // For legacy responses and direct status setting
	InitialStatus   string `json:"initial_status"` // AI-assigned status from impact assessment (todo/pending)
	TaskIndex       int    `json:"task_index"`     // New: index within comment (0, 1, 2...)
	URL             string `json:"url"`            // GitHub comment URL for direct navigation
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
	aiProvider ClaudeClient // Reuse ClaudeClient interface for AI provider
}

func NewTaskValidator(cfg *config.Config) *TaskValidator {
	return &TaskValidator{
		config:     cfg,
		maxRetries: cfg.AISettings.MaxRetries,
		aiProvider: nil, // Will be injected when needed
	}
}

func NewTaskValidatorWithClient(cfg *config.Config, client ClaudeClient) *TaskValidator {
	return &TaskValidator{
		config:     cfg,
		maxRetries: cfg.AISettings.MaxRetries,
		aiProvider: client,
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
			// Skip nitpick-only reviews when nitpick processing is disabled
			if !a.config.AISettings.ProcessNitpickComments && a.isNitpickOnlyReview(review.Body) {
				if a.config.AISettings.VerboseMode {
					fmt.Printf("🧹 Skipping nitpick-only review body %d (nitpick processing disabled)\n", review.ID)
				}
			} else {
				// Create a pseudo-comment from the review body
				reviewBodyComment := github.Comment{
					ID:        review.ID, // Use review ID
					File:      "",        // Review body is not file-specific
					Line:      0,         // Review body is not line-specific
					Body:      review.Body,
					Author:    review.Reviewer,
					CreatedAt: review.SubmittedAt,
					URL:       "", // Review bodies don't have direct URLs
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

// GenerateTasksWithRealtimeSaving processes reviews with real-time task saving
func (a *Analyzer) GenerateTasksWithRealtimeSaving(reviews []github.Review, prNumber int, storageManager *storage.Manager) ([]storage.Task, error) {
	if a.config.AISettings.VerboseMode {
		fmt.Printf("🔍 GenerateTasksWithRealtimeSaving called with %d reviews\n", len(reviews))
	}
	// Clear validation feedback to ensure clean state for each PR analysis
	a.clearValidationFeedback()

	if len(reviews) == 0 {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("🔍 No reviews to process, returning empty tasks\n")
		}
		return []storage.Task{}, nil
	}

	// Extract all comments from all reviews, filtering out resolved comments
	var allComments []CommentContext
	resolvedCommentCount := 0

	if a.config.AISettings.VerboseMode {
		fmt.Printf("🔍 Extracting comments from %d reviews\n", len(reviews))
	}
	for _, review := range reviews {
		// Process review body as a comment if it exists and contains content
		if review.Body != "" {
			// Skip nitpick-only reviews when nitpick processing is disabled
			if !a.config.AISettings.ProcessNitpickComments && a.isNitpickOnlyReview(review.Body) {
				if a.config.AISettings.VerboseMode {
					fmt.Printf("🧹 Skipping nitpick-only review body %d (nitpick processing disabled)\n", review.ID)
				}
			} else {
				// Create a pseudo-comment from the review body
				reviewBodyComment := github.Comment{
					ID:        review.ID, // Use review ID
					File:      "",        // Review body is not file-specific
					Line:      0,         // Review body is not line-specific
					Body:      review.Body,
					Author:    review.Reviewer,
					CreatedAt: review.SubmittedAt,
					URL:       "", // Review bodies don't have direct URLs
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
		if a.config.AISettings.VerboseMode {
			fmt.Printf("🔍 No comments to process after filtering, returning empty tasks\n")
		}
		return []storage.Task{}, nil
	}

	if a.config.AISettings.VerboseMode {
		fmt.Printf("🔍 Processing %d comments with real-time saving\n", len(allComments))
	}

	// Check if batch processing with existing task awareness is enabled
	if a.config.AISettings.EnableBatchProcessing {
		if a.config.AISettings.VerboseMode {
			fmt.Printf("🚀 Using batch processing with existing task awareness\n")
		}
		return a.ProcessCommentsWithBatchAI(allComments, storageManager, prNumber)
	}

	// Use stream processor with real-time saving (default behavior)
	streamProcessor := NewStreamProcessor(a)
	return streamProcessor.ProcessCommentsWithRealtimeSaving(allComments, storageManager, prNumber)
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
	// Switch by prompt profile; default to legacy for full backward compatibility
	profile := strings.ToLower(strings.TrimSpace(a.config.AISettings.PromptProfile))
	if profile == "" || profile == "legacy" {
		return a.buildAnalysisPromptLegacy(reviews)
	}
	return a.buildAnalysisPromptV2(reviews, profile)
}

// buildAnalysisPromptLegacy preserves the original prompt construction
func (a *Analyzer) buildAnalysisPromptLegacy(reviews []github.Review) string {
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

	// Build review data (full detail)
	reviewsDataStr := a.renderReviewsData(reviews, "rich")
	a.promptSizeTracker.TrackReviewsData(reviewsDataStr, reviews)

	// System prompt and schema
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

	return fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n%s", systemPrompt, languageInstruction, priorityPrompt, nitpickInstruction, reviewsDataStr)
}

// buildAnalysisPromptV2 builds a profile-aware prompt with compact/rich options
func (a *Analyzer) buildAnalysisPromptV2(reviews []github.Review, profile string) string {
	// Initialize prompt size tracker
	a.promptSizeTracker = NewPromptSizeTracker()

	// Language
	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" {
		languageInstruction = fmt.Sprintf("IMPORTANT: Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage)
	}
	a.promptSizeTracker.TrackLanguagePrompt(languageInstruction)

	// Priority
	priorityPrompt := a.config.GetPriorityPrompt()
	a.promptSizeTracker.TrackPriorityPrompt(priorityPrompt)

	// Nitpick guidance
	nitpickInstruction := a.buildNitpickInstruction()
	a.promptSizeTracker.TrackNitpickPrompt(nitpickInstruction)

	// Choose detail level based on profile
	detail := "rich"
	switch profile {
	case "rich", "v2":
		detail = "rich"
	case "compact":
		detail = "compact"
	case "minimal":
		detail = "minimal"
	}

	reviewsDataStr := a.renderReviewsData(reviews, detail)
	a.promptSizeTracker.TrackReviewsData(reviewsDataStr, reviews)

	// Enhanced schema with unified task generation philosophy and impact assessment
	systemPrompt := `You are analyzing GitHub PR review comments to produce actionable developer tasks with AI-powered impact assessment.

===== TASK GENERATION PHILOSOPHY =====

FUNDAMENTAL PRINCIPLE: Understand the reviewer's INTENDED GOAL before creating any tasks.
Your job is to capture what the reviewer wants to ACHIEVE, not to break down HOW to implement it.

CRITICAL RULES:
1. CREATE EXACTLY ONE TASK PER COMMENT (unless multiple completely unrelated topics exist)
2. PRESERVE the reviewer's intent and reasoning in full detail
3. DO NOT reduce information - especially when tools like CodeRabbit provide "Prompt for AI Agents" summaries
4. Focus on the END GOAL, not implementation steps
5. Think from the reviewer's perspective: "What outcome do they want to see?"

===== COMPREHENSIVE COMMENT ANALYSIS =====

ANALYZE ALL REVIEW COMMENTS without filtering:
- Include nitpicks (minor suggestions like typos, variable naming, formatting)
- Include questions that may need code changes or documentation
- Include all suggestions and recommendations regardless of size
- Include all feedback, even seemingly trivial items

===== AI-POWERED IMPACT ASSESSMENT =====

For EACH task, evaluate the implementation effort/impact and assign initial_status:

**TODO (Small/Medium Impact) - Assign when:**
- Changes to existing code < 50 lines
- No architecture or design changes required
- No new dependencies needed
- Quick fixes and improvements
- Examples: typo fixes, variable renaming, adding comments, formatting changes, simple logic fixes, adding error handling, adding validation

**PENDING (Large Impact) - Assign when:**
- Changes to existing code > 50 lines expected
- Architecture or design changes required
- New dependencies or major refactoring needed
- Requires significant discussion or planning
- Examples: design changes, new features, major refactoring, API changes

===== SPECIAL HANDLING FOR CODERABBIT COMMENTS =====

When CodeRabbit provides "Prompt for AI Agents" sections:
- These are ALREADY well-structured summaries of what needs to be done
- Use this information AS-IS for your task description
- Do NOT break these down into smaller pieces
- The AI prompt already represents the reviewer's intended goal

Return ONLY a valid JSON array. Each element MUST contain exactly these fields:
- description (string): actionable instruction capturing the reviewer's OVERALL GOAL
- origin_text (string): verbatim original review comment text
- priority (string): one of critical|high|medium|low
- initial_status (string): "todo" or "pending" based on impact assessment
- source_review_id (number)
- source_comment_id (number)
- file (string): file path or empty if not applicable
- line (number): line number or 0 if not applicable
- task_index (number): 0-based index within the comment (usually 0)

Rules:
- Do NOT include any explanations outside the JSON array
- CREATE ONE UNIFIED TASK per comment that captures the reviewer's complete intent
- Only create multiple tasks when a comment discusses COMPLETELY SEPARATE topics
- Preserve origin_text exactly; do not translate or summarize it
- Focus on WHAT needs to be achieved, not step-by-step implementation details
- When in doubt, create ONE comprehensive task rather than multiple small ones
- ALWAYS assign initial_status based on impact assessment criteria above

CRITICAL JSON FORMATTING RULES:
- Return RAW JSON only - NO markdown code blocks, NO ` + "```json" + ` wrappers, NO explanations
- ESCAPE all special characters in JSON strings: use \\n for newlines, \\t for tabs, \\r for carriage returns
- Do NOT include literal newlines or tabs inside JSON string values
- All string values must be properly escaped and on a single logical line`
	a.promptSizeTracker.TrackSystemPrompt(systemPrompt)

	return fmt.Sprintf("%s\n\n%s\n%s\n%s\n\n%s", systemPrompt, languageInstruction, priorityPrompt, nitpickInstruction, reviewsDataStr)
}

// renderReviewsData renders reviews and comments with varying detail levels
func (a *Analyzer) renderReviewsData(reviews []github.Review, detail string) string {
	var b strings.Builder
	b.WriteString("PR Reviews to analyze:\n\n")
	for i, review := range reviews {
		b.WriteString(fmt.Sprintf("Review %d (ID: %d):\n", i+1, review.ID))
		switch detail {
		case "rich":
			b.WriteString(fmt.Sprintf("Reviewer: %s\n", review.Reviewer))
			b.WriteString(fmt.Sprintf("State: %s\n", review.State))
			if review.Body != "" {
				b.WriteString(fmt.Sprintf("Review Body: %s\n", review.Body))
			}
			if len(review.Comments) > 0 {
				b.WriteString("Comments:\n")
				for _, comment := range review.Comments {
					b.WriteString(fmt.Sprintf("  Comment ID: %d\n", comment.ID))
					b.WriteString(fmt.Sprintf("  File: %s:%d\n", comment.File, comment.Line))
					b.WriteString(fmt.Sprintf("  Author: %s\n", comment.Author))
					b.WriteString(fmt.Sprintf("  Text: %s\n", comment.Body))
					if len(comment.Replies) > 0 {
						b.WriteString("  Replies:\n")
						for _, reply := range comment.Replies {
							b.WriteString(fmt.Sprintf("    - %s: %s\n", reply.Author, reply.Body))
						}
					}
					b.WriteString("\n")
				}
			}
		case "compact":
			// Keep only essential fields; omit reviewer/state; include comment summaries
			if len(review.Comments) > 0 {
				b.WriteString("Comments:\n")
				for _, comment := range review.Comments {
					b.WriteString(fmt.Sprintf("  ID:%d File:%s:%d Author:%s\n", comment.ID, comment.File, comment.Line, comment.Author))
					b.WriteString(fmt.Sprintf("  Text: %s\n\n", comment.Body))
				}
			} else if review.Body != "" {
				b.WriteString(fmt.Sprintf("Body: %s\n\n", review.Body))
			}
		case "minimal":
			// Only IDs and raw text; smallest footprint
			if review.Body != "" {
				b.WriteString("Body:\n")
				b.WriteString(review.Body)
				b.WriteString("\n")
			}
			for _, comment := range review.Comments {
				b.WriteString(fmt.Sprintf("Comment %d: %s\n", comment.ID, comment.Body))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// callClaudeForSimpleTasks calls Claude and returns simple task objects
func (a *Analyzer) callClaudeForSimpleTasks(prompt string) ([]SimpleTaskRequest, error) {
	// Use existing client infrastructure
	client := a.claudeClient
	if client == nil {
		var err error
		client, err = NewRealClaudeClientWithConfig(a.config)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	output, err := client.Execute(ctx, prompt, "json")
	if err != nil {
		return nil, err
	}

	// The output should be a simple JSON array
	result := strings.TrimSpace(output)

	// Try to extract JSON from various wrapper formats
	// 1. Check for <response> tags
	if strings.Contains(result, "<response>") {
		start := strings.Index(result, "<response>") + len("<response>")
		end := strings.Index(result, "</response>")
		if end > start {
			result = strings.TrimSpace(result[start:end])
		}
	}
	// 2. Check for code blocks
	if strings.Contains(result, "```") {
		result = a.extractJSON(result)
	}
	// 3. Find JSON array even with prefix text
	if !strings.HasPrefix(result, "[") {
		idx := strings.Index(result, "[")
		if idx > 0 {
			result = result[idx:]
		}
	}

	// Parse the simple tasks
	var simpleTasks []SimpleTaskRequest
	if err := json.Unmarshal([]byte(result), &simpleTasks); err != nil {
		// Try JSON recovery if enabled
		if a.config.AISettings.EnableJSONRecovery {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  🔧 Attempting JSON recovery for simple tasks (error: %v)\n", err)
			}

			// Try to recover truncated JSON arrays
			recoverer := NewJSONRecoverer(
				a.config.AISettings.EnableJSONRecovery,
				a.config.AISettings.VerboseMode,
			)

			// Convert to TaskRequest format for recovery
			recoveryResult := recoverer.RecoverJSON(result, err)
			if recoveryResult.IsRecovered && len(recoveryResult.Tasks) > 0 {
				// Convert recovered TaskRequest back to SimpleTaskRequest
				for _, task := range recoveryResult.Tasks {
					simpleTasks = append(simpleTasks, SimpleTaskRequest{
						Description:   task.Description,
						Priority:      task.Priority,
						InitialStatus: task.Status, // Preserve initial status from recovery
					})
				}
				if a.config.AISettings.VerboseMode {
					fmt.Printf("  ✅ Recovered %d simple tasks from truncated response\n", len(simpleTasks))
				}
				return simpleTasks, nil
			}

			// Try enhanced recovery
			enhancedRecoverer := NewEnhancedJSONRecovery(
				a.config.AISettings.EnableJSONRecovery,
				a.config.AISettings.VerboseMode,
			)
			recoveryResult = enhancedRecoverer.RepairAndRecover(result, err)
			if recoveryResult.IsRecovered && len(recoveryResult.Tasks) > 0 {
				// Convert recovered TaskRequest back to SimpleTaskRequest
				for _, task := range recoveryResult.Tasks {
					simpleTasks = append(simpleTasks, SimpleTaskRequest{
						Description:   task.Description,
						Priority:      task.Priority,
						InitialStatus: task.Status, // Preserve initial status from recovery
					})
				}
				if a.config.AISettings.VerboseMode {
					fmt.Printf("  ✅ Enhanced recovery: recovered %d simple tasks\n", len(simpleTasks))
				}
				return simpleTasks, nil
			}
		}

		// Log for debugging
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  ❌ Failed to parse simple tasks: %v\n", err)
			fmt.Printf("  🐛 Raw output: %.500s\n", result)
		}
		return []SimpleTaskRequest{}, nil // Return empty array if all recovery attempts fail
	}

	return simpleTasks, nil
}

func (a *Analyzer) callClaudeCode(prompt string) ([]TaskRequest, error) {
	return a.callClaudeCodeWithRetryStrategy(prompt, 0)
}

// RenderAnalysisPrompt exposes the internal prompt builder for tooling/debug.
// It does not perform any AI calls and is safe for local/offline usage.
func (a *Analyzer) RenderAnalysisPrompt(reviews []github.Review) string {
	return a.buildAnalysisPrompt(reviews)
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

	// Use the AI provider client (should always be initialized by NewAnalyzer)
	client := a.claudeClient
	if client == nil {
		return nil, fmt.Errorf("AI provider not initialized - this should not happen")
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

	// The claude_client.Execute already extracts the result field when outputFormat is "json"
	// So 'output' here is already the raw result string, not the wrapper
	result := strings.TrimSpace(output)

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
			fmt.Printf("  🐛 Full Claude response: %s\n", output)
		}
		// For CodeRabbit nitpick comments, return empty array instead of error
		if a.config.AISettings.ProcessNitpickComments && a.isCodeRabbitNitpickResponse(output) {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  🔄 CodeRabbit nitpick detected with no actionable tasks - returning empty array\n")
			}
			return []TaskRequest{}, nil
		}
		return nil, fmt.Errorf("no valid JSON found in Claude response")
	}

	// Parse the actual task array with recovery mechanism
	var tasks []TaskRequest

	// STEP 0: Try to repair common cursor-agent(grok) JSON issues before parsing
	repairedResult, repairErr := RepairJSONResponse(result)
	if repairErr == nil {
		result = repairedResult // Use repaired version
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  🔧 JSON auto-repair successful\n")
		}
	}

	if err := json.Unmarshal([]byte(result), &tasks); err != nil {
		// First attempt JSON recovery for incomplete/malformed responses
		recoverer := NewJSONRecoverer(
			a.config.AISettings.EnableJSONRecovery,
			a.config.AISettings.VerboseMode,
		)

		recoveryResult := recoverer.RecoverJSON(result, err)
		recoverer.LogRecoveryAttempt(recoveryResult)

		// If standard recovery failed, try enhanced recovery
		if !recoveryResult.IsRecovered || len(recoveryResult.Tasks) == 0 {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("  🚀 Trying enhanced JSON recovery...\n")
			}
			enhancedRecoverer := NewEnhancedJSONRecovery(
				a.config.AISettings.EnableJSONRecovery,
				a.config.AISettings.VerboseMode,
			)
			recoveryResult = enhancedRecoverer.RepairAndRecover(result, err)
		}

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

// generateDeterministicTaskID generates a deterministic UUID v5 based on comment ID, task index,
// and comment content hash. This ensures that the same comment content and task index always produce
// the same ID, but edited comments generate new IDs, preventing task loss while maintaining deduplication.
//
// UUID v5 Generation Strategy:
// - Uses SHA-1 based UUID v5 (deterministic) instead of v4 (random)
// - Namespace: Standard DNS namespace (6ba7b810-9dad-11d1-80b4-00c04fd430c8)
// - Name: "comment-{commentID}-task-{taskIndex}-content-{contentHash}"
// - Same input always produces same output (idempotent)
// - Comment edits change contentHash, generating new task IDs
// - Compatible with existing UUID infrastructure
// - No database schema changes needed
//
// Benefits:
// - Prevents duplicate tasks from same comment across multiple reviewtask runs
// - Allows MergeTasks to properly handle comment edits (new IDs → new tasks)
// - Leverages existing WriteWorker deduplication logic
// - Maintains RFC 4122 compliance
// - Provides stable task references until comment is edited
//
// Example:
//
//	generateDeterministicTaskID(12345, 0, "Fix bug") → "a1b2c3d4-e5f6-5789-abcd-ef0123456789"
//	generateDeterministicTaskID(12345, 0, "Fix bug") → "a1b2c3d4-e5f6-5789-abcd-ef0123456789" (same)
//	generateDeterministicTaskID(12345, 0, "Fix bug X") → "b2c3d4e5-f6a7-5890-bcde-f12345678901" (edited)
func (a *Analyzer) generateDeterministicTaskID(commentID int64, taskIndex int, commentContent string) string {
	// Standard DNS namespace UUID for v5 generation
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	// Create SHA-256 hash of comment content for stable content-based ID component
	contentHashBytes := sha256.Sum256([]byte(commentContent))
	contentHash := hex.EncodeToString(contentHashBytes[:8]) // Use first 8 bytes (16 hex chars)

	// Create deterministic name including comment ID, task index, and content hash
	// This ensures:
	// 1. Same comment content → same ID (deduplication)
	// 2. Edited comment → different ID (allows new tasks)
	name := fmt.Sprintf("comment-%d-task-%d-content-%s", commentID, taskIndex, contentHash)

	// Generate UUID v5 (SHA-1 based, deterministic)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

// convertToStorageTasks converts AI-generated TaskRequest objects to storage.Task objects.
//
// SPECIFICATION: Deterministic UUID-based Task ID Generation
// Task IDs are generated using UUID v5 (deterministic) to ensure:
// 1. Idempotency - same comment content (content hash) + task index = same ID across runs
// 2. Deduplication - prevents duplicate tasks when reviewtask runs multiple times
// 3. Standards compliance - follows RFC 4122
// 4. Compatibility - works with existing UUID infrastructure
//
// This implementation fixes Issue #247 by ensuring the same comment content always generates
// the same task ID, allowing WriteWorker to properly deduplicate tasks.
func (a *Analyzer) convertToStorageTasks(tasks []TaskRequest) []storage.Task {
	var result []storage.Task
	now := time.Now().UTC().Format(time.RFC3339)

	for _, task := range tasks {
		// Priority order: InitialStatus (AI impact assessment) > Status (legacy) > fallback patterns
		status := task.InitialStatus
		if status == "" {
			status = task.Status
		}
		if status == "" {
			// Fallback: Determine initial status based on low-priority patterns
			status = a.config.TaskSettings.DefaultStatus
			if a.isLowPriorityComment(task.OriginText) {
				status = a.config.TaskSettings.LowPriorityStatus
			}
		}

		// Override priority for nitpick comments if configured
		priority := task.Priority
		if a.config.AISettings.ProcessNitpickComments && a.isLowPriorityComment(task.OriginText) {
			priority = a.config.AISettings.NitpickPriority
		}

		storageTask := storage.Task{
			// Deterministic UUID generation based on comment ID, task index, and content hash
			// This ensures idempotency: same comment content generates same task ID
			// But edited comments generate new IDs, allowing MergeTasks to handle updates properly
			ID:              a.generateDeterministicTaskID(task.SourceCommentID, task.TaskIndex, task.OriginText),
			Description:     task.Description,
			OriginText:      task.OriginText,
			Priority:        priority,
			SourceReviewID:  task.SourceReviewID,
			SourceCommentID: task.SourceCommentID,
			TaskIndex:       task.TaskIndex,
			File:            task.File,
			Line:            task.Line,
			Status:          status, // Now respects AI-assigned initial status
			CreatedAt:       now,
			UpdatedAt:       now,
			URL:             task.URL,
		}
		result = append(result, storageTask)
	}

	return result
}

// ConvertToStorageTasksForTesting is a test helper that exposes convertToStorageTasks for integration tests
func (a *Analyzer) ConvertToStorageTasksForTesting(taskRequests []TaskRequest, prNumber int) []storage.Task {
	tasks := a.convertToStorageTasks(taskRequests)
	// Set PR number for all tasks
	for i := range tasks {
		tasks[i].PRNumber = prNumber
	}
	return tasks
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
	return a.isCodeRabbitNitpickComment(lowerBody)
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
	return a.hasStructuredNitpickContent(lowerBody)
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
		tasks   []TaskRequest
		err     error
		index   int
		context CommentContext
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
				tasks:   tasks,
				err:     err,
				index:   index,
				context: ctx,
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
			// Record error in error tracker
			if a.errorTracker != nil {
				errorType := "processing_failed"
				if strings.Contains(result.err.Error(), "json") {
					errorType = "json_parse"
				} else if strings.Contains(result.err.Error(), "API") || strings.Contains(result.err.Error(), "execution failed") {
					errorType = "api_failure"
				} else if strings.Contains(result.err.Error(), "context") || strings.Contains(result.err.Error(), "size") {
					errorType = "context_overflow"
				}
				a.errorTracker.RecordCommentError(result.context, errorType, result.err.Error(), 0, false, 0, 0)
			}
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
			// Show error summary when all processing failed
			if a.errorTracker != nil && a.config.AISettings.VerboseMode {
				a.errorTracker.PrintErrorSummary()
			}
			return nil, fmt.Errorf("all comment processing failed")
		}
		// Show error summary when some processing failed
		if a.errorTracker != nil && a.config.AISettings.VerboseMode {
			a.errorTracker.PrintErrorSummary()
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

	// Use stream processor if enabled, otherwise use traditional parallel processing
	if a.config.AISettings.StreamProcessingEnabled {
		streamProcessor := NewStreamProcessor(a)
		tasks, err := streamProcessor.ProcessCommentsStream(comments, a.processComment)
		if err == nil && a.config.AISettings.VerboseMode {
			fmt.Printf("✓ Generated %d tasks from %d comments (stream mode)\n", len(tasks), len(comments))
		}
		return tasks, err
	}

	// Traditional parallel processing
	tasks, err := a.processCommentsParallel(comments, a.processComment)
	if err == nil && a.config.AISettings.VerboseMode {
		fmt.Printf("✓ Generated %d tasks from %d comments\n", len(tasks), len(comments))
	}
	return tasks, err
}

// processCommentSimple uses simplified AI prompts for better reliability
func (a *Analyzer) processCommentSimple(ctx CommentContext) ([]TaskRequest, error) {
	// Try template-based prompt first, fall back to hardcoded if needed
	prompt := a.buildSimpleCommentPromptFromTemplate(ctx)

	// Call AI with simple prompt
	simpleTasks, err := a.callClaudeForSimpleTasks(prompt)
	if err != nil {
		return nil, err
	}

	// Apply task consolidation if multiple tasks were generated from single comment
	consolidatedTasks := a.consolidateTasksIfNeeded(simpleTasks)

	// Convert simple tasks to full TaskRequest objects
	var fullTasks []TaskRequest
	for i, simpleTask := range consolidatedTasks {
		// Use AI-assigned initial status if provided
		// If empty, convertToStorageTasks will handle fallback logic
		initialStatus := simpleTask.InitialStatus

		fullTask := TaskRequest{
			Description:     simpleTask.Description,
			Priority:        simpleTask.Priority,
			OriginText:      ctx.Comment.Body, // Preserve original comment
			SourceReviewID:  ctx.SourceReview.ID,
			SourceCommentID: ctx.Comment.ID,
			File:            ctx.Comment.File,
			Line:            ctx.Comment.Line,
			Status:          initialStatus, // Use AI-assigned status (may be empty)
			TaskIndex:       i,
			URL:             ctx.Comment.URL,
		}
		fullTasks = append(fullTasks, fullTask)
	}

	return fullTasks, nil
}

// consolidateTasksIfNeeded merges multiple tasks from the same comment into a single comprehensive task
func (a *Analyzer) consolidateTasksIfNeeded(tasks []SimpleTaskRequest) []SimpleTaskRequest {
	// Skip consolidation in test mode if requested
	if os.Getenv("REVIEWTASK_TEST_NO_CONSOLIDATE") == "true" {
		return tasks
	}

	// If only one task, no consolidation needed
	if len(tasks) <= 1 {
		return tasks
	}

	// Log the consolidation attempt
	if a.config.AISettings.VerboseMode {
		fmt.Printf("  🔄 Consolidating %d tasks from single comment into unified task\n", len(tasks))
	}

	// Find the highest priority and most restrictive status
	highestPriority := "low"
	priorityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
	consolidatedStatus := "" // Empty by default to allow fallback logic

	for _, task := range tasks {
		if priorityOrder[task.Priority] > priorityOrder[highestPriority] {
			highestPriority = task.Priority
		}
		// Use explicit status from any source task
		if task.InitialStatus != "" {
			// If any task is pending, the consolidated task should be pending
			if task.InitialStatus == "pending" {
				consolidatedStatus = "pending"
			} else if consolidatedStatus == "" {
				// Use first non-empty status found
				consolidatedStatus = task.InitialStatus
			}
		}
	}

	// Combine all descriptions into a comprehensive task
	var descriptions []string
	for _, task := range tasks {
		descriptions = append(descriptions, task.Description)
	}

	// Create unified description
	unifiedDescription := a.createUnifiedTaskDescription(descriptions)

	// Return single consolidated task
	consolidatedTask := SimpleTaskRequest{
		Description:   unifiedDescription,
		Priority:      highestPriority,
		InitialStatus: consolidatedStatus,
	}

	if a.config.AISettings.VerboseMode {
		fmt.Printf("  ✅ Consolidated into single task: %s\n", unifiedDescription)
	}

	return []SimpleTaskRequest{consolidatedTask}
}

// createUnifiedTaskDescription combines multiple task descriptions into a coherent single description
func (a *Analyzer) createUnifiedTaskDescription(descriptions []string) string {
	if len(descriptions) == 1 {
		return descriptions[0]
	}

	// Try to identify the main action and combine related sub-actions
	mainAction := descriptions[0]

	// For configuration/environment issues, create unified configuration task (check first)
	if a.containsConfigurationKeywords(descriptions) {
		return a.unifyConfigurationTasks(descriptions)
	}

	// For code duplication issues, create unified refactoring task
	if a.containsCodeDuplicationKeywords(descriptions) {
		return a.unifyCodeDuplicationTasks(descriptions)
	}

	// For missing implementation issues, create unified implementation task
	if a.containsImplementationKeywords(descriptions) {
		return a.unifyImplementationTasks(descriptions)
	}

	// Default: combine all descriptions with context
	return fmt.Sprintf("%s (including all related implementation steps)", mainAction)
}

// Helper methods for task unification patterns
func (a *Analyzer) containsCodeDuplicationKeywords(descriptions []string) bool {
	keywords := []string{"duplicate", "refactor", "delegate", "shared", "centralize"}
	for _, desc := range descriptions {
		lowerDesc := strings.ToLower(desc)
		for _, keyword := range keywords {
			if strings.Contains(lowerDesc, keyword) {
				return true
			}
		}
	}
	return false
}

func (a *Analyzer) containsImplementationKeywords(descriptions []string) bool {
	keywords := []string{"implement", "add", "create", "missing", "function"}
	for _, desc := range descriptions {
		lowerDesc := strings.ToLower(desc)
		for _, keyword := range keywords {
			if strings.Contains(lowerDesc, keyword) {
				return true
			}
		}
	}
	return false
}

func (a *Analyzer) containsConfigurationKeywords(descriptions []string) bool {
	keywords := []string{"environment", "config", "variable", "setting", "override"}
	for _, desc := range descriptions {
		lowerDesc := strings.ToLower(desc)
		for _, keyword := range keywords {
			if strings.Contains(lowerDesc, keyword) {
				return true
			}
		}
	}
	return false
}

func (a *Analyzer) unifyCodeDuplicationTasks(descriptions []string) string {
	// Extract the main component being refactored
	mainComponent := "component"
	for _, desc := range descriptions {
		if strings.Contains(strings.ToLower(desc), "cursor") {
			mainComponent = "CursorClient"
			break
		}
	}

	return fmt.Sprintf("Remove code duplication by refactoring %s to use shared implementation", mainComponent)
}

func (a *Analyzer) unifyImplementationTasks(descriptions []string) string {
	// Find the main thing being implemented
	for _, desc := range descriptions {
		if strings.Contains(strings.ToLower(desc), "implement") || strings.Contains(strings.ToLower(desc), "add") {
			// Use the first implementation task as the main description
			return fmt.Sprintf("%s with comprehensive implementation", desc)
		}
	}
	return "Implement missing functionality with all required components"
}

func (a *Analyzer) unifyConfigurationTasks(descriptions []string) string {
	// Find environment or config related task
	for _, desc := range descriptions {
		if strings.Contains(strings.ToLower(desc), "environment") || strings.Contains(strings.ToLower(desc), "config") {
			return fmt.Sprintf("%s with proper configuration handling", desc)
		}
	}
	return "Update configuration with all required settings"
}

// processComment handles a single comment and returns tasks for it
func (a *Analyzer) processComment(ctx CommentContext) ([]TaskRequest, error) {
	// Use simplified processing for better reliability
	return a.processCommentSimple(ctx)
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

// processLargeCommentWithSummarization handles large comments by summarizing them
func (a *Analyzer) processLargeCommentWithSummarization(ctx CommentContext) ([]TaskRequest, error) {
	if a.config.AISettings.VerboseMode {
		fmt.Printf("  📝 Large comment detected (ID: %d, size: %d bytes), summarizing...\n",
			ctx.Comment.ID, len(ctx.Comment.Body))
	}

	// Create content summarizer
	summarizer := NewContentSummarizer(18000, a.config.AISettings.VerboseMode) // 18KB to leave room for prompt

	// Summarize the comment
	summarizedComment := summarizer.SummarizeComment(ctx.Comment)

	// Create new context with summarized comment
	summarizedCtx := CommentContext{
		Comment:      summarizedComment,
		SourceReview: ctx.SourceReview,
	}

	// Process the summarized comment
	var tasks []TaskRequest
	var err error

	if a.config.AISettings.ValidationEnabled != nil && *a.config.AISettings.ValidationEnabled {
		tasks, err = a.processCommentWithValidation(summarizedCtx)
	} else {
		prompt := a.buildCommentPrompt(summarizedCtx)
		tasks, err = a.callClaudeCode(prompt)
	}

	if err != nil {
		// If summarization failed, fall back to chunking
		if a.config.AISettings.VerboseMode {
			fmt.Printf("    ⚠️ Summarization failed, falling back to chunking: %v\n", err)
		}
		chunker := NewCommentChunker(20000)
		return a.processLargeComment(ctx, chunker)
	}

	if a.config.AISettings.VerboseMode {
		fmt.Printf("  ✅ Successfully processed summarized comment: %d tasks generated\n", len(tasks))
	}

	return tasks, nil
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

	// Create validator with same AI provider as analyzer
	validator := NewTaskValidatorWithClient(a.config, a.claudeClient)

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

// buildSimpleCommentPromptWithTags creates a prompt using XML tags for clearer response format
func (a *Analyzer) buildSimpleCommentPromptWithTags(ctx CommentContext) string {
	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" && a.config.AISettings.UserLanguage != "English" {
		languageInstruction = fmt.Sprintf("Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage)
	}

	prompt := fmt.Sprintf(`You are a GitHub PR review assistant that extracts actionable tasks from comments.

%sGenerate 0 to N tasks from the following comment. Return empty array if no action is needed.

## Examples:

Comment: "This function lacks error handling. Add nil check and error logging."
<response>
[
  {"description": "Add nil check to function", "priority": "high"},
  {"description": "Implement error logging", "priority": "medium"}
]
</response>

Comment: "LGTM! Great implementation."
<response>
[]
</response>

Comment: "Missing timeout handling. Add 30 second timeout. URGENT."
<response>
[
  {"description": "Implement 30 second timeout handling", "priority": "critical"}
]
</response>

Priority levels: critical (security/data loss), high (bugs/performance), medium (improvements), low (style/naming)

## Now analyze this comment:

File: %s:%d
Author: %s
Comment:
%s

Provide your response in <response> tags. Include ONLY the JSON array:
<response>
`, languageInstruction, ctx.Comment.File, ctx.Comment.Line, ctx.Comment.Author, ctx.Comment.Body)

	return prompt
}

// loadPromptTemplate loads a prompt template from the prompts directory
func (a *Analyzer) loadPromptTemplate(filename string, data interface{}) (string, error) {
	// Try multiple locations for the prompt file
	possiblePaths := []string{
		fmt.Sprintf("prompts/%s", filename),
		fmt.Sprintf("./prompts/%s", filename),
		fmt.Sprintf("../../prompts/%s", filename), // For tests running from subdirectories
		fmt.Sprintf("../prompts/%s", filename),    // Alternative relative path
	}

	var templateContent []byte
	var err error
	var foundPath string

	for _, path := range possiblePaths {
		templateContent, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if templateContent == nil {
		return "", fmt.Errorf("prompt template %s not found in any location", filename)
	}

	// Parse and execute the template
	tmpl, err := template.New(filepath.Base(foundPath)).Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// buildSimpleCommentPromptFromTemplate builds prompt using external template file
func (a *Analyzer) buildSimpleCommentPromptFromTemplate(ctx CommentContext) string {
	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" && a.config.AISettings.UserLanguage != "English" {
		languageInstruction = fmt.Sprintf("Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage)
	}

	// Prepare template data
	data := struct {
		LanguageInstruction string
		File                string
		Line                int
		Author              string
		Comment             string
	}{
		LanguageInstruction: languageInstruction,
		File:                ctx.Comment.File,
		Line:                ctx.Comment.Line,
		Author:              ctx.Comment.Author,
		Comment:             ctx.Comment.Body,
	}

	// Try to load from template file
	prompt, err := a.loadPromptTemplate("simple_task_generation.md", data)
	if err != nil {
		// Fall back to hardcoded prompt if template not found
		if a.config.AISettings.VerboseMode {
			fmt.Printf("  ⚠️  Failed to load prompt template: %v, using fallback\n", err)
		}
		return a.buildSimpleCommentPrompt(ctx)
	}

	return prompt
}

// buildSimpleCommentPrompt creates a minimal prompt for AI to generate only description and priority
func (a *Analyzer) buildSimpleCommentPrompt(ctx CommentContext) string {
	var languageInstruction string
	if a.config.AISettings.UserLanguage != "" && a.config.AISettings.UserLanguage != "English" {
		languageInstruction = fmt.Sprintf("Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage)
	}

	prompt := fmt.Sprintf("You are a GitHub PR review assistant that extracts actionable tasks from comments.\n\n"+
		"%sGenerate 0 to N tasks from the following comment. Return empty array if no action is needed.\n\n"+
		"## Examples:\n\n"+
		"Comment: \"This function lacks error handling. Add nil check and error logging.\"\n"+
		"Response:\n"+
		"```json\n"+
		"[\n"+
		"  {\"description\": \"Add nil check to function\", \"priority\": \"high\"},\n"+
		"  {\"description\": \"Implement error logging\", \"priority\": \"medium\"}\n"+
		"]\n"+
		"```\n\n"+
		"Comment: \"LGTM! Great implementation.\"\n"+
		"Response:\n"+
		"```json\n"+
		"[]\n"+
		"```\n\n"+
		"Comment: \"Missing timeout handling. Add 30 second timeout. URGENT.\"\n"+
		"Response:\n"+
		"```json\n"+
		"[\n"+
		"  {\"description\": \"Implement 30 second timeout handling\", \"priority\": \"critical\"}\n"+
		"]\n"+
		"```\n\n"+
		"Priority levels: critical (security/data loss), high (bugs/performance), medium (improvements), low (style/naming)\n\n"+
		"## Now analyze this comment:\n\n"+
		"File: %s:%d\n"+
		"Author: %s\n"+
		"Comment:\n%s\n\n"+
		"## Your response:\n"+
		"Return ONLY the JSON array below. No explanations, no markdown wrapper, just the raw JSON:\n",
		languageInstruction, ctx.Comment.File, ctx.Comment.Line, ctx.Comment.Author, ctx.Comment.Body)

	return prompt
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
		"url":               ctx.Comment.URL,
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
    "task_index": 0,
    "url": "%s"
  }
]`, ctx.SourceReview.ID, ctx.Comment.ID, ctx.Comment.File, ctx.Comment.Line, ctx.Comment.URL))
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

IMPORTANT: You MUST return ONLY a JSON array with NO markdown formatting, NO code blocks, NO backticks.
Return ONLY the raw JSON array, nothing else.

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
	// Use stream processor if enabled, otherwise use traditional parallel processing
	if a.config.AISettings.StreamProcessingEnabled {
		streamProcessor := NewStreamProcessor(a)
		tasks, err := streamProcessor.ProcessCommentsStream(comments, a.processCommentWithValidation)
		if err == nil && a.config.AISettings.VerboseMode {
			fmt.Printf("✓ Generated %d tasks from %d comments with validation (stream mode)\n", len(tasks), len(comments))
		}
		return tasks, err
	}

	// Traditional parallel processing
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
	// Check GitHub thread resolved status first
	// This is the authoritative source of truth from GitHub API
	if comment.GitHubThreadResolved {
		return true
	}

	// Fallback: Check for common resolution markers in the comment body or replies
	// This handles cases where reviews.json hasn't been updated yet
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

// isNitpickOnlyReview checks if a review body only contains nitpick content and no actionable tasks
func (a *Analyzer) isNitpickOnlyReview(body string) bool {
	lowerBody := strings.ToLower(body)

	// Check for CodeRabbit nitpick-only pattern
	hasActionableZero := strings.Contains(lowerBody, "actionable comments posted: 0")
	hasNitpickSection := strings.Contains(lowerBody, "nitpick comments") ||
		strings.Contains(lowerBody, "🧹")

	// If it has "Actionable comments posted: 0" and nitpick sections, it's likely nitpick-only
	if hasActionableZero && hasNitpickSection {
		// Check if there's any other significant content outside nitpick sections
		// Remove the nitpick section to see if there's other content
		beforeNitpick := strings.Split(body, "<details>")[0]
		// Remove the "Actionable comments posted: 0" line
		beforeNitpick = strings.ReplaceAll(beforeNitpick, "**Actionable comments posted: 0**", "")
		beforeNitpick = strings.TrimSpace(beforeNitpick)

		// If there's no other content, it's nitpick-only
		return beforeNitpick == ""
	}

	return false
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
		pi := priorityOrder[sorted[i].Priority]
		pj := priorityOrder[sorted[j].Priority]

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

// ============================================================================
// Batch Processing with Existing Task Awareness
// ============================================================================

// BatchTaskResponse represents AI response for a single comment
type BatchTaskResponse struct {
	CommentID int64 `json:"comment_id"`
	Tasks     []struct {
		Description string `json:"description"`
		Priority    string `json:"priority"`
	} `json:"tasks"`
}

// buildBatchPromptWithExistingTasks creates a single markdown prompt for all comments
func (a *Analyzer) buildBatchPromptWithExistingTasks(
	comments []CommentContext,
	existingTasksByComment map[int64][]storage.Task,
) string {
	var buf strings.Builder

	// Header
	buf.WriteString("# PR Review Task Generation Request\n\n")
	buf.WriteString("You are a GitHub PR review assistant. Analyze multiple review comments and generate necessary tasks for each comment.\n\n")

	// Instructions
	buf.WriteString("## Important Instructions\n\n")
	buf.WriteString("1. **Avoid duplicates with existing tasks**\n")
	buf.WriteString("   - Each comment lists its existing tasks\n")
	buf.WriteString("   - If existing tasks are sufficient, return empty array `[]`\n")
	buf.WriteString("   - Semantically identical tasks (different wording) are duplicates\n")
	buf.WriteString("   - Example: \"Fix memory leak\" and \"Fix the memory leak\" are the same\n\n")

	buf.WriteString("2. **Generate only new tasks**\n")
	buf.WriteString("   - Add only truly missing tasks\n")
	buf.WriteString("   - Priorities: critical (security/data loss), high (bugs/performance), medium (improvements), low (style/naming)\n\n")

	// Language instruction
	if a.config.AISettings.UserLanguage != "" && a.config.AISettings.UserLanguage != "English" {
		buf.WriteString(fmt.Sprintf("3. **Language**: Generate all task descriptions in %s\n\n", a.config.AISettings.UserLanguage))
	}

	buf.WriteString("---\n\n")

	// Process each comment
	for i, ctx := range comments {
		buf.WriteString(fmt.Sprintf("## Comment #%d: %d\n\n", i+1, ctx.Comment.ID))
		buf.WriteString(fmt.Sprintf("**File:** %s", ctx.Comment.File))
		if ctx.Comment.Line > 0 {
			buf.WriteString(fmt.Sprintf(":%d", ctx.Comment.Line))
		}
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("**Author:** %s\n", ctx.Comment.Author))
		buf.WriteString(fmt.Sprintf("**Content:**\n```\n%s\n```\n\n", ctx.Comment.Body))

		// Existing tasks
		existingTasks := existingTasksByComment[ctx.Comment.ID]
		buf.WriteString(fmt.Sprintf("### Existing Tasks for Comment #%d\n\n", i+1))

		if len(existingTasks) == 0 {
			buf.WriteString("*No existing tasks for this comment.*\n\n")
		} else {
			for j, task := range existingTasks {
				statusIcon := getStatusIcon(task.Status)
				buf.WriteString(fmt.Sprintf("%d. %s **%s** - %s\n",
					j+1, statusIcon, strings.ToUpper(task.Status), task.Description))
				buf.WriteString(fmt.Sprintf("   - Priority: %s\n", task.Priority))
				if task.Status == "cancel" && task.CancelReason != "" {
					buf.WriteString(fmt.Sprintf("   - Cancelled: %s\n", task.CancelReason))
				}
				buf.WriteString("\n")
			}
		}

		buf.WriteString("---\n\n")
	}

	// Footer - Response format
	buf.WriteString("# Response Format\n\n")
	buf.WriteString("Return a JSON array with responses for ALL comments above.\n")
	buf.WriteString("For each comment, return only NEW tasks that don't duplicate existing ones.\n\n")
	buf.WriteString("```json\n")
	buf.WriteString("[\n")
	buf.WriteString("  {\n")
	buf.WriteString("    \"comment_id\": 12345,\n")
	buf.WriteString("    \"tasks\": [\n")
	buf.WriteString("      {\"description\": \"Task description\", \"priority\": \"high\"},\n")
	buf.WriteString("      {\"description\": \"Another task\", \"priority\": \"medium\"}\n")
	buf.WriteString("    ]\n")
	buf.WriteString("  },\n")
	buf.WriteString("  {\n")
	buf.WriteString("    \"comment_id\": 67890,\n")
	buf.WriteString("    \"tasks\": []\n")
	buf.WriteString("  }\n")
	buf.WriteString("]\n")
	buf.WriteString("```\n\n")
	buf.WriteString("Return ONLY the JSON array. No explanations, no markdown wrapper, just the raw JSON.\n")

	return buf.String()
}

// getStatusIcon returns an emoji icon for task status
func getStatusIcon(status string) string {
	switch status {
	case "done":
		return "✅"
	case "doing":
		return "🔄"
	case "todo":
		return "📝"
	case "pending":
		return "⏸️"
	case "cancel":
		return "❌"
	default:
		return "•"
	}
}

// parseBatchTaskResponse parses AI response into structured data
func (a *Analyzer) parseBatchTaskResponse(response string) ([]BatchTaskResponse, error) {
	// Clean up response (remove markdown wrapper if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var batchResponses []BatchTaskResponse
	if err := json.Unmarshal([]byte(response), &batchResponses); err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	return batchResponses, nil
}

// ProcessCommentsWithBatchAI processes all comments in a single AI call with existing task context
func (a *Analyzer) ProcessCommentsWithBatchAI(
	comments []CommentContext,
	storageManager *storage.Manager,
	prNumber int,
) ([]storage.Task, error) {
	if a.config.AISettings.VerboseMode {
		fmt.Printf("🔍 Processing %d comments with batch AI (existing task awareness)\n", len(comments))
	}

	// Load existing tasks
	existingTasks, err := storageManager.GetTasksByPR(prNumber)
	if err != nil && a.config.AISettings.VerboseMode {
		fmt.Printf("⚠️  Could not load existing tasks: %v\n", err)
	}

	// Group existing tasks by comment ID
	existingTasksByComment := make(map[int64][]storage.Task)
	for _, task := range existingTasks {
		if task.SourceCommentID != 0 {
			existingTasksByComment[task.SourceCommentID] = append(
				existingTasksByComment[task.SourceCommentID],
				task,
			)
		}
	}

	if a.config.AISettings.VerboseMode {
		fmt.Printf("📋 Loaded %d existing tasks from %d comments\n",
			len(existingTasks), len(existingTasksByComment))
	}

	// Build batch prompt
	batchPrompt := a.buildBatchPromptWithExistingTasks(comments, existingTasksByComment)

	if a.config.AISettings.VerboseMode {
		fmt.Printf("📝 Generated batch prompt: %d chars\n", len(batchPrompt))
	}

	// Call AI once for all comments
	response, err := a.callClaudeForBatchTasks(batchPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call AI for batch processing: %w", err)
	}

	// Parse response
	batchResponses, err := a.parseBatchTaskResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	if a.config.AISettings.VerboseMode {
		totalNewTasks := 0
		for _, resp := range batchResponses {
			totalNewTasks += len(resp.Tasks)
		}
		fmt.Printf("✅ AI generated %d new tasks across %d comments\n",
			totalNewTasks, len(batchResponses))
	}

	// Convert to TaskRequest objects
	var allTaskRequests []TaskRequest
	commentMap := make(map[int64]CommentContext)
	for _, ctx := range comments {
		commentMap[ctx.Comment.ID] = ctx
	}

	for _, batchResp := range batchResponses {
		ctx, exists := commentMap[batchResp.CommentID]
		if !exists {
			if a.config.AISettings.VerboseMode {
				fmt.Printf("⚠️  Comment ID %d not found in original comments\n", batchResp.CommentID)
			}
			continue
		}

		for i, taskData := range batchResp.Tasks {
			taskReq := TaskRequest{
				Description:     taskData.Description,
				Priority:        taskData.Priority,
				OriginText:      ctx.Comment.Body,
				SourceReviewID:  ctx.SourceReview.ID,
				SourceCommentID: ctx.Comment.ID,
				File:            ctx.Comment.File,
				Line:            ctx.Comment.Line,
				TaskIndex:       i,
				URL:             ctx.Comment.URL,
			}
			allTaskRequests = append(allTaskRequests, taskReq)
		}
	}

	// Convert to storage tasks
	storageTasks := a.convertToStorageTasks(allTaskRequests)

	if a.config.AISettings.VerboseMode {
		fmt.Printf("✅ Converted to %d storage tasks\n", len(storageTasks))
	}

	return storageTasks, nil
}

// callClaudeForBatchTasks calls Claude with batch prompt
func (a *Analyzer) callClaudeForBatchTasks(prompt string) (string, error) {
	// Reuse existing Claude Code CLI integration
	return a.callClaudeCodeWithPrompt(prompt)
}

// callClaudeCodeWithPrompt calls Claude and returns raw response (for batch processing)
func (a *Analyzer) callClaudeCodeWithPrompt(prompt string) (string, error) {
	ctx := context.Background()
	
	// Use existing Claude client
	response, err := a.claudeClient.Execute(ctx, prompt, "text")
	if err != nil {
		return "", fmt.Errorf("failed to execute Claude: %w", err)
	}
	
	return response, nil
}
