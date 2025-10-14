# Architecture Overview

This document provides a comprehensive overview of reviewtask's architecture, design decisions, and implementation details.

## Core Philosophy

reviewtask is built on several foundational principles:

### 1. Zero Feedback Loss Policy
- **Every actionable review comment must be captured and tracked**
- **No developer should need to manually track what needs to be done**
- **Review discussions should translate directly into work items**

### 2. State Preservation is Sacred
- **Developer work progress is never lost due to tool operations**
- **Task statuses reflect real work and must be preserved across all operations**
- **Tool should adapt to developer workflow, not force workflow changes**

### 3. AI-Assisted, Human-Controlled
- **AI provides intelligent task generation and prioritization**
- **Developers maintain full control over task status and workflow**
- **Automation reduces cognitive overhead without removing agency**

### 4. Simplicity Over Features
- **Core workflow should be immediately intuitive**
- **Advanced features are optional and discoverable**
- **CLI commands follow standard patterns and conventions**

## Technology Stack

### Core Technology Choices

#### Go Programming Language
- **Rationale**: CLI tools benefit from Go's single-binary distribution and cross-platform support
- **Benefits**: Fast compilation, excellent concurrency support, rich standard library
- **Rule**: All core functionality implemented in Go with minimal external dependencies

#### Claude Code CLI Integration
- **Rationale**: Provides best-in-class AI analysis while maintaining local control
- **Benefits**: Leverages Anthropic's advanced language models, no direct API management
- **Rule**: All AI processing goes through Claude Code CLI, no direct API calls

#### JSON-based Local Storage
- **Rationale**: Human-readable, git-trackable, and easily debuggable
- **Benefits**: Simple format, version control friendly, easy inspection and modification
- **Rule**: All data stored as structured JSON with clear schema

#### GitHub API Integration
- **Rationale**: Direct integration provides real-time data and comprehensive access
- **Benefits**: Official API, comprehensive access to PR and review data
- **Rule**: Multi-source authentication with fallback strategies

## Project Structure

```
reviewtask/
├── cmd/                    # CLI command implementations (Cobra pattern)
│   ├── auth.go            # Authentication management
│   ├── claude.go          # AI provider integration
│   ├── config.go          # Configuration management
│   ├── debug.go           # Debug and troubleshooting commands
│   ├── fetch.go           # Main PR analysis workflow
│   ├── root.go            # Root command and global flags
│   ├── show.go            # Task display and details
│   ├── stats.go           # Statistics and analytics
│   ├── status.go          # Task status management
│   ├── update.go          # Task status updates
│   └── version.go         # Version management and updates
├── internal/              # Private implementation packages
│   ├── ai/               # AI integration and task generation
│   ├── config/           # Configuration management
│   ├── git/              # Git operations and commit generation
│   ├── github/           # GitHub API client and authentication
│   ├── guidance/         # Context-aware guidance system (v3.0.0)
│   ├── progress/         # Progress tracking and reporting
│   ├── setup/            # Repository initialization
│   ├── storage/          # Data persistence and task management
│   ├── tasks/            # Task management utilities
│   ├── threads/          # GitHub review thread resolution
│   ├── tui/              # Terminal UI components
│   ├── ui/               # UI components and formatting
│   ├── verification/     # Task verification and quality checks
│   └── version/          # Version checking and updates
├── docs/                 # Documentation
├── scripts/              # Build, release, and installation scripts
└── .pr-review/           # Per-repository data storage (gitignored auth)
    ├── config.json       # Project configuration
    ├── auth.json         # Authentication (gitignored)
    └── PR-{number}/      # Per-PR data
        ├── info.json     # PR metadata
        ├── reviews.json  # Review data
        └── tasks.json    # Generated tasks
```

### Architectural Rules

- **cmd/** contains only CLI interface logic
- **internal/** packages are single-responsibility focused
- **No circular dependencies between internal packages**
- **Configuration-driven behavior over hard-coded logic**

## Data Architecture

### Local-First Approach

```mermaid
graph TB
    A[GitHub API] --> B[Local Storage]
    B --> C[AI Processing]
    C --> D[Task Generation]
    D --> B
    B --> E[CLI Interface]
```

**Benefits:**
- No cloud dependencies for core functionality
- Git integration for sharing configuration (not sensitive data)
- Fast access to historical data
- Works offline for existing data

### Data Flow

1. **Fetch Phase**: GitHub API → Local JSON storage
2. **Analysis Phase**: Local storage → AI provider → Task generation
3. **Management Phase**: Task updates → Local storage
4. **Display Phase**: Local storage → CLI output

### State Preservation Strategy

```mermaid
graph LR
    A[New Comments] --> B[Change Detection]
    B --> C{Comment Changed?}
    C -->|Yes| D[Cancel Old Tasks]
    C -->|No| E[Preserve Tasks]
    D --> F[Generate New Tasks]
    E --> F
    F --> G[Merge with Existing]
```

- Task statuses are treated as source of truth
- Tool operations never overwrite user work progress
- Merge conflicts resolved in favor of preserving human work

## Core Components

### AI Processing Pipeline

#### Comment Analysis
```go
type CommentProcessor struct {
    client    ClaudeClient
    chunker   CommentChunker
    validator TaskValidator
    monitor   ResponseMonitor
}
```

**Features:**
- Parallel processing of multiple comments
- Automatic chunking for large comments (>20KB)
- JSON recovery for incomplete responses
- Quality validation with retry logic

#### Task Generation

##### Simplified Task Request Structure
```go
// AI generates only essential fields
type SimpleTaskRequest struct {
    Description string `json:"description"`  // Task description in user's language
    Priority    string `json:"priority"`     // critical|high|medium|low
}

// Full task structure with all fields
type TaskRequest struct {
    Description     string // From AI
    Priority        string // From AI
    OriginText      string // From comment body
    SourceReviewID  int64  // From review metadata
    SourceCommentID int64  // From comment metadata
    File            string // From comment location
    Line            int    // From comment location
    Status          string // Default: "todo"
    TaskIndex       int    // Order within comment
    URL             string // GitHub comment URL
}
```

**Benefits:**
- Minimal AI response size (less prone to errors)
- Mechanical fields populated programmatically
- Complete origin text preservation

#### Task ID Generation Strategy

**Deterministic UUID v5 Generation** (Issue #247):

```go
func (a *Analyzer) generateDeterministicTaskID(commentID int64, taskIndex int) string {
    // Standard DNS namespace UUID for v5 generation (RFC 4122)
    namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

    // Create deterministic name from comment ID and task index
    name := fmt.Sprintf("comment-%d-task-%d", commentID, taskIndex)

    // Generate UUID v5 (SHA-1 based, deterministic)
    return uuid.NewSHA1(namespace, []byte(name)).String()
}
```

**Design Rationale:**
- **UUID v5 (not v4)**: SHA-1 based, deterministic, RFC 4122 compliant
- **Idempotency**: Same comment + task index always produces same UUID
- **Uniqueness**: Different comments or task indexes produce different UUIDs
- **Collision-resistant**: SHA-1 hash ensures extremely low collision probability
- **Backward compatible**: Co-exists with legacy random UUID v4 tasks

**Key Benefits:**
1. **Prevents duplicate tasks**: Running `reviewtask` multiple times on same PR doesn't create duplicates
2. **Leverages existing deduplication**: WriteWorker's ID-based deduplication works automatically
3. **No migration needed**: Old random UUIDs and new deterministic UUIDs work together
4. **RFC compliance**: Standard UUID format, works with all UUID tooling

**Input Parameters:**
- `commentID`: GitHub comment ID (stable, unique identifier)
- `taskIndex`: Task position within comment (0, 1, 2...)

**Example:**
```go
// Comment 12345, Task 0
generateDeterministicTaskID(12345, 0)
// => "485370cd-3594-5380-896e-0d646eb34ac4" (always the same)

// Comment 12345, Task 1
generateDeterministicTaskID(12345, 1)
// => "a1b2c3d4-5678-5901-234e-56789abcdef0" (different from task 0)
```

**Implementation Details:**
- Namespace: DNS namespace UUID (`6ba7b810-9dad-11d1-80b4-00c04fd430c8`)
- Name format: `"comment-{commentID}-task-{taskIndex}"`
- Hash algorithm: SHA-1 (UUID v5 standard)
- Output format: Standard UUID string representation

#### Task Data Structure
```go
type Task struct {
    ID          string    `json:"id"`           // Deterministic UUID v5
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Priority    string    `json:"priority"`
    Status      string    `json:"status"`
    CommentID   string    `json:"comment_id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

#### Deduplication Engine
```go
type TaskDeduplicator struct {
    client             ClaudeClient
    similarityThreshold float64
    enabled            bool
}
```

**Algorithm:**
1. Compare new tasks against existing tasks
2. Use AI-powered similarity analysis
3. Merge similar tasks while preserving status
4. Configurable similarity threshold

### Review Source Integration

#### Multi-Source Review Support

reviewtask supports multiple AI code review tools with different comment formats:

##### CodeRabbit Integration
- **Detection**: `coderabbitai[bot]` username
- **Format**: Standard GitHub review comments + actionable summary
- **Processing**:
  - Summary body cleared but individual comments preserved
  - Nitpick comments configurable via `process_nitpick_comments`
  - HTML element removal for clean task extraction

##### Codex Integration (NEW)
- **Detection**: `chatgpt-codex-connector` username or contains "codex"
- **Format**: Embedded comments within review body
- **Processing**:
  - Parse structured markdown from review body
  - Extract GitHub permalinks, priority badges, titles, descriptions
  - Convert to standard Comment format for task generation

**Codex Comment Structure:**
```go
type EmbeddedComment struct {
    FilePath    string // Extracted from GitHub permalink
    StartLine   int    // From permalink line range
    EndLine     int    // From permalink line range
    Priority    string // P1/P2/P3 from badge
    Title       string // From markdown heading
    Description string // Comment body text
    Permalink   string // Full GitHub URL
}
```

**Priority Mapping:**
- P1 (orange badge) → HIGH priority
- P2 (yellow badge) → MEDIUM priority
- P3 (green badge) → LOW priority

**Deduplication:**
- Codex sometimes submits duplicate reviews
- Content-based fingerprinting detects duplicates
- Keeps most recent review when duplicates found

##### Integration Flow
```mermaid
graph TB
    A[GitHub Reviews] --> B{Review Source?}
    B -->|CodeRabbit| C[Clear Summary Body]
    B -->|Codex| D[Parse Embedded Comments]
    B -->|Standard| E[Process Normally]
    C --> F[Extract Comments]
    D --> G[Convert to Comment Format]
    E --> F
    F --> H[Deduplicate Reviews]
    G --> H
    H --> I[Task Generation]
```

### GitHub Integration

#### GraphQL API Integration

**Thread Auto-Resolution:**
```go
type GraphQLClient struct {
    token      string
    httpClient *http.Client
}

func (c *GraphQLClient) ResolveReviewThread(ctx context.Context, threadID string) error
func (c *GraphQLClient) GetReviewThreadID(ctx context.Context, owner, repo string, prNumber int, commentID int64) (string, error)
```

**Features:**
- Automatically resolve review threads based on configurable mode
- Maps comment IDs to thread IDs via GraphQL API with pagination support
- Handles large PRs with >100 threads or >100 comments per thread
- Only applies to standard GitHub comments (not Codex embedded comments)
- Configurable via `auto_resolve_mode` setting (default: "complete")

**Auto-Resolve Modes:**
- `complete` - Resolve when ALL tasks from a comment are completed (smart resolution)
- `immediate` - Resolve thread immediately when each task is marked as done
- `disabled` - Never auto-resolve (use manual `reviewtask resolve` command)

**Pagination Support:**
The GraphQL client implements nested pagination to support large PRs:
- Outer loop: Paginates through review threads (100 per page)
- Inner loop: Paginates through comments within each thread (100 per page)
- Returns immediately when target comment is found
- Exhausts all pages before returning "not found" error

**Implementation:**
```go
// Comment-level completion check
func (m *Manager) AreAllCommentTasksCompleted(prNumber int, commentID int64) (bool, error) {
    // Check all tasks from the same comment
    // Rules:
    // - done: always OK
    // - cancel: requires CancelCommentPosted=true
    // - pending/todo/doing: blocks resolution
}

// Auto-resolve with mode support
if config.AutoResolveMode != "disabled" {
    if config.AutoResolveMode == "immediate" && task.Status == "done" {
        // Resolve immediately
        resolveThread(task)
    } else if config.AutoResolveMode == "complete" {
        // Check if all tasks from comment are completed
        if allCompleted, _ := manager.AreAllCommentTasksCompleted(prNumber, commentID); allCompleted {
            resolveThread(task)
        }
    }
}
```

#### Authentication Hierarchy
```go
type AuthManager struct {
    sources []AuthSource
}

type AuthSource interface {
    GetToken() (string, error)
    GetUser() (*github.User, error)
    Validate() error
}
```

**Priority order:**
1. Environment variable (`GITHUB_TOKEN`)
2. Local configuration file (`.pr-review/auth.json`)
3. GitHub CLI integration (`gh auth token`)

#### API Client
```go
type Client struct {
    github   *github.Client
    cache    *Cache
    rateLim  *RateLimiter
}
```

**Features:**
- Automatic rate limiting
- Response caching
- Retry logic for transient failures
- Multi-source authentication

### Storage System

#### Write Worker for Concurrent Task Saving
```go
type WriteWorker struct {
    manager      *Manager
    taskQueue    chan Task
    errorQueue   chan WriteError
    wg           sync.WaitGroup
    mu           sync.Mutex
    isRunning    bool
    shutdownChan chan struct{}
}
```

**Features:**
- Queue-based concurrent writes
- Thread-safe file operations with mutex
- Real-time task persistence
- PR-specific directory management
- Error tracking and recovery

**Operation Flow:**
1. Tasks queued as they're generated
2. Worker processes queue continuously
3. Each task written to PR-specific `tasks.json`
4. Mutex ensures file consistency
5. Errors collected for reporting

#### File Structure
```
.pr-review/
├── config.json              # Project configuration
├── auth.json                 # Authentication (gitignored)
├── cache/                    # API response cache
│   └── reviews-{pr}-{hash}.json
└── PR-{number}/
    ├── info.json            # PR metadata
    ├── reviews.json         # Review data with nested comments
    └── tasks.json           # AI-generated tasks
```

#### Data Models
```go
type PRInfo struct {
    Number      int       `json:"number"`
    Title       string    `json:"title"`
    Branch      string    `json:"branch"`
    LastUpdated time.Time `json:"last_updated"`
}

type ReviewData struct {
    Reviews  []Review  `json:"reviews"`
    Comments []Comment `json:"comments"`
    FetchedAt time.Time `json:"fetched_at"`
}

type TaskData struct {
    Tasks     []Task    `json:"tasks"`
    Generated time.Time `json:"generated_at"`
    Version   string    `json:"version"`
}
```

## Performance Architecture

### Parallel Processing

```mermaid
graph TB
    A[PR Reviews] --> B[Comment Extraction]
    B --> C[Parallel Processing Pool]
    C --> D[Worker 1]
    C --> E[Worker 2]
    C --> F[Worker N]
    D --> G[Task Aggregation]
    E --> G
    F --> G
    G --> H[Deduplication]
    H --> I[Final Tasks]
```

**Benefits:**
- Reduced processing time for large PRs
- Better AI provider reliability (smaller prompts)
- Improved error isolation (one comment failure doesn't affect others)

### Caching Strategy

```go
type Cache struct {
    storage map[string]CacheEntry
    ttl     time.Duration
}

type CacheEntry struct {
    Data      interface{} `json:"data"`
    ExpiresAt time.Time   `json:"expires_at"`
    Hash      string      `json:"hash"`
}
```

**Levels:**
1. **API Response Cache**: GitHub API responses cached for 1 hour
2. **Processing Cache**: Avoid reprocessing unchanged comments
3. **Task Cache**: Preserve task statuses across runs

### Optimization Features

#### Automatic Performance Scaling
- Small PRs: Fast, simple processing
- Large PRs: Parallel processing, chunking, and optimization
- Auto-detection based on comment count and size

#### Resource Management
- Configurable concurrency limits
- Memory-efficient streaming for large responses
- Graceful degradation under resource constraints

## Security Architecture

### Authentication Security
```go
type SecureAuth struct {
    tokenValidator TokenValidator
    permChecker    PermissionChecker
    rateLimiter    RateLimiter
}
```

**Features:**
- Token validation and permission checking
- Secure storage with restricted file permissions
- No token logging or exposure in errors
- Rate limiting to prevent abuse

### Data Security
- Local storage only (no cloud data transmission)
- Gitignore patterns for sensitive files
- File permission restrictions (600 for auth files)
- No sensitive data in log output

### AI Provider Security
- No direct API key management
- All AI processing through local CLI tools
- No data transmission to external services
- Local prompt processing and response handling

## Extensibility Architecture

### AI Provider Interface
```go
type AIProvider interface {
    GenerateTasks(comments []Comment, config Config) ([]Task, error)
    ValidateTasks(tasks []Task) (ValidationResult, error)
    DeduplicateTasks(existing, new []Task) ([]Task, error)
}
```

**Current providers:**
- Claude Code CLI
- Stdout (for testing and debugging)

**Future providers:**
- OpenAI API
- Local models (Ollama)
- Custom providers

### Prompt Template System

```go
type PromptTemplate struct {
    Path     string
    Content  string
    Variables map[string]interface{}
}
```

**Features:**
- External markdown templates in `prompts/` directory
- Go template syntax for variable substitution
- Hot-reloadable without recompilation
- Language-specific customization support

**Template Variables:**
- `{{.LanguageInstruction}}` - User's language preference
- `{{.File}}` - Source file path
- `{{.Line}}` - Line number in file
- `{{.Author}}` - Comment author
- `{{.Comment}}` - Comment body text

**Benefits:**
- Easy prompt iteration without code changes
- Version control friendly markdown format
- Customizable per-project through config
- Testable with golden tests

### Plugin Architecture
```go
type Plugin interface {
    Name() string
    Version() string
    Process(data PluginData) (PluginResult, error)
}
```

**Extension points:**
- Custom task processors
- Additional authentication sources
- Custom output formatters
- Integration with external tools

## Error Handling and Recovery

### Resilience Patterns

#### Circuit Breaker
```go
type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
    state       CircuitState
}
```

**Application:**
- GitHub API failures
- AI provider unavailability
- Network connectivity issues

#### Retry Strategy
```go
type RetryStrategy struct {
    maxAttempts int
    backoff     BackoffStrategy
    conditions  []RetryCondition
}
```

**Retry conditions:**
- Transient network errors
- Rate limiting (with exponential backoff)
- AI provider temporary failures

#### JSON Recovery
```go
type JSONRecovery struct {
    parser      *PartialParser
    validator   *DataValidator
    threshold   float64
}
```

**Capabilities:**
- Recover partial task data from truncated responses
- Validate and clean malformed JSON
- Extract usable content from incomplete API responses

## Monitoring and Observability

### Performance Monitoring
```go
type ResponseMonitor struct {
    metrics    map[string]Metric
    analytics  *Analytics
    thresholds map[string]float64
}
```

**Tracked metrics:**
- API response times
- Task generation success rates
- Error patterns and frequency
- Resource usage patterns

### Analytics
- Response size analysis
- Truncation pattern detection
- Success rate tracking
- Performance optimization recommendations

### Logging
```go
type Logger struct {
    level   LogLevel
    writers []io.Writer
    format  LogFormat
}
```

**Log levels:**
- Error: Critical failures and errors
- Warn: Non-critical issues and warnings
- Info: General operational information
- Debug: Detailed debugging information (verbose mode)

## Development and Operational Guidelines

### Code Organization Rules
- Follow Go standard project layout
- Each command gets its own file in `cmd/`
- Business logic stays in `internal/` packages
- Configuration changes require documentation updates

### CLI Design Principles
- Commands follow `gh` CLI patterns and conventions
- Help text includes practical examples
- Error messages provide actionable guidance
- Progressive disclosure: simple commands first, advanced features discoverable

### Testing Strategy
- Focus on workflow testing over unit testing
- Test real user scenarios end-to-end
- Mock external dependencies (GitHub API, Claude CLI)
- Manual testing of authentication flows

### Release Management
- Semantic versioning with automated releases
- Cross-platform binary distribution
- Automated testing and quality checks
- Clear migration guides for breaking changes

## Future Architecture Considerations

### Planned Enhancements

#### Multi-Provider AI Support
- Plugin-based AI provider system
- Provider selection and fallback logic
- Performance comparison and optimization

#### Enhanced Caching
- Distributed caching for team environments
- Intelligent cache invalidation
- Cross-repository cache sharing

#### Integration Expansion
- IDE plugins and extensions
- CI/CD pipeline integration
- Webhook support for real-time updates

#### Performance Optimization
- Streaming processing for very large PRs
- Advanced parallel processing patterns
- Resource usage optimization

### Scalability Considerations

#### Team Environments
- Shared configuration management
- Team-wide analytics and reporting
- Collaborative task management

#### Enterprise Features
- GitHub Enterprise support
- SSO integration
- Audit logging and compliance
- Policy enforcement

This architecture enables reviewtask to be both simple for individual developers and powerful enough for team and enterprise environments, while maintaining the core principles of reliability, performance, and user control.