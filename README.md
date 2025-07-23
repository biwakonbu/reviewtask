# reviewtask - AI-Powered PR Review Management Tool

[![Latest Release](https://img.shields.io/github/v/release/biwakonbu/reviewtask)](https://github.com/biwakonbu/reviewtask/releases/latest)
[![CI](https://github.com/biwakonbu/reviewtask/workflows/CI/badge.svg)](https://github.com/biwakonbu/reviewtask/actions)
[![codecov](https://codecov.io/gh/biwakonbu/reviewtask/branch/main/graph/badge.svg)](https://codecov.io/gh/biwakonbu/reviewtask)
[![Go Report Card](https://goreportcard.com/badge/github.com/biwakonbu/reviewtask)](https://goreportcard.com/report/github.com/biwakonbu/reviewtask)
[![GoDoc](https://godoc.org/github.com/biwakonbu/reviewtask?status.svg)](https://godoc.org/github.com/biwakonbu/reviewtask)

A CLI tool that fetches GitHub Pull Request reviews, analyzes them using AI, and generates actionable tasks for developers to address feedback systematically.

## Features

- **🔍 PR Review Fetching**: Automatically retrieves reviews from GitHub API with nested comment structure
- **🤖 AI Analysis**: Supports multiple AI providers for generating structured, actionable tasks from review content
- **💾 Local Storage**: Stores data in structured JSON format under `.pr-review/` directory
- **📋 Task Management**: Full lifecycle management with status tracking (todo/doing/done/pending/cancel)
- **⚡ Parallel Processing**: Processes multiple comments concurrently for improved performance
- **🔒 Authentication**: Multi-source token detection with interactive setup
- **🎯 Priority-based Analysis**: Customizable priority rules for task generation
- **🔄 Task State Preservation**: Maintains existing task statuses during subsequent runs
- **🆔 UUID-based Task IDs**: Unique task identification to eliminate duplication issues
- **🔌 Extensible AI Provider Support**: Architecture designed for easy integration of multiple AI providers

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
# Initialize the tool in your repository
./reviewtask init
```

This will:
- Create `.pr-review/` directory structure
- Generate default configuration files
- Set up `.gitignore` entries
- Check repository permissions

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
# - Process comments in parallel using configured AI provider
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
| `reviewtask status [options]` | Show task status and statistics |
| `reviewtask show [task-id]` | Show current/next task or specific task details |
| `reviewtask update <id> <status>` | Update task status |
| `reviewtask stats [PR_NUMBER] [options]` | Show detailed task statistics with comment breakdown |
| `reviewtask version [VERSION]` | Show version information or switch to specific version |
| `reviewtask versions` | List available versions from GitHub releases |
| `reviewtask prompt <provider> <target>` | Generate AI provider command templates |
| `reviewtask claude <target>` | (Deprecated) Use `reviewtask prompt claude <target>` |
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
- `reviewtask prompt stdout <target>` - Output prompts to stdout for redirection or piping
- `reviewtask prompt <provider> <target>` - Generate templates for various AI providers (extensible)

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

### Permission Requirements

Required GitHub API permissions:
- `repo` (for private repositories)
- `public_repo` (for public repositories)
- `read:org` (for organization repositories)

Use `reviewtask auth check` for comprehensive permission validation.

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
