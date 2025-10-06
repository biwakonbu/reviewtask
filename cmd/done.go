package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"reviewtask/internal/config"
	"reviewtask/internal/git"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
	"reviewtask/internal/tasks"
	"reviewtask/internal/threads"
	"reviewtask/internal/verification"
)

var doneCmd = &cobra.Command{
	Use:   "done <task-id>",
	Short: "Mark a task as \"done\" with automated workflow",
	Long: `Complete a task with an automated workflow that includes:
1. Verification (build/test/lint)
2. Auto-commit with structured message
3. Thread resolution (when all comment tasks complete)
4. Next task suggestion

This is the recommended way to mark tasks as complete, as it ensures
proper verification and maintains a clean git history.

Examples:
  reviewtask done task-1
  reviewtask done abc123
  reviewtask done abc123 --skip-verification
  reviewtask done abc123 --skip-commit`,
	Args: cobra.ExactArgs(1),
	RunE: runDone,
}

var (
	doneSkipVerification bool
	doneSkipCommit       bool
	doneSkipResolve      bool
	doneSkipSuggestion   bool
)

func init() {
	doneCmd.Flags().BoolVar(&doneSkipVerification, "skip-verification", false, "Skip verification checks")
	doneCmd.Flags().BoolVar(&doneSkipCommit, "skip-commit", false, "Skip automatic commit")
	doneCmd.Flags().BoolVar(&doneSkipResolve, "skip-resolve", false, "Skip thread resolution")
	doneCmd.Flags().BoolVar(&doneSkipSuggestion, "skip-suggestion", false, "Skip next task suggestion")
}

func runDone(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Display AI provider info
	_, err = DisplayAIProviderIfNeeded()
	if err != nil {
		// Continue without AI provider display
	}

	// Initialize storage
	storageManager := storage.NewManager()

	// Get task
	task, err := getTaskByID(storageManager, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	fmt.Printf("\n")
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Task Completion Workflow")
	fmt.Println("════════════════════════════════════════")
	fmt.Printf("Task: %s\n", task.Description)
	fmt.Printf("\n")

	// Phase 1: Verification
	if cfg.DoneWorkflow.EnableVerification && !doneSkipVerification {
		if err := runVerificationPhase(task); err != nil {
			return err
		}
	} else {
		fmt.Println("⏭️  Skipping verification")
	}

	// Update task status to done
	if err := storageManager.UpdateTaskStatus(taskID, "done"); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Reload task with updated status
	task, err = getTaskByID(storageManager, taskID)
	if err != nil {
		return fmt.Errorf("failed to reload task: %w", err)
	}

	// Phase 2: Auto-commit
	if cfg.DoneWorkflow.EnableAutoCommit && !doneSkipCommit {
		if err := runAutoCommitPhase(task); err != nil {
			fmt.Printf("⚠️  Auto-commit failed: %v\n", err)
			fmt.Println("   (Task is still marked as done)")
		}
	} else {
		fmt.Println("⏭️  Skipping auto-commit")
	}

	// Phase 3: Thread resolution
	if cfg.DoneWorkflow.EnableAutoResolve != "disabled" && !doneSkipResolve {
		if err := runThreadResolutionPhase(cfg, storageManager, task); err != nil {
			fmt.Printf("⚠️  Thread resolution failed: %v\n", err)
		}
	} else {
		fmt.Println("⏭️  Skipping thread resolution")
	}

	// Phase 4: Next task suggestion
	if cfg.DoneWorkflow.EnableNextTaskSuggestion && !doneSkipSuggestion {
		runNextTaskSuggestionPhase(storageManager, taskID)
	}

	fmt.Printf("\n")
	fmt.Println("════════════════════════════════════════")
	fmt.Printf("✅ Task %s completed\n", formatTaskID(taskID))
	fmt.Println("════════════════════════════════════════")
	fmt.Printf("\n")

	return nil
}

func runVerificationPhase(task *storage.Task) error {
	fmt.Println("──────────────────────────────────────")
	fmt.Println("Verifying task completion...")
	fmt.Println("──────────────────────────────────────")

	verifier, err := verification.NewVerifier()
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	results, err := verifier.VerifyTask(task.ID)
	if err != nil {
		fmt.Printf("❌ Verification failed: %v\n", err)
		return err
	}

	// Check results
	allPassed := true
	titleCaser := cases.Title(language.English)
	for _, result := range results {
		if result.Success {
			fmt.Printf("  ✓ %s passed\n", titleCaser.String(string(result.Type)))
		} else {
			fmt.Printf("  ✗ %s failed\n", titleCaser.String(string(result.Type)))
			if result.Message != "" {
				fmt.Printf("    %s\n", result.Message)
			}
			allPassed = false
		}
	}

	if !allPassed {
		fmt.Println("❌ Verification checks failed")
		return fmt.Errorf("verification failed")
	}

	fmt.Println()
	return nil
}

func runAutoCommitPhase(task *storage.Task) error {
	fmt.Println("──────────────────────────────────────")
	fmt.Println("Creating commit...")
	fmt.Println("──────────────────────────────────────")

	committer, err := git.NewGitCommitter()
	if err != nil {
		return fmt.Errorf("failed to create git committer: %w", err)
	}

	// Check for staged changes
	checker := git.NewStagingChecker()
	hasStagedChanges, err := checker.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	if !hasStagedChanges {
		fmt.Println("  ⚠️  No staged changes to commit")
		fmt.Println()
		return nil
	}

	result, err := committer.CreateCommitForTask(task)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	if result.Success {
		fmt.Printf("  ✓ Created commit %s\n", result.CommitHash)
	} else {
		fmt.Printf("  ⚠️  %s\n", result.Message)
	}

	fmt.Println()
	return nil
}

func runThreadResolutionPhase(cfg *config.Config, storageManager *storage.Manager, task *storage.Task) error {
	fmt.Println("──────────────────────────────────────")
	fmt.Println("Resolving review thread...")
	fmt.Println("──────────────────────────────────────")

	// Get repository info
	owner, repo, err := getGitHubRepoInfo()
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w", err)
	}

	// Get GitHub token
	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Create GraphQL client
	graphqlClient := github.NewGraphQLClient(token)

	// Create thread resolver
	resolver := threads.NewThreadResolver(cfg, storageManager, graphqlClient)

	// Check if thread should be resolved
	status, err := resolver.GetResolutionStatus(context.Background(), task)
	if err != nil {
		return fmt.Errorf("failed to get resolution status: %w", err)
	}

	if !status.ThreadResolved {
		fmt.Printf("  ⏸️  %s\n", status.Message)
		if status.RemainingTasks > 0 {
			fmt.Printf("     (%d of %d tasks complete)\n", status.CompletedTasks, status.TotalTasks)
		}
		fmt.Println()
		return nil
	}

	// Resolve thread
	ctx := context.Background()
	if err := resolver.ResolveThreadForTask(ctx, task, owner, repo); err != nil {
		return fmt.Errorf("failed to resolve thread: %w", err)
	}

	fmt.Println("  ✓ Thread resolved (all tasks from this comment completed)")
	fmt.Println()
	return nil
}

func runNextTaskSuggestionPhase(storageManager *storage.Manager, completedTaskID string) {
	fmt.Println("──────────────────────────────────────")
	fmt.Println("Next Steps")
	fmt.Println("──────────────────────────────────────")

	recommender := tasks.NewTaskRecommender(storageManager)
	recommendation, err := recommender.GetRecommendationAfterCompletion(completedTaskID)
	if err != nil {
		fmt.Printf("⚠️  Failed to get next task recommendation: %v\n", err)
		return
	}

	if recommendation.AllComplete {
		fmt.Println("  🎉 " + recommendation.Message)
		fmt.Println()
		return
	}

	if !recommendation.HasNext {
		fmt.Println("  ℹ️  No more tasks available")
		fmt.Println()
		return
	}

	// Display progress
	fmt.Printf("\n")
	fmt.Println("Progress")
	fmt.Println("────────")
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		fmt.Printf("⚠️  Failed to calculate progress: %v\n", err)
		fmt.Println()
		return
	}
	totalTasks := len(allTasks)
	completedTasks := 0
	for _, t := range allTasks {
		if t.Status == "done" {
			completedTasks++
		}
	}
	percentage := 0
	if totalTasks > 0 {
		percentage = (completedTasks * 100) / totalTasks
	}
	fmt.Printf("%d of %d tasks complete (%d%%)\n", completedTasks, totalTasks, percentage)

	// Display next task
	fmt.Printf("\n")
	fmt.Println("Next Task")
	fmt.Println("─────────")
	fmt.Printf("%-8s %-8s %s\n", formatTaskID(recommendation.NextTask.ID),
		strings.ToUpper(recommendation.NextTask.Priority),
		recommendation.NextTask.Description)

	// Display next steps
	fmt.Printf("\n")
	fmt.Println("Next Steps")
	fmt.Println("──────────")
	fmt.Println("→ Continue with next task")
	fmt.Printf("  reviewtask show %s\n", recommendation.NextTask.ID)
	fmt.Println()
	fmt.Println("→ Start immediately")
	fmt.Printf("  reviewtask start %s\n", recommendation.NextTask.ID)
	fmt.Println()
}

func getTaskByID(storageManager *storage.Manager, taskID string) (*storage.Task, error) {
	allTasks, err := storageManager.GetAllTasks()
	if err != nil {
		return nil, err
	}

	for i := range allTasks {
		if allTasks[i].ID == taskID || strings.HasPrefix(allTasks[i].ID, taskID) {
			return &allTasks[i], nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

func getGitHubRepoInfo() (owner, repo string, err error) {
	// Use the provider to get repo info
	provider := &github.DefaultRepoInfoProvider{}
	return provider.GetRepoInfo()
}

func formatTaskID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// hasStagedChanges is kept for compatibility
func hasStagedChanges() (bool, error) {
	checker := git.NewStagingChecker()
	return checker.HasStagedChanges()
}
