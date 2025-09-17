# PR Review Workflow with ReviewTask

## Initial Setup
```bash
# Fetch reviews for your PR
reviewtask fetch review <PR_NUMBER>

# Generate tasks from reviews
reviewtask

# Check overall status
reviewtask status
```

## Working on Tasks
```bash
# Show next task to work on
reviewtask show

# Start working on a specific task
reviewtask update <TASK_ID> doing

# After completing the implementation
reviewtask update <TASK_ID> done

# If blocked on a task
reviewtask update <TASK_ID> pending
```

## Progress Monitoring
```bash
# View all tasks with progress bars
reviewtask status

# View wide format with more details
reviewtask status -w

# Show specific task details
reviewtask show <TASK_ID>
```

## Handling Updated Reviews
```bash
# When reviewers add new comments
reviewtask fetch review <PR_NUMBER>
reviewtask

# The tool automatically:
# - Preserves your work progress
# - Adds only new tasks
# - Maintains task status history
```

## Configuration
```bash
# View current configuration
reviewtask config show

# Set AI provider (cursor/claude/auto)
reviewtask config set ai_provider cursor

# Enable verbose mode for debugging
reviewtask config set verbose_mode true
```

## Tips
- Task IDs can be partial (first few characters of UUID)
- Use tab completion for command and task ID suggestions
- Run commands from repository root for best results
