package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/setup"
	"reviewtask/internal/storage"
	"reviewtask/internal/sync"
	"reviewtask/internal/version"

	"github.com/spf13/cobra"
)

// Version information (set at build time)
var (
	appVersion    = "dev"
	appCommitHash = "unknown"
	appBuildDate  = "unknown"
)

// osExit is a variable that holds the os.Exit function for testing purposes
var osExit = os.Exit

// SetVersionInfo sets the version information from build-time variables
func SetVersionInfo(version, commitHash, buildDate string) {
	appVersion = version
	appCommitHash = commitHash
	appBuildDate = buildDate
}

// NewRootCmd creates a new instance of the root command for testing
// This prevents shared state issues in concurrent tests
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reviewtask [PR_NUMBER]",
		Short: "AI-powered PR review management tool",
		Long: `reviewtask fetches GitHub Pull Request reviews, saves them locally,
and uses AI to analyze review content for task generation.

When called without subcommands, reviewtask runs the integrated workflow:
fetch → analyze → generate tasks with AI impact assessment.

Examples:
  reviewtask              # Analyze current branch's PR (integrated workflow)
  reviewtask 123          # Analyze PR #123 (integrated workflow)
  reviewtask status       # Show current task status
  reviewtask show         # Show current/next task details
  reviewtask done <id>    # Complete task with automation

Common Workflow:
  reviewtask              # 1. Fetch reviews and analyze
  reviewtask status       # 2. Check tasks
  reviewtask start <id>   # 3. Start working on a task
  reviewtask done <id>    # 4. Complete with full automation`,
		Args: cobra.MaximumNArgs(1),
		RunE: runReviewTask,
	}

	// Add all subcommands
	cmd.AddCommand(statusCmd)
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(startCmd)
	cmd.AddCommand(doneCmd)
	cmd.AddCommand(holdCmd)
	cmd.AddCommand(cancelCmd)
	cmd.AddCommand(verifyCmd)
	cmd.AddCommand(configCmd)
	cmd.AddCommand(showCmd)
	cmd.AddCommand(statsCmd)
	cmd.AddCommand(versionCmd)
	cmd.AddCommand(versionsCmd)
	cmd.AddCommand(authCmd)
	cmd.AddCommand(debugCmd)
	cmd.AddCommand(initCmd)
	cmd.AddCommand(claudeCmd)
	cmd.AddCommand(cursorCmd)
	cmd.AddCommand(promptCmd)

	return cmd
}

var rootCmd = &cobra.Command{
	Use:   "reviewtask [PR_NUMBER]",
	Short: "AI-powered PR review management tool",
	Long: `reviewtask fetches GitHub Pull Request reviews, saves them locally,
and uses AI to analyze review content for task generation.

When called without subcommands, reviewtask runs the integrated workflow:
fetch → analyze → generate tasks with AI impact assessment.

Examples:
  reviewtask              # Analyze current branch's PR (integrated workflow)
  reviewtask 123          # Analyze PR #123 (integrated workflow)
  reviewtask status       # Show current task status
  reviewtask show         # Show current/next task details
  reviewtask done <id>    # Complete task with automation

Common Workflow:
  reviewtask              # 1. Fetch reviews and analyze
  reviewtask status       # 2. Check tasks
  reviewtask start <id>   # 3. Start working on a task
  reviewtask done <id>    # 4. Complete with full automation`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReviewTask,
}

func Execute() error {
	return rootCmd.Execute()
}

// Injectable factory for GitHub client (for tests)
var newGitHubClient = github.NewClient

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(claudeCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(cursorCmd)
	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(holdCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(promptCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(versionsCmd)
}

func runReviewTask(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if initialization is needed
	if setup.ShouldPromptInit() {
		fmt.Println("🔧 This repository is not initialized for reviewtask.")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Initialize now? (Y/n): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			fmt.Println()
			if err := runInit(cmd, args); err != nil {
				return err
			}
			fmt.Println("Now continuing with your request...")
			fmt.Println()
		} else {
			fmt.Println()
			fmt.Println("To initialize later, run:")
			fmt.Println("  reviewtask init")
			fmt.Println()
			return fmt.Errorf("repository not initialized")
		}
	}

	// Load configuration and display AI provider
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	DisplayAIProvider(cfg)

	// Check for updates if enabled and needed
	checkForUpdatesAsync(cfg)

	// Initialize GitHub client
	ghClient, err := newGitHubClient()
	if err != nil {
		fmt.Println("✗ GitHub authentication required")
		fmt.Println()
		fmt.Println("To authenticate with GitHub, run:")
		fmt.Println("  reviewtask auth login")
		fmt.Println()
		fmt.Println("Or set the GITHUB_TOKEN environment variable")
		return fmt.Errorf("authentication required")
	}

	// Initialize storage manager
	storageManager := storage.NewManager()

	// Auto-cleanup closed PRs
	fmt.Println("Checking for closed PRs to clean up...")
	if err := storageManager.CleanupClosedPRs(func(prNumber int) (bool, error) {
		return ghClient.IsPROpen(ctx, prNumber)
	}); err != nil {
		// Don't fail the command if cleanup fails, just warn
		fmt.Printf("Warning: Failed to cleanup closed PRs: %v\n", err)
	}

	// Determine PR number
	var prNumber int
	if len(args) > 0 {
		prNumber, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}
	} else {
		// Get PR number from current branch
		prNumber, err = ghClient.GetCurrentBranchPR(ctx)
		if err != nil {
			// Check if it's a "no PR found" error
			if errors.Is(err, github.ErrNoPRFound) {
				// Get current branch name for helpful message
				currentBranch, branchErr := storageManager.GetCurrentBranch()
				if branchErr != nil {
					currentBranch = "current branch"
				}

				fmt.Printf("No pull request found for branch '%s'.\n\n", currentBranch)
				fmt.Println("To use reviewtask, you need to:")
				fmt.Println("1. Create a pull request for your branch:")
				fmt.Printf("   gh pr create --title \"Your PR title\" --body \"Your PR description\"\n\n")
				fmt.Println("2. Or specify a PR number directly:")
				fmt.Println("   reviewtask <PR_NUMBER>")
				fmt.Println("\nFor more information, run: reviewtask --help")
				// Exit gracefully - this is not an error condition
				osExit(0)
			}
			return fmt.Errorf("failed to get PR for current branch: %w", err)
		}
	}

	// Simple status messages
	fmt.Printf("Fetching reviews for PR #%d...\n", prNumber)

	// Fetch PR information
	fmt.Println("  Fetching PR information...")
	prInfo, err := ghClient.GetPRInfo(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR info: %w", err)
	}

	// Fetch PR reviews
	fmt.Println("  Fetching PR reviews and comments...")
	reviews, err := ghClient.GetPRReviews(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR reviews: %w", err)
	}

	// Check if self-reviews should be processed
	if cfg.AISettings.ProcessSelfReviews {
		fmt.Println("  Fetching self-review comments...")
		selfReviews, err := ghClient.GetSelfReviews(ctx, prNumber, prInfo.Author)
		if err != nil {
			// Log warning but don't fail the entire process
			fmt.Printf("  ⚠️  Warning: failed to fetch self-reviews: %v\n", err)
		} else if len(selfReviews) > 0 {
			// Merge self-reviews with external reviews
			reviews = append(reviews, selfReviews...)
			fmt.Printf("  Found %d self-review comments\n", len(selfReviews[0].Comments))
		}
	}

	// Fetch thread resolution state for all comments using batch API
	fmt.Println("  Checking thread resolution state...")

	// Use batch fetch for better performance (Issue #222)
	threadStates, err := ghClient.GetAllThreadStates(ctx, prNumber)
	if err != nil {
		fmt.Printf("  ⚠️  Warning: failed to fetch thread states: %v\n", err)
		// Continue without thread state information
		threadStates = make(map[int64]bool)
	}

	// Update resolution state for all comments
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	unresolvedCount := 0
	for i := range reviews {
		for j := range reviews[i].Comments {
			comment := &reviews[i].Comments[j]
			// Skip comments without ID (e.g., embedded Codex comments)
			if comment.ID == 0 {
				continue
			}

			// Get resolution state from batch result
			if isResolved, exists := threadStates[comment.ID]; exists {
				comment.GitHubThreadResolved = isResolved
				comment.LastCheckedAt = now

				if !isResolved {
					unresolvedCount++
				}
			}
		}
	}

	if unresolvedCount > 0 {
		fmt.Printf("  Found %d unresolved comment threads\n", unresolvedCount)
	}

	// Reconcile local task state with GitHub thread state
	fmt.Println("  Reconciling local task state with GitHub...")
	reconciler := sync.NewReconciler(ghClient, storageManager)
	reconcileResult, err := reconciler.ReconcileWithGitHub(ctx, prNumber, reviews)
	if err != nil {
		fmt.Printf("  ⚠️  Warning: failed to reconcile with GitHub: %v\n", err)
	} else {
		// Show reconciliation results
		if reconcileResult.LocalTasksNeedingResolve > 0 {
			fmt.Printf("  ✓ Resolved %d threads that were completed locally\n", len(reconcileResult.ResolvedThreads))
		}

		if reconcileResult.CancelTasksWithoutReply > 0 {
			fmt.Printf("  ⚠️  Found %d cancelled tasks without reply comments\n", reconcileResult.CancelTasksWithoutReply)
		}

		// Show warnings if any
		if len(reconcileResult.Warnings) > 0 {
			for _, warning := range reconcileResult.Warnings {
				fmt.Printf("  %s\n", warning)
			}
		}
	}

	// Save PR info and reviews
	fmt.Println("  Saving data...")
	if err := storageManager.SavePRInfo(prNumber, prInfo); err != nil {
		return fmt.Errorf("failed to save PR info: %w", err)
	}

	if err := storageManager.SaveReviews(prNumber, reviews); err != nil {
		return fmt.Errorf("failed to save reviews: %w", err)
	}

	// Calculate total comments for information
	totalComments := 0
	for _, review := range reviews {
		if review.Body != "" {
			totalComments++
		}
		totalComments += len(review.Comments)
	}

	// Show final results
	fmt.Printf("\n✓ Saved PR info to .pr-review/PR-%d/info.json\n", prNumber)
	fmt.Printf("✓ Saved reviews to .pr-review/PR-%d/reviews.json\n", prNumber)

	if totalComments > 0 {
		fmt.Printf("📊 Found %d comments ready for analysis\n", totalComments)
		fmt.Println()

		// Automatically run analysis (integrated workflow)
		fmt.Println("🔄 Analyzing reviews and generating tasks...")
		fmt.Println()

		// Call the analyze function directly
		return runAnalyzeIntegrated(cmd, prNumber, cfg)
	} else {
		fmt.Printf("ℹ️  No comments found in the reviews\n")
	}

	return nil
}

// runAnalyzeIntegrated runs the analysis workflow automatically after fetch
func runAnalyzeIntegrated(cmd *cobra.Command, prNumber int, cfg *config.Config) error {
	// Import AI package
	analyzer := ai.NewAnalyzer(cfg)
	storageManager := storage.NewManager()

	// Load saved reviews
	reviews, err := storageManager.LoadReviews(prNumber)
	if err != nil {
		return fmt.Errorf("failed to load reviews: %w", err)
	}

	// Generate tasks with realtime saving
	_, err = analyzer.GenerateTasksWithRealtimeSaving(reviews, prNumber, storageManager)
	if err != nil {
		return fmt.Errorf("failed to generate tasks: %w", err)
	}

	return nil
}

// getReviewerName extracts the reviewer name for a given comment index
func getReviewerName(reviews []github.Review, commentIndex int) string {
	currentIndex := 0
	for _, review := range reviews {
		if review.Body != "" {
			if currentIndex == commentIndex {
				return review.Reviewer
			}
			currentIndex++
		}
		for _, comment := range review.Comments {
			if currentIndex == commentIndex {
				return comment.Author
			}
			currentIndex++
		}
	}
	return "reviewer"
}

// checkForUpdatesAsync performs update check in background if needed
func checkForUpdatesAsync(cfg *config.Config) {
	if !version.ShouldCheckForUpdates(cfg.UpdateCheck.Enabled, cfg.UpdateCheck.IntervalHours, cfg.UpdateCheck.LastCheck) {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		checker := version.NewChecker(0)
		notification, err := checker.CheckAndNotify(ctx, appVersion, cfg.UpdateCheck.NotifyPrereleases)
		if err != nil {
			// Silently fail - don't interrupt user workflow
			return
		}

		// Update last check time
		cfg.UpdateCheck.LastCheck = time.Now()
		_ = cfg.Save() // Ignore error - not critical

		// Show notification if available
		if notification != "" {
			fmt.Println()
			fmt.Printf("💡 %s\n", notification)
			fmt.Println()
		}
	}()
}
