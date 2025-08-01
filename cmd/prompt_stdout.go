package cmd

import (
	"fmt"
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

// getPRReviewPromptTemplate returns the PR review workflow prompt template
// This is shared between claude.go and prompt_stdout.go to maintain consistency
func getPRReviewPromptTemplate() string {
	// Use § as placeholder for backticks to enable true heredoc format
	template := `---
name: review-task-workflow
description: Execute PR review tasks systematically using reviewtask
---

You are tasked with executing PR review tasks systematically using the reviewtask tool. 

## Available Commands:

The reviewtask tool provides the following commands for managing PR review tasks:

- **§reviewtask§** - Fetch latest PR reviews and generate/update tasks
- **§reviewtask status§** - Check overall task status and get summary
- **§reviewtask show§** - Get next recommended task based on priority
- **§reviewtask show <task-id>§** - Show detailed information for a specific task
- **§reviewtask update <task-id> <status>§** - Update task status
  - Status options: §todo§, §doing§, §done§, §pending§, §cancel§

### Task Completion Verification Commands:

- **§reviewtask verify <task-id>§** - Run verification checks before task completion
- **§reviewtask complete <task-id>§** - Complete task with automatic verification
- **§reviewtask complete <task-id> --skip-verification§** - Complete task without verification
- **§reviewtask config set-verifier <task-type> <command>§** - Configure custom verification commands
- **§reviewtask config show§** - Display current verification configuration

## Task Priority System:

Tasks are automatically assigned priority levels that determine processing order:
- **§critical§** - Security issues, critical bugs, breaking changes
- **§high§** - Important functionality issues, major improvements
- **§medium§** - Moderate improvements, refactoring suggestions
- **§low§** - Minor improvements, style suggestions

## Initial Setup (Execute Once Per Command Invocation):

**Fetch Latest Reviews**: Run §reviewtask§ without arguments to fetch the latest PR reviews and generate/update tasks. This ensures you're working with the most current review feedback and tasks.

After completing the initial setup, follow this exact workflow:

## Workflow Steps:

1. **Check Status**: Use §reviewtask status§ to check current task status and identify any tasks in progress
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
      - If needed: Use §reviewtask show§ to get the next recommended task, then run §reviewtask update <task-id> doing§
      - If duplicate/unnecessary: Update to §cancel§
      - If uncertain: Update to §pending§
   
   d) **For pending-only scenario**: 
      - List all pending tasks and their reasons for being blocked
      - For each pending task, decide:
        - §doing§: If you can now resolve the blocking issue
        - §todo§: If the task should be attempted again
        - §cancel§: If the task is no longer relevant or cannot be completed
      - Update task status: §reviewtask update <task-id> <new-status>§

3. **Verify Task Start**: Confirm the status change was successful before proceeding

4. **Execute Task**: Implement the required changes in the current branch based on the task description and original review comment

5. **Verify and Complete Task**: When implementation is finished:
   a) **Verify Implementation**: Run verification checks to ensure quality:
      - §reviewtask verify <task-id>§ - Check if implementation meets verification requirements
      - If verification fails: Review and fix issues, then retry verification
      - If verification passes: Continue to completion
   
   b) **Complete Task**: Choose completion method:
      - **Recommended**: §reviewtask complete <task-id>§ - Complete with automatic verification
      - **Alternative**: §reviewtask complete <task-id> --skip-verification§ - Skip verification if needed
      - **Manual**: §reviewtask update <task-id> done§ - Direct status update (no verification)
   - Commit changes using this message template (adjust language based on §user_language§ setting in §.pr-review/config.json§):
     §§§
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
     §§§

6. **Commit Changes**: After successful task completion, commit with proper message format

7. **Continue Workflow**: After committing:
   - Check status again with §reviewtask status§
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Handle pending tasks as described in Step 2d
   - **If todo or doing tasks remain**: Repeat this entire workflow from step 1

## Task Classification Guidelines:

**CANCEL** tasks that are:
- Clear duplicates of already implemented features
- References to changes already completed in previous commits
- Obsolete suggestions that no longer apply to current code
- Tasks that would introduce conflicts with existing implementations

**PENDING** tasks that are:
- Ambiguous requirements that need clarification
- Low-priority improvements that could be done later
- Tasks dependent on external decisions or unclear specifications
- Enhancements that could be implemented but aren't critical

**TODO** tasks that are:
- New functional requirements not yet implemented
- Critical bug fixes or security issues
- Clear improvement suggestions for current code
- Tasks with specific actionable requirements

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
- Verification settings stored in §.pr-review/config.json§

## Current Tool Features:

This workflow leverages the full capabilities of the current reviewtask implementation:
- **Multi-source Authentication**: Supports GitHub CLI, environment variables, and configuration files
- **Task Management**: Complete lifecycle management with status tracking and validation
- **Task Completion Verification**: Automated verification checks before task completion
- **AI-Enhanced Analysis**: Intelligent task generation and classification
- **Progress Tracking**: Comprehensive status reporting and workflow optimization

## Important Notes:
- Work only in the current branch
- Always verify status changes before proceeding
- Include proper commit message format with task details and comment references
- **Task Priority**: Always work on §doing§ tasks first, then §todo§ tasks (by priority level), then handle §pending§ tasks
- **Priority-Based Processing**: Within todo tasks, process critical → high → medium → low priority items
- **Task Completion Verification**: Use §reviewtask complete <task-id>§ for verified completion (recommended)
- **Verification Failure Handling**: If verification fails, fix issues and retry before completing
- **Verification Configuration**: Custom verification commands can be set per task type for project-specific requirements
- **Pending Task Management**: Pending tasks must be resolved by changing status to §doing§, §todo§, or §cancel§
- **Efficient Task Management**: Classify duplicate/unnecessary tasks as §cancel§ and uncertain tasks as §pending§ to focus on actionable work
- **Automatic Task Generation**: The tool intelligently creates tasks from review feedback with appropriate priorities
- Continue until all tasks are completed or no more actionable tasks remain
- The initial review fetch is executed only once per command invocation, not during the iterative workflow steps

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
You can now safely complete this task with: reviewtask complete task-001
§§§

**§reviewtask complete§ output example:**
§§§text
Running verification checks for task 'task-001'...
Task 'task-001' completed successfully!
All verification checks passed
§§§

Execute this workflow now.
`
	// Replace § placeholders with backticks
	return strings.ReplaceAll(template, "§", "`")
}
