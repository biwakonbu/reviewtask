# Claude Code One-Shot Integration Implementation Plan

## Requirements Analysis

### Information Preservation and User Language Support

**Key Requirements:**
1. **Information Integrity**: Preserve original review text to prevent data loss during AI translation/interpretation
2. **Dual Text Structure**: Store both `origin_text` (original review content) and `task_description` (AI-generated actionable task)
3. **User Language Configuration**: Allow users to specify preferred language for task descriptions
4. **Format Validation**: Implement robust validation with fallback mechanisms for malformed AI responses

### Current State Analysis

The existing `gh-review-task` tool currently uses a **placeholder implementation** for Claude Code integration:

```go
// Line 134 in internal/ai/analyzer.go
cmd := exec.Command("claude", "code", "--input", tempFile, "--output", "json")
```

This approach has several limitations:
1. Uses incorrect command syntax
2. Relies on external CLI command instead of proper SDK integration
3. Has fallback to dummy data when Claude Code is unavailable
4. Uses temporary files instead of direct prompt passing
5. **Missing origin text preservation**
6. **No user language configuration**

## Enhanced Claude Code Integration Plan

### 1. Command Line Integration (Immediate Implementation)

Based on the SDK documentation, Claude Code supports proper one-shot mode with these options:

**Current Incorrect Usage:**
```bash
claude code --input tempFile --output json
```

**Correct One-Shot Usage:**
```bash
claude -p "prompt text" --output-format json
```

### 2. Enhanced Data Structure Design

#### Updated Task Structure
```go
type TaskRequest struct {
    Description      string `json:"description"`        // AI-generated task description (user language)
    OriginText       string `json:"origin_text"`        // Original review comment text
    Priority         string `json:"priority"`
    SourceReviewID   int64  `json:"source_review_id"`
    SourceCommentID  int64  `json:"source_comment_id"`  // Required: specific comment ID
    File             string `json:"file"`
    Line             int    `json:"line"`
    Status           string `json:"status"`
    TaskIndex        int    `json:"task_index"`         // New: index within comment (0, 1, 2...)
}
```

#### Updated Storage Task Structure
```go
type Task struct {
    ID               string `json:"id"`                 // Format: "comment-{commentID}-task-{index}"
    Description      string `json:"description"`        // Display to user (user language)
    OriginText       string `json:"origin_text"`        // Original review text (for reference)
    Priority         string `json:"priority"`
    SourceReviewID   int64  `json:"source_review_id"`
    SourceCommentID  int64  `json:"source_comment_id"`  // Required: comment this task belongs to
    TaskIndex        int    `json:"task_index"`         // Index within the comment (for multiple tasks per comment)
    File             string `json:"file"`
    Line             int    `json:"line"`
    Status           string `json:"status"`
    CreatedAt        string `json:"created_at"`
    UpdatedAt        string `json:"updated_at"`
    PRNumber         int    `json:"pr_number"`
}
```

#### Comment Statistics Structure
```go
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
    PRNumber        int            `json:"pr_number"`
    GeneratedAt     string         `json:"generated_at"`
    TotalComments   int            `json:"total_comments"`
    TotalTasks      int            `json:"total_tasks"`
    CommentStats    []CommentStats `json:"comment_stats"`
    StatusSummary   StatusSummary  `json:"status_summary"`
}

type StatusSummary struct {
    Todo        int `json:"todo"`
    Doing       int `json:"doing"`
    Done        int `json:"done"`
    Pending     int `json:"pending"`
    Cancelled   int `json:"cancelled"`
}
```

### 3. Configuration Enhancement

#### Updated Config Structure
```go
type Config struct {
    PriorityRules    PriorityRules    `json:"priority_rules"`
    ProjectSpecific  ProjectSpecific  `json:"project_specific"`
    TaskSettings     TaskSettings     `json:"task_settings"`
    AISettings       AISettings       `json:"ai_settings"`     // New
}

type AISettings struct {
    UserLanguage      string  `json:"user_language"`       // e.g., "Japanese", "English"
    OutputFormat      string  `json:"output_format"`       // "json"
    MaxRetries        int     `json:"max_retries"`         // Validation retry attempts (default: 5)
    FallbackEnabled   bool    `json:"fallback_enabled"`    // Enable fallback to dummy tasks
    ValidationEnabled bool    `json:"validation_enabled"`  // Enable two-stage validation
    QualityThreshold  float64 `json:"quality_threshold"`   // Minimum score to accept (0.0-1.0)
}
```

### 4. Implementation Strategy

#### Option A: Enhanced CLI Integration (Recommended)
Replace the current `callClaudeCode` function with improved implementation:

```go
func (a *Analyzer) callClaudeCode(prompt string) ([]TaskRequest, error) {
    // Use proper Claude Code one-shot syntax
    cmd := exec.Command("claude", "-p", prompt, "--output-format", "json")
    output, err := cmd.Output()
    
    if err != nil {
        return nil, fmt.Errorf("claude code execution failed: %w", err)
    }
    
    // Parse and validate JSON response
    var tasks []TaskRequest
    if err := json.Unmarshal(output, &tasks); err != nil {
        // Try format correction before falling back
        if correctedTasks, corrErr := a.attemptFormatCorrection(output); corrErr == nil {
            return correctedTasks, nil
        }
        
        if a.config.AISettings.FallbackEnabled {
            return a.createFallbackTasks(), nil
        }
        return nil, fmt.Errorf("failed to parse claude response: %w", err)
    }
    
    // Validate required fields
    return a.validateAndEnrichTasks(tasks)
}
```

#### Option B: SDK Integration (Advanced)
For more robust integration, consider using Claude Code SDK directly:

1. **TypeScript SDK**: Requires Node.js runtime
2. **Python SDK**: Requires Python runtime  
3. **HTTP API**: Direct API calls

### 5. Enhanced Prompt Engineering

Update the prompt structure to include user language and origin text preservation:

```go
func (a *Analyzer) buildAnalysisPrompt(reviews []github.Review) string {
    var prompt strings.Builder
    
    // System instruction with language specification
    prompt.WriteString("You are an AI assistant helping to analyze GitHub PR reviews and generate actionable tasks.\n\n")
    
    // User language configuration
    if a.config.AISettings.UserLanguage != "" {
        prompt.WriteString(fmt.Sprintf("IMPORTANT: Generate task descriptions in %s language.\n", a.config.AISettings.UserLanguage))
    }
    
    // Priority rules based on project config
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
```

### 6. Enhanced Validation System

#### Two-Stage Validation Architecture

```go
type ValidationResult struct {
    IsValid     bool                `json:"is_valid"`
    Score       float64             `json:"score"`        // 0.0-1.0 quality score
    Issues      []ValidationIssue   `json:"issues"`
    Tasks       []TaskRequest       `json:"tasks"`
}

type ValidationIssue struct {
    Type        string `json:"type"`        // "format", "content", "missing", "incorrect"
    TaskIndex   int    `json:"task_index"`  // -1 for general issues
    Field       string `json:"field"`       // specific field with issue
    Description string `json:"description"` // human-readable issue description
    Severity    string `json:"severity"`    // "critical", "major", "minor"
}

type TaskValidator struct {
    config   *config.Config
    maxRetries int
}

func NewTaskValidator(cfg *config.Config) *TaskValidator {
    return &TaskValidator{
        config:     cfg,
        maxRetries: cfg.AISettings.MaxRetries,
    }
}
```

#### Stage 1: Format Validation (Mechanical)

```go
func (tv *TaskValidator) validateFormat(rawOutput []byte) (*ValidationResult, error) {
    result := &ValidationResult{
        IsValid: false,
        Score:   0.0,
        Issues:  []ValidationIssue{},
    }
    
    // Try to parse JSON
    cleaned := tv.cleanOutput(rawOutput)
    var tasks []TaskRequest
    if err := json.Unmarshal([]byte(cleaned), &tasks); err != nil {
        result.Issues = append(result.Issues, ValidationIssue{
            Type:        "format",
            TaskIndex:   -1,
            Field:       "json",
            Description: fmt.Sprintf("Invalid JSON format: %v", err),
            Severity:    "critical",
        })
        return result, nil
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

func (tv *TaskValidator) cleanOutput(rawOutput []byte) string {
    cleaned := string(rawOutput)
    
    // Remove common markdown artifacts
    cleaned = strings.ReplaceAll(cleaned, "```json", "")
    cleaned = strings.ReplaceAll(cleaned, "```", "")
    cleaned = strings.TrimSpace(cleaned)
    
    // Remove leading/trailing whitespace and control characters
    cleaned = strings.Trim(cleaned, " \t\n\r")
    
    return cleaned
}
```

#### Stage 2: Content Validation (AI-Powered)

```go
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
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("claude validation execution failed: %w", err)
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
```

#### Retry Mechanism with Quality Scoring

```go
func (a *Analyzer) GenerateTasksWithValidation(reviews []github.Review) ([]storage.Task, error) {
    validator := NewTaskValidator(a.config)
    var bestResult *ValidationResult
    var bestTasks []TaskRequest
    maxScore := 0.0
    
    for attempt := 1; attempt <= validator.maxRetries; attempt++ {
        fmt.Printf("🔄 Task generation attempt %d/%d...\n", attempt, validator.maxRetries)
        
        // Generate tasks
        prompt := a.buildAnalysisPrompt(reviews)
        tasks, err := a.callClaudeCode(prompt)
        if err != nil {
            fmt.Printf("  ❌ Generation failed: %v\n", err)
            continue
        }
        
        // Stage 1: Format validation
        formatResult, err := validator.validateFormat([]byte(fmt.Sprintf("%+v", tasks)))
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
        if contentResult.IsValid && contentResult.Score >= 0.8 {
            fmt.Printf("  ✅ Validation passed!\n")
            return a.convertToStorageTasks(formatResult.Tasks), nil
        }
        
        // If not valid, create improvement prompt for next iteration
        if attempt < validator.maxRetries {
            fmt.Printf("  🔧 Preparing improved prompt for next attempt...\n")
            // Add validation feedback to next prompt
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

func (a *Analyzer) addValidationFeedback(issues []ValidationIssue) {
    // Store validation feedback for next iteration
    // This would modify the prompt generation to include previous issues
    a.validationFeedback = issues
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

// Enhanced prompt building with feedback integration
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

// Update the main method to use feedback-enhanced prompts
func (a *Analyzer) callClaudeCodeWithRetry(reviews []github.Review, attempt int) ([]TaskRequest, error) {
    var prompt string
    if attempt == 1 {
        prompt = a.buildAnalysisPrompt(reviews)
    } else {
        prompt = a.buildAnalysisPromptWithFeedback(reviews)
    }
    
    return a.callClaudeCode(prompt)
}
```

### 7. Default Configuration Updates

Update the default configuration to include AI settings:

```go
func defaultConfig() *Config {
    return &Config{
        PriorityRules: PriorityRules{
            Critical: "Security vulnerabilities, authentication bypasses, data exposure risks",
            High:     "Performance bottlenecks, memory leaks, database optimization issues",
            Medium:   "Functional bugs, logic improvements, error handling",
            Low:      "Code style, naming conventions, comment improvements",
        },
        ProjectSpecific: ProjectSpecific{
            Critical: "",
            High:     "",
            Medium:   "",
            Low:      "",
        },
        TaskSettings: TaskSettings{
            DefaultStatus:  "todo",
            AutoPrioritize: true,
        },
        AISettings: AISettings{
            UserLanguage:     "English",
            OutputFormat:     "json",
            MaxRetries:       5,                    // Allow up to 5 validation attempts
            FallbackEnabled:  true,
            ValidationEnabled: true,                // Enable two-stage validation
            QualityThreshold: 0.8,                 // Minimum quality score to accept
        },
    }
}
```

### 5. Error Handling Improvements

Implement robust error handling for Claude Code integration:

```go
func (a *Analyzer) callClaudeCode(prompt string) ([]TaskRequest, error) {
    cmd := exec.Command("claude", "-p", prompt, "--output-format", "json")
    
    var stderr bytes.Buffer
    cmd.Stderr = &stderr
    
    output, err := cmd.Output()
    if err != nil {
        // Provide specific error guidance
        if strings.Contains(stderr.String(), "command not found") {
            return nil, fmt.Errorf("claude code not installed. Please install from: https://docs.anthropic.com/claude-code")
        }
        
        return nil, fmt.Errorf("claude code execution failed: %s", stderr.String())
    }
    
    // ... parsing logic
}
```

### 6. Testing Strategy

1. **Unit Tests**: Mock Claude Code responses
2. **Integration Tests**: Test with actual Claude Code installation
3. **Fallback Tests**: Ensure graceful degradation when Claude Code unavailable

### 7. Documentation Updates

Update `CLAUDE.md` and `README.md` to include:
- Claude Code installation requirements
- Configuration options
- Troubleshooting guide for common issues

### 8. Statistics and Analytics Implementation

#### Task Statistics Manager
```go
type StatisticsManager struct {
    storageManager *storage.Manager
}

func (sm *StatisticsManager) GenerateTaskStatistics(prNumber int) (*TaskStatistics, error) {
    tasks, err := sm.storageManager.GetTasksByPR(prNumber)
    if err != nil {
        return nil, err
    }
    
    // Group tasks by comment ID
    commentGroups := make(map[int64][]storage.Task)
    for _, task := range tasks {
        commentGroups[task.SourceCommentID] = append(commentGroups[task.SourceCommentID], task)
    }
    
    var commentStats []CommentStats
    statusSummary := StatusSummary{}
    
    for commentID, commentTasks := range commentGroups {
        stats := CommentStats{
            CommentID:  commentID,
            TotalTasks: len(commentTasks),
            File:       commentTasks[0].File,
            Line:       commentTasks[0].Line,
            OriginText: commentTasks[0].OriginText,
        }
        
        // Count by status
        for _, task := range commentTasks {
            switch task.Status {
            case "todo":
                stats.PendingTasks++
                statusSummary.Todo++
            case "doing":
                stats.InProgressTasks++
                statusSummary.Doing++
            case "done":
                stats.CompletedTasks++
                statusSummary.Done++
            case "pending":
                stats.PendingTasks++
                statusSummary.Pending++
            case "cancelled":
                stats.CancelledTasks++
                statusSummary.Cancelled++
            }
        }
        
        commentStats = append(commentStats, stats)
    }
    
    return &TaskStatistics{
        PRNumber:       prNumber,
        GeneratedAt:    time.Now().Format("2006-01-02T15:04:05Z"),
        TotalComments:  len(commentGroups),
        TotalTasks:     len(tasks),
        CommentStats:   commentStats,
        StatusSummary:  statusSummary,
    }, nil
}
```

#### Enhanced Storage Manager Methods
```go
// Add to storage/manager.go
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
```

#### Enhanced CLI Commands
```go
// Add new statistics command
var statsCmd = &cobra.Command{
    Use:   "stats [PR_NUMBER]",
    Short: "Show task statistics by comment",
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... get PR number logic ...
        
        statsManager := NewStatisticsManager(storageManager)
        stats, err := statsManager.GenerateTaskStatistics(prNumber)
        if err != nil {
            return err
        }
        
        // Display formatted statistics
        fmt.Printf("📊 Task Statistics for PR #%d\n\n", prNumber)
        fmt.Printf("Total Comments: %d\n", stats.TotalComments)
        fmt.Printf("Total Tasks: %d\n\n", stats.TotalTasks)
        
        fmt.Println("Status Summary:")
        fmt.Printf("  ✅ Done: %d\n", stats.StatusSummary.Done)
        fmt.Printf("  🔄 Doing: %d\n", stats.StatusSummary.Doing)
        fmt.Printf("  📋 Todo: %d\n", stats.StatusSummary.Todo)
        fmt.Printf("  ⏸️ Pending: %d\n", stats.StatusSummary.Pending)
        fmt.Printf("  ❌ Cancelled: %d\n\n", stats.StatusSummary.Cancelled)
        
        fmt.Println("By Comment:")
        for _, comment := range stats.CommentStats {
            fmt.Printf("  Comment #%d (%s:%d) - %d tasks\n", 
                comment.CommentID, comment.File, comment.Line, comment.TotalTasks)
            fmt.Printf("    Done: %d, Doing: %d, Todo: %d\n", 
                comment.CompletedTasks, comment.InProgressTasks, comment.PendingTasks)
        }
        
        return nil
    },
}
```

## Implementation Phases

### Phase 1: Enhanced Data Structure Updates (High Priority)
1. Update `TaskRequest` struct to include `OriginText`, `SourceCommentID`, and `TaskIndex`
2. Update `Task` struct to include new fields and ID format
3. Add `CommentStats` and `TaskStatistics` structures
4. Update `Config` struct to include `AISettings`
5. Modify default configuration to include AI settings

### Phase 2: Prompt Engineering Enhancement (High Priority)  
1. Update `buildAnalysisPrompt` to include user language specification
2. Add task splitting guidelines and requirements
3. Enhance format specification with `task_index` field
4. Add comment ID tracking for better source attribution
5. Include multiple-task-per-comment examples

### Phase 3: Claude Code CLI Integration (High Priority)
1. Fix CLI command syntax from `claude code` to `claude -p`
2. Update task generation to create proper IDs and indexes
3. Implement format validation with comment ID validation
4. Add robust error handling with specific guidance
5. Implement fallback mechanisms for malformed responses

### Phase 4: Statistics and Analytics (High Priority)
1. Implement `StatisticsManager` with comment-based analytics
2. Add `GetTasksByPR` and `GetTasksByComment` methods to storage
3. Create `stats` CLI command for comment-level statistics
4. Add task status summary and progress tracking
5. Implement comment completion rate calculation

### Phase 5: Enhanced CLI and User Experience (Medium Priority)
1. Add task update by comment ID and index
2. Update help text and documentation
3. Add validation for configuration values
4. Implement configuration migration for existing users
5. Add comment-focused status displays

### Phase 6: Testing and Validation (Medium Priority)
1. Unit tests for comment-based task grouping
2. Integration tests with multiple tasks per comment
3. Statistics generation testing
4. End-to-end testing with various review formats
5. Language-specific testing for task descriptions

## Breaking Changes

**Minimal Breaking Changes:**
- New fields in Task structure (backward compatible with JSON unmarshaling)
- New configuration section (defaults provided for existing configs)
- Updated CLI behavior (improved functionality, same interface)

## Dependencies

- Claude Code CLI must be installed and accessible in PATH
- Existing Go dependencies remain unchanged
- No additional runtime dependencies required

## Migration Strategy

1. **Existing Data**: Add migration logic to populate `OriginText` from existing `Description` for old tasks
2. **Configuration**: Merge new AI settings with existing configuration files
3. **User Guidance**: Provide clear upgrade instructions and language configuration steps

## Testing Requirements

### Pre-Implementation Testing:
1. Verify Claude Code CLI installation and syntax
2. Test one-shot command with sample prompts
3. Validate JSON output parsing with various response formats

### Post-Implementation Testing:
1. Test with various review comment formats and languages
2. Validate fallback mechanisms when Claude Code is unavailable
3. Test configuration migration from existing setups
4. Verify origin text preservation across different comment types

## Success Metrics

1. **Information Integrity**: Original review text preserved in all cases
2. **User Experience**: Task descriptions displayed in user's preferred language
3. **Reliability**: Robust handling of malformed AI responses with fallbacks
4. **Configurability**: Easy language and behavior configuration
5. **Performance**: No degradation in analysis speed with enhanced features

## Risk Mitigation

1. **Format Validation Failures**: Multiple retry mechanisms and format correction
2. **Language Support**: Fallback to English if specified language fails
3. **CLI Dependency**: Clear error messages and installation guidance
4. **Data Migration**: Safe migration with backup preservation of existing data