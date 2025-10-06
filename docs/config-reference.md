# Configuration Parameter Reference

This document describes all available configuration parameters for reviewtask.

## Configuration Levels

### Level 1: Minimal Configuration (Recommended for 90% of users)

```json
{
  "language": "English",
  "ai_provider": "auto"
}
```

### Level 2: Basic Configuration with Customization

```json
{
  "language": "English",
  "ai_provider": "cursor",
  "model": "grok",
  "priorities": {
    "project_specific": {
      "critical": "Authentication vulnerabilities",
      "high": "Payment processing errors"
    }
  }
}
```

### Level 3: Advanced Configuration

```json
{
  "language": "English",
  "ai_provider": "cursor",
  "model": "grok",
  "ai": {
    "provider": "cursor",
    "model": "grok",
    "prompt_profile": "v2",
    "verbose": true,
    "validation": false
  },
  "advanced": {
    "max_retries": 3,
    "timeout_seconds": 120,
    "deduplication_threshold": 0.8
  }
}
```

## Parameter Reference

### Basic Parameters (Simplified Format)

| Parameter | Type | Default | Description | Example Values |
|-----------|------|---------|-------------|----------------|
| `language` | string | `"English"` | Language for task descriptions | `"English"`, `"Japanese"` |
| `ai_provider` | string | `"auto"` | AI provider to use | `"auto"`, `"cursor"`, `"claude"` |
| `model` | string | `"auto"` | AI model to use | `"grok"`, `"sonnet"`, `"opus"`, `"auto"` |

### Priority Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `priorities.project_specific.critical` | string | `""` | Project-specific critical priority items |
| `priorities.project_specific.high` | string | `""` | Project-specific high priority items |
| `priorities.project_specific.medium` | string | `""` | Project-specific medium priority items |
| `priorities.project_specific.low` | string | `""` | Project-specific low priority items |

### AI Configuration (Advanced)

| Parameter | Type | Default | Description | Impact |
|-----------|------|---------|-------------|---------|
| `ai.provider` | string | `"auto"` | AI provider selection | Determines which AI tool is used |
| `ai.model` | string | `"auto"` | Model selection | Controls AI response quality/speed |
| `ai.prompt_profile` | string | `"v2"` | Prompt template version | `"legacy"`, `"v2"`, `"compact"`, `"minimal"` |
| `ai.verbose` | bool | `false` | Enable verbose output | Shows detailed progress and debugging info |
| `ai.validation` | bool | `true` | Enable task validation | Double-checks generated tasks for quality |

### Advanced Settings

| Parameter | Type | Default | Description | When to Change |
|-----------|------|---------|-------------|----------------|
| `advanced.max_retries` | int | `5` | Max retry attempts for AI calls | Increase if experiencing network issues |
| `advanced.timeout_seconds` | int | `120` | Timeout for operations | Increase for large PRs |
| `advanced.deduplication_threshold` | float | `0.8` | Task similarity threshold (0.0-1.0) | Lower to reduce duplicate tasks |

## Full Configuration Format (Legacy/Detailed)

For backward compatibility, the following detailed format is also supported:

### Priority Rules

```json
{
  "priority_rules": {
    "critical": "Security vulnerabilities, authentication bypasses, data exposure risks",
    "high": "Performance bottlenecks, memory leaks, database optimization issues",
    "medium": "Functional bugs, logic improvements, error handling",
    "low": "Code style, naming conventions, comment improvements"
  }
}
```

**Default Priority Rules:**
- **Critical**: Security vulnerabilities, authentication bypasses, data exposure risks
- **High**: Performance bottlenecks, memory leaks, database optimization issues
- **Medium**: Functional bugs, logic improvements, error handling
- **Low**: Code style, naming conventions, comment improvements

### Task Settings

```json
{
  "task_settings": {
    "default_status": "todo",
    "auto_prioritize": true,
    "low_priority_patterns": ["nit:", "style:", "minor:"],
    "low_priority_status": "pending"
  }
}
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `default_status` | string | `"todo"` | Default status for new tasks |
| `auto_prioritize` | bool | `true` | Automatically set task priorities |
| `low_priority_patterns` | array | `["nit:", "style:", ...]` | Patterns marking low priority comments |
| `low_priority_status` | string | `"pending"` | Status for low priority tasks |

### AI Settings (Detailed)

```json
{
  "ai_settings": {
    "user_language": "English",
    "output_format": "json",
    "max_retries": 5,
    "ai_provider": "auto",
    "model": "auto",
    "prompt_profile": "v2",
    "validation_enabled": true,
    "quality_threshold": 0.8,
    "verbose_mode": false,
    "claude_path": "",
    "cursor_path": "",
    "max_tasks_per_comment": 2,
    "deduplication_enabled": true,
    "similarity_threshold": 0.8,
    "process_nitpick_comments": true,
    "nitpick_priority": "low",
    "enable_json_recovery": true,
    "max_recovery_attempts": 3,
    "partial_response_threshold": 0.7,
    "log_truncated_responses": true,
    "process_self_reviews": false,
    "error_tracking_enabled": true,
    "stream_processing_enabled": true,
    "auto_summarize_enabled": true,
    "realtime_saving_enabled": true,
    "skip_claude_auth_check": false
  }
}
```

| Parameter | Type | Default | Description | When to Change |
|-----------|------|---------|-------------|----------------|
| `user_language` | string | `"English"` | Language for task descriptions | Set to your preferred language |
| `max_retries` | int | `5` | Retry attempts for AI calls | Increase for unreliable connections |
| `ai_provider` | string | `"auto"` | AI provider (`auto`, `cursor`, `claude`) | Set specific provider if auto-detection fails |
| `model` | string | `"auto"` | AI model selection | Choose specific model for consistency |
| `prompt_profile` | string | `"v2"` | Prompt template (`legacy`, `v2`, `compact`, `minimal`) | Use `compact` for faster responses |
| `validation_enabled` | bool | `true` | Enable task validation | Disable for faster processing |
| `quality_threshold` | float | `0.8` | Min quality score (0.0-1.0) | Lower if too many tasks rejected |
| `verbose_mode` | bool | `false` | Show detailed output | Enable for debugging |
| `max_tasks_per_comment` | int | `2` | Max tasks per review comment | Increase for complex comments |
| `deduplication_enabled` | bool | `true` | Remove duplicate tasks | Disable if unique tasks being removed |
| `similarity_threshold` | float | `0.8` | Task similarity threshold | Lower to catch more duplicates |
| `process_nitpick_comments` | bool | `true` | Process CodeRabbit nitpick comments | Disable to skip minor issues |
| `process_self_reviews` | bool | `false` | Process PR author's own reviews | Enable for self-review workflows |

**Note:** Thread auto-resolution is now configured via `done_workflow.enable_auto_resolve` setting. See [Done Workflow Settings](#done-workflow-settings) for details.

#### AI Review Tool Integration

**Supported Review Tools:**

1. **CodeRabbit** (`coderabbitai[bot]`)
   - Standard GitHub review comments
   - Nitpick comment handling via `process_nitpick_comments`
   - Thread auto-resolution supported

2. **Codex** (`chatgpt-codex-connector`)
   - Embedded comments in review body
   - Automatic priority badge detection (P1/P2/P3)
   - GitHub permalink parsing
   - Duplicate review detection
   - Thread auto-resolution NOT supported (no comment ID)

3. **Standard GitHub Reviews**
   - Direct comment processing
   - Thread auto-resolution supported

### Done Workflow Settings

Configure automation behavior for the `reviewtask done` command. This provides a complete workflow automation including verification, commit, thread resolution, and next task suggestion.

```json
{
  "done_workflow": {
    "enable_auto_resolve": "when_all_complete",
    "enable_verification": true,
    "enable_auto_commit": true,
    "enable_next_task_suggestion": true,
    "verifiers": {
      "build": "go build ./...",
      "test": "go test ./...",
      "lint": "golangci-lint run",
      "format": "gofmt -l ."
    }
  }
}
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enable_auto_resolve` | string | `"when_all_complete"` | Thread resolution mode: `"immediate"`, `"when_all_complete"`, or `"disabled"` |
| `enable_verification` | bool | `true` | Run verification checks before marking task as done |
| `enable_auto_commit` | bool | `true` | Automatically commit changes with structured message |
| `enable_next_task_suggestion` | bool | `true` | Show next recommended task after completion |
| `verifiers` | object | *project-specific* | Verification commands by type (build, test, lint, format) |

**Auto-Resolve Modes:**
- `"immediate"`: Resolve GitHub review thread immediately after task completion
- `"when_all_complete"`: Resolve only when all tasks from same comment are done (recommended)
- `"disabled"`: No automatic resolution (use `reviewtask resolve` manually)

**Verification Commands:**
The `verifiers` object maps verification types to shell commands. Default commands are auto-detected based on project type:
- **Go projects**: `go build`, `go test`, `golangci-lint run`, `gofmt -l .`
- **Node.js projects**: `npm run build`, `npm test`, `npm run lint`, `npm run format`
- **Python projects**: `python setup.py build`, `pytest`, `flake8`, `black --check`
- **Rust projects**: `cargo build`, `cargo test`, `cargo clippy`, `cargo fmt --check`

**Skip Options:**
You can skip individual automation phases:
```bash
reviewtask done <task-id> --skip-verification
reviewtask done <task-id> --skip-commit
reviewtask done <task-id> --skip-resolve
reviewtask done <task-id> --skip-suggestion
```

### Update Check Settings

```json
{
  "update_check": {
    "enabled": true,
    "interval_hours": 24,
    "notify_prereleases": false
  }
}
```

## Project Type Auto-Detection

The following project types are automatically detected and configured:

| Project Type | Detection | Build Command | Test Command | Lint Command |
|--------------|-----------|---------------|--------------|--------------|
| **Go** | `go.mod` | `go build ./...` | `go test ./...` | `golangci-lint run` |
| **Node.js** | `package.json` | `npm run build` | `npm test` | `npm run lint` |
| **Rust** | `Cargo.toml` | `cargo build` | `cargo test` | `cargo clippy` |
| **Python** | `requirements.txt`, `setup.py` | `python -m py_compile .` | `pytest` | `pylint .` |
| **Java (Maven)** | `pom.xml` | `mvn compile` | `mvn test` | `mvn checkstyle:check` |
| **Java (Gradle)** | `build.gradle` | `gradle build` | `gradle test` | `gradle check` |
| **Ruby** | `Gemfile` | `bundle install` | `bundle exec rspec` | `rubocop` |
| **PHP** | `composer.json` | `composer install` | `phpunit` | `phpcs` |
| **.NET** | `*.csproj`, `*.sln` | `dotnet build` | `dotnet test` | `dotnet format --verify-no-changes` |

## AI Provider Details

### Auto Mode (Default)
- Tries Cursor CLI first
- Falls back to Claude Code
- Automatically selects best available option

### Cursor CLI
- Models: `grok` (recommended), `auto`
- Best for: Fast responses, general development
- Required: Cursor editor installed

### Claude Code
- Models: `sonnet` (default), `opus` (premium), `haiku` (fast)
- Best for: Complex analysis, detailed explanations
- Required: Claude Code CLI installed

## Migration from Old to New Format

To convert your existing detailed configuration to simplified format:

```bash
reviewtask config migrate
```

This will:
1. Create a backup of your current config
2. Convert to minimal format preserving customizations
3. Remove default values
4. Show the new simplified configuration

## Configuration Commands

| Command | Description | Example |
|---------|-------------|---------|
| `reviewtask config show` | Display current configuration | - |
| `reviewtask config validate` | Check configuration for issues | - |
| `reviewtask config migrate` | Convert to simplified format | - |
| `reviewtask config set-verifier <type> <cmd>` | Set custom verification command | `reviewtask config set-verifier security-task "npm audit"` |
| `reviewtask init` | Interactive setup wizard | - |

## Best Practices

1. **Start Simple**: Use the minimal 2-line configuration
2. **Add As Needed**: Only add parameters when you need to change defaults
3. **Use Auto-Detection**: Let the tool detect your project type
4. **Test Changes**: Use `config validate` after making changes
5. **Keep Backups**: `config migrate` creates backups automatically

## Troubleshooting

### AI Provider Not Found
```json
{
  "ai_provider": "cursor",
  "cursor_path": "/usr/local/bin/cursor"
}
```

### Wrong Project Commands
```json
{
  "verification_settings": {
    "build_command": "npm run build",
    "test_command": "npm test"
  }
}
```

### Tasks Not Being Generated
```json
{
  "ai": {
    "verbose": true,
    "validation": false
  }
}
```

### Too Many Duplicate Tasks
```json
{
  "advanced": {
    "deduplication_threshold": 0.6
  }
}
```