# gh-review-task

An AI-powered CLI tool that fetches GitHub Pull Request reviews, analyzes them intelligently, and generates actionable tasks for developers.

## Features

### 🔍 Intelligent PR Review Analysis
- Fetches PR reviews and comments from GitHub API
- Analyzes comment chains and conversation context
- Uses AI to generate contextual, actionable tasks
- Supports nested comment structures and reply threading

### 📊 Task Management
- Full task lifecycle: `todo` → `doing` → `done` (+ `pending`/`cancel`)
- Priority-based organization (critical/high/medium/low)
- Status tracking and progress statistics
- Cross-PR task aggregation and reporting

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
# Analyze reviews for a specific PR
./gh-review-task 123

# Analyze reviews for current branch's PR  
./gh-review-task

# Check current task status
./gh-review-task status

# Update task status
./gh-review-task update task-123 doing
```

## Commands

### Core Commands
- `gh-review-task [PR_NUMBER]` - Analyze PR reviews and generate tasks
- `gh-review-task status` - Show current task status and statistics
- `gh-review-task update <task-id> <status>` - Update task status

### Setup & Authentication  
- `gh-review-task init` - Initialize repository for gh-review-task
- `gh-review-task auth login` - Authenticate with GitHub
- `gh-review-task auth status` - Check authentication status
- `gh-review-task auth check` - Comprehensive authentication and permission check
- `gh-review-task auth logout` - Remove local authentication

## Configuration

### Priority Rules

The tool uses intelligent priority assignment based on configurable rules:

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
  }
}
```

### Data Storage

```
.pr-review/
├── config.json              # Priority rules and project settings
├── auth.json                # Local authentication (gitignored)
└── PR-<number>/
    ├── info.json            # PR metadata
    ├── reviews.json         # Review data with nested comments  
    └── tasks.json           # AI-generated tasks
```

## AI Integration

### Contextual Analysis
- Analyzes entire comment threads, not just individual comments
- Considers reply context to avoid duplicate or resolved tasks
- Understands conversation flow and resolution status
- Generates specific, actionable tasks with file/line references

### Priority Intelligence
- Uses project-specific rules for task prioritization
- Considers comment author, review type, and discussion context
- Adapts to team communication patterns and priorities

## Requirements

- Go 1.20+
- GitHub Personal Access Token with `repo` and `pull_requests` scopes
- Git repository with GitHub remote

## Development

### Project Structure

```
├── cmd/                    # CLI commands and subcommands
├── internal/
│   ├── github/            # GitHub API client and authentication
│   ├── storage/           # Local data persistence
│   ├── ai/                # AI analysis and task generation
│   ├── config/            # Configuration management
│   └── setup/             # Repository initialization
├── PRD.md                 # Product requirements document
└── CLAUDE.md              # Project memory and context
```

### Testing

The tool has been comprehensively tested across all major user flows:
- First-time setup and initialization
- Authentication with multiple token sources
- PR review processing and task generation  
- Task lifecycle management
- Permission validation and error scenarios
