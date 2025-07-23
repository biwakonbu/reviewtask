package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var claudeCmd = &cobra.Command{
	Use:   "claude [TARGET]",
	Short: "Output command templates for Claude Code to .claude/commands directory",
	Long: `Output command templates for Claude Code to .claude/commands directory for better organization and discoverability.

Available targets:
  pr-review    Output PR review workflow command template to .claude/commands/pr-review/

Examples:
  reviewtask claude pr-review    # Output review-task-workflow command template for Claude Code

DEPRECATION WARNING:
  This command is deprecated. Please use 'reviewtask prompt claude' instead.
  The 'reviewtask claude' command will be removed in a future major version.
  
  Migration examples:
    Old: reviewtask claude pr-review
    New: reviewtask prompt claude pr-review`,
	Args:       cobra.ExactArgs(1),
	RunE:       runClaude,
	Deprecated: "use 'reviewtask prompt claude' instead. This command will be removed in a future major version.",
}

func init() {
	// Command registration moved to root.go
}

func runClaude(cmd *cobra.Command, args []string) error {
	// Show deprecation warning
	fmt.Fprintf(os.Stderr, "⚠️  DEPRECATION WARNING: The 'reviewtask claude' command is deprecated.\n")
	fmt.Fprintf(os.Stderr, "⚠️  Please use 'reviewtask prompt claude %s' instead.\n", args[0])
	fmt.Fprintf(os.Stderr, "⚠️  This command will be removed in a future major version.\n\n")

	target := args[0]

	switch target {
	case "pr-review":
		return outputClaudePRReviewCommands()
	default:
		return fmt.Errorf("unknown target: %s\n\nAvailable targets:\n  pr-review    Output PR review workflow command template", target)
	}
}

func outputClaudePRReviewCommands() error {
	// Create the output directory
	outputDir := ".claude/commands/pr-review"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Define the review-task-workflow command template
	workflowTemplate := `---
name: review-task-workflow
description: Execute PR review tasks systematically using reviewtask
---

You are tasked with executing PR review tasks systematically using the reviewtask tool. 

## Initial Setup (Execute Once Per Command Invocation):

**Fetch Latest Reviews**: Run ` + "`reviewtask`" + ` without arguments to fetch the latest PR reviews and generate/update tasks. This ensures you're working with the most current review feedback and tasks.

After completing the initial setup, follow this exact workflow:

## Workflow Steps:

1. **Check Status**: Use ` + "`reviewtask status`" + ` to check current task status and identify any tasks in progress
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Review each pending task and decide action (see Step 2b)
   - **Continue only if todo, doing, or pending tasks exist**

2. **Identify Task**: 
   a) **Priority order**: Always work on tasks in this order:
      - **doing** tasks first (resume interrupted work)
      - **todo** tasks next (new work)
      - **pending** tasks last (blocked work requiring decision)
   
   b) **For doing tasks**: Continue with the task already in progress
   
   c) **For todo tasks**: Use ` + "`reviewtask show`" + ` to get the next recommended task, then run ` + "`reviewtask update <task-id> doing`" + `
   
   d) **For pending-only scenario**: 
      - List all pending tasks and their reasons for being blocked
      - For each pending task, decide:
        - ` + "`doing`" + `: If you can now resolve the blocking issue
        - ` + "`todo`" + `: If the task should be attempted again
        - ` + "`cancel`" + `: If the task is no longer relevant or cannot be completed
      - Update task status: ` + "`reviewtask update <task-id> <new-status>`" + `

3. **Verify Task Start**: Confirm the status change was successful before proceeding

4. **Execute Task**: Implement the required changes in the current branch based on the task description and original review comment

5. **Complete Task**: When implementation is finished:
   - Mark task as completed: ` + "`reviewtask update <task-id> done`" + `
   - Commit changes using this message template (adjust language based on ` + "`user_language`" + ` setting in ` + "`.pr-review/config.json`" + `):
     ` + "```" + `
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
     ` + "```" + `

6. **Continue Workflow**: After committing:
   - Check status again with ` + "`reviewtask status`" + `
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Handle pending tasks as described in Step 2d
   - **If todo or doing tasks remain**: Repeat this entire workflow from step 1

## Important Notes:
- Work only in the current branch
- Always verify status changes before proceeding
- Include proper commit message format with task details and comment references
- **Task Priority**: Always work on ` + "`doing`" + ` tasks first, then ` + "`todo`" + ` tasks, then handle ` + "`pending`" + ` tasks
- **Pending Task Management**: Pending tasks must be resolved by changing status to ` + "`doing`" + `, ` + "`todo`" + `, or ` + "`cancel`" + `
- Continue until all tasks are completed or no more actionable tasks remain
- The initial review fetch is executed only once per command invocation, not during the iterative workflow steps

Execute this workflow now.
`

	// Write the workflow template
	workflowPath := filepath.Join(outputDir, "review-task-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write workflow template: %w", err)
	}

	fmt.Printf("✓ Created Claude Code command template at %s\n", workflowPath)
	fmt.Println()
	fmt.Println("Claude Code commands have been organized in .claude/commands/pr-review/")
	fmt.Println("You can now use the /review-task-workflow command in Claude Code.")
	fmt.Println()
	fmt.Println("Future expansion possibilities:")
	fmt.Println("  reviewtask claude pr-review    # Current functionality")
	fmt.Println("  reviewtask vscode pr-review    # Future: VSCode extensions")
	fmt.Println("  reviewtask cursor pr-review    # Future: Cursor rules")

	return nil
}
