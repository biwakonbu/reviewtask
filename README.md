# reviewtask - AI-Powered PR Review Management Tool

[![Latest Release](https://img.shields.io/github/v/release/biwakonbu/reviewtask)](https://github.com/biwakonbu/reviewtask/releases/latest)
[![CI](https://github.com/biwakonbu/reviewtask/workflows/CI/badge.svg)](https://github.com/biwakonbu/reviewtask/actions)
[![codecov](https://codecov.io/gh/biwakonbu/reviewtask/branch/main/graph/badge.svg)](https://codecov.io/gh/biwakonbu/reviewtask)
[![Go Report Card](https://goreportcard.com/badge/github.com/biwakonbu/reviewtask)](https://goreportcard.com/report/github.com/biwakonbu/reviewtask)
[![GoDoc](https://godoc.org/github.com/biwakonbu/reviewtask?status.svg)](https://godoc.org/github.com/biwakonbu/reviewtask)

A CLI tool that fetches GitHub Pull Request reviews, analyzes them using AI, and generates actionable tasks for developers to address feedback systematically.

## Features

- **🔍 PR Review Fetching**: Automatically retrieves reviews from GitHub API with nested comment structure
- **🤖 AI Analysis**: Supports multiple AI providers (Claude Code, Cursor CLI) for generating structured, actionable tasks
- **💾 Local Storage**: Stores data in structured JSON format under `.pr-review/` directory
- **📋 Task Management**: Full lifecycle management with status tracking (todo/doing/done/pending/cancel)
- **⚡ Parallel Processing**: Processes multiple comments concurrently for improved performance
- **🔒 Authentication**: Multi-source token detection with interactive setup
- **🎯 Priority-based Analysis**: Customizable priority rules for task generation
- **🔄 Task State Preservation**: Maintains existing task statuses during subsequent runs
- **🆔 UUID-based Task IDs**: Unique task identification to eliminate duplication issues
- **🔌 Extensible AI Provider Support**: Architecture designed for easy integration of multiple AI providers
- **👁️ AI Provider Transparency**: Displays current AI provider and model at the start of every command
- **🏷️ Low-Priority Detection**: Automatically identifies and assigns "pending" status to low-priority comments (nits, suggestions)
- **⏱️ Smart Performance**: Automatic optimization based on PR size with no configuration needed
- **💨 API Caching**: Reduces redundant GitHub API calls automatically
- **📊 Auto-Resume**: Seamlessly continues from where it left off if interrupted
- **🔧 Debug Commands**: Test specific phases independently for troubleshooting
- **📏 Prompt Size Optimization**: Automatic chunking for large comments (>20KB) and pre-validation size checks
- **✅ Task Validation**: AI-powered validation with configurable quality thresholds and retry logic
- **🖥️ Verbose Mode**: Detailed logging and debugging output for development and troubleshooting
- **🔄 Smart Deduplication**: AI-powered task deduplication with similarity threshold control
- **🛡️ JSON Recovery**: Automatic recovery from incomplete Claude API responses with partial task extraction
- **🔁 Intelligent Retry**: Smart retry strategies with pattern detection and prompt size adjustment
- **📊 Response Monitoring**: Performance analytics and optimization recommendations for API usage
- **🧠 AI Prompt Preservation**: Preserves "🤖 Prompt for AI Agents" blocks from CodeRabbit while removing verbose metadata
- **📦 File Size Optimization**: Achieves up to 66% reduction in reviews.json size (200KB → 67KB) while maintaining essential content
- **🔤 HTML Entity Processing**: Properly handles Unicode HTML entities and GitHub API response variations

## Installation

### Quick Install (Recommended)

**Unix/Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash
```

**Windows (PowerShell):**
```powershell
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex
```

### Default Installation Locations

- **Unix/Linux/macOS**: `~/.local/bin` (user's local directory, no sudo required)
- **Windows**: `%USERPROFILE%\bin` (e.g., `C:\Users\username\bin`)

### PATH Configuration

The installation script will automatically detect your shell and provide specific instructions. If `~/.local/bin` is not in your PATH, you'll see instructions like:

**For Bash users:**
```bash
# Add to ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"

# Reload configuration
source ~/.bashrc
```

**For Zsh users:**
```bash
# Add to ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"

# Reload configuration
source ~/.zshrc
```

**For Fish users:**
```fish
# Add to ~/.config/fish/config.fish
set -gx PATH $HOME/.local/bin $PATH

# Reload configuration
source ~/.config/fish/config.fish
```

### System-wide Installation

For system-wide installation (requires sudo):
```bash
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | sudo bash -s -- --bin-dir /usr/local/bin
```

For detailed installation information including PATH configuration and troubleshooting, see [Installation Guide](docs/INSTALLATION.md).

### Installation Options

**Install specific version:**
```bash
# Unix/Linux/macOS
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --version v1.2.3

# Windows
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-Version", "v1.2.3"
```

**Install to custom directory:**
```bash
# Unix/Linux/macOS
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --bin-dir ~/bin

# Windows
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-BinDir", "C:\tools"
```

**Force overwrite existing installation:**
```bash
# Unix/Linux/macOS
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --force

# Windows
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-Force"
```

### Manual Installation

**Download Release Binary:**

Download the latest release for your platform:

```bash
# Download latest release (Linux/macOS/Windows)
curl -L https://github.com/biwakonbu/reviewtask/releases/latest/download/reviewtask-<version>-<os>-<arch>.tar.gz | tar xz

# Make executable and move to PATH
chmod +x reviewtask-<version>-<os>-<arch>
sudo mv reviewtask-<version>-<os>-<arch> /usr/local/bin/reviewtask
```

**Install with Go:**

```bash
go install github.com/biwakonbu/reviewtask@latest
```

### Build from Source

1. Clone the repository:
```bash
git clone https://github.com/biwakonbu/reviewtask.git
cd reviewtask
```

2. Build the binary:
```bash
go build -o reviewtask
```

3. Install AI Provider CLI (required for AI analysis):
```bash
# For Claude Code (default)
# Follow Claude Code installation instructions
# https://docs.anthropic.com/en/docs/claude-code

# For other providers (future support)
# Install the respective provider's CLI tool
```

### Verify Installation

```bash
# Check version and build information
reviewtask version
```

## Quick Start

### 1. Initialize Repository

```bash
# Interactive setup wizard
./reviewtask init
```

The wizard will:
- Ask for your preferred language (English/Japanese)
- Auto-detect available AI providers (Cursor CLI, Claude Code)
- Create a minimal 2-line configuration
- Set up `.gitignore` entries
- Check GitHub authentication

**Example session:**
```
Welcome to reviewtask setup!
What language do you prefer? [English/Japanese]: English
Detecting AI providers...
Found: Cursor CLI
Use Cursor CLI as AI provider? [Y/n]: Y
✓ Minimal configuration created at .pr-review/config.json
```

### 2. Authentication Setup

```bash
# Login with GitHub token
./reviewtask auth login

# Check authentication status
./reviewtask auth status

# Logout
./reviewtask auth logout
```

Authentication sources (in order of preference):
1. `GITHUB_TOKEN` environment variable
2. Local config file (`.pr-review/auth.json`)
3. GitHub CLI (`gh auth token`)

### 3. Analyze PR Reviews

```bash
# Analyze current branch's PR
./reviewtask

# Analyze specific PR
./reviewtask 123

# The tool will:
# - Fetch PR reviews and comments
# - Automatically optimize performance based on PR size
# - Process comments in parallel batches
# - Cache API responses to reduce redundant calls
# - Support automatic resume if interrupted
# - Generate actionable tasks with priorities
# - Save results to .pr-review/PR-{number}/
```

### 4. Task Management

```bash
# View all task status
./reviewtask status

# Show current/next task details
./reviewtask show

# Show specific task details
./reviewtask show <task-id>

# Update specific task status
./reviewtask update <task-id> <status>

# Valid statuses: todo, doing, done, pending, cancel
```


## Command Reference

| Command | Description |
|---------|-------------|
| `reviewtask [PR_NUMBER]` | Analyze current branch's PR or specific PR |
| `reviewtask --refresh-cache` | Clear cache and reprocess all comments |
| `reviewtask fetch [PR_NUMBER]` | Same as reviewtask (alias) |
| `reviewtask status [options]` | Show task status and statistics |
| `reviewtask show [task-id]` | Show current/next task or specific task details |
| `reviewtask update <id> <status>` | Update task status |
| `reviewtask stats [PR_NUMBER] [options]` | Show detailed task statistics with comment breakdown |
| `reviewtask version [VERSION]` | Show version information or switch to specific version |
| `reviewtask versions` | List available versions from GitHub releases |
| `reviewtask prompt <provider> <target>` | Generate AI provider command templates |
| `reviewtask claude <target>` | (Deprecated) Use `reviewtask prompt claude <target>` |
| `reviewtask debug fetch <phase> [PR]` | Test specific phases independently |
| `reviewtask init` | Initialize repository |
| `reviewtask auth <cmd>` | Authentication management |

### Command Options

#### Global Options
- `--refresh-cache` - Clear cache and reprocess all comments (available with main command)

#### Status and Stats Options  
- `--all` - Show information for all PRs
- `--pr <number>` - Show information for specific PR
- `--branch <name>` - Show information for specific branch

#### Authentication Commands
- `reviewtask auth login` - Interactive GitHub token setup
- `reviewtask auth status` - Show current authentication source and user
- `reviewtask auth logout` - Remove local authentication
- `reviewtask auth check` - Comprehensive validation of token and permissions

#### Version Commands
- `reviewtask version` - Show current version with update check
- `reviewtask version <VERSION>` - Switch to specific version (e.g., `v1.2.3`, `latest`)
- `reviewtask version --check` - Check for available updates
- `reviewtask versions` - List recent 5 versions with release information

#### AI Provider Integration
- `reviewtask prompt claude pr-review` - Generate PR review workflow template for Claude Code
- `reviewtask cursor pr-review` - Generate Cursor IDE integration files (.cursorrules and custom commands)
- `reviewtask prompt stdout <target>` - Output prompts to stdout for redirection or piping
- `reviewtask prompt <provider> <target>` - Generate templates for various AI providers (extensible)

#### Debug Commands
- `reviewtask debug fetch review <PR>` - Fetch and save PR reviews only (no task generation)
- `reviewtask debug fetch task <PR>` - Generate tasks from previously saved reviews only
- Debug commands automatically enable verbose mode for detailed logging

## Configuration

### 🚀 Quick Configuration (NEW!)

**Start with just 2 lines of configuration:**

```json
{
  "language": "English",
  "ai_provider": "auto"
}
```

That's it! The tool will automatically:
- Detect your project type (Go, Node.js, Rust, Python, etc.)
- Configure appropriate build/test/lint commands
- Find and use available AI providers (Cursor CLI or Claude Code)
- Apply sensible defaults for all other settings

### Configuration Management Commands

```bash
# Interactive setup wizard
reviewtask init

# Validate your configuration
reviewtask config validate

# Migrate existing config to simplified format
reviewtask config migrate

# Show current configuration
reviewtask config show
```

### Configuration Levels

#### Level 1: Minimal (90% of users)
```json
{
  "language": "English",
  "ai_provider": "auto"
}
```

#### Level 2: Basic Customization
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

#### Level 3: Advanced (Power Users)
See [Configuration Reference](docs/CONFIG_REFERENCE.md) for all available parameters.

### Prompt Profiles

Control the prompt style used for task generation. Default is `v2`.

```json
{
  "ai_settings": {
    "prompt_profile": "v2"  // one of: v2 (alias: rich), compact, minimal, legacy
  }
}
```

Render the exact prompt (offline, no AI) from saved reviews for inspection or A/B comparison:

```bash
reviewtask debug fetch review 123          # Save .pr-review/PR-123/reviews.json
reviewtask debug prompt 123 --profile v2   # Print v2 prompt to stdout
reviewtask debug prompt 123 --profile legacy
```

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

### Low-Priority Comment Detection

The tool can automatically detect and handle low-priority comments (such as "nits" from code review tools):

- **`low_priority_patterns`**: List of patterns to identify low-priority comments (case-insensitive)
  - Default patterns: `["nit:", "nits:", "minor:", "suggestion:", "consider:", "optional:", "style:"]`
  - Matches comments starting with these patterns or containing them after a newline
- **`low_priority_status`**: Status to assign to tasks from matching comments (default: `"pending"`)
  - This allows developers to focus on critical issues first
  - Low-priority tasks can be addressed later or promoted to active status

Example: A comment like "nit: Consider using const instead of let" will create a task with `"pending"` status instead of `"todo"`.

### AI Provider Configuration

Configure AI provider and model settings:

```json
{
  "ai_settings": {
    "ai_provider": "auto",                // Options: "claude", "cursor", "auto" (tries cursor then claude)
    "model": "auto",                      // Model selection (auto lets provider choose best model)
    "cursor_path": "",                    // Optional custom path to cursor-agent CLI
    "claude_path": ""                     // Optional custom path to Claude CLI
  }
}
```

**Supported AI Providers:**
- **Claude Code CLI**: The original Claude AI provider (`npm install -g @anthropic-ai/claude-code`)
- **Cursor CLI**: Cursor's AI with automatic model selection (`npm install -g cursor-agent`)
- **Auto**: Automatically tries Cursor first, falls back to Claude if unavailable

### Advanced AI Settings

Configure advanced processing features in `.pr-review/config.json`:

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
    "log_truncated_responses": true,     // Log truncated responses for debugging
    "preserve_ai_prompts": true,         // Preserve "🤖 Prompt for AI Agents" blocks from CodeRabbit
    "optimize_file_size": true,          // Enable file size optimization (removes verbose metadata)
    "html_entity_processing": true       // Process HTML entities in GitHub API responses
    "process_self_reviews": false        // Process self-review comments from PR author
  }
}
```

### AI Prompt Preservation & File Size Optimization

The tool intelligently optimizes review data storage while preserving essential AI guidance:

#### AI Prompt Preservation
- **Preserves CodeRabbit AI Prompts**: Keeps "🤖 Prompt for AI Agents" blocks intact for enhanced task generation
- **Smart Content Filtering**: Removes verbose GitHub suggestion blocks and metadata while maintaining review essence
- **HTML Entity Support**: Properly processes both HTML-escaped (`\u003c`, `\u003e`) and normal HTML content

#### File Size Optimization
- **Significant Size Reduction**: Achieves up to 66% reduction in `reviews.json` file size (e.g., 200KB → 67KB)
- **Intelligent Metadata Removal**: Strips GitHub suggestion blocks, committable suggestions, and fingerprinting comments
- **Content Structure Preservation**: Maintains markdown formatting, code references, and essential feedback

#### Configuration Options
```json
{
  "ai_settings": {
    "preserve_ai_prompts": true,         // Keep AI prompt blocks from CodeRabbit
    "optimize_file_size": true,          // Enable comprehensive size optimization
    "html_entity_processing": true       // Handle HTML entity variations
  }
}
```

#### Benefits
- **💾 Reduced Storage**: Smaller JSON files for faster processing and reduced disk usage
- **🧠 Enhanced AI Analysis**: Preserved AI prompts provide better context for task generation
- **⚡ Improved Performance**: Smaller data files lead to faster processing and analysis
- **🔧 Better Compatibility**: Handles various GitHub API response formats consistently

### Self-Review Processing

The tool can process self-reviews (comments made by the PR author on their own PR):

- **`process_self_reviews`**: Enable processing of PR author's own comments (default: `false`)
  - When enabled, fetches both issue comments and PR review comments from the author
  - Self-review comments are processed through the same AI task generation pipeline
  - Useful for capturing TODO comments, known issues, and self-documentation

Example use cases:
- Authors documenting known issues or technical debt
- TODO comments for follow-up work
- Self-review before requesting external reviews
- Design decisions and trade-offs documentation

To enable self-review processing:
```json
{
  "ai_settings": {
    "process_self_reviews": true
  }
}
```

### JSON Recovery and Retry Features

The tool now includes advanced recovery mechanisms for handling incomplete Claude API responses:

- **JSON Recovery**: Automatically recovers valid tasks from truncated or malformed JSON responses
  - Extracts complete task objects from partial arrays
  - Cleans up malformed JSON syntax
  - Validates recovered data before processing
  - Configurable recovery attempts and thresholds

- **Intelligent Retry**: Smart retry strategies based on error patterns
  - Automatic prompt size reduction for token limit errors
  - Exponential backoff for rate limiting
  - Pattern detection for common truncation issues
  - Configurable retry attempts and delays

- **Response Monitoring**: Tracks API performance and provides optimization insights
  - Response size and truncation pattern analysis
  - Success rate tracking and error distribution
  - Optimal prompt size recommendations
  - Performance analytics and reporting

### Processing Modes

- **Parallel Mode** (`validation_enabled: false`): Fast processing with individual comment analysis
- **Validation Mode** (`validation_enabled: true`): Two-stage validation with retry logic and quality scoring
- **Verbose Mode** (`verbose_mode: true`): Detailed logging for debugging and development
- **Automatic Chunking**: Large comments (>20KB) are automatically split for optimal processing

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

## IDE Integration

### Cursor IDE Integration

Generate Cursor-specific integration files for enhanced development experience:

```bash
# Generate Cursor IDE integration files
reviewtask cursor pr-review
```

This creates:
- **`.cursorrules`**: AI context rules that help Cursor understand your reviewtask workflow
- **`.cursor/commands/pr-review/`**: Custom command templates for common operations

After running this command, Cursor IDE will:
- Understand reviewtask commands and suggest appropriate usage
- Provide context-aware assistance for PR review workflows
- Support custom commands through the command palette

### Claude Code Integration

Generate Claude Code command templates:

```bash
# Generate Claude Code workflow template
reviewtask prompt claude pr-review
```

This creates workflow templates in `.claude/commands/` for streamlined PR review management.

## Advanced Features

### Task State Preservation

- Existing task statuses are preserved during subsequent review fetches
- Comment content changes trigger automatic task cancellation
- New tasks are added without overwriting existing work progress

### Parallel Processing

- Each comment is processed independently using goroutines
- Reduced prompt sizes (3,000-6,000 characters vs 57,760)
- Better performance and AI provider reliability

### Comment Change Detection

- Automatically detects significant changes in comment content
- Cancels outdated tasks and creates new ones as needed
- Preserves completed work and prevents duplicate tasks

### Statistics and Analytics

Use the `reviewtask stats` command to get detailed task analytics:

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

#### Statistics Output Format
- **Comment-level breakdown**: Task counts per review comment
- **Priority distribution**: Critical/high/medium/low task counts  
- **Status distribution**: Todo/doing/done/pending/cancel counts
- **Completion metrics**: Task completion rates and progress tracking
- **File-level summary**: Tasks grouped by affected files

### Version Management and Updates

The tool includes built-in version management capabilities:

```bash
# Show current version and check for updates
reviewtask version

# List available versions from GitHub releases
reviewtask versions

# Switch to specific version
reviewtask version v1.2.3
reviewtask version latest

# Check for updates only
reviewtask version --check
```

#### Self-Update Features
- **Automatic update detection**: Checks for newer versions on startup
- **GitHub releases integration**: Downloads binaries directly from GitHub
- **Version switching**: Easy switching between versions
- **Rollback capability**: Return to previous versions if needed

### Cache Management

Improve performance and handle data consistency with cache controls:

```bash
# Force cache refresh (reprocess all comments)
reviewtask --refresh-cache

# When to use --refresh-cache:
# - After significant PR changes
# - When comment content has been updated
# - To regenerate tasks with updated priority rules
# - Troubleshooting inconsistent task generation
```

#### Cache Behavior
- **Performance optimization**: Avoids re-processing unchanged comments
- **Consistency preservation**: Maintains task state across runs  
- **Selective refresh**: Only processes changed or new content
- **Manual override**: `--refresh-cache` bypasses all caching

### AI Provider Integration

Streamline your AI workflows with generated templates for various providers:

```bash
# Generate PR review workflow template for Claude Code (writes to .claude/commands/)
reviewtask prompt claude pr-review

# Output prompts to stdout for redirection or piping
reviewtask prompt stdout pr-review                    # Display on terminal
reviewtask prompt stdout pr-review > my-workflow.md   # Save to custom file
reviewtask prompt stdout pr-review | pbcopy           # Copy to clipboard (macOS)
reviewtask prompt stdout pr-review | xclip            # Copy to clipboard (Linux)

# Extensible architecture for future AI providers
# reviewtask prompt <provider> <target>
```

This provides flexible options for AI integration:
- **Claude provider**: Creates optimized command templates in `.claude/commands/` directory
- **Stdout provider**: Outputs prompts to standard output for maximum flexibility
- Structured PR review analysis workflows
- Task generation and management integration
- Consistent review quality and format
- Integration with existing reviewtask data structures

**Note**: The `reviewtask claude` command is deprecated. Please use `reviewtask prompt claude` for future compatibility.

## Troubleshooting

### Authentication Issues

```bash
# Check token permissions and repository access
reviewtask auth check

# View current authentication status
reviewtask auth status

# Re-authenticate if needed
reviewtask auth logout
reviewtask auth login

# Common solutions:
export GITHUB_TOKEN="your_token_here"
# or
gh auth login
```

### Version and Update Issues

```bash
# Check current version and available updates
reviewtask version

# View available versions
reviewtask versions

# Switch to stable version if experiencing issues
reviewtask version latest

# Manually check GitHub releases
# https://github.com/biwakonbu/reviewtask/releases
```

### Cache and Performance Issues

```bash
# Clear cache and reprocess all data
reviewtask --refresh-cache

# Check statistics for diagnostic information
reviewtask stats --all

# Symptoms requiring cache refresh:
# - Inconsistent task generation
# - Missing tasks for recent comments
# - Outdated task content
```

### AI Provider Integration

Ensure your AI provider CLI is properly installed and accessible:

```bash
# Test Claude Code availability (for Claude provider)
claude --version

# Generate integration templates if missing
reviewtask prompt claude pr-review

# Common issues:
# - AI provider CLI not in PATH
# - Authentication required
# - Network connectivity
```

### JSON Recovery and API Response Issues

Handle incomplete or truncated Claude API responses:

```bash
# Enable verbose mode to see recovery attempts
# Edit .pr-review/config.json:
{
  "ai_settings": {
    "verbose_mode": true,
    "enable_json_recovery": true
  }
}

# Common recovery scenarios:
# - "unexpected end of JSON input" errors
# - Truncated responses at token limits
# - Malformed JSON from API timeouts
# - Partial task arrays

# Monitor API performance:
# Check .pr-review/response_analytics.json for patterns
```

### Permission Requirements

Required GitHub API permissions:
- `repo` (for private repositories)
- `public_repo` (for public repositories)
- `read:org` (for organization repositories)

Use `reviewtask auth check` for comprehensive permission validation.

## Documentation

### 📚 For Users
Start here if you want to use reviewtask to manage PR reviews:
- [Installation Guide](docs/user-guide/installation.md)
- [Quick Start Tutorial](docs/user-guide/quick-start.md)
- [Authentication Setup](docs/user-guide/authentication.md)
- [Command Reference](docs/user-guide/commands.md)
- [Configuration Guide](docs/user-guide/configuration.md)
- [Workflow Guide](docs/user-guide/workflow.md)
- [Troubleshooting](docs/user-guide/troubleshooting.md)

### 🔧 For Developers
Start here if you want to contribute or extend reviewtask:
- [Architecture Overview](docs/developer-guide/architecture.md)
- [Development Setup](docs/developer-guide/development-setup.md)
- [Project Structure](docs/developer-guide/project-structure.md)
- [Testing Strategy](docs/developer-guide/testing.md)
- [Contributing Guidelines](docs/developer-guide/contributing.md)
- [Versioning & Releases](docs/developer-guide/versioning.md)

## Contributing

Please see our [Contributing Guide](CONTRIBUTING.md) for detailed information on:
- Development setup and guidelines
- Pull request process
- Release labeling system
- Code style and testing

### Quick Start for Contributors

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add appropriate release label (`release:major`, `release:minor`, or `release:patch`)
5. Submit a pull request

### Development Documentation

- [Contributing Guide](CONTRIBUTING.md) - Detailed contribution guidelines
- [Versioning Guide](docs/VERSIONING.md) - Semantic versioning rules and release process
- [Project Requirements](PRD.md) - Project vision and development guidelines

## License

MIT License - see [LICENSE](LICENSE) file for details.
