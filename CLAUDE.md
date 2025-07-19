## Project Language

- Memories and documents should be written in English
- Conversations should be conducted in the language specified by the user

# AI-Powered PR Review Management Tool

## Project Overview

This project implements `gh-review-task`, a CLI tool that fetches GitHub Pull Request reviews, analyzes them using AI, and generates actionable tasks for developers with advanced parallel processing and task state management.

## Current Implementation Status

### Core System Architecture
- **Enhanced AI Analysis**: Claude Code integration with parallel processing and UUID-based task IDs
- **Task State Preservation**: Comprehensive task merging and state management system
- **Parallel Processing**: Goroutine-based concurrent comment processing
- **Local Storage**: Structured JSON persistence with task lifecycle management
- **Configuration Management**: Fixed boolean field merging with validation modes

### Recent Major Enhancements (Latest Implementation)
- **UUID-based Task IDs**: Eliminated duplication issues using `github.com/google/uuid`
- **Parallel Comment Processing**: Each comment processed independently in goroutines
- **Task Merging System**: Preserves existing task statuses during subsequent runs
- **Comment Change Detection**: Automatic task cancellation for content changes
- **Individual JSON Validation**: Per-comment validation in parallel processing mode
- **Boolean Config Fix**: Resolved `validation_enabled: false` override issue

### Command Structure
```
gh-review-task [PR_NUMBER]     # Analyze PR reviews and generate tasks
gh-review-task status          # Show current task status and statistics  
gh-review-task update <id> <status>  # Update task status
gh-review-task init            # Initialize repository
gh-review-task auth <cmd>      # Authentication management
```

## Detailed Workflow Documentation

### 1. Review Fetching Workflow

```bash
# Initialize repository (one-time setup)
./gh-review-task init

# Authenticate with GitHub
./gh-review-task auth login

# Fetch and analyze PR reviews
./gh-review-task [PR_NUMBER]
```

**Internal Process:**
1. **Authentication Check**: Multi-source token detection (env → local → gh CLI)
2. **Repository Validation**: Check GitHub API access and permissions
3. **PR Data Fetching**: Retrieve PR info, reviews, and nested comments
4. **Parallel Processing**: Each comment processed in separate goroutine
5. **AI Analysis**: Claude Code generates tasks with priority assignment
6. **Task Merging**: Combine new tasks with existing ones (preserves statuses)
7. **Data Persistence**: Save to `.pr-review/PR-{number}/` directory

### 2. Task Execution Workflow

**Generated Task Structure:**
```json
{
  "id": "uuid-v4-string",
  "description": "Actionable task description",
  "origin_text": "Original review comment text",
  "priority": "critical|high|medium|low",
  "source_review_id": 12345,
  "source_comment_id": 67890,
  "file": "src/file.go",
  "line": 42,
  "status": "todo",
  "created_at": "2025-01-19T12:00:00Z",
  "updated_at": "2025-01-19T12:00:00Z"
}
```

**Task Execution Steps:**
1. **View Tasks**: `./gh-review-task status` - Shows all tasks with current statuses
2. **Start Work**: `./gh-review-task update <task-uuid> doing` - Mark task as in progress
3. **Complete Work**: `./gh-review-task update <task-uuid> done` - Mark as completed
4. **Cancel Task**: `./gh-review-task update <task-uuid> cancel` - Cancel if not needed

### 3. Status Update and Management Workflow

**Status Management Commands:**
```bash
# View overall status across all PRs
./gh-review-task status

# Update specific task status
./gh-review-task update <task-uuid> <new-status>

# Valid statuses: todo, doing, done, pending, cancel
```

**Status Lifecycle:**
- `todo` → `doing` → `done` (normal completion)
- `todo` → `cancel` (task no longer needed)
- `doing` → `pending` (blocked/waiting)
- `pending` → `doing` (unblocked, resume work)

### 4. Subsequent Review Fetches

**Task State Preservation Logic:**
1. **Load Existing Tasks**: Read current tasks from `tasks.json`
2. **Generate New Tasks**: Process current comments with AI
3. **Comment Comparison**: Check for content changes in each comment
4. **Merge Strategy**:
   - **No existing tasks**: Add all new tasks
   - **No new tasks**: Cancel existing non-completed tasks
   - **Content unchanged**: Preserve existing task statuses, add new tasks if more generated
   - **Content changed**: Cancel existing tasks, add new tasks
5. **Save Merged Results**: Update `tasks.json` with preserved statuses

## Technical Architecture Details

### Data Storage Structure
```
.pr-review/
├── config.json              # Priority rules and project settings
├── auth.json                # Local authentication (gitignored)
└── PR-<number>/
    ├── info.json            # PR metadata
    ├── reviews.json         # Review data with nested comments
    └── tasks.json           # AI-generated tasks
```

### Key Components
- **GitHub API Client**: Repository detection, PR/review fetching, permission checking
- **AI Analyzer**: Claude Code integration for contextual task generation  
- **Storage Manager**: JSON-based local data persistence with task merging
- **Config System**: Priority rules with project-specific overrides
- **Setup Manager**: Repository initialization and git integration

### Parallel Processing Implementation
```go
// Each comment processed in separate goroutine
func (a *Analyzer) generateTasksParallel(comments []CommentContext) ([]storage.Task, error) {
    results := make(chan commentResult, len(comments))
    var wg sync.WaitGroup
    
    for i, commentCtx := range comments {
        wg.Add(1)
        go func(index int, ctx CommentContext) {
            defer wg.Done()
            tasks, err := a.processComment(ctx)
            results <- commentResult{tasks: tasks, err: err, index: index}
        }(i, commentCtx)
    }
    // ... collection and error handling
}
```

### Task Merging Strategy
- **By Comment ID**: Tasks grouped by `source_comment_id`
- **Content Detection**: Compare `origin_text` for significant changes
- **Status Preservation**: Maintain `doing`, `done`, `pending` statuses
- **Automatic Cancellation**: Mark outdated tasks as `cancelled`
- **UUID Consistency**: Preserve existing task IDs during merges

### Configuration Modes
- **Parallel Mode** (`validation_enabled: false`): Fast, concurrent processing
- **Validation Mode** (`validation_enabled: true`): Two-stage validation with retry logic
- **Debug Mode** (`debug_mode: true`): Detailed logging and response previews

## Key File Locations and Responsibilities

### Core Implementation Files
- **`cmd/root.go`**: Main command logic, calls `MergeTasks()` instead of `SaveTasks()`
- **`internal/ai/analyzer.go`**: Parallel processing, UUID generation, Claude integration
- **`internal/storage/manager.go`**: Task merging logic, state preservation, change detection
- **`internal/config/config.go`**: Boolean field merge fix for configuration
- **`.pr-review/config.json`**: Project settings with `validation_enabled: false`

### Data Persistence
- **PR Info**: `.pr-review/PR-{number}/info.json`
- **Reviews**: `.pr-review/PR-{number}/reviews.json`
- **Tasks**: `.pr-review/PR-{number}/tasks.json`
- **Config**: `.pr-review/config.json`
- **Auth**: `.pr-review/auth.json` (gitignored)

## Performance Characteristics

### Current Optimization Results
- **Prompt Size Reduction**: 57,760 → 3,000-6,000 characters per comment
- **Parallel Processing**: 12 comments processed concurrently
- **Response Time**: Improved Claude Code reliability and speed
- **Memory Usage**: Efficient goroutine management with WaitGroup synchronization
- **Error Handling**: Partial success scenarios, graceful degradation

## Testing Coverage

### Verified Workflows
- ✅ **Initial Setup**: Repository initialization and authentication
- ✅ **Parallel Processing**: Multiple comments processed concurrently
- ✅ **Task Generation**: UUID-based IDs with proper structure
- ✅ **State Preservation**: Existing task statuses maintained
- ✅ **Comment Changes**: Automatic task cancellation and replacement
- ✅ **Configuration**: Boolean field handling and validation modes
- ✅ **Error Scenarios**: Partial failures, network issues, permission problems

### Latest Test Results
- **Comments Processed**: 12 comments in parallel
- **Tasks Generated**: 16 tasks with UUID IDs
- **Processing Mode**: Parallel processing with individual validation
- **Merge Success**: Existing task IDs preserved during subsequent runs
- **Performance**: Significant improvement in processing speed and reliability

## Development Patterns

### Error Handling
- Graceful degradation with clear user guidance
- Specific error messages with actionable remediation steps
- Progressive authentication checking with fallback options

### User Experience
- Self-guided setup with minimal manual configuration
- Consistent CLI patterns following `gh` conventions  
- Progressive disclosure of advanced features

### Security Considerations
- Token stored with restricted file permissions (600)
- Sensitive data excluded from git tracking
- Clear separation of local vs global authentication
