package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve [task-id]",
	Short: "Resolve GitHub review thread for task(s)",
	Long: `Resolve the GitHub review thread associated with a task or all done tasks.

This command manually resolves the GitHub review thread for a specific task or all done tasks.
This is useful when:
- Auto-resolve is disabled
- You want to resolve threads for completed tasks manually
- Review tools like Codex don't auto-resolve threads

Examples:
  reviewtask resolve abc123           # Resolve thread for task abc123
  reviewtask resolve abc123 --force   # Force resolve even if task is not done
  reviewtask resolve --all            # Resolve all done tasks
  reviewtask resolve --all --force    # Resolve all tasks regardless of status
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runResolve,
}

var (
	forceResolve bool
	resolveAll   bool
)

func init() {
	rootCmd.AddCommand(resolveCmd)
	resolveCmd.Flags().BoolVar(&forceResolve, "force", false, "Force resolve even if task is not done")
	resolveCmd.Flags().BoolVar(&resolveAll, "all", false, "Resolve all done tasks")
}

func runResolve(cmd *cobra.Command, args []string) error {
	// Validate arguments
	if resolveAll && len(args) > 0 {
		return fmt.Errorf("cannot specify task-id with --all flag")
	}
	if !resolveAll && len(args) == 0 {
		return fmt.Errorf("task-id is required (or use --all to resolve all done tasks)")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage manager
	storageManager := storage.NewManager()

	// Initialize GitHub client
	client, err := github.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	ctx := context.Background()

	if resolveAll {
		return resolveAllTasks(ctx, cfg, storageManager, client)
	}

	taskID := args[0]
	return resolveSingleTask(ctx, cfg, storageManager, client, taskID)
}

func resolveSingleTask(ctx context.Context, cfg *config.Config, storageManager *storage.Manager, client *github.Client, taskID string) error {
	// Get all tasks and find the specific one
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	var task *storage.Task
	for i := range allTasks {
		if allTasks[i].ID == taskID {
			task = &allTasks[i]
			break
		}
	}

	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Check task status
	if task.Status != "done" && !forceResolve {
		return fmt.Errorf("task is not done (status: %s). Use --force to resolve anyway", task.Status)
	}

	// Check if comment ID is available
	if task.SourceCommentID == 0 {
		return fmt.Errorf("task does not have a source comment ID (embedded comments are not supported)")
	}

	// Resolve the thread
	fmt.Fprintf(os.Stderr, "🔄 Resolving review thread for task '%s'...\n", taskID)
	if err := client.ResolveCommentThread(ctx, task.PRNumber, task.SourceCommentID); err != nil {
		return fmt.Errorf("failed to resolve thread: %w", err)
	}

	// Show success message
	fmt.Fprintf(os.Stderr, "✅ Thread resolved successfully!\n")
	fmt.Fprintf(os.Stderr, "✓ Review thread for task '%s' has been marked as resolved\n", taskID)
	fmt.Fprintf(os.Stderr, "💬 Comment URL: %s\n", task.URL)

	return nil
}

func resolveAllTasks(ctx context.Context, cfg *config.Config, storageManager *storage.Manager, client *github.Client) error {
	// Get all tasks
	tasks, err := storageManager.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	// Filter tasks to resolve
	var tasksToResolve []storage.Task
	for _, task := range tasks {
		// Skip if not done and not forcing
		if task.Status != "done" && !forceResolve {
			continue
		}

		// Skip if no comment ID
		if task.SourceCommentID == 0 {
			continue
		}

		tasksToResolve = append(tasksToResolve, task)
	}

	if len(tasksToResolve) == 0 {
		fmt.Fprintf(os.Stderr, "✅ No tasks to resolve\n")
		return nil
	}

	// Resolve all tasks
	fmt.Fprintf(os.Stderr, "🔄 Resolving %d review thread(s)...\n", len(tasksToResolve))

	resolved := 0
	failed := 0
	skipped := 0

	// Track unique comment IDs to avoid resolving the same thread multiple times
	resolvedComments := make(map[int64]bool)

	for i := range tasksToResolve {
		task := &tasksToResolve[i]

		// Skip if already resolved this comment
		if resolvedComments[task.SourceCommentID] {
			skipped++
			continue
		}

		if err := client.ResolveCommentThread(ctx, task.PRNumber, task.SourceCommentID); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ Failed to resolve task '%s': %v\n", task.ID, err)
			failed++
			continue
		}

		resolvedComments[task.SourceCommentID] = true
		resolved++

		descPreview := task.Description
		if len(descPreview) > 50 {
			descPreview = descPreview[:50] + "..."
		}
		fmt.Fprintf(os.Stderr, "  ✓ Resolved task '%s' (%s)\n", task.ID, descPreview)
	}

	// Show summary
	fmt.Fprintf(os.Stderr, "\n📊 Summary:\n")
	fmt.Fprintf(os.Stderr, "  ✓ Resolved: %d\n", resolved)
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "  ⊘ Skipped (already resolved): %d\n", skipped)
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "  ✗ Failed: %d\n", failed)
	}

	if failed > 0 {
		return fmt.Errorf("%d thread(s) failed to resolve", failed)
	}

	return nil
}
