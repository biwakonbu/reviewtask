package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"reviewtask/internal/ai"
	"reviewtask/internal/config"
	"reviewtask/internal/github"
	"reviewtask/internal/storage"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug commands for testing specific phases",
	Long:  `Debug commands allow testing specific phases of the review task workflow.`,
}

var debugFetchCmd = &cobra.Command{
	Use:   "fetch [phase] [PR_NUMBER]",
	Short: "Debug fetch command phases",
	Long: `Debug specific phases of the fetch command:
  - review: Fetch and save PR reviews only
  - task: Generate tasks from saved reviews only

Examples:
  reviewtask debug fetch review 123    # Fetch reviews for PR #123
  reviewtask debug fetch task 123      # Generate tasks from saved reviews`,
	Args: cobra.ArbitraryArgs,
	RunE: runDebugFetch,
}

func init() {
	debugCmd.AddCommand(debugFetchCmd)
}

func runDebugFetch(cmd *cobra.Command, args []string) error {
	// Debug: Print all arguments
	fmt.Printf("DEBUG: Received %d arguments: %v\n", len(args), args)

	// Validate arguments manually
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("invalid number of arguments: expected 1 or 2, got %d", len(args))
	}

	phase := args[0]
	if phase != "review" && phase != "task" {
		return fmt.Errorf("invalid phase: %s (must be 'review' or 'task')", phase)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Enable verbose mode for debugging
	cfg.AISettings.VerboseMode = true

	// Initialize storage manager
	storageManager := storage.NewManager()

	// Determine PR number
	var prNumber int
	if len(args) > 1 {
		prNumber, err = strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[1])
		}
	} else {
		// Get PR number from current branch
		ghClient, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
		prNumber, err = ghClient.GetCurrentBranchPR(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get PR for current branch: %w", err)
		}
	}

	fmt.Printf("🔧 Debug mode: %s phase for PR #%d\n\n", phase, prNumber)

	switch phase {
	case "review":
		return debugFetchReviews(cfg, storageManager, prNumber)
	case "task":
		return debugGenerateTasks(cfg, storageManager, prNumber)
	}

	return nil
}

func debugFetchReviews(cfg *config.Config, storageManager *storage.Manager, prNumber int) error {
	ctx := context.Background()

	// Initialize GitHub client
	ghClient, err := github.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	fmt.Println("📥 Fetching PR information...")
	prInfo, err := ghClient.GetPRInfo(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR info: %w", err)
	}

	fmt.Println("📥 Fetching PR reviews and comments...")
	reviews, err := ghClient.GetPRReviews(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch PR reviews: %w", err)
	}

	// Save data
	fmt.Println("💾 Saving PR data...")
	if err := storageManager.SavePRInfo(prNumber, prInfo); err != nil {
		return fmt.Errorf("failed to save PR info: %w", err)
	}

	if err := storageManager.SaveReviews(prNumber, reviews); err != nil {
		return fmt.Errorf("failed to save reviews: %w", err)
	}

	// Show summary
	fmt.Printf("\n✅ Saved PR info to .pr-review/PR-%d/info.json\n", prNumber)
	fmt.Printf("✅ Saved reviews to .pr-review/PR-%d/reviews.json\n", prNumber)

	// Display review statistics
	totalComments := 0
	largestCommentSize := 0
	var largestCommentID int64

	for _, review := range reviews {
		if review.Body != "" {
			totalComments++
			if len(review.Body) > largestCommentSize {
				largestCommentSize = len(review.Body)
				largestCommentID = review.ID
			}
		}
		for _, comment := range review.Comments {
			totalComments++
			if len(comment.Body) > largestCommentSize {
				largestCommentSize = len(comment.Body)
				largestCommentID = comment.ID
			}
		}
	}

	fmt.Printf("\n📊 Review Statistics:\n")
	fmt.Printf("  - Total reviews: %d\n", len(reviews))
	fmt.Printf("  - Total comments: %d\n", totalComments)
	if largestCommentSize > 0 {
		fmt.Printf("  - Largest comment: ID %d (%d bytes)\n", largestCommentID, largestCommentSize)
		if largestCommentSize > 20000 {
			fmt.Printf("    ⚠️  This comment will be chunked during task generation\n")
		}
	}

	return nil
}

func debugGenerateTasks(cfg *config.Config, storageManager *storage.Manager, prNumber int) error {
	// Load saved reviews
	reviews, err := storageManager.LoadReviews(prNumber)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no saved reviews found for PR #%d. Run 'reviewtask debug fetch review %d' first", prNumber, prNumber)
		}
		return fmt.Errorf("failed to load reviews: %w", err)
	}

	fmt.Printf("📚 Loaded %d reviews from cache\n", len(reviews))

	// Generate tasks using AI
	fmt.Println("🤖 Analyzing reviews with AI...")
	analyzer := ai.NewAnalyzer(cfg)

	// Count total comments
	totalComments := 0
	for _, review := range reviews {
		if review.Body != "" {
			totalComments++
		}
		totalComments += len(review.Comments)
	}

	fmt.Printf("  - Total comments to analyze: %d\n", totalComments)

	// Use standard task generation (it will show verbose output)
	tasks, err := analyzer.GenerateTasks(reviews)
	if err != nil {
		// Check if it's a prompt size error
		if err.Error() != "" {
			fmt.Printf("\n❌ Error: %v\n", err)

			// If it's a prompt size error, show helpful information
			if len(err.Error()) > 0 {
				fmt.Println("\n💡 Debug Tips:")
				fmt.Println("  1. Check the error message above for details")
				fmt.Println("  2. Large comments should be automatically chunked")
				fmt.Println("  3. If chunking failed, check the comment structure")
			}
		}
		return fmt.Errorf("failed to generate tasks: %w", err)
	}

	// Save tasks
	if err := storageManager.SaveTasks(prNumber, tasks); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	fmt.Printf("\n✅ Generated %d tasks and saved to .pr-review/PR-%d/tasks.json\n", len(tasks), prNumber)

	// Show task summary
	if len(tasks) > 0 {
		fmt.Println("\n📋 Generated Tasks:")
		for i, task := range tasks {
			if i >= 5 {
				fmt.Printf("  ... and %d more tasks\n", len(tasks)-5)
				break
			}
			fmt.Printf("  %d. %s (Priority: %s)\n", i+1, task.Description, task.Priority)
		}
	}

	return nil
}
