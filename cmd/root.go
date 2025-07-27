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

	"github.com/spf13/cobra"
	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/setup"
	"reviewtask/internal/storage"
	"reviewtask/internal/version"
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
		Use:   "reviewtask",
		Short: "AI-powered PR review management tool",
		Long: `reviewtask fetches GitHub Pull Request reviews, saves them locally,
and uses AI to analyze review content for task generation.

Examples:
  reviewtask fetch        # Check reviews for current branch's PR
  reviewtask fetch 123    # Check reviews for PR #123
  reviewtask status       # Show current task status
  reviewtask show         # Show current/next task details
  reviewtask show <task-id>  # Show specific task details
  reviewtask update <task-id> doing  # Update task status`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add all subcommands
	cmd.AddCommand(fetchCmd)
	cmd.AddCommand(statusCmd)
	cmd.AddCommand(updateCmd)
	cmd.AddCommand(showCmd)
	cmd.AddCommand(statsCmd)
	cmd.AddCommand(versionCmd)
	cmd.AddCommand(versionsCmd)
	cmd.AddCommand(authCmd)
	cmd.AddCommand(initCmd)
	cmd.AddCommand(claudeCmd)
	cmd.AddCommand(promptCmd)

	return cmd
}

var rootCmd = &cobra.Command{
	Use:   "reviewtask",
	Short: "AI-powered PR review management tool",
	Long: `reviewtask fetches GitHub Pull Request reviews, saves them locally,
and uses AI to analyze review content for task generation.

Examples:
  reviewtask fetch        # Check reviews for current branch's PR
  reviewtask fetch 123    # Check reviews for PR #123
  reviewtask status       # Show current task status
  reviewtask show         # Show current/next task details
  reviewtask show <task-id>  # Show specific task details
  reviewtask update <task-id> doing  # Update task status`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(claudeCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(promptCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(updateCmd)
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

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check for updates if enabled and needed
	checkForUpdatesAsync(cfg)

	// Initialize GitHub client
	ghClient, err := github.NewClient()
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

	// Save PR info and reviews
	fmt.Println("  Saving data...")
	if err := storageManager.SavePRInfo(prNumber, prInfo); err != nil {
		return fmt.Errorf("failed to save PR info: %w", err)
	}

	if err := storageManager.SaveReviews(prNumber, reviews); err != nil {
		return fmt.Errorf("failed to save reviews: %w", err)
	}

	// Generate tasks using AI - always use optimized processing
	fmt.Println("  Analyzing reviews with AI...")
	analyzer := ai.NewAnalyzer(cfg)

	// Calculate optimal batch size based on number of comments
	totalComments := 0
	for _, review := range reviews {
		if review.Body != "" {
			totalComments++
		}
		totalComments += len(review.Comments)
	}

	// Show total comments count
	if totalComments > 0 {
		fmt.Printf("  Found %d comments to analyze\n", totalComments)
	}

	// Auto-detect batch size: smaller batches for large PRs
	batchSize := 10 // Default for normal PRs
	if totalComments > 50 {
		batchSize = 20 // Larger batches for big PRs
	} else if totalComments < 10 {
		batchSize = totalComments // Process all at once for small PRs
	}

	// Always use incremental processing for better performance
	incrementalOpts := ai.IncrementalOptions{
		BatchSize:    batchSize,
		Resume:       true, // Always support resume in case of failure
		FastMode:     false,
		MaxTimeout:   10 * time.Minute, // Generous timeout
		ShowProgress: true,
		OnProgress: func(processed, total int) {
			// Simple progress indicator
			if processed%10 == 0 || processed == total {
				fmt.Printf("  Processed %d/%d comments\r", processed, total)
			}
		},
	}

	tasks, err := analyzer.GenerateTasksIncremental(reviews, prNumber, storageManager, incrementalOpts)
	if err != nil {
		// Check if it's a timeout error and suggest retry
		if strings.Contains(err.Error(), "timed out") {
			fmt.Println("\n⚠️  Processing timed out. Run the command again to resume from where it left off.")
		}
		return fmt.Errorf("failed to generate tasks: %w", err)
	}
	fmt.Println() // New line after progress

	// Merge tasks with existing ones (preserves task statuses)
	if err := storageManager.MergeTasks(prNumber, tasks); err != nil {
		return fmt.Errorf("failed to merge tasks: %w", err)
	}

	// Show final results
	fmt.Printf("\n✓ Saved PR info to .pr-review/PR-%d/info.json\n", prNumber)
	fmt.Printf("✓ Saved reviews to .pr-review/PR-%d/reviews.json\n", prNumber)
	fmt.Printf("✓ Generated %d tasks and saved to .pr-review/PR-%d/tasks.json\n", len(tasks), prNumber)

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
