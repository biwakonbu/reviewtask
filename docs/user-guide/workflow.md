# Workflow Guide

Learn how to integrate reviewtask into your daily development workflow for maximum productivity and systematic PR review management.

## Core Workflow Philosophy

reviewtask is designed around three core principles:

1. **Zero Feedback Loss**: Every actionable review comment becomes a tracked task
2. **State Preservation**: Your work progress is never lost during tool operations
3. **AI-Assisted, Human-Controlled**: AI provides intelligent analysis while you maintain full control

## Daily Development Routine

### Morning Startup

Start your day by understanding what needs attention:

```bash
reviewtask show           # What should I work on today?
reviewtask status         # Overall progress across all PRs
```

**What you'll see:**
- Current or next recommended task with full context
- Progress overview across all your active PRs
- Priority distribution and completion status

### During Implementation

As you work on tasks, maintain visibility and track progress:

```bash
# Get full context for a specific task
reviewtask show <task-id>

# Start working on the task (v3.0.0)
reviewtask start <task-id>

# Complete the work...

# Mark task as completed with full automation (v3.0.0 - RECOMMENDED)
reviewtask done <task-id>
# What happens automatically:
# 1. Verification (build/test/lint checks)
# 2. Status update to "done"
# 3. Auto-commit with structured message
# 4. Review thread resolution (when all comment tasks complete)
# 5. Next task suggestion

```

**Skip Options** - Skip individual automation phases when needed:

Skip verification (build, test, lint checks):
```bash
reviewtask done <task-id> --skip-verification
```
Task will be marked as done without running verification checks.

Skip automatic commit:
```bash
reviewtask done <task-id> --skip-commit
```
Task will be marked as done without creating a git commit.

Skip thread resolution:
```bash
reviewtask done <task-id> --skip-resolve
```
Task will be marked as done without resolving the GitHub review thread.

Skip next task suggestion:
```bash
reviewtask done <task-id> --skip-suggestion
```
Task will be marked as done without suggesting the next task.

**Done Command Automation:**

The `reviewtask done` command provides complete workflow automation:

1. **Verification Phase** - Runs build, test, lint, and format checks
   - Configured via `done_workflow.verifiers` in config
   - Auto-detected based on project type (Go, Node.js, Python, Rust)
   - Fails if checks don't pass (prevents incomplete work)

2. **Status Update** - Marks task as "done" in local storage

3. **Auto-Commit** - Creates structured commit with:
   - Task description and context
   - Reference to review comment
   - Co-authored-by tag for attribution

4. **Thread Resolution** - Resolves GitHub review thread based on mode:
   - `immediate`: Resolve right after task completion
   - `when_all_complete`: Resolve only when all comment tasks done (recommended)
   - `disabled`: Manual resolution

5. **Next Task Suggestion** - Shows next recommended task by priority

**Note:** Traditional commands (`reviewtask update <task-id> doing/done`) are still supported, but `start` and `done` provide better workflow integration.

### When Blocked

Handle blocked or low-priority work systematically:

```bash
# Put task on hold (v3.0.0)
reviewtask hold <task-id>

# Find next available task
reviewtask show

# Check overall progress
reviewtask status
```

**Note:** The traditional `reviewtask update <task-id> pending` command is still supported, but the new `hold` command is more intuitive.

### Handling Updated Reviews

When reviewers add new comments or feedback:

```bash
# Re-run analysis to get new feedback
reviewtask

# Tool automatically:
# - Preserves existing task statuses
# - Adds new tasks for new comments
# - Cancels outdated tasks if comments change significantly
```

## PR Review Response Workflow

### The Golden Path

This is the recommended workflow for handling PR reviews:

#### 1. Receive Review Notification
- PR receives reviews with comments and feedback
- Developer needs to address feedback systematically

#### 2. Generate Actionable Tasks
```bash
# Convert all review feedback into tracked tasks
reviewtask
```

#### 3. Review What Needs to be Done
```bash
# See current work or next recommended task
reviewtask show
```

#### 4. Work on Tasks Systematically
```bash
# Start working on a task (v3.0.0)
reviewtask start <task-id>

# Complete implementation (v3.0.0)
reviewtask done <task-id>
```

#### 5. Handle Updated Reviews
```bash
# Re-run when reviewers add new comments
reviewtask
# Tool automatically preserves your work progress
```

## Task Lifecycle Management

### Task Statuses

Understanding and using task statuses effectively:

- **`todo`** - Ready to work on (default for new tasks)
- **`doing`** - Currently in progress (use for active work)
- **`done`** - Completed (work finished)
- **`pending`** - Blocked or low priority (use for blockers or nits)
- **`cancel`** - No longer relevant (auto-assigned when comments change)

### Status Transitions

Typical task status flow:

```
todo → doing → done     (normal completion)
todo → pending          (blocked or low priority)
pending → doing → done  (unblocked and completed)
any → cancel           (comment becomes outdated)
```

### Best Practices

1. **Use `doing` status**: Mark tasks as in progress to track active work
2. **Don't skip statuses**: Move tasks through appropriate states
3. **Mark blockers as `pending`**: Use pending for blocked or low-priority work
4. **Let tool handle `cancel`**: Don't manually cancel tasks; let the tool detect changes

## Team Collaboration Rules

### For PR Authors

**When you receive reviews:**
1. Run `reviewtask` immediately after receiving reviews
2. Update task statuses as you complete work  
3. Never manually edit `.pr-review/` files
4. Use task completion as readiness indicator

**Daily practice:**
```bash
# Morning: Check what needs attention
reviewtask status

# During work: Track progress (v3.0.0)
reviewtask start <task-id>
# ... work ...
reviewtask done <task-id>

# When stuck: Put on hold and find next task (v3.0.0)
reviewtask hold <task-id>
reviewtask show
```

### For Reviewers

**Writing effective reviews:**
1. Write actionable, specific feedback
2. Use clear priority indicators (security issues, nits, etc.)
3. Trust that feedback will be systematically addressed
4. Follow up reviews add incremental tasks automatically

**Review workflow:**
- Initial review creates comprehensive task list
- Follow-up reviews add only new/changed feedback
- Completed tasks remain completed across review iterations

### For Teams

**Integration points:**
1. Integrate task status into standup discussions
2. Use task completion as PR readiness indicator
3. Treat persistent `pending` tasks as team blockers
4. Share configuration patterns across team repositories

## Advanced Workflow Patterns

### Multi-PR Management

Working across multiple PRs simultaneously:

```bash
# Check all PRs
reviewtask status --all

# Work on specific PR (v3.0.0: simplified syntax)
cd /path/to/pr-branch
reviewtask show

# Check specific PR status (v3.0.0: positional argument)
reviewtask status 123
reviewtask status 456
```

### Priority-Based Workflow

Focus on high-impact work first:

```bash
# Review task priorities
reviewtask stats

# Configure priority rules in .pr-review/config.json
# Work on critical/high priority tasks first
# Leave low-priority tasks for later or mark as pending
```

### Batch Processing

Handle multiple tasks efficiently:

```bash
# Get overview of all tasks
reviewtask status

# Process similar tasks together
# Use task descriptions to group related work
# Update multiple task statuses as you complete batches
```

## Debugging and Troubleshooting Workflow

### When Things Go Wrong

**Missing tasks:**
```bash
# Force refresh to reprocess all comments
reviewtask --refresh-cache
```

**Authentication issues:**
```bash
# Check authentication status
reviewtask auth status
reviewtask auth check

# Re-authenticate if needed
reviewtask auth logout
reviewtask auth login
```

**Task generation problems:**
```bash
# Test specific phases independently
reviewtask debug fetch review 123    # Fetch reviews only
reviewtask debug fetch task 123      # Generate tasks only

# Render analysis prompt locally (no AI) for inspection/A/B comparison
reviewtask debug prompt 123 --profile v2
reviewtask debug prompt 123 --profile legacy

# Enable verbose mode in .pr-review/config.json
{
  "ai_settings": {
    "verbose_mode": true
  }
}
```

### Performance Issues

**Large PRs:**
- Tool automatically optimizes for large PRs
- Uses parallel processing and automatic chunking
- Enable verbose mode to see optimization details

**API rate limits:**
```bash
# Check rate limit status
reviewtask auth check

# Use cache to reduce API calls
# Avoid --refresh-cache unless necessary
```

## Integration with Development Tools

### Git Integration

reviewtask works seamlessly with Git workflows:

```bash
# Standard Git workflow
git checkout feature-branch
# ... receive PR reviews ...
reviewtask                    # Generate tasks
reviewtask show              # Work on tasks
# ... make changes ...
git add .
git commit -m "Address PR feedback"
git push
```

### IDE Integration

**Task context in your editor:**
- Use `reviewtask show <task-id>` to get full task context
- Copy task descriptions into commit messages
- Reference task IDs in commit messages for traceability

### CI/CD Integration

**Automated checks:**
```bash
# In CI pipeline, check for pending tasks
reviewtask status --pr $PR_NUMBER
# Fail build if critical tasks remain incomplete
```

## Configuration for Different Workflows

### Individual Developer

```json
{
  "task_settings": {
    "default_status": "todo",
    "low_priority_status": "pending"
  },
  "ai_settings": {
    "verbose_mode": false,
    "validation_enabled": false
  }
}
```

### Team Development

```json
{
  "priority_rules": {
    "critical": "Security, breaking changes, production issues",
    "high": "Performance, functionality, API changes",
    "medium": "Bugs, improvements, error handling",
    "low": "Style, documentation, minor suggestions"
  },
  "task_settings": {
    "auto_prioritize": true,
    "low_priority_patterns": ["nit:", "style:", "suggestion:"]
  }
}
```

### High-Quality Projects

```json
{
  "ai_settings": {
    "validation_enabled": true,
    "quality_threshold": 0.9,
    "deduplication_enabled": true,
    "max_retries": 5
  }
}
```

## Measuring Success

### Key Metrics

Track these indicators of successful reviewtask adoption:

1. **Task Completion Rate**: Percentage of tasks marked as done
2. **Review Cycle Time**: Time from review to PR approval
3. **Feedback Loss**: Number of review comments not addressed
4. **Developer Satisfaction**: Subjective measure of workflow improvement

### Monitoring Commands

```bash
# Overall progress
reviewtask status --all

# Detailed analytics
reviewtask stats --all

# Individual PR analysis
reviewtask stats --pr 123
```

## Getting Help

### Self-Service Debugging

1. **Enable verbose mode**: Add detailed logging to understand tool behavior
2. **Use debug commands**: Test specific functionality independently
3. **Check configuration**: Verify settings match your workflow needs
4. **Review documentation**: Check command reference and troubleshooting guides

### Community Support

1. **GitHub Issues**: Report bugs and request features
2. **Documentation**: Comprehensive guides and examples
3. **Examples**: Real-world configuration patterns and workflows

For detailed troubleshooting steps, see the [Troubleshooting Guide](troubleshooting.md).

For specific command usage, see the [Command Reference](commands.md).
