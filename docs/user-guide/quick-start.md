# Quick Start Guide

This guide will get you up and running with reviewtask in just a few minutes.

## Prerequisites

- Go 1.19 or later (if building from source)
- GitHub repository with pull requests
- AI provider CLI (Claude Code recommended)

## 1. Installation

Choose your preferred installation method:

=== "One-liner Install (Recommended)"

    **Unix/Linux/macOS:**
    ```bash
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash
    ```

    **Windows (PowerShell):**
    ```powershell
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex
    ```

=== "Manual Download"

    Download the latest release for your platform from the [releases page](https://github.com/biwakonbu/reviewtask/releases/latest).

=== "Go Install"

    ```bash
    go install github.com/biwakonbu/reviewtask@latest
    ```

For detailed installation instructions, see the [Installation Guide](installation.md).

## 2. Verify Installation

```bash
reviewtask version
```

## 3. Repository Initialization

Navigate to your Git repository and initialize reviewtask:

```bash
cd /path/to/your/repository
reviewtask init
```

This creates:
- `.pr-review/` directory structure
- Default configuration files
- Appropriate `.gitignore` entries

## 4. Authentication Setup

Set up GitHub authentication:

```bash
reviewtask auth login
```

This will guide you through:
- GitHub token creation
- Permission verification
- Authentication testing

### Alternative Authentication Methods

reviewtask checks for authentication in this order:

1. `GITHUB_TOKEN` environment variable
2. Local config file (`.pr-review/auth.json`)
3. GitHub CLI (`gh auth token`)

## 5. Analyze Your First PR

Now you're ready to analyze PR reviews:

```bash
# Analyze current branch's PR
reviewtask

# Or analyze a specific PR
reviewtask 123
```

The tool will:
- Fetch PR reviews and comments from GitHub
- Process comments using AI analysis
- Generate actionable tasks with priorities
- Save results to `.pr-review/PR-{number}/`

## 6. Task Management

View and manage your tasks:

```bash
# View all task status
reviewtask status

# Show current/next task details
reviewtask show

# Show specific task details
reviewtask show <task-id>

# Update task status as you work
reviewtask update <task-id> doing
reviewtask update <task-id> done
```

## Task Management Commands (v3.0.0)

reviewtask provides intuitive commands for managing task status:

```bash
# Start working on a task
reviewtask start <task-id>

# Mark a task as completed
reviewtask done <task-id>

# Put a task on hold
reviewtask hold <task-id>

# Traditional update command (still supported)
reviewtask update <task-id> doing
reviewtask update <task-id> done
reviewtask update <task-id> pending
```

## Task Statuses

- `todo` - Ready to work on
- `doing` - Currently in progress (use `reviewtask start`)
- `done` - Completed (use `reviewtask done`)
- `pending` - Blocked or low priority (use `reviewtask hold`)
- `cancel` - No longer relevant

## Next Steps

Now that you have reviewtask set up:

1. **[Configure](configuration.md)** priority rules and AI settings
2. **[Learn the commands](commands.md)** for advanced task management
3. **[Understand the workflow](workflow.md)** for daily development
4. **[Troubleshoot](troubleshooting.md)** any issues you encounter

## Daily Workflow

Here's how reviewtask fits into your daily development routine:

### Morning Startup
```bash
reviewtask show           # What should I work on today?
reviewtask status         # Overall progress across all PRs
```

### During Implementation
```bash
reviewtask show <task-id> # Full context for current task
reviewtask start <task-id> # Mark as in progress
# Work on the task...
reviewtask done <task-id>  # Mark as completed
```

### When Blocked
```bash
reviewtask hold <task-id>  # Put task on hold
reviewtask show            # Find next task to work on
```

### When Reviews Are Updated
```bash
reviewtask                # Re-run to get new feedback
# Tool automatically preserves your work progress
```

## Common Issues

If you encounter issues during setup:

- **Permission errors**: Use `reviewtask auth check` to verify GitHub token permissions
- **Binary not found**: Ensure the installation directory is in your PATH
- **AI provider errors**: Verify Claude Code CLI is installed and accessible

For detailed troubleshooting, see the [Troubleshooting Guide](troubleshooting.md).