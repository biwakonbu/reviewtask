# ReviewTask Quick Reference

## Essential Commands
| Command | Description |
|---------|-------------|
| reviewtask | Generate tasks from PR reviews |
| reviewtask status | Show all tasks with progress |
| reviewtask show | Display next task to work on |
| reviewtask update <id> doing | Start working on task |
| reviewtask update <id> done | Mark task as completed |

## Task Status Values
- **todo**: Not started yet
- **doing**: Currently in progress
- **done**: Completed
- **pending**: Blocked or waiting

## Keyboard Shortcuts (when configured)
- Cmd+Shift+T: Show current task
- Cmd+Shift+S: Show status
- Cmd+Shift+U: Update task status

## Environment Variables
- REVIEWTASK_AI_PROVIDER: Set AI provider (cursor/claude/auto)
- SKIP_CURSOR_AUTH_CHECK: Skip authentication check (true/false)
- REVIEWTASK_DEBUG: Enable debug output (true/false)
