package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gh-review-task/internal/ai"
	"gh-review-task/internal/config"
	"gh-review-task/internal/github"
	"gh-review-task/internal/setup"
	"gh-review-task/internal/storage"
	"github.com/spf13/cobra"
)

// Version information (set at build time)
var (
	appVersion    = "dev"
	appCommitHash = "unknown"
	appBuildDate  = "unknown"
)

// SetVersionInfo sets the version information from build-time variables
func SetVersionInfo(version, commitHash, buildDate string) {
	appVersion = version
	appCommitHash = commitHash
	appBuildDate = buildDate
}

var (
	refreshCache bool
)

var rootCmd = &cobra.Command{
	Use:   "gh-review-task [PR_NUMBER]",
	Short: "AI-powered PR review management tool",
	Long: `gh-review-task fetches GitHub Pull Request reviews, saves them locally,
and uses AI to analyze review content for task generation.

Examples:
  gh-review-task          # Check reviews for current branch's PR
  gh-review-task 123      # Check reviews for PR #123
  gh-review-task --refresh-cache  # Force refresh cache and reprocess all comments
  gh-review-task status   # Show current task status
  gh-review-task show     # Show current/next task details
  gh-review-task show <task-id>  # Show specific task details
  gh-review-task update <task-id> doing  # Update task status`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReviewTask,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&refreshCache, "refresh-cache", false, "Clear cache and reprocess all comments")
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(versionCmd)
}

func runReviewTask(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if initialization is needed
	if setup.ShouldPromptInit() {
		fmt.Println("🔧 This repository is not initialized for gh-review-task.")
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
			fmt.Println("  gh-review-task init")
			fmt.Println()
			return fmt.Errorf("repository not initialized")
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize GitHub client
	ghClient, err := github.NewClient()
	if err != nil {
		fmt.Println("✗ GitHub authentication required")
		fmt.Println()
		fmt.Println("To authenticate with GitHub, run:")
		fmt.Println("  gh-review-task auth login")
		fmt.Println()
		fmt.Println("Or set the GITHUB_TOKEN environment variable")
		return fmt.Errorf("authentication required")
	}

	// Initialize storage manager
	storageManager := storage.NewManager()

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
			return fmt.Errorf("failed to get PR for current branch: %w", err)
		}
	}

	fmt.Printf("Fetching reviews for PR #%d...\n", prNumber)

	// Fetch PR information
	prInfo, err := ghClient.GetPRInfo(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR info: %w", err)
	}

	// Fetch PR reviews
	reviews, err := ghClient.GetPRReviews(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR reviews: %w", err)
	}

	// Save PR info and reviews
	if err := storageManager.SavePRInfo(prNumber, prInfo); err != nil {
		return fmt.Errorf("failed to save PR info: %w", err)
	}

	if err := storageManager.SaveReviews(prNumber, reviews); err != nil {
		return fmt.Errorf("failed to save reviews: %w", err)
	}

	// Clear cache if refresh flag is set
	if refreshCache {
		fmt.Printf("🔄 Clearing cache for PR #%d...\n", prNumber)
		if err := storageManager.ClearCache(prNumber); err != nil {
			fmt.Printf("⚠️  Failed to clear cache: %v\n", err)
		}
	}

	// Generate tasks using AI with smart caching
	analyzer := ai.NewAnalyzer(cfg)
	var tasks []storage.Task
	if refreshCache {
		// Force reprocessing all comments when cache is refreshed
		tasks, err = analyzer.GenerateTasks(reviews)
	} else {
		// Use smart caching for normal operation
		tasks, err = analyzer.GenerateTasksWithCache(reviews, prNumber, storageManager)
	}
	if err != nil {
		return fmt.Errorf("failed to generate tasks: %w", err)
	}

	// Merge tasks with existing ones (preserves task statuses)
	if err := storageManager.MergeTasks(prNumber, tasks); err != nil {
		return fmt.Errorf("failed to merge tasks: %w", err)
	}

	fmt.Printf("✓ Saved PR info to .pr-review/PR-%d/info.json\n", prNumber)
	fmt.Printf("✓ Saved reviews to .pr-review/PR-%d/reviews.json\n", prNumber)
	fmt.Printf("✓ Generated %d tasks and saved to .pr-review/PR-%d/tasks.json\n", len(tasks), prNumber)

	return nil
}
