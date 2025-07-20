# AI-Powered PR Review Management Tool

[![CI](https://github.com/biwakonbu/ai-pr-review-checker/workflows/CI/badge.svg)](https://github.com/biwakonbu/ai-pr-review-checker/actions)
[![codecov](https://codecov.io/gh/biwakonbu/ai-pr-review-checker/branch/main/graph/badge.svg)](https://codecov.io/gh/biwakonbu/ai-pr-review-checker)
[![Go Report Card](https://goreportcard.com/badge/github.com/biwakonbu/ai-pr-review-checker)](https://goreportcard.com/report/github.com/biwakonbu/ai-pr-review-checker)
[![GoDoc](https://godoc.org/github.com/biwakonbu/ai-pr-review-checker?status.svg)](https://godoc.org/github.com/biwakonbu/ai-pr-review-checker)

A CLI tool that fetches GitHub Pull Request reviews, analyzes them using AI, and generates actionable tasks for developers.

## Features

- **🔍 PR Review Fetching**: Automatically retrieves reviews from GitHub API with nested comment structure
- **🤖 AI Analysis**: Uses Claude Code integration to generate structured, actionable tasks from review content
- **💾 Local Storage**: Stores data in structured JSON format under `.pr-review/` directory
- **📋 Task Management**: Full lifecycle management with status tracking (todo/doing/done/pending/cancel)
- **⚡ Parallel Processing**: Processes multiple comments concurrently for improved performance
- **🔒 Authentication**: Multi-source token detection with interactive setup
- **🎯 Priority-based Analysis**: Customizable priority rules for task generation
- **🔄 Task State Preservation**: Maintains existing task statuses during subsequent runs
- **🆔 UUID-based Task IDs**: Unique task identification to eliminate duplication issues

## Installation

1. Clone the repository:
```bash
git clone https://github.com/biwakonbu/ai-pr-review-checker.git
cd ai-pr-review-checker
```

2. Build the binary:
```bash
go build -o gh-review-task
```

3. Install Claude Code CLI (required for AI analysis):
```bash
# Follow Claude Code installation instructions
# https://docs.anthropic.com/en/docs/claude-code
```

## Quick Start

### 1. Initialize Repository

```bash
# Initialize the tool in your repository
./gh-review-task init
```

This will:
- Create `.pr-review/` directory structure
- Generate default configuration files
- Set up `.gitignore` entries
- Check repository permissions

### 2. Authentication Setup

```bash
# Login with GitHub token
./gh-review-task auth login

# Check authentication status
./gh-review-task auth status

# Logout
./gh-review-task auth logout
```

Authentication sources (in order of preference):
1. `GITHUB_TOKEN` environment variable
2. Local config file (`.pr-review/auth.json`)
3. GitHub CLI (`gh auth token`)

### 3. Analyze PR Reviews

```bash
# Analyze current branch's PR
./gh-review-task

# Analyze specific PR
./gh-review-task 123

# The tool will:
# - Fetch PR reviews and comments
# - Process comments in parallel using Claude Code
# - Generate actionable tasks with priorities
# - Save results to .pr-review/PR-{number}/
```

### 4. Task Management

```bash
# View all task status
./gh-review-task status

# Show current/next task details
./gh-review-task show

# Show specific task details
./gh-review-task show <task-id>

# Update specific task status
./gh-review-task update <task-id> <status>

# Valid statuses: todo, doing, done, pending, cancel
```

## Command Reference

| Command | Description |
|---------|-------------|
| `gh-review-task` | Analyze current branch's PR |
| `gh-review-task <PR_NUMBER>` | Analyze specific PR |
| `gh-review-task status` | Show task status and statistics |
| `gh-review-task show [task-id]` | Show current/next task or specific task details |
| `gh-review-task update <id> <status>` | Update task status |
| `gh-review-task init` | Initialize repository |
| `gh-review-task auth <cmd>` | Authentication management |

## Configuration

### Priority Rules

Edit `.pr-review/config.json` to customize priority rules:

```json
{
  "priority_rules": {
    "critical": "Security vulnerabilities, authentication bypasses, data exposure risks",
    "high": "Performance bottlenecks, memory leaks, database optimization issues",
    "medium": "Functional bugs, logic improvements, error handling",
    "low": "Code style, naming conventions, comment improvements"
  },
  "ai_settings": {
    "user_language": "English",
    "validation_enabled": false,
    "debug_mode": true
  }
}
```

### Processing Modes

- **Parallel Mode** (`validation_enabled: false`): Fast processing with individual comment analysis
- **Validation Mode** (`validation_enabled: true`): Two-stage validation with retry logic

## Data Structure

```
.pr-review/
├── config.json              # Priority rules and project settings
├── auth.json                # Local authentication (gitignored)
└── PR-<number>/
    ├── info.json            # PR metadata
    ├── reviews.json         # Review data with nested comments
    └── tasks.json           # AI-generated tasks
```

## Task Lifecycle

1. **Generation**: AI analyzes review comments and creates tasks
2. **Assignment**: Tasks get UUID-based IDs and default "todo" status
3. **Execution**: Developers update status as they work (todo → doing → done)
4. **Preservation**: Subsequent runs preserve existing task statuses
5. **Cancellation**: Outdated tasks are automatically cancelled when comments change

## Advanced Features

### Task State Preservation

- Existing task statuses are preserved during subsequent review fetches
- Comment content changes trigger automatic task cancellation
- New tasks are added without overwriting existing work progress

### Parallel Processing

- Each comment is processed independently using goroutines
- Reduced prompt sizes (3,000-6,000 characters vs 57,760)
- Better performance and Claude Code reliability

### Comment Change Detection

- Automatically detects significant changes in comment content
- Cancels outdated tasks and creates new ones as needed
- Preserves completed work and prevents duplicate tasks

## Troubleshooting

### Authentication Issues

```bash
# Check token permissions
./gh-review-task auth check

# Common solutions:
export GITHUB_TOKEN="your_token_here"
# or
gh auth login
```

### Claude Code Integration

Ensure Claude Code CLI is properly installed and accessible:

```bash
# Test Claude Code availability
claude --version

# Common issues:
# - Claude Code not in PATH
# - Authentication required
# - Network connectivity
```

### Permission Requirements

Required GitHub API permissions:
- `repo` (for private repositories)
- `public_repo` (for public repositories)
- `read:org` (for organization repositories)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.
