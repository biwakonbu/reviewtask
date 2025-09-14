# Configuration Guide

reviewtask uses a flexible configuration system that allows customization of priority rules, AI processing, and task management behavior.

## Configuration File

Configuration is stored in `.pr-review/config.json` in your repository:

```json
{
  "priority_rules": {
    "critical": "Security vulnerabilities, authentication bypasses, data exposure risks",
    "high": "Performance bottlenecks, memory leaks, database optimization issues",
    "medium": "Functional bugs, logic improvements, error handling",
    "low": "Code style, naming conventions, comment improvements"
  },
  "task_settings": {
    "default_status": "todo",
    "auto_prioritize": true,
    "low_priority_patterns": ["nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"],
    "low_priority_status": "pending"
  },
  "ai_settings": {
    "user_language": "English",
    "validation_enabled": false,
    "verbose_mode": true
  }
}
```

## Priority Rules

Customize how AI analyzes and prioritizes review comments:

### Critical Priority
Comments matching this description get highest priority:
```json
"critical": "Security vulnerabilities, authentication bypasses, data exposure risks"
```

Examples of critical issues:
- SQL injection vulnerabilities
- Authentication bypasses
- Exposed API keys or secrets
- Data privacy violations

### High Priority
Important functionality and performance issues:
```json
"high": "Performance bottlenecks, memory leaks, database optimization issues"
```

Examples of high priority issues:
- Memory leaks
- N+1 query problems
- Blocking I/O operations
- Resource exhaustion risks

### Medium Priority
Standard functional improvements:
```json
"medium": "Functional bugs, logic improvements, error handling"
```

Examples of medium priority issues:
- Logic errors
- Missing error handling
- Edge case bugs
- API inconsistencies

### Low Priority
Style and minor improvements:
```json
"low": "Code style, naming conventions, comment improvements"
```

Examples of low priority issues:
- Code formatting
- Variable naming
- Comment quality
- Documentation improvements

## Task Settings

### Default Status
Set the default status for new tasks:
```json
"default_status": "todo"
```

Options: `todo`, `doing`, `done`, `pending`, `cancel`

### Auto-Prioritization
Enable automatic priority assignment based on priority rules:
```json
"auto_prioritize": true
```

### Low-Priority Detection

Automatically detect and handle low-priority comments:

```json
"low_priority_patterns": ["nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"],
"low_priority_status": "pending"
```

**Pattern Matching:**
- Patterns are case-insensitive
- Matches comments starting with patterns or containing them after newlines
- Common patterns for code review tools (like CodeRabbit nits)

**Example matches:**
- "nit: Consider using const instead of let"
- "minor: This could be optimized"
- "suggestion: Maybe extract this to a function"

## AI Settings

### User Language
Set the language for AI-generated content:
```json
"user_language": "English"
```

Supported languages include English, Japanese, and others based on AI provider capabilities.

### Processing Modes

#### Standard Mode (Default)
```json
"validation_enabled": false
```

Fast processing with individual comment analysis.

#### Validation Mode
```json
"validation_enabled": true,
"max_retries": 5,
"quality_threshold": 0.8
```

Two-stage validation with retry logic and quality scoring.

### Advanced AI Settings

```json
{
  "ai_settings": {
    "verbose_mode": false,               // Enable detailed debug logging
    "validation_enabled": true,          // Enable AI task validation
    "max_retries": 5,                    // Validation retry attempts
    "quality_threshold": 0.8,            // Minimum validation score (0.0-1.0)
    "deduplication_enabled": true,       // AI-powered task deduplication
    "similarity_threshold": 0.8,         // Task similarity detection threshold
    "process_nitpick_comments": false,   // Process CodeRabbit nitpick comments
    "nitpick_priority": "low",           // Priority for nitpick-generated tasks
    "enable_json_recovery": true,        // Enable JSON recovery for incomplete responses
    "max_recovery_attempts": 3,          // Maximum JSON recovery attempts
    "partial_response_threshold": 0.7,   // Minimum threshold for partial responses
"log_truncated_responses": true      // Log truncated responses for debugging
  }
}
```

### Prompt Profiles (Default: v2)

Select the prompt style used for task generation:

```json
{
  "ai_settings": {
    "prompt_profile": "v2"  // one of: v2 (alias: rich), compact, minimal, legacy
  }
}
```

Tips:
- `v2` is the default and recommended for most cases.
- Use `legacy` only to compare with previous behavior or for fallback.
- Render prompts locally without AI to inspect differences:
```bash
reviewtask debug fetch review 123
reviewtask debug prompt 123 --profile v2
reviewtask debug prompt 123 --profile legacy
```

### JSON Recovery and Retry Features

Advanced recovery mechanisms for handling incomplete AI responses:

#### JSON Recovery
```json
"enable_json_recovery": true,
"max_recovery_attempts": 3,
"partial_response_threshold": 0.7
```

- Extracts valid tasks from truncated JSON responses
- Cleans up malformed JSON syntax
- Validates recovered data before processing

#### Intelligent Retry
```json
"max_retries": 5
```

Smart retry strategies based on error patterns:
- Automatic prompt size reduction for token limit errors
- Exponential backoff for rate limiting
- Pattern detection for common truncation issues

#### Response Monitoring
```json
"log_truncated_responses": true
```

Tracks API performance and provides optimization insights:
- Response size and truncation pattern analysis
- Success rate tracking and error distribution
- Performance analytics in `.pr-review/response_analytics.json`

### Task Deduplication

AI-powered task deduplication prevents duplicate tasks:

```json
"deduplication_enabled": true,
"similarity_threshold": 0.8
```

- Compares task content using AI similarity analysis
- Configurable similarity threshold (0.0-1.0)
- Preserves existing task statuses during deduplication

### CodeRabbit Integration

Special handling for CodeRabbit code review comments:

```json
"process_nitpick_comments": false,
"nitpick_priority": "low"
```

- Control whether to process nitpick comments as tasks
- Set priority level for nitpick-generated tasks
- Integrates with low-priority detection patterns

## Environment Variables

Override configuration using environment variables:

```bash
# GitHub authentication
export GITHUB_TOKEN="your_github_token"

# AI provider settings
export REVIEWTASK_VERBOSE=true
export REVIEWTASK_VALIDATION=true

# GitHub Enterprise (if applicable)
export GITHUB_API_URL="https://github.company.com/api/v3"
```

## Configuration Examples

### Development Team Setup
For a development team focusing on security and performance:

```json
{
  "priority_rules": {
    "critical": "Security vulnerabilities, authentication issues, data exposure, production failures",
    "high": "Performance problems, memory leaks, database issues, breaking changes",
    "medium": "Functional bugs, error handling, business logic issues",
    "low": "Code style, documentation, minor refactoring suggestions"
  },
  "task_settings": {
    "default_status": "todo",
    "auto_prioritize": true,
    "low_priority_patterns": ["nit:", "style:", "docs:", "typo:", "minor:"],
    "low_priority_status": "pending"
  },
  "ai_settings": {
    "validation_enabled": true,
    "quality_threshold": 0.9,
    "deduplication_enabled": true,
    "verbose_mode": false
  }
}
```

### Open Source Project Setup
For open source projects with many contributors:

```json
{
  "priority_rules": {
    "critical": "Security issues, breaking changes, license violations",
    "high": "API changes, performance regressions, compatibility issues",
    "medium": "Feature bugs, documentation errors, test failures",
    "low": "Code style, variable naming, comment improvements"
  },
  "task_settings": {
    "default_status": "todo",
    "low_priority_patterns": ["nit:", "suggestion:", "consider:", "optional:"],
    "low_priority_status": "pending"
  },
  "ai_settings": {
    "process_nitpick_comments": false,
    "deduplication_enabled": true,
    "similarity_threshold": 0.85
  }
}
```

### Debug and Development Setup
For debugging and development work:

```json
{
  "ai_settings": {
    "verbose_mode": true,
    "validation_enabled": true,
    "max_retries": 3,
    "enable_json_recovery": true,
    "log_truncated_responses": true,
    "deduplication_enabled": false
  }
}
```

## Configuration Best Practices

### Priority Rules
- Use specific, actionable descriptions
- Align with your team's development priorities
- Include examples that match your codebase
- Review and update regularly based on experience

### Task Settings
- Set realistic default statuses for your workflow
- Customize low-priority patterns for your review style
- Consider your team's task management preferences

### AI Settings
- Enable validation mode for critical projects
- Use verbose mode during initial setup and troubleshooting
- Adjust similarity thresholds based on your task deduplication needs
- Enable JSON recovery for better reliability

### Performance Tuning
- Start with default settings
- Enable verbose mode to understand processing behavior
- Adjust retry and threshold settings based on your AI provider's characteristics
- Monitor response analytics for optimization opportunities

## Troubleshooting Configuration

### Invalid Configuration
```bash
# Validate configuration syntax
reviewtask init  # Re-initializes with defaults if config is invalid
```

### Priority Assignment Issues
```bash
# Enable verbose mode to see priority assignment logic
# Edit .pr-review/config.json:
{
  "ai_settings": {
    "verbose_mode": true
  }
}
```

### Task Deduplication Problems
```bash
# Adjust similarity threshold or disable deduplication
{
  "ai_settings": {
    "deduplication_enabled": false,
    "similarity_threshold": 0.7  // Lower = less strict
  }
}
```

### Processing Performance
```bash
# Enable validation and recovery for better reliability
{
  "ai_settings": {
    "validation_enabled": true,
    "enable_json_recovery": true,
    "max_retries": 5
  }
}
```

For more advanced configuration scenarios, see the [Troubleshooting Guide](troubleshooting.md) or check the project's [GitHub repository](https://github.com/biwakonbu/reviewtask) for examples.
