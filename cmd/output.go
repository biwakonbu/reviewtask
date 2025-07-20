package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var outputCmd = &cobra.Command{
	Use:   "output [TYPE]",
	Short: "Output command templates to .claude/commands directory",
	Long: `Output command templates to .claude/commands directory for better organization and discoverability.

Available types:
  pr-review    Output PR review workflow command template to .claude/commands/pr-review/

Examples:
  gh-review-task output pr-review    # Output review-task-workflow command template`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOutput,
}

func init() {
	rootCmd.AddCommand(outputCmd)
}

func runOutput(cmd *cobra.Command, args []string) error {
	outputType := "pr-review" // default
	if len(args) > 0 {
		outputType = args[0]
	}

	switch outputType {
	case "pr-review":
		return outputPRReviewCommands()
	default:
		return fmt.Errorf("unknown output type: %s", outputType)
	}
}

func outputPRReviewCommands() error {
	// Create the output directory
	outputDir := ".claude/commands/pr-review"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Define the review-task-workflow command template
	workflowTemplate := `---
name: review-task-workflow
description: Execute PR review tasks systematically using gh-review-task
---

You are tasked with executing PR review tasks systematically using the gh-review-task tool. 

## Initial Setup (Execute Once Per Command Invocation):

**Fetch Latest Reviews**: Run ` + "`gh-review-task`" + ` without arguments to fetch the latest PR reviews and generate/update tasks. This ensures you're working with the most current review feedback and tasks.

After completing the initial setup, follow this exact workflow:

## Workflow Steps:

1. **Check Status**: Use ` + "`gh-review-task status`" + ` to check current task status and identify any tasks in progress
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Review each pending task and decide action (see Step 2b)
   - **Continue only if todo, doing, or pending tasks exist**

2. **Identify Task**: 
   a) **Priority order**: Always work on tasks in this order:
      - **doing** tasks first (resume interrupted work)
      - **todo** tasks next (new work)
      - **pending** tasks last (blocked work requiring decision)
   
   b) **For doing tasks**: Continue with the task already in progress
   
   c) **For todo tasks**: Use ` + "`gh-review-task show`" + ` to get the next recommended task, then run ` + "`gh-review-task update <task-id> doing`" + `
   
   d) **For pending-only scenario**: 
      - List all pending tasks and their reasons for being blocked
      - For each pending task, decide:
        - ` + "`doing`" + `: If you can now resolve the blocking issue
        - ` + "`todo`" + `: If the task should be attempted again
        - ` + "`cancel`" + `: If the task is no longer relevant or cannot be completed
      - Update task status: ` + "`gh-review-task update <task-id> <new-status>`" + `

3. **Verify Task Start**: Confirm the status change was successful before proceeding

4. **Execute Task**: Implement the required changes in the current branch based on the task description and original review comment

5. **Complete Task**: When implementation is finished:
   - Mark task as completed: ` + "`gh-review-task update <task-id> done`" + `
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
   - Check status again with ` + "`gh-review-task status`" + `
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

	fmt.Printf("✓ Created PR review command template at %s\n", workflowPath)
	fmt.Println()
	fmt.Println("PR review commands have been organized in .claude/commands/pr-review/")
	fmt.Println("You can now use the /review-task-workflow command in Claude Code.")

	return nil
}