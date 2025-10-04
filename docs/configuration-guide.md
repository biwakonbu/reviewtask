# Configuration Guide

This guide helps you configure reviewtask for your specific needs.

## Getting Started

### Option 1: Interactive Setup (Recommended)

```bash
reviewtask init
```

This interactive wizard will:
1. Ask for your preferred language
2. Auto-detect available AI providers
3. Create a minimal configuration
4. Set up your repository

### Option 2: Manual Minimal Configuration

Create `.pr-review/config.json`:

```json
{
  "language": "English",
  "ai_provider": "auto"
}
```

That's all you need! The tool handles everything else automatically.

## Common Configuration Scenarios

### Scenario: Japanese Language Support

```json
{
  "language": "Japanese",
  "ai_provider": "auto"
}
```

### Scenario: Using Specific AI Provider

**For Cursor CLI with Grok model:**
```json
{
  "language": "English",
  "ai_provider": "cursor",
  "model": "grok"
}
```

**For Claude Code with Opus model:**
```json
{
  "language": "English",
  "ai_provider": "claude",
  "model": "opus"
}
```

### Scenario: Custom Priority Rules for Security Project

```json
{
  "language": "English",
  "ai_provider": "auto",
  "priorities": {
    "project_specific": {
      "critical": "SQL injection, XSS vulnerabilities, Auth bypass",
      "high": "Session management, Input validation errors",
      "medium": "CORS issues, Rate limiting",
      "low": "Documentation, Code comments"
    }
  }
}
```

### Scenario: Custom Build Commands for Monorepo

```json
{
  "language": "English",
  "ai_provider": "auto",
  "verification_settings": {
    "build_command": "npm run build:all",
    "test_command": "npm run test:all",
    "lint_command": "npm run lint:all"
  }
}
```

## Configuration Management

### Check Your Configuration

```bash
# Show current configuration
reviewtask config show

# Validate configuration
reviewtask config validate
```

### Migrate from Old Format

If you have an existing detailed configuration:

```bash
# Convert to simplified format
reviewtask config migrate
```

Before (46+ lines):
```json
{
  "priority_rules": {
    "critical": "...",
    "high": "...",
    // ... many more settings
  },
  "ai_settings": {
    "user_language": "English",
    "output_format": "json",
    "max_retries": 5,
    // ... 20+ more settings
  }
  // ... more sections
}
```

After (4 lines):
```json
{
  "language": "English",
  "ai_provider": "cursor",
  "model": "grok"
}
```

## Project Type Auto-Detection

The tool automatically detects your project type and configures appropriate commands:

| If it finds... | Project Type | Build | Test | Lint |
|---------------|--------------|-------|------|------|
| `go.mod` | Go | `go build ./...` | `go test ./...` | `golangci-lint run` |
| `package.json` | Node.js | `npm run build` | `npm test` | `npm run lint` |
| `Cargo.toml` | Rust | `cargo build` | `cargo test` | `cargo clippy` |
| `requirements.txt` | Python | `python -m py_compile .` | `pytest` | `pylint .` |
| `pom.xml` | Java (Maven) | `mvn compile` | `mvn test` | `mvn checkstyle:check` |
| `Gemfile` | Ruby | `bundle install` | `bundle exec rspec` | `rubocop` |

## AI Provider Selection

### Auto Mode (Default)

When `ai_provider: "auto"` is set:
1. Tries Cursor CLI first (if installed)
2. Falls back to Claude Code (if installed)
3. Shows error if neither is available

### Manual Provider Selection

**Cursor CLI:**
- Best for: Fast responses, general development
- Models: `grok` (recommended), `auto`

**Claude Code:**
- Best for: Complex analysis, detailed explanations
- Models: `sonnet` (default), `opus` (premium)

## Advanced Configuration

For power users who need fine control, you can use advanced settings:

```json
{
  "language": "English",
  "ai_provider": "cursor",
  "model": "grok",
  "ai_settings": {
    "verbose_mode": true,
    "validation_enabled": false,
    "prompt_profile": "compact",
    "advanced": {
      "max_retries": 3,
      "timeout_seconds": 180,
      "deduplication_threshold": 0.7
    }
  }
}
```

See [Configuration Reference](config-reference.md) for all parameters.

## Troubleshooting

### Issue: AI Provider Not Found

**Solution:** Specify the path explicitly:

```json
{
  "language": "English",
  "ai_provider": "cursor",
  "cursor_path": "/usr/local/bin/cursor"
}
```

### Issue: Wrong Build Commands Detected

**Solution:** Override with custom commands:

```json
{
  "language": "English",
  "ai_provider": "auto",
  "verification_settings": {
    "build_command": "make build",
    "test_command": "make test"
  }
}
```

### Issue: Too Many Similar Tasks

**Solution:** Adjust deduplication threshold:

```json
{
  "language": "English",
  "ai_provider": "auto",
  "ai_settings": {
    "advanced": {
      "deduplication_threshold": 0.6
    }
  }
}
```

### Issue: Tasks Not Being Generated

**Solution:** Enable verbose mode for debugging:

```json
{
  "language": "English",
  "ai_provider": "auto",
  "ai_settings": {
    "verbose_mode": true
  }
}
```

## Best Practices

1. **Start Simple**: Begin with the 2-line minimal configuration
2. **Use Auto-Detection**: Let the tool detect your project type
3. **Add Gradually**: Only add settings when you need to override defaults
4. **Validate Changes**: Run `reviewtask config validate` after editing
5. **Keep Backups**: The migrate command automatically creates backups

## Migration Path

### From No Configuration
1. Run `reviewtask init`
2. Answer the interactive prompts
3. Done!

### From Old Detailed Configuration
1. Run `reviewtask config migrate`
2. Review the simplified configuration
3. Test with `reviewtask config validate`
4. Original config is backed up as `config.json.backup`

## Environment Variables

You can override configuration with environment variables:

```bash
# Override AI provider
REVIEWTASK_AI_PROVIDER=claude reviewtask analyze

# Skip authentication checks (useful in CI)
SKIP_CLAUDE_AUTH_CHECK=true reviewtask analyze
```

## Next Steps

- See [Configuration Reference](config-reference.md) for all parameters
- Learn about [AI Providers](ai-providers.md)
- Read the [Troubleshooting Guide](troubleshooting.md)