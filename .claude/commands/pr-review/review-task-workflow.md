---
name: review-task-workflow
description: Execute PR review tasks systematically using reviewtask
---

You are tasked with executing PR review tasks systematically using the reviewtask tool. 

## Initial Setup (Execute Once Per Command Invocation):

**Fetch Latest Reviews**: Run `reviewtask` without arguments to fetch the latest PR reviews and generate/update tasks. This ensures you're working with the most current review feedback and tasks.

After completing the initial setup, follow this exact workflow:

## Workflow Steps:

1. **Check Status**: Use `reviewtask status` to check current task status and identify any tasks in progress
   - **If all tasks are completed (no todo, doing, or pending tasks remaining)**: Stop here - all work is done!
   - **If only pending tasks remain**: Review each pending task and decide action (see Step 2b)
   - **Continue only if todo, doing, or pending tasks exist**

2. **Identify Task**: 
   a) **Priority order**: Always work on tasks in this order:
      - **doing** tasks first (resume interrupted work)
      - **todo** tasks next (new work)
      - **pending** tasks last (blocked work requiring decision)
   
   b) **For doing tasks**: Continue with the task already in progress
   
   c) **For todo tasks**: 
      - First, evaluate if the task is needed using the **Task Classification Guidelines** below
      - If needed: Use `reviewtask show` to get the next recommended task, then run `reviewtask update <task-id> doing`
      - If duplicate/unnecessary: Update to `cancel`
      - If uncertain: Update to `pending`
   
   d) **For pending-only scenario**: 
      - List all pending tasks and their reasons for being blocked
      - For each pending task, decide:
        - `doing`: If you can now resolve the blocking issue
        - `todo`: If the task should be attempted again
        - `cancel`: If the task is no longer relevant or cannot be completed
      - Update task status: `reviewtask update <task-id> <new-status>`

3. **Verify Task Start**: Confirm the status change was successful before proceeding

4. **Execute Task**: Implement the required changes in the current branch based on the task description and original review comment

5. **Complete Task**: When implementation is finished:
   - Mark task as completed: `reviewtask update <task-id> done`
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

6. **Continue Workflow**: After committing:
   - Check status again with `reviewtask status`
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

## Important Notes:
- Work only in the current branch
- Always verify status changes before proceeding
- Include proper commit message format with task details and comment references
- **Task Priority**: Always work on `doing` tasks first, then `todo` tasks, then handle `pending` tasks
- **Pending Task Management**: Pending tasks must be resolved by changing status to `doing`, `todo`, or `cancel`
- **Efficient Task Management**: Classify duplicate/unnecessary tasks as `cancel` and uncertain tasks as `pending` to focus on actionable work
- Continue until all tasks are completed or no more actionable tasks remain
- The initial review fetch is executed only once per command invocation, not during the iterative workflow steps

Execute this workflow now.
