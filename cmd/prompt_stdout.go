package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var promptStdoutCmd = &cobra.Command{
	Use:   "stdout <target>",
	Short: "Output AI provider prompts to stdout for redirection or piping",
	Long: `Output AI provider prompts to standard output instead of writing to files.
This allows you to redirect the output to any file or pipe it to other tools.

Available targets:
  pr-review    Output PR review workflow prompt

Examples:
  reviewtask prompt stdout pr-review                    # Display prompt on stdout
  reviewtask prompt stdout pr-review > my-workflow.md   # Save to custom file
  reviewtask prompt stdout pr-review | pbcopy           # Copy to clipboard (macOS)
  reviewtask prompt stdout pr-review | xclip            # Copy to clipboard (Linux)`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptStdout,
}

func init() {
	promptCmd.AddCommand(promptStdoutCmd)
}

func runPromptStdout(cmd *cobra.Command, args []string) error {
	target := args[0]

	switch target {
	case "pr-review":
		return outputPRReviewPromptToStdout(cmd)
	default:
		return fmt.Errorf("unknown target: %s\n\nAvailable targets:\n  pr-review    Output PR review workflow prompt", target)
	}
}

func outputPRReviewPromptToStdout(cmd *cobra.Command) error {
	// Get the same prompt template used in claude.go
	prompt := getPRReviewPromptTemplate()
	// Use cmd.OutOrStdout() to allow tests to capture output
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	return nil
}

// loadTemplateFromPaths attempts to load a template from the given file paths.
// Returns the content if successful, or an empty string if all paths fail.
func loadTemplateFromPaths(paths []string) string {
	for _, path := range paths {
		if content, err := os.ReadFile(path); err == nil {
			return string(content)
		}
	}
	return ""
}

// getPRReviewPromptTemplate returns the PR review workflow prompt template
// This is shared between claude.go and prompt_stdout.go to maintain consistency
func getIssueToPRTemplate() string {
	// For tests and when running from different directories,
	// try multiple possible locations for the template
	possiblePaths := []string{
		".claude/commands/issue-to-pr.md",
		"../.claude/commands/issue-to-pr.md",
		"../../.claude/commands/issue-to-pr.md",
	}

	if content := loadTemplateFromPaths(possiblePaths); content != "" {
		return content
	}

	// Return a minimal working template if file not found
	return `# Issue to PR Workflow

## Core Principle: Complete Automation of Issues-Driven Development

**Create systematic PRs and implementation from specified GitHub Issues numbers.**

## Development Workflow

1. **Branch Preparation**: Create feature branch from issue
2. **Draft PR Creation**: Create draft PR linked to issue
3. **Implementation**: Implement features step by step
4. **Testing**: Create comprehensive tests
5. **Completion**: Convert draft to open PR

## Usage

Run with issue number: /issue-to-pr 42

This workflow automates the complete development process from GitHub Issues.`
}

func getLabelIssuesTemplate() string {
	// For tests and when running from different directories,
	// try multiple possible locations for the template
	possiblePaths := []string{
		".claude/commands/label-issues.md",
		"../.claude/commands/label-issues.md",
		"../../.claude/commands/label-issues.md",
	}

	if content := loadTemplateFromPaths(possiblePaths); content != "" {
		return content
	}

	// Return a minimal working template if file not found
	return `# Label Issues

## Basic Principle: Automatic Label Setting Based on Release Impact

**Automatically set appropriate release labels (release:major/minor/patch) for GitHub issues.**

## Label Setting Flow

1. **Fetch GitHub Issues**: Get open issues without release labels
2. **Analyze Content**: Determine if it's a development-related issue
3. **Assign Labels**: Apply appropriate release label based on impact

## Release Label Criteria

**release:major**: Breaking changes, CLI structure changes
**release:minor**: New features, enhancements
**release:patch**: Bug fixes, documentation updates

## Usage

Analyze and label all open issues: /label-issues

This workflow helps maintain consistent semantic versioning.`
}

func getPRReviewPromptTemplate() string {
	// Try to read from external file first (preferred method for keeping in sync)
	possiblePaths := []string{
		".cursor/commands/pr-review/review-task-workflow.md",
		".claude/commands/pr-review/review-task-workflow.md",
		"../.cursor/commands/pr-review/review-task-workflow.md",
		"../.claude/commands/pr-review/review-task-workflow.md",
	}

	if content := loadTemplateFromPaths(possiblePaths); content != "" {
		return content
	}

	// Fallback to embedded template if external files not found
	// IMPORTANT: This template must be kept in sync with:
	// - .claude/commands/pr-review/review-task-workflow.md
	// - .cursor/commands/pr-review/review-task-workflow.md
	// Any updates to the workflow prompt should be reflected in all three locations.

	// Use § as placeholder for backticks to enable true heredoc format
	template := `---
name: review-task-workflow
description: Execute PR review tasks systematically using reviewtask
---

You are tasked with executing PR review tasks systematically using the reviewtask tool.

## Available Commands:

The reviewtask tool provides the following commands for managing PR review tasks:

### Core Workflow Commands:

- **§reviewtask [PR_NUMBER]§** - Fetch reviews and analyze with AI (integrated workflow)
- **§reviewtask status§** - Check overall task status and get summary
- **§reviewtask show§** - Get next recommended task based on priority
- **§reviewtask show <task-id>§** - Show detailed information for a specific task
- **§reviewtask update <task-id> <status>§** - Update task status
  - Status options: §todo§, §doing§, §done§, §pending§, §cancel§

### Task Lifecycle Management Commands:

- **§reviewtask done <task-id>§** - Complete task with full automation (verification, commit, resolve, next task)
- **§reviewtask done <task-id> --skip-verification§** - Skip verification phase
- **§reviewtask done <task-id> --skip-commit§** - Skip automatic commit
- **§reviewtask done <task-id> --skip-resolve§** - Skip thread resolution
- **§reviewtask done <task-id> --skip-suggestion§** - Skip next task suggestion
- **§reviewtask start <task-id>§** - Start working on a task
- **§reviewtask hold <task-id>§** - Put task on hold
- **§reviewtask cancel <task-id> --reason "..."§** - Cancel a task and post reason to GitHub review thread
- **§reviewtask cancel --all-pending --reason "..."§** - Cancel all pending tasks with same reason
- **§reviewtask verify <task-id>§** - Run verification checks before task completion

### Thread Management Commands:

- **§reviewtask resolve <task-id>§** - Manually resolve GitHub review thread for a completed task
- **§reviewtask resolve --all§** - Resolve threads for all done tasks
- **§reviewtask resolve --all --force§** - Force resolve all tasks regardless of status

### Statistics and Configuration:

- **§reviewtask stats§** - Show task statistics by comment for current branch
- **§reviewtask stats --all§** - Show statistics for all PRs
- **§reviewtask stats --pr 123§** - Show statistics for specific PR
- **§reviewtask config show§** - Display current verification configuration
- **§reviewtask config set-verifier <task-type> <command>§** - Configure custom verification commands

## Task Priority System:

Tasks are automatically assigned priority levels that determine processing order:
- **§critical§** - Security issues, critical bugs, breaking changes
- **§high§** - Important functionality issues, major improvements
- **§medium§** - Moderate improvements, refactoring suggestions
- **§low§** - Minor improvements, style suggestions

## Initial Setup (Execute ALWAYS at Start):

**CRITICAL: Always fetch latest reviews first to detect new comments and ensure local data is up-to-date.**

Run the integrated workflow to sync with GitHub:

- §reviewtask§ - Fetches PR reviews from GitHub and analyzes them with AI to generate actionable tasks
- §reviewtask [PR_NUMBER]§ - Same workflow for a specific PR number

**This step is MANDATORY even if you think you have the latest data.** New review comments may have been added since your last check, and the tool will automatically:
- Detect new unresolved comment threads
- Generate new tasks from new review feedback
- Update task counts and progress tracking
- Identify any discrepancies between local and remote state

After fetching the latest reviews, follow this exact workflow:

## Workflow Steps:

1. **Check Status**: Use §reviewtask status§ to check current task status and identify any tasks in progress
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Check if new comments exist by running §reviewtask§ again
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
      - If needed: Use §reviewtask show§ to get the next recommended task, then run §reviewtask update <task-id> doing§
      - If duplicate/unnecessary: Use §reviewtask cancel <task-id> --reason "explanation"§ to cancel and notify reviewers
      - If uncertain: Update to §pending§ with §reviewtask update <task-id> pending§

   d) **For pending-only scenario**:
      - List all pending tasks and their reasons for being blocked
      - For each pending task, decide:
        - §doing§: If you can now resolve the blocking issue
        - §todo§: If the task should be attempted again
        - §cancel§: If the task is no longer relevant or cannot be completed
      - Update task status: §reviewtask update <task-id> <new-status>§

3. **Verify Task Start**: Confirm the status change was successful before proceeding

4. **Execute Task**: Implement the required changes in the current branch based on the task description and original review comment

5. **Complete Task**: When implementation is finished, use the done command for full automation:
   - **Recommended (Full Automation)**: §reviewtask done <task-id>§
     - Automatically runs verification checks
     - Creates structured commit with task details
     - Resolves GitHub review thread (if configured)
     - Suggests next task to work on

   - **Skip Options** (when needed):
     - §reviewtask done <task-id> --skip-verification§ - Skip verification checks
     - §reviewtask done <task-id> --skip-commit§ - Skip automatic commit
     - §reviewtask done <task-id> --skip-resolve§ - Skip thread resolution
     - §reviewtask done <task-id> --skip-suggestion§ - Skip next task suggestion

   - **Alternative Commands**:
     - §reviewtask verify <task-id>§ - Run verification checks only
     - §reviewtask update <task-id> done§ - Direct status update (no automation)

   **Note**: The §done§ command automatically creates commits with proper formatting when auto-commit is enabled.

6. **Review Automation Results**: After running §reviewtask done§:
   - Check verification results (if verification enabled)
   - Review the generated commit (if auto-commit enabled)
   - Verify thread resolution status (if auto-resolve enabled)
   - Note the suggested next task (if suggestion enabled)

7. **Continue Workflow**: After committing:
   - Check status again with §reviewtask status§
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Handle pending tasks as described in Step 2d
   - **If todo or doing tasks remain**: Repeat this entire workflow from step 1

## Task Classification Guidelines:

### When to CANCEL Tasks:

Use §reviewtask cancel <task-id> --reason "explanation"§ for tasks that are:
- **Clear duplicates**: Already implemented features or duplicate review comments
- **Already completed**: Changes finished in previous commits (reference commit hash in reason)
- **Obsolete suggestions**: No longer applicable to current code
- **Conflicting changes**: Would introduce conflicts with existing implementations

**Important**: Always provide a clear cancellation reason to help reviewers understand why feedback wasn't addressed. The reason will be posted as a comment on the GitHub review thread.

**Examples**:
§§§bash
reviewtask cancel task-abc123 --reason "This was already addressed in commit 1a2b3c4"
reviewtask cancel task-xyz789 --reason "Duplicate of task-def456 which is already completed"
reviewtask cancel --all-pending --reason "Deferring all remaining tasks to follow-up PR #125"
§§§

### When to PENDING Tasks:

Use §reviewtask update <task-id> pending§ for tasks that are:
- **Ambiguous requirements**: Need clarification from reviewer
- **Low-priority improvements**: Could be done later
- **External dependencies**: Dependent on external decisions or unclear specifications
- **Non-critical enhancements**: Could be implemented but aren't critical now

### When to Process as TODO:

Keep tasks as §todo§ when they are:
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
- **Build Verification**: Runs build/compile checks (default: §go build ./...§)
- **Test Execution**: Runs test suites (default: §go test ./...§)
- **Code Quality**: Lint and format checks (default: §golangci-lint run§, §gofmt -l .§)
- **Custom Verification**: Project-specific commands based on task type

**Task Type Detection:**
Tasks are automatically categorized for custom verification:
- §test-task§: Tasks containing "test" or "testing"
- §build-task§: Tasks containing "build" or "compile"
- §style-task§: Tasks containing "lint" or "format"
- §bug-fix§: Tasks containing "bug" or "fix"
- §feature-task§: Tasks containing "feature" or "implement"
- §general-task§: All other tasks

**Configuration:**
- §reviewtask config show§ - View current verification settings
- §reviewtask config set-verifier <task-type> <command>§ - Set custom verification commands
- Done workflow settings stored in §.pr-review/config.json§

**Done Workflow Configuration Example:**
§§§json
{
  "done_workflow": {
    "enable_auto_resolve": "complete",
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
§§§

## Current Tool Features:

This workflow leverages the full capabilities of the current reviewtask implementation:
- **Multi-source Authentication**: Supports GitHub CLI, environment variables, and configuration files
- **Task Management**: Complete lifecycle management with status tracking and validation
- **Done Command Automation**: Full 5-phase automation for task completion
- **Task Cancellation**: Cancel tasks with GitHub comment notification to reviewers
- **Thread Resolution**: Automatic and manual resolution of review threads
- **Task Completion Verification**: Automated verification checks before task completion
- **Auto-commit**: Structured commit creation with task details and references
- **AI-Enhanced Analysis**: Intelligent task generation and classification with batch processing
- **Progress Tracking**: Comprehensive status reporting and workflow optimization
- **Statistics**: Per-comment task breakdown and progress analysis
- **Next Task Recommendation**: Priority-based suggestion of next task to work on

## Important Notes:

### Workflow Best Practices:
- Work only in the current branch
- Always verify status changes before proceeding
- Include proper commit message format with task details and comment references
- Continue until all tasks are completed or no more actionable tasks remain
- The initial review fetch/analyze is executed only once per command invocation

### Task Priority and Processing:
- **Task Priority**: Always work on §doing§ tasks first, then §todo§ tasks (by priority level), then handle §pending§ tasks
- **Priority-Based Processing**: Within todo tasks, process critical → high → medium → low priority items
- **Automatic Task Generation**: The tool intelligently creates tasks from review feedback with appropriate priorities

### Task Cancellation:
- **Use cancel command**: Use §reviewtask cancel <task-id> --reason "..."§ instead of §reviewtask update <task-id> cancel§
- **Provide clear reasons**: Cancellation reasons are posted to GitHub to notify reviewers why feedback wasn't addressed
- **Batch cancellation**: Use §--all-pending§ flag to cancel multiple tasks with the same reason
- **Error handling**: Cancel command returns non-zero exit code on failure (safe for CI/CD scripts)

### Task Completion:
- **Recommended approach**: Use §reviewtask done <task-id>§ for full automation
- **Automation Features**: The done command provides 5-phase automation:
  1. **Verification**: Runs configured verification checks
  2. **Status Update**: Marks task as done
  3. **Auto-commit**: Creates structured commit with task details
  4. **Thread Resolution**: Resolves GitHub review thread
  5. **Next Task**: Suggests next task to work on
- **Skip Options**: Use §--skip-verification§, §--skip-commit§, §--skip-resolve§, §--skip-suggestion§ as needed
- **Configuration**: Enable/disable features in §.pr-review/config.json§ under §done_workflow§ section

### Thread Management:
- **Manual resolution**: Use §reviewtask resolve <task-id>§ when auto-resolve is disabled
- **Batch resolution**: Use §reviewtask resolve --all§ to resolve all done tasks at once
- **Force resolution**: Use §--force§ flag to resolve threads regardless of task status

### Task Management Tips:
- **Efficient classification**: Use §cancel§ command for duplicates/unnecessary tasks to notify reviewers
- **Pending tasks**: Must be resolved by changing status to §doing§, §todo§, or using §cancel§ command
- **Statistics**: Use §reviewtask stats§ to track progress and identify bottlenecks

## Example Tool Output:

Here are examples of actual reviewtask command outputs to demonstrate expected behavior:

**§reviewtask status§ output example:**
§§§text
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
§§§

**§reviewtask show§ output example:**
§§§text
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
§§§

**§reviewtask verify§ output example:**
§§§text
Running verification checks for task 'task-001'...

BUILD: verification passed (0.45s)
TEST: verification passed (2.3s)

All verification checks passed for task 'task-001'
You can now safely complete this task with: reviewtask done task-001
§§§

**§reviewtask done§ output example:**
§§§text
🔍 Phase 1/5: Verification
  ✓ Running verification checks...
  ✓ All checks passed

📝 Phase 2/5: Status Update
  ✓ Task 'task-001' marked as done

💾 Phase 3/5: Auto-commit
  ✓ Created commit: fix: Add input validation for user data (abc1234)

🔗 Phase 4/5: Thread Resolution
  ✓ Resolved review thread (Comment ID: r123456789)

💡 Phase 5/5: Next Task Suggestion
  ✓ Next recommended task: task-002 (critical priority)

✅ Task completed successfully with full automation
   All 5 phases completed

📊 Progress Update:
   Completed: 6/8 tasks (75%)
   Remaining: 2 tasks (1 critical, 1 high priority)

Next: reviewtask done task-002
§§§

**§reviewtask cancel§ output example:**
§§§text
✓ Cancelled task 'task-duplicate-123' and posted reason to PR #42
§§§

**§reviewtask cancel --all-pending§ output example:**
§§§text
Found 3 pending task(s) to cancel

✓ Successfully cancelled 3 task(s)
§§§

**§reviewtask resolve§ output example:**
§§§text
✓ Resolved review thread for task 'task-001' (Comment ID: r123456789)
§§§

**§reviewtask stats§ output example:**
§§§text
Task Statistics for PR #42 (feature/new-feature)

Overall Progress: 75% (6/8 tasks completed)

By Comment:
┌─────────────┬───────┬───────┬──────┬─────────┬────────┐
│ Comment ID  │ Total │ Done  │ Todo │ Pending │ Cancel │
├─────────────┼───────┼───────┼──────┼─────────┼────────┤
│ r123456789  │   3   │   3   │  0   │    0    │   0    │
│ r987654321  │   5   │   3   │  1   │    0    │   1    │
└─────────────┴───────┴───────┴──────┴─────────┴────────┘
§§§

Execute this workflow now.
`
	// Replace § placeholders with backticks
	return strings.ReplaceAll(template, "§", "`")
}
