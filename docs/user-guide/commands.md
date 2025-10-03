# Command Reference

This comprehensive guide covers all reviewtask commands, their options, and usage examples.

## AI Provider Transparency

All major commands now display the active AI provider and model at startup:

```
🤖 AI Provider: Cursor CLI (grok)
```

This feature ensures you always know:
- Which AI tool is being used
- What model is active
- Whether configuration is working correctly

## Core Commands

### Main Command

#### `reviewtask [PR_NUMBER]`

Analyze PR reviews and generate actionable tasks.

```bash
# Analyze current branch's PR
reviewtask

# Analyze specific PR
reviewtask 123

# Clear cache and reprocess all comments
reviewtask --refresh-cache
```

**Options:**
- `--refresh-cache` - Clear cache and reprocess all comments

**What it does:**
- Fetches PR reviews and comments from GitHub
- Processes comments using AI analysis
- Generates actionable tasks with priorities
- Saves results to `.pr-review/PR-{number}/`
- Preserves existing task statuses

### Task Management

#### `reviewtask status [options]`

Show task status and statistics across PRs.

```bash
# Current branch status
reviewtask status

# All PRs
reviewtask status --all

# Specific PR
reviewtask status --pr 123

# Specific branch
reviewtask status --branch feature/new-feature
```

**Options:**
- `--all` - Show information for all PRs
- `--pr <number>` - Show information for specific PR
- `--branch <name>` - Show information for specific branch

#### `reviewtask show [task-id]`

Show current/next task or specific task details.

```bash
# Show current/next task
reviewtask show

# Show specific task details
reviewtask show task-uuid-here
```

#### `reviewtask update <task-id> <status>`

Update task status.

```bash
reviewtask update task-uuid-here doing
reviewtask update task-uuid-here done
reviewtask update task-uuid-here pending
reviewtask update task-uuid-here cancel
```

**Valid statuses:**
- `todo` - Ready to work on
- `doing` - Currently in progress
- `done` - Completed
- `pending` - Blocked or low priority
- `cancel` - No longer relevant

### Thread Resolution

#### `reviewtask resolve [task-id] [options]`

Manually resolve GitHub review threads for completed tasks.

```bash
# Resolve thread for a specific task
reviewtask resolve task-uuid-here

# Resolve all completed tasks' threads
reviewtask resolve --all

# Force resolution even if task isn't marked as done
reviewtask resolve task-uuid-here --force
```

**Options:**
- `--all` - Resolve threads for all tasks marked as done
- `--force` - Resolve thread even if task status is not done

**What it does:**
- Resolves the GitHub review thread associated with the task
- Requires task to be in `done` status (unless `--force` is used)
- Uses GitHub GraphQL API to mark thread as resolved
- Prevents duplicate resolution attempts

**Auto-resolve mode:**
By default, threads are automatically resolved when all tasks from a comment are completed. Use `reviewtask resolve` for manual control or when auto-resolve is disabled.

### Statistics

#### `reviewtask stats [PR_NUMBER] [options]`

Show detailed task statistics with comment breakdown.

```bash
# Current branch statistics
reviewtask stats

# Statistics for specific PR
reviewtask stats 123
reviewtask stats --pr 123

# Statistics for all PRs
reviewtask stats --all

# Statistics for specific branch
reviewtask stats --branch feature/new-feature
```

**Output includes:**
- Comment-level breakdown with task counts
- Priority distribution (critical/high/medium/low)
- Status distribution (todo/doing/done/pending/cancel)
- Completion metrics and progress tracking
- File-level summary with affected files

## Setup and Configuration

### Repository Setup

#### `reviewtask init`

Interactive setup wizard that initializes repository with reviewtask configuration.

```bash
reviewtask init
```

**Features:**
- Interactive language selection
- Automatic AI provider detection (Cursor CLI, Claude Code)
- Creates minimal 2-line configuration
- Sets up `.pr-review/` directory structure
- Adds appropriate `.gitignore` entries
- Verifies GitHub authentication

### Configuration Management

#### `reviewtask config <command>`

Manage and optimize configuration settings.

```bash
# Show current configuration
reviewtask config show

# Validate configuration and suggest improvements
reviewtask config validate

# Migrate to simplified format (46+ lines → 2-8 lines)
reviewtask config migrate
```

**Benefits of simplified configuration:**
- Auto-detects project type (Go, Node.js, Rust, Python, etc.)
- Applies smart defaults for build/test commands
- Reduces configuration complexity by 90%
- Maintains full backward compatibility

### Authentication

#### `reviewtask auth <command>`

Manage GitHub authentication.

```bash
# Interactive GitHub token setup
reviewtask auth login

# Show current authentication source and user
reviewtask auth status

# Remove local authentication
reviewtask auth logout

# Comprehensive validation of token and permissions
reviewtask auth check
```

**Authentication sources (priority order):**
1. `GITHUB_TOKEN` environment variable
2. Local config file (`.pr-review/auth.json`)
3. GitHub CLI (`gh auth token`)

## Version Management

#### `reviewtask version [VERSION]`

Show version information or switch to specific version.

```bash
# Show current version with update check
reviewtask version

# Switch to specific version
reviewtask version v1.2.3
reviewtask version latest

# Check for available updates only
reviewtask version --check
```

#### `reviewtask versions`

List available versions from GitHub releases.

```bash
reviewtask versions
```

Shows recent 5 versions with release information.

## AI Provider Integration

### Prompt Generation

#### `reviewtask prompt <provider> <target>`

Generate AI provider command templates.

```bash
# Generate PR review workflow template for Claude Code
reviewtask prompt claude pr-review

# Output prompts to stdout for redirection or piping
reviewtask prompt stdout pr-review
reviewtask prompt stdout pr-review > my-workflow.md
reviewtask prompt stdout pr-review | pbcopy  # macOS clipboard
reviewtask prompt stdout pr-review | xclip   # Linux clipboard
```

**Providers:**
- `claude` - Creates optimized command templates in `.claude/commands/` directory
- `stdout` - Outputs prompts to standard output

**Targets:**
- `pr-review` - PR review analysis workflow

#### `reviewtask claude <target>` (Deprecated)

Legacy command for Claude integration. Use `reviewtask prompt claude <target>` instead.

## Debug and Development

### Debug Commands

#### `reviewtask debug fetch <phase> [PR]`

Test specific phases independently for troubleshooting.

```bash
# Fetch and save PR reviews only (no task generation)
reviewtask debug fetch review 123

# Generate tasks from previously saved reviews only
reviewtask debug fetch task 123
```

**Features:**
- Automatically enables verbose mode for detailed logging
- Isolates specific functionality for testing
- Useful for troubleshooting issues

#### `reviewtask debug prompt <PR> [--profile <profile>]`

Render the analysis prompt locally from saved reviews (no AI calls). Useful for A/B comparisons between profiles.

```bash
# Save reviews for a PR, then render the prompt
reviewtask debug fetch review 123
reviewtask debug prompt 123 --profile v2
reviewtask debug prompt 123 --profile legacy
```

**Options:**
- `--profile` — one of: `v2` (default, alias: `rich`), `compact`, `minimal`, `legacy`

## Command Examples

### Daily Workflow

```bash
# Morning startup
reviewtask show           # What should I work on today?
reviewtask status         # Overall progress across all PRs

# During implementation
reviewtask show <task-id> # Full context for current task
# Work on the task...
reviewtask update <task-id> done

# When blocked
reviewtask update <task-id> pending  # Mark as blocked
reviewtask show                      # Find next task to work on

# When reviews are updated
reviewtask                # Re-run to get new feedback
# Tool automatically preserves your work progress
```

### Advanced Usage

```bash
# Force complete refresh of all data
reviewtask --refresh-cache

# Comprehensive system check
reviewtask auth check
reviewtask version --check

# Generate documentation workflows
reviewtask prompt claude pr-review
reviewtask prompt stdout pr-review > custom-workflow.md

# Detailed statistics analysis
reviewtask stats --all
reviewtask stats --branch main
```

### Troubleshooting Commands

```bash
# Authentication debugging
reviewtask auth status
reviewtask auth check

# Version management
reviewtask versions
reviewtask version latest

# Debug specific functionality
reviewtask debug fetch review 123
reviewtask debug fetch task 123

# Performance analysis
reviewtask stats --pr 123
```

## Global Options

Most commands support these global options:

- `--help` - Show command help
- `--version` - Show version information (for main command)

## Command Aliases

Some commands have aliases for convenience:

- `reviewtask fetch` - Alias for main `reviewtask` command
- `reviewtask` (no arguments) - Analyzes current branch's PR

## Exit Codes

reviewtask uses standard exit codes:

- `0` - Success
- `1` - General error
- `2` - Authentication error
- `3` - Configuration error
- `4` - Network error

## Performance Considerations

### Cache Management

- Commands automatically cache GitHub API responses
- Use `--refresh-cache` when you need to bypass caching
- Cache improves performance and reduces API rate limit usage

### Large PRs

- Tool automatically optimizes performance based on PR size
- Comments >20KB are automatically chunked
- Parallel processing handles multiple comments efficiently
- Auto-resume functionality for interrupted processing

### Rate Limiting

- Authenticated requests: 5,000/hour
- Use `reviewtask auth check` to monitor rate limit status
- Tool includes automatic retry logic for rate limit handling

For more detailed information on specific commands, use the `--help` flag with any command.
