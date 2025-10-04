---
name: review-task-workflow
description: Execute PR review tasks systematically using reviewtask
---

You are tasked with executing PR review tasks systematically using the reviewtask tool.

## Available Commands:

The reviewtask tool provides the following commands for managing PR review tasks:

### Core Workflow Commands:

- **`reviewtask fetch [PR_NUMBER]`** - Fetch PR reviews from GitHub and save locally
- **`reviewtask analyze [PR_NUMBER]`** - Analyze saved reviews and generate tasks using AI (batch processing)
- **`reviewtask status`** - Check overall task status and get summary
- **`reviewtask show`** - Get next recommended task based on priority
- **`reviewtask show <task-id>`** - Show detailed information for a specific task
- **`reviewtask update <task-id> <status>`** - Update task status
  - Status options: `todo`, `doing`, `done`, `pending`, `cancel`

### Task Lifecycle Management Commands:

- **`reviewtask cancel <task-id> --reason "..."`** - Cancel a task and post reason to GitHub review thread
- **`reviewtask cancel --all-pending --reason "..."`** - Cancel all pending tasks with same reason
- **`reviewtask verify <task-id>`** - Run verification checks before task completion
- **`reviewtask complete <task-id>`** - Complete task with automatic verification
- **`reviewtask complete <task-id> --skip-verification`** - Complete task without verification

### Thread Management Commands:

- **`reviewtask resolve <task-id>`** - Manually resolve GitHub review thread for a completed task
- **`reviewtask resolve --all`** - Resolve threads for all done tasks
- **`reviewtask resolve --all --force`** - Force resolve all tasks regardless of status

### Statistics and Configuration:

- **`reviewtask stats`** - Show task statistics by comment for current branch
- **`reviewtask stats --all`** - Show statistics for all PRs
- **`reviewtask stats --pr 123`** - Show statistics for specific PR
- **`reviewtask config show`** - Display current verification configuration
- **`reviewtask config set-verifier <task-type> <command>`** - Configure custom verification commands

## Task Priority System:

Tasks are automatically assigned priority levels that determine processing order:
- **`critical`** - Security issues, critical bugs, breaking changes
- **`high`** - Important functionality issues, major improvements
- **`medium`** - Moderate improvements, refactoring suggestions
- **`low`** - Minor improvements, style suggestions

## Initial Setup (Execute Once Per Command Invocation):

**Fetch and Analyze Reviews**: The workflow consists of two steps:

1. **Fetch Reviews**: `reviewtask fetch` - Downloads PR reviews from GitHub and saves them locally
2. **Generate Tasks**: `reviewtask analyze` - Analyzes reviews using AI and generates actionable tasks

You can also use the combined command `reviewtask` (without arguments) which runs both fetch and analyze in sequence. Run these commands to ensure you're working with the most current review feedback and tasks.

After completing the initial setup, follow this exact workflow:

## Workflow Steps:

1. **Check Status**: Use `reviewtask status` to check current task status and identify any tasks in progress
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Review each pending task and decide action (see Step 2d)
   - **Continue only if todo, doing, or pending tasks exist**

2. **Identify Task**:
   a) **Priority order**: Always work on tasks in this order:
      - **doing** tasks first (resume interrupted work)
      - **todo** tasks next (new work, prioritized by: critical → high → medium → low)
      - **pending** tasks last (blocked work requiring decision)

   b) **For doing tasks**: Continue with the task already in progress

   c) **For todo tasks**:
      - First, evaluate if the task is needed using the **Task Classification Guidelines** below
      - If needed: Use `reviewtask show` to get the next recommended task, then run `reviewtask update <task-id> doing`
      - If duplicate/unnecessary: Use `reviewtask cancel <task-id> --reason "explanation"` to cancel and notify reviewers
      - If uncertain: Update to `pending` with `reviewtask update <task-id> pending`

   d) **For pending-only scenario**:
      - List all pending tasks and their reasons for being blocked
      - For each pending task, decide:
        - `doing`: If you can now resolve the blocking issue
        - `todo`: If the task should be attempted again
        - `cancel`: If the task is no longer relevant or cannot be completed
      - Update task status: `reviewtask update <task-id> <new-status>`

3. **Verify Task Start**: Confirm the status change was successful before proceeding

4. **Execute Task**: Implement the required changes in the current branch based on the task description and original review comment

5. **Verify and Complete Task**: When implementation is finished:
   a) **Verify Implementation**: Run verification checks to ensure quality:
      - `reviewtask verify <task-id>` - Check if implementation meets verification requirements
      - If verification fails: Review and fix issues, then retry verification
      - If verification passes: Continue to completion

   b) **Complete Task**: Choose completion method:
      - **Recommended**: `reviewtask complete <task-id>` - Complete with automatic verification
      - **Alternative**: `reviewtask complete <task-id> --skip-verification` - Skip verification if needed
      - **Manual**: `reviewtask update <task-id> done` - Direct status update (no verification)
   - Commit changes using this message template (adjust language based on `user_language` setting in `.pr-review/config.json`):
     ```
     fix: [Clear, concise description of what was fixed or implemented]

     **Feedback:** [Brief summary of the issue identified in the review]
     The original review comment pointed out [specific problem/concern]. This issue
     occurred because [root cause explanation]. The reviewer suggested [any specific
     recommendations if provided].

     **Solution:** [What was implemented to resolve the issue]
     Implemented the following changes to address the feedback:
     - [Specific change 1 with file/location details]
     - [Specific change 2 with file/location details]
     - [Additional changes as needed]

     The implementation approach involved [brief technical explanation of how the
     solution works].

     **Rationale:** [Why this solution approach was chosen]
     This solution was selected because it [primary benefit/advantage]. Additionally,
     it [secondary benefits such as improved security, performance, maintainability,
     code quality, etc.]. This approach ensures [long-term benefits or compliance
     with best practices].

     Comment ID: [source_comment_id]
     Review Comment: https://github.com/[owner]/[repo]/pull/[pr-number]#discussion_r[comment-id]
     ```

6. **Commit Changes**: After successful task completion, commit with proper message format

7. **Continue Workflow**: After committing:
   - Check status again with `reviewtask status`
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Handle pending tasks as described in Step 2d
   - **If todo or doing tasks remain**: Repeat this entire workflow from step 1

## Task Classification Guidelines:

### When to CANCEL Tasks:

Use `reviewtask cancel <task-id> --reason "explanation"` for tasks that are:
- **Clear duplicates**: Already implemented features or duplicate review comments
- **Already completed**: Changes finished in previous commits (reference commit hash in reason)
- **Obsolete suggestions**: No longer applicable to current code
- **Conflicting changes**: Would introduce conflicts with existing implementations

**Important**: Always provide a clear cancellation reason to help reviewers understand why feedback wasn't addressed. The reason will be posted as a comment on the GitHub review thread.

**Examples**:
```bash
reviewtask cancel task-abc123 --reason "This was already addressed in commit 1a2b3c4"
reviewtask cancel task-xyz789 --reason "Duplicate of task-def456 which is already completed"
reviewtask cancel --all-pending --reason "Deferring all remaining tasks to follow-up PR #125"
```

### When to PENDING Tasks:

Use `reviewtask update <task-id> pending` for tasks that are:
- **Ambiguous requirements**: Need clarification from reviewer
- **Low-priority improvements**: Could be done later
- **External dependencies**: Dependent on external decisions or unclear specifications
- **Non-critical enhancements**: Could be implemented but aren't critical now

### When to Process as TODO:

Keep tasks as `todo` when they are:
- **New functional requirements**: Not yet implemented
- **Critical bug fixes**: Security issues or important bugs
- **Clear improvements**: Specific, actionable suggestions for current code
- **Actionable requirements**: Tasks with clear implementation path

## AI Processing and Task Generation:

The reviewtask tool includes intelligent AI processing that:
- **Automatic Task Creation**: Analyzes PR review comments and automatically generates actionable tasks
- **Task Deduplication**: Identifies and removes duplicate or similar tasks to avoid redundant work
- **Priority Assignment**: Automatically assigns priority levels based on comment content and context
- **Task Validation**: Ensures generated tasks are actionable and properly scoped

## Task Completion Verification:

The tool includes verification capabilities to ensure task quality before completion:

**Verification Types:**
- **Build Verification**: Runs build/compile checks (default: `go build ./...`)
- **Test Execution**: Runs test suites (default: `go test ./...`)
- **Code Quality**: Lint and format checks (default: `golangci-lint run`, `gofmt -l .`)
- **Custom Verification**: Project-specific commands based on task type

**Task Type Detection:**
Tasks are automatically categorized for custom verification:
- `test-task`: Tasks containing "test" or "testing"
- `build-task`: Tasks containing "build" or "compile"
- `style-task`: Tasks containing "lint" or "format"
- `bug-fix`: Tasks containing "bug" or "fix"
- `feature-task`: Tasks containing "feature" or "implement"
- `general-task`: All other tasks

**Configuration:**
- `reviewtask config show` - View current verification settings
- `reviewtask config set-verifier <task-type> <command>` - Set custom verification commands
- Verification settings stored in `.pr-review/config.json`

## Current Tool Features:

This workflow leverages the full capabilities of the current reviewtask implementation:
- **Multi-source Authentication**: Supports GitHub CLI, environment variables, and configuration files
- **Task Management**: Complete lifecycle management with status tracking and validation
- **Task Cancellation**: Cancel tasks with GitHub comment notification to reviewers
- **Thread Resolution**: Manually resolve review threads for completed tasks
- **Task Completion Verification**: Automated verification checks before task completion
- **AI-Enhanced Analysis**: Intelligent task generation and classification with batch processing
- **Progress Tracking**: Comprehensive status reporting and workflow optimization
- **Statistics**: Per-comment task breakdown and progress analysis

## Important Notes:

### Workflow Best Practices:
- Work only in the current branch
- Always verify status changes before proceeding
- Include proper commit message format with task details and comment references
- Continue until all tasks are completed or no more actionable tasks remain
- The initial review fetch/analyze is executed only once per command invocation

### Task Priority and Processing:
- **Task Priority**: Always work on `doing` tasks first, then `todo` tasks (by priority level), then handle `pending` tasks
- **Priority-Based Processing**: Within todo tasks, process critical → high → medium → low priority items
- **Automatic Task Generation**: The tool intelligently creates tasks from review feedback with appropriate priorities

### Task Cancellation:
- **Use cancel command**: Use `reviewtask cancel <task-id> --reason "..."` instead of `reviewtask update <task-id> cancel`
- **Provide clear reasons**: Cancellation reasons are posted to GitHub to notify reviewers why feedback wasn't addressed
- **Batch cancellation**: Use `--all-pending` flag to cancel multiple tasks with the same reason
- **Error handling**: Cancel command returns non-zero exit code on failure (safe for CI/CD scripts)

### Task Completion:
- **Recommended approach**: Use `reviewtask complete <task-id>` for verified completion
- **Verification Failure Handling**: If verification fails, fix issues and retry before completing
- **Verification Configuration**: Custom verification commands can be set per task type for project-specific requirements

### Thread Management:
- **Manual resolution**: Use `reviewtask resolve <task-id>` when auto-resolve is disabled
- **Batch resolution**: Use `reviewtask resolve --all` to resolve all done tasks at once
- **Force resolution**: Use `--force` flag to resolve threads regardless of task status

### Task Management Tips:
- **Efficient classification**: Use `cancel` command for duplicates/unnecessary tasks to notify reviewers
- **Pending tasks**: Must be resolved by changing status to `doing`, `todo`, or using `cancel` command
- **Statistics**: Use `reviewtask stats` to track progress and identify bottlenecks

## Example Tool Output:

Here are examples of actual reviewtask command outputs to demonstrate expected behavior:

**`reviewtask status` output example:**
```text
PR Review Tasks Status:
┌────────┬──────────────────────────────────────────┬──────────┬──────────┐
│ Status │ Task Description                         │ Priority │ ID       │
├────────┼──────────────────────────────────────────┼──────────┼──────────┤
│ doing  │ Add input validation for user data       │ critical │ task-001 │
│ todo   │ Refactor error handling logic            │ high     │ task-002 │
│ todo   │ Update documentation for API changes     │ medium   │ task-003 │
│ pending│ Consider alternative data structure      │ low      │ task-004 │
└────────┴──────────────────────────────────────────┴──────────┴──────────┘

Next recommended task: task-001 (doing - critical priority)
```

**`reviewtask show` output example:**
```text
Task ID: task-001
Status: doing
Priority: critical
Implementation: Implemented
Verification: Verified
Description: Add input validation for user data

Original Review Comment:
"The function doesn't validate input parameters, which could lead to security vulnerabilities."

Comment ID: r123456789
Review URL: https://github.com/owner/repo/pull/42#discussion_r123456789

Files to modify:
- src/handlers/user.go
- test/handlers/user_test.go

Last Verification: 2025-08-01 18:19:53
Verification History:
  1. PASSED 2025-08-01 18:19:53 (checks: build, test)
```

**`reviewtask verify` output example:**
```text
Running verification checks for task 'task-001'...

BUILD: verification passed (0.45s)
TEST: verification passed (2.3s)

All verification checks passed for task 'task-001'
You can now safely complete this task with: reviewtask complete task-001
```

**`reviewtask complete` output example:**
```text
Running verification checks for task 'task-001'...
Task 'task-001' completed successfully!
All verification checks passed
```

**`reviewtask cancel` output example:**
```text
✓ Cancelled task 'task-duplicate-123' and posted reason to PR #42
```

**`reviewtask cancel --all-pending` output example:**
```text
Found 3 pending task(s) to cancel

✓ Successfully cancelled 3 task(s)
```

**`reviewtask resolve` output example:**
```text
✓ Resolved review thread for task 'task-001' (Comment ID: r123456789)
```

**`reviewtask stats` output example:**
```text
Task Statistics for PR #42 (feature/new-feature)

Overall Progress: 75% (6/8 tasks completed)

By Comment:
┌─────────────┬───────┬───────┬──────┬─────────┬────────┐
│ Comment ID  │ Total │ Done  │ Todo │ Pending │ Cancel │
├─────────────┼───────┼───────┼──────┼─────────┼────────┤
│ r123456789  │   3   │   3   │  0   │    0    │   0    │
│ r987654321  │   5   │   3   │  1   │    0    │   1    │
└─────────────┴───────┴───────┴──────┴─────────┴────────┘
```

Execute this workflow now.
