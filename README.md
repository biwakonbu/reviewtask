# gh-review-task

An AI-powered CLI tool that fetches GitHub Pull Request reviews, analyzes them with Claude Code integration, and generates actionable tasks with advanced validation and multi-language support.

## Features

### 🔍 Intelligent PR Review Analysis
- Fetches PR reviews and comments from GitHub API with full metadata
- **Two-Stage Validation**: Mechanical format validation + AI-powered content validation
- **Claude Code Integration**: Uses proper one-shot mode with quality scoring (0.0-1.0)
- **Information Preservation**: Original review text preserved alongside AI-generated descriptions
- Supports nested comment structures and reply threading
- **Multi-language Support**: Task descriptions in user's preferred language

### 📊 Advanced Task Management
- **Comment-Based Tracking**: Multiple tasks per comment with individual status management
- Task ID format: `comment-{commentID}-task-{index}` for precise tracking
- Full task lifecycle: `todo` → `doing` → `done` (+ `pending`/`cancelled`)
- Priority-based organization (critical/high/medium/low) with AI-powered assignment
- **Enhanced Statistics**: Comment-level progress tracking with new `stats` command
- Cross-PR task aggregation and detailed reporting

### 🤖 AI Quality Assurance
- **Iterative Improvement**: Up to 5 retry attempts with validation feedback
- **Quality Scoring**: Configurable quality threshold (default: 0.8)
- **Task Splitting**: Automatically splits complex comments into multiple actionable tasks
- **Best Result Fallback**: Uses highest-scoring result when perfect validation fails
- Graceful degradation with fallback mechanisms

### 🔐 Smart Authentication
- Multi-source token detection (env vars, local config, gh CLI)
- Interactive setup with `gh auth login` style flow
- Comprehensive permission validation
- Detailed troubleshooting and error guidance

### ⚙️ Project-Specific Configuration
- Customizable priority rules and AI analysis criteria
- Automatic repository initialization
- Git integration with automatic `.gitignore` management
- Per-project settings and preferences

## Quick Start

### Installation

```bash
# Build from source
git clone https://github.com/biwakonbu/ai-pr-review-checker.git
cd ai-pr-review-checker
go build -o gh-review-task
```

### First-Time Setup

```bash
# Initialize the repository (creates .pr-review/ directory and config)
./gh-review-task init

# Authenticate with GitHub (if needed)
./gh-review-task auth login
```

### Basic Usage

```bash
# Analyze reviews for a specific PR (with AI validation)
./gh-review-task 123

# Analyze reviews for current branch's PR  
./gh-review-task

# Check current task status across all PRs
./gh-review-task status

# View detailed statistics by comment
./gh-review-task stats 123

# Update task status (new comment-based format)
./gh-review-task update comment-67890-task-0 doing
```

## Commands

### Core Commands
- `gh-review-task [PR_NUMBER]` - Analyze PR reviews with two-stage validation and generate tasks
- `gh-review-task status` - Show current task status and statistics across all PRs
- `gh-review-task stats [PR_NUMBER]` - **NEW**: Detailed comment-level statistics and progress tracking
- `gh-review-task update <task-id> <status>` - Update task status (format: `comment-{id}-task-{index}`)

### Setup & Authentication  
- `gh-review-task init` - Initialize repository for gh-review-task
- `gh-review-task auth login` - Authenticate with GitHub
- `gh-review-task auth status` - Check authentication status
- `gh-review-task auth check` - Comprehensive authentication and permission check
- `gh-review-task auth logout` - Remove local authentication

## Configuration

### Enhanced Configuration

The tool supports comprehensive configuration including AI settings and priority rules:

```json
{
  "priority_rules": {
    "critical": "Security vulnerabilities, authentication bypasses, data exposure risks",
    "high": "Performance bottlenecks, memory leaks, database optimization issues", 
    "medium": "Functional bugs, logic improvements, error handling",
    "low": "Code style, naming conventions, comment improvements"
  },
  "project_specific": {
    "critical": "Custom rules for your project...",
    "high": "API response time over 500ms, concurrent user handling issues"
  },
  "ai_settings": {
    "user_language": "English",
    "max_retries": 5,
    "validation_enabled": true,
    "quality_threshold": 0.8,
    "fallback_enabled": true
  }
}
```

### AI Configuration Options

- **`user_language`**: Language for task descriptions (default: "English")
- **`max_retries`**: Maximum validation attempts (default: 5)
- **`validation_enabled`**: Enable two-stage validation (default: true)
- **`quality_threshold`**: Minimum quality score to accept (default: 0.8)
- **`fallback_enabled`**: Enable graceful degradation (default: true)

### Enhanced Data Storage

```
.pr-review/
├── config.json              # Priority rules, AI settings, and project configuration
├── auth.json                # Local authentication (gitignored)
└── PR-<number>/
    ├── info.json            # PR metadata
    ├── reviews.json         # Review data with nested comments and full metadata
    └── tasks.json           # AI-generated tasks with validation scores and origin text
```

#### Task Data Structure

Each task now includes comprehensive information:

```json
{
  "id": "comment-67890-task-0",
  "description": "Fix the memory leak in the connection pool",
  "origin_text": "This connection pool implementation might cause memory leaks...",
  "priority": "high",
  "source_comment_id": 67890,
  "task_index": 0,
  "file": "src/database/pool.go",
  "line": 42,
  "status": "todo"
}
```

## Advanced AI Integration

### Two-Stage Validation Process

#### Stage 1: Format Validation (Mechanical)
- JSON syntax and structure validation
- Required field presence verification
- Data type and format consistency checks
- Priority value validation against allowed values

#### Stage 2: Content Validation (AI-Powered)
- Task actionability and specificity assessment
- Language consistency with user preferences
- Original comment intent preservation
- Duplicate task detection across comment chains
- Priority appropriateness for issue severity

### Enhanced Contextual Analysis
- **Multi-language Task Generation**: Task descriptions in user's preferred language while preserving original review text
- **Comment Chain Analysis**: Understands entire conversation threads and reply context
- **Task Splitting Intelligence**: Automatically identifies and splits complex comments into multiple actionable tasks
- **Resolution Detection**: Avoids creating tasks for already-resolved issues
- **Quality Scoring**: Each generation attempt receives a 0.0-1.0 quality score

### Iterative Improvement System
- **Feedback Loop**: Failed validations provide specific guidance for next attempt
- **Retry Logic**: Up to 5 attempts with progressive improvement
- **Best Result Selection**: Uses highest-scoring result when perfect validation fails
- **Graceful Degradation**: Falls back to functional tasks when AI is unavailable

## Requirements

### Essential Dependencies
- **Go 1.20+** - For building and running the application
- **GitHub Personal Access Token** - With `repo` and `pull_requests` scopes
- **Git repository** - With GitHub remote configured
- **[Claude Code CLI](https://docs.anthropic.com/claude-code)** - For AI-powered task generation and validation

### Installation Steps

1. **Install Claude Code CLI**:
   ```bash
   # Follow installation instructions at: https://docs.anthropic.com/claude-code
   # Verify installation:
   claude --version
   ```

2. **GitHub Authentication**:
   - Create a Personal Access Token with `repo` and `pull_requests` scopes
   - Set via environment variable: `export GITHUB_TOKEN=your_token`
   - Or authenticate via: `./gh-review-task auth login`

3. **Build gh-review-task**:
   ```bash
   git clone https://github.com/biwakonbu/ai-pr-review-checker.git
   cd ai-pr-review-checker
   go build -o gh-review-task
   ```

### Optional Dependencies
- **gh CLI** - For seamless GitHub integration (token auto-detection)
- **jq** - For manual JSON manipulation of stored data

## Development

### Project Structure

```
├── cmd/                    # CLI commands and subcommands
│   ├── auth.go            # Authentication management
│   ├── init.go            # Repository initialization
│   ├── root.go            # Main command and PR analysis
│   ├── stats.go           # NEW: Comment-level statistics
│   ├── status.go          # Task status overview
│   └── update.go          # Task status updates
├── internal/
│   ├── ai/                # Enhanced AI analysis with validation
│   │   ├── analyzer.go    # Core AI integration with Claude Code
│   │   ├── validator.go   # NEW: Two-stage validation system
│   │   └── statistics.go  # NEW: Comment-based statistics
│   ├── github/            # GitHub API client and authentication
│   ├── storage/           # Enhanced data persistence with new structures
│   ├── config/            # Configuration with AI settings
│   └── setup/             # Repository initialization
├── PRD.md                 # Product requirements document
└── CLAUDE.md              # Project memory and context
```

### Recent Enhancements

#### New Files Added
- `cmd/stats.go` - Comment-level statistics and progress tracking
- `internal/ai/validator.go` - Two-stage validation system implementation
- `internal/ai/statistics.go` - Statistical analysis and reporting

#### Enhanced Files
- `internal/ai/analyzer.go` - Claude Code integration with validation
- `internal/storage/manager.go` - Extended data structures and comment-based queries
- `internal/config/config.go` - AI settings and multi-language configuration

### Testing Coverage

The tool has been comprehensively tested across all major user flows:
- **Enhanced**: Two-stage validation system with quality scoring
- **Enhanced**: Multi-language task generation and validation
- **New**: Comment-based task management and statistics
- First-time setup and initialization
- Authentication with multiple token sources
- PR review processing with advanced AI integration
- Task lifecycle management with comment tracking
- Permission validation and error scenarios
