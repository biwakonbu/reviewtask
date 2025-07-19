---
name: review-task-workflow
description: Execute PR review tasks systematically using gh-review-task
---

You are tasked with executing PR review tasks systematically using the gh-review-task tool. Follow this exact workflow:

## Workflow Steps:

1. **Check Status**: Use `gh-review-task status` to check current task status and identify any tasks in progress

2. **Identify Task**: 
   - If there's a task with "doing" status, work on that task
   - If no "doing" task exists, use `gh-review-task show` to get the next recommended task
   - Start the task by running `gh-review-task update <task-id> doing`

3. **Verify Task Start**: Confirm the status change was successful before proceeding

4. **Execute Task**: Implement the required changes in the current branch based on the task description and original review comment

5. **Complete Task**: When implementation is finished:
   - Mark task as completed: `gh-review-task update <task-id> done`
   - Commit changes using this message template (adjust language based on `user_language` setting in `.pr-review/config.json`):
     ```
     fix: [Task description summary]
     
     **Feedback:** [Original comment excerpt - what was pointed out]
     **Solution:** [What was implemented to address the feedback]
     **Rationale:** [Why this approach was chosen - security/performance/maintainability/etc.]
     
     Comment ID: [source_comment_id]
     Review Comment: https://github.com/[owner]/[repo]/pull/[pr-number]#discussion_r[comment-id]
     ```

6. **Continue Workflow**: After committing:
   - Check status again with `gh-review-task status`
   - If remaining tasks exist, repeat this entire workflow from step 1

## Important Notes:
- Work only in the current branch
- Always verify status changes before proceeding
- Include proper commit message format with task details and comment references
- Continue until all tasks are completed or no more actionable tasks remain

Execute this workflow now.