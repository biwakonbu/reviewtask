package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var cursorCmd = &cobra.Command{
	Use:   "cursor [TARGET]",
	Short: "Output command templates and integration files for Cursor IDE",
	Long: `Output command templates and integration files for Cursor IDE to provide seamless reviewtask workflows.

Available targets:
  pr-review    Output PR review workflow files to .cursor/ directory

Examples:
  reviewtask cursor pr-review    # Generate Cursor IDE integration files

The command generates:
  .cursor/commands/pr-review/     # Custom command templates
  .cursorrules                    # AI context rules for reviewtask`,
	Args: cobra.ExactArgs(1),
	RunE: runCursor,
}

func init() {
	// Command registration will be done in root.go
}

func runCursor(cmd *cobra.Command, args []string) error {
	target := args[0]

	switch target {
	case "pr-review":
		return outputCursorPRReviewFiles()
	default:
		return fmt.Errorf("unknown target: %s\n\nAvailable targets:\n  pr-review    Output PR review workflow files", target)
	}
}

func outputCursorPRReviewFiles() error {
	// Create the .cursor/commands directory
	commandDir := ".cursor/commands/pr-review"
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", commandDir, err)
	}

	// Generate .cursorrules file
	if err := generateCursorRules(); err != nil {
		return err
	}

	// Generate command templates
	if err := generateCursorCommands(); err != nil {
		return err
	}

	fmt.Println("✓ Created Cursor IDE integration files:")
	fmt.Println("  - .cursorrules (AI context rules)")
	fmt.Println("  - .cursor/commands/pr-review/ (workflow commands)")
	fmt.Println()
	fmt.Println("You can now use reviewtask seamlessly within Cursor IDE.")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("  1. Open Cursor IDE in your project")
	fmt.Println("  2. Use the command palette to run reviewtask commands")
	fmt.Println("  3. The AI assistant will understand reviewtask context")

	return nil
}

func generateCursorRules() error {
	cursorRules := `# ReviewTask Integration Rules for Cursor IDE

## About ReviewTask
This project uses reviewtask for managing GitHub PR review feedback. All review tasks are stored in the .pr-review/ directory and tracked systematically.

## Available Commands
- reviewtask status              # Show overview of all tasks
- reviewtask show                # Display current/next task details
- reviewtask show <task-id>      # Show specific task details
- reviewtask update <id> <status> # Update task status (todo/doing/done/pending)
- reviewtask fetch review <PR>   # Fetch reviews for a PR
- reviewtask                     # Generate tasks from fetched reviews

## Workflow Guidelines
When working on PR review tasks:
1. Always check task status before starting work: reviewtask status
2. Update task to "doing" when starting: reviewtask update <id> doing
3. Mark as "done" when completed: reviewtask update <id> done
4. Use "pending" for blocked tasks: reviewtask update <id> pending

## Task Priority Levels
- critical: Security issues, data exposure, authentication problems
- high: Performance issues, memory leaks, critical bugs
- medium: Functional bugs, logic improvements, error handling
- low: Code style, naming conventions, documentation

## Configuration
- Config file: .pr-review/config.json
- AI provider: auto (tries Cursor CLI first, then Claude)
- Model: auto (Cursor selects best model automatically)

## Best Practices
- Run reviewtask immediately after receiving PR reviews
- Keep task statuses updated in real-time
- Use reviewtask show to get full context for current task
- Never manually edit .pr-review/ files
- Re-run reviewtask when new review comments are added

## AI Assistance Context
When helping with reviewtask operations:
- Tasks have unique UUIDs for identification
- Each task links to its source GitHub comment
- Task descriptions are in the configured user language
- The tool preserves work progress when re-analyzing reviews
`

	// Write .cursorrules file
	if err := os.WriteFile(".cursorrules", []byte(cursorRules), 0644); err != nil {
		return fmt.Errorf("failed to write .cursorrules: %w", err)
	}

	return nil
}

func generateCursorCommands() error {
	// Create review workflow command
	reviewWorkflow := `# PR Review Workflow with ReviewTask

## Initial Setup
` + "```bash" + `
# Fetch reviews for your PR
reviewtask fetch review <PR_NUMBER>

# Generate tasks from reviews
reviewtask

# Check overall status
reviewtask status
` + "```" + `

## Working on Tasks
` + "```bash" + `
# Show next task to work on
reviewtask show

# Start working on a specific task
reviewtask update <TASK_ID> doing

# After completing the implementation
reviewtask update <TASK_ID> done

# If blocked on a task
reviewtask update <TASK_ID> pending
` + "```" + `

## Progress Monitoring
` + "```bash" + `
# View all tasks with progress bars
reviewtask status

# View wide format with more details
reviewtask status -w

# Show specific task details
reviewtask show <TASK_ID>
` + "```" + `

## Handling Updated Reviews
` + "```bash" + `
# When reviewers add new comments
reviewtask fetch review <PR_NUMBER>
reviewtask

# The tool automatically:
# - Preserves your work progress
# - Adds only new tasks
# - Maintains task status history
` + "```" + `

## Configuration
` + "```bash" + `
# View current configuration
reviewtask config show

# Set AI provider (cursor/claude/auto)
reviewtask config set ai_provider cursor

# Enable verbose mode for debugging
reviewtask config set verbose_mode true
` + "```" + `

## Tips
- Task IDs can be partial (first few characters of UUID)
- Use tab completion for command and task ID suggestions
- Run commands from repository root for best results
`

	// Write workflow template
	workflowPath := filepath.Join(".cursor/commands/pr-review", "review-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(reviewWorkflow), 0644); err != nil {
		return fmt.Errorf("failed to write workflow template: %w", err)
	}

	// Create quick reference command
	quickRef := `# ReviewTask Quick Reference

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
`

	// Write quick reference
	quickRefPath := filepath.Join(".cursor/commands/pr-review", "quick-reference.md")
	if err := os.WriteFile(quickRefPath, []byte(quickRef), 0644); err != nil {
		return fmt.Errorf("failed to write quick reference: %w", err)
	}

	return nil
}
